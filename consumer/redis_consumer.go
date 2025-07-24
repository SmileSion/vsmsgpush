package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vxmsgpush/config"
	"vxmsgpush/logger"
	"vxmsgpush/vxmsg"

	"github.com/go-redis/redis/v8"
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
	maxRetryCount    = 5                       // 最大重试次数
	deadLetterQueue  = "wx_template_msg_dlq"   // 死信队列
	delayQueue       = "wx_template_msg_delay" // 延迟队列（ZSet）
	delayStepSeconds = 3                       // 每次重试递增秒数
)

// 启动多个消费者
func StartRedisConsumers(rdb *redis.Client, queueName string, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go worker(rdb, queueName, i+1)
	}
	logger.Infof("[redis] 启动 %d 个 worker 监听队列 %s", workerCount, queueName)
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

// worker 消费循环
func worker(rdb *redis.Client, queueName string, id int) {
	for {
		result, err := rdb.BRPop(ctx, 5*time.Second, queueName).Result()
		if err == redis.Nil || len(result) < 2 {
			continue
		}
		if err != nil {
			logger.Errorf("[worker-%d] Redis BRPop 错误: %v", id, err)
			time.Sleep(time.Second)
			continue
		}

		raw := result[1]
		var msg RedisTemplateMessage
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			logger.Errorf("[worker-%d] JSON 解析失败: %v，内容: %s", id, err, raw)
			AddFail()
			continue
		}

		// 黑名单优先判断
		if config.IsMobileBlocked(msg.Mobile) {
			logger.Warnf("[worker-%d] 手机号 %s 在黑名单中，跳过", id, msg.Mobile)
			continue
		}

		// 添加白名单校验
		if !config.IsMobileAllowed(msg.Mobile) {
			logger.Warnf("[worker-%d] 手机号 %s 不在白名单中，跳过", id, msg.Mobile)
			continue
		}

		openid, err := vxmsg.GetUserOpenIDByMobile(msg.Mobile)
		if err != nil {
			logger.Errorf("[worker-%d] 获取 OpenID 失败: %v", id, err)
			AddFail()
			continue
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
			if msg.RetryCount == 0 {
				AddFail()
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
				continue
			}

			// 加入延迟队列
			bs, _ := json.Marshal(msg)
			delay := msg.RetryCount * delayStepSeconds
			score := float64(time.Now().Add(time.Duration(delay) * time.Second).Unix())
			if err := rdb.ZAdd(ctx, delayQueue, &redis.Z{Score: score, Member: bs}).Err(); err != nil {
				logger.Errorf("[worker-%d] 延迟入队失败: %v", id, err)
			} else {
				logger.Infof("[worker-%d] 延迟消息入队，%ds 后重试，第 %d 次", id, delay, msg.RetryCount)
			}
			continue
		}

		AddSuccess()
		logger.Infof("[worker-%d] 模板消息发送成功: %s", id, openid)
	}
}
