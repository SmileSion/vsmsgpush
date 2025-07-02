package consumer

import (
	"context"
	"encoding/json"
	"time"
	"vxmsgpush/logger"
	"vxmsgpush/vxmsg"

	"github.com/go-redis/redis/v8"
)

type RedisTemplateMessage struct {
	Mobile     string                 `json:"mobile"`
	TemplateID string                 `json:"template_id"`
	URL        string                 `json:"url"`
	Data       map[string]interface{} `json:"data"`

	// 新增字段，记录重试次数（json里可选）
	RetryCount int `json:"retry_count,omitempty"`
}

var ctx = context.Background()

const (
	maxRetryCount = 5                    // 最大重试次数
	deadLetterQueue = "wx_template_msg_dlq" // 死信队列名字
)

// 启动多个消费者
func StartRedisConsumers(rdb *redis.Client, queueName string, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go worker(rdb, queueName, i+1)
	}
	logger.Logger.Infof("[redis] 启动 %d 个 worker 监听队列 %s", workerCount, queueName)
}

// 单个 worker 消费循环
func worker(rdb *redis.Client, queueName string, id int) {
	for {
		result, err := rdb.BRPop(ctx, 5*time.Second, queueName).Result()
		if err == redis.Nil || len(result) < 2 {
			continue
		}
		if err != nil {
			logger.Logger.Errorf("[worker-%d] Redis BRPop 错误: %v", id, err)
			time.Sleep(time.Second)
			continue
		}

		raw := result[1]
		var msg RedisTemplateMessage
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			logger.Logger.Errorf("[worker-%d] JSON 解析失败: %v，消息内容: %s", id, err, raw)
			continue
		}

		openid, err := vxmsg.GetUserOpenIDByMobile(msg.Mobile)
		if err != nil {
			logger.Logger.Errorf("[worker-%d] 获取 OpenID 失败: %v", id, err)
			// 这里可以考虑是否放回队列，暂不回退
			continue
		}

		tpl := vxmsg.TemplateMsg{
			ToUser:     openid,
			TemplateID: msg.TemplateID,
			URL:        msg.URL,
			Data:       msg.Data,
		}

		err = vxmsg.SendTemplateMsg(tpl)
		if err != nil {
			logger.Logger.Errorf("[worker-%d] 模板消息发送失败: %v", id, err)
			// 重试次数+1
			msg.RetryCount++
			if msg.RetryCount > maxRetryCount {
				// 超过最大重试次数，写入死信队列
				bs, _ := json.Marshal(msg)
				if dlqErr := rdb.RPush(ctx, deadLetterQueue, bs).Err(); dlqErr != nil {
					logger.Logger.Errorf("[worker-%d] 写入死信队列失败: %v", id, dlqErr)
				} else {
					logger.Logger.Warnf("[worker-%d] 消息放入死信队列，丢弃: %s", id, string(bs))
				}
				continue // 丢弃当前消息，避免死循环
			}

			// 重试未超过阈值，重新放回队列尾部，稍微延迟防止刷屏
			bs, _ := json.Marshal(msg)
			if pushErr := rdb.RPush(ctx, queueName, bs).Err(); pushErr != nil {
				logger.Logger.Errorf("[worker-%d] 重试消息入队失败: %v", id, pushErr)
			} else {
				logger.Logger.Infof("[worker-%d] 重试消息重新入队，重试次数: %d", id, msg.RetryCount)
			}
			time.Sleep(time.Second)
			continue
		}

		logger.Logger.Infof("[worker-%d] 模板消息发送成功: %s", id, openid)
	}
}
