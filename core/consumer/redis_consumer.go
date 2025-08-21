package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vxmsgpush/config"
	"vxmsgpush/core/db"
	"vxmsgpush/core/vxmsg"
	"vxmsgpush/logger"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type RedisTemplateMessage struct {
	Mobile      string                 `json:"mobile"`
	TemplateID  string                 `json:"template_id"`
	URL         string                 `json:"url"`
	Data        map[string]interface{} `json:"data"`
	MiniProgram *vxmsg.MiniProgram     `json:"miniprogram,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"` // 重试次数
	AppID       string                 `json:"appid,omitempty"`       // 用来存 Header 的 AppID
}

var ctx = context.Background()

const (
	maxRetryCount     = 5                       // 最大重试次数
	deadLetterQueue   = "wx_template_msg_dlq"   // 死信队列
	delayQueue        = "wx_template_msg_delay" // 延迟队列（ZSet）
	delayStepSeconds  = 3                       // 每次重试递增秒数
	sendRatePerSecond = 200                     // 限制发送频率， 条/秒
)

// StartRedisConsumers 启动 dispatcher 和多个 worker
func StartRedisConsumers(rdb *redis.Client, queueName string, dispatcherCount, workerCount int, chanBuffer int) {
	msgChan := make(chan string, chanBuffer) // 可调缓冲区大小

	// 创建限流器（每秒 sendRatePerSecond 个请求，突发容量为1）
	limiter := rate.NewLimiter(rate.Limit(sendRatePerSecond), 5)

	// 启动多个 dispatcher 负责从 Redis 读取消息，放入 msgChan
	for i := 0; i < dispatcherCount; i++ {
		go func(id int) {
			for {
				result, err := rdb.BRPop(ctx, 5*time.Second, queueName).Result()
				if err == redis.Nil || len(result) < 2 {
					continue
				}
				if err != nil {
					logger.Errorf("[dispatcher-%d] Redis BRPop 错误: %v", id, err)
					time.Sleep(time.Second)
					continue
				}

				raw := result[1]

				// 阻塞等待，直到有空闲 worker 从 chan 读取
				msgChan <- raw
			}
		}(i + 1)
	}

	// 启动 worker 数量，负责处理消息
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			for raw := range msgChan {
				// 等待限流器令牌，控制速率
				if err := limiter.Wait(ctx); err != nil {
					logger.Errorf("[worker-%d] 限流等待失败: %v", id, err)
					continue
				}
				processMessage(rdb, raw, id)
			}
		}(i + 1)
	}

	logger.Infof("[redis] 启动 %d 个 dispatcher + %d 个 worker，队列 %s，chan 缓冲 %d", dispatcherCount, workerCount, queueName, chanBuffer)
}

// 启动延迟队列调度器（定时扫描）
func StartRetryScheduler(rdb *redis.Client, delayQueue, mainQueue string, batchSize int) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := float64(time.Now().Unix())
			// 批量获取到期消息
			msgs, err := rdb.ZRangeByScore(ctx, delayQueue, &redis.ZRangeBy{
				Min:   "0",
				Max:   fmt.Sprintf("%f", now),
				Count: int64(batchSize),
			}).Result()
			if err != nil {
				logger.Errorf("[scheduler] 获取延迟消息失败: %v", err)
				continue
			}
			if len(msgs) == 0 {
				continue
			}

			var successful []interface{} // 成功投递到主队列的消息，用于批量删除

			for _, raw := range msgs {
				if err := rdb.RPush(ctx, mainQueue, raw).Err(); err != nil {
					logger.Errorf("[scheduler] 消息重投失败: %v，内容: %s", err, raw)
					continue
				}
				successful = append(successful, raw)
			}

			// 批量从延迟队列删除已经成功投递的消息
			if len(successful) > 0 {
				if _, err := rdb.ZRem(ctx, delayQueue, successful...).Result(); err != nil {
					logger.Errorf("[scheduler] 删除延迟队列已投递消息失败: %v", err)
				} else {
					logger.Infof("[scheduler] 成功将 %d 条延迟消息投递到主队列", len(successful))
				}
			}
		}
	}()
}

func processMessage(rdb *redis.Client, raw string, id int) {
	var msg RedisTemplateMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		logger.Errorf("[worker-%d] JSON 解析失败: %v，内容: %s", id, err, raw)
		AddFailWithReason("invalid_json", "") // 无法解析时没有 AppID
		return
	}

	if config.IsMobileBlocked(msg.Mobile) || !config.IsMobileAllowed(msg.Mobile) {
		logger.Warnf("[worker-%d] 手机号 %s 被过滤，跳过", id, msg.Mobile)
		return
	}

	openid, err := vxmsg.GetUserOpenIDByMobile(msg.Mobile)
	if err != nil {
		logger.Errorf("[worker-%d] 获取 OpenID 失败: %v", id, err)
		AddFailWithReason("geterror_openid", msg.AppID)
		// 仅在 openid 非空时更新统计

		_ = db.UpdateUserSendStatWithAppID(msg.Mobile, openid, msg.AppID, false)
		if err := db.UpdatePushStatWithAppID(time.Now(), false, msg.AppID); err != nil {
			logger.Warnf("[worker-%d] push_stat 更新失败: %v", id, err)
		}

		return
	}

	tpl := vxmsg.TemplateMsg{
		ToUser:      openid,
		TemplateID:  msg.TemplateID,
		URL:         msg.URL,
		Data:        msg.Data,
		MiniProgram: msg.MiniProgram,
	}

	err = vxmsg.SendTemplateMsg(tpl)
	if err != nil {
		msg.RetryCount++
		if we, ok := err.(*vxmsg.WechatError); ok {
			logger.Errorf("[worker-%d] 微信发送失败 errcode=%d errmsg=%s", id, we.ErrCode, we.ErrMsg)
			if msg.RetryCount == 1 {
				_ = db.UpdateUserSendStatWithAppID(msg.Mobile, openid, msg.AppID, false)
				if err := db.UpdatePushStatWithAppID(time.Now(), false, msg.AppID); err != nil {
					logger.Warnf("[worker-%d] push_stat 更新失败: %v", id, err)
				}
				switch we.ErrCode {
				case 40003:
					AddFailWithReason("invalid_openid", msg.AppID)
				case 43004:
					AddFailWithReason("user_not_followed", msg.AppID)
				case 42001:
					AddFailWithReason("token_expired", msg.AppID)
				default:
					AddFailWithReason(fmt.Sprintf("wx_%d", we.ErrCode), msg.AppID)
				}
			}
		} else {
			logger.Errorf("[worker-%d] 模板消息发送失败: %v", id, err)
			if msg.RetryCount == 1 {
				AddFailWithReason("send_error", msg.AppID)
			}
		}

		if msg.RetryCount > maxRetryCount {
			bs, _ := json.Marshal(msg)
			if err := rdb.RPush(ctx, deadLetterQueue, bs).Err(); err != nil {
				logger.Errorf("[worker-%d] 死信入队失败: %v", id, err)
			} else {
				logger.Warnf("[worker-%d] 消息进入死信队列: %s", id, string(bs))
			}
			return
		}

		bs, _ := json.Marshal(msg)
		delay := msg.RetryCount * delayStepSeconds
		score := float64(time.Now().Add(time.Duration(delay) * time.Second).Unix())
		if err := rdb.ZAdd(ctx, delayQueue, redis.Z{Score: score, Member: bs}).Err(); err != nil {
			logger.Errorf("[worker-%d] 延迟入队失败: %v", id, err)
		} else {
			logger.Infof("[worker-%d] 延迟消息入队，%ds 后重试，第 %d 次", id, delay, msg.RetryCount)
		}
		return
	}

	// 发送成功
	AddSuccess()

	// 更新 push_stat 表
	if err := db.UpdatePushStatWithAppID(time.Now(), true, msg.AppID); err != nil {
		logger.Warnf("[worker-%d] push_stat 更新失败: %v", id, err)
	}

	if err := db.UpdateUserSendStatWithAppID(msg.Mobile, openid, msg.AppID, true); err != nil {
		logger.Warnf("[worker-%d] 成功统计更新失败: %v", id, err)
	}

	// 更新 OpenID
	if err := db.UpdateUserOpenIDWithAppID(msg.Mobile, openid, msg.AppID); err != nil {
		logger.Warnf("[worker-%d] 更新openid失败: %v", id, err)
	}
	logger.Infof("[worker-%d] 模板消息发送成功: %s", id, openid)

}
