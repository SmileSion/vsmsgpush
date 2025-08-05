package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vxmsgpush/config"
	"vxmsgpush/logger"
	"vxmsgpush/vxmsg"

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
func StartRetryScheduler(rdb *redis.Client, delayQueue, mainQueue string) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := float64(time.Now().Unix())
			msgs, err := rdb.ZRangeByScore(ctx, delayQueue, &redis.ZRangeBy{
				Min: "0", Max: fmt.Sprintf("%f", now), Count: 10,
			}).Result()
			if err != nil {
				logger.Errorf("[scheduler] 获取延迟消息失败: %v", err)
				continue
			}
			for _, raw := range msgs {
				// 投入主队列
				if err := rdb.RPush(ctx, mainQueue, raw).Err(); err != nil {
					logger.Errorf("[scheduler] 消息重投失败: %v", err)
					continue
				}
				// 从延迟队列移除
				rdb.ZRem(ctx, delayQueue, raw)
				logger.Infof("[scheduler] 消息重新投递成功")
			}
		}
	}()
}

// 处理消息逻辑，原来 worker 中的业务逻辑拆成函数
func processMessage(rdb *redis.Client, raw string, id int) {
	var msg RedisTemplateMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		logger.Errorf("[worker-%d] JSON 解析失败: %v，内容: %s", id, err, raw)
		AddFailWithReason("invalid_json")
		return
	}

	// 黑名单优先判断
	if config.IsMobileBlocked(msg.Mobile) {
		logger.Warnf("[worker-%d] 手机号 %s 在黑名单中，跳过", id, msg.Mobile)
		return
	}

	// 添加白名单校验
	if !config.IsMobileAllowed(msg.Mobile) {
		logger.Warnf("[worker-%d] 手机号 %s 不在白名单中，跳过", id, msg.Mobile)
		return
	}

	openid, err := vxmsg.GetUserOpenIDByMobile(msg.Mobile)
	if err != nil {
		logger.Errorf("[worker-%d] 获取 OpenID 失败: %v", id, err)
		AddFailWithReason("geterror_openid")
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
		if we, ok := err.(*vxmsg.WechatError); ok {
			logger.Errorf("[worker-%d] 微信发送失败 errcode=%d errmsg=%s", id, we.ErrCode, we.ErrMsg)
			switch we.ErrCode {
			case 40003:
				AddFailWithReason("invalid_openid")
			case 43004:
				AddFailWithReason("user_not_followed")
			case 42001:
				AddFailWithReason("token_expired")
			default:
				AddFailWithReason(fmt.Sprintf("wx_%d", we.ErrCode))
			}
		} else {
			logger.Errorf("[worker-%d] 模板消息发送失败: %v", id, err)
			AddFailWithReason("send_error")
		}
		msg.RetryCount++
		logger.Errorf("[worker-%d] 模板消息发送失败: %v（重试 %d 次）", id, err, msg.RetryCount)

		if msg.RetryCount > maxRetryCount {
			bs, _ := json.Marshal(msg)
			if err := rdb.RPush(ctx, deadLetterQueue, bs).Err(); err != nil {
				logger.Errorf("[worker-%d] 死信入队失败: %v", id, err)
			} else {
				logger.Warnf("[worker-%d] 消息进入死信队列: %s", id, string(bs))
			}
			return
		}

		// 加入延迟队列
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

	AddSuccess()
	logger.Infof("[worker-%d] 模板消息发送成功: %s", id, openid)
}
