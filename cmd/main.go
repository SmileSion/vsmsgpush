package main

import (
	"vxmsgpush/api"
	"vxmsgpush/config"
	"vxmsgpush/consumer"
	"vxmsgpush/logger"
)

const (
	mainQueue  = "wx_template_msg_queue"
	delayQueue = "wx_template_msg_delay"
)

func main() {
	// 初始化配置和日志
	config.InitConfig()
	logger.InitLogger()

	defer func() {
		if err := logger.CloseAsyncWriters(); err != nil {
			logger.Logger.Errorf("关闭日志写入器失败: %v", err)
		}
	}()

	// 初始化 Redis 客户端并启动消费者
	rdb := consumer.InitRedis()
	consumer.StartRedisConsumers(rdb, mainQueue, 1)
	consumer.StartRetryScheduler(rdb, delayQueue, mainQueue)


	// 初始化 Gin 路由
	r := api.SetupRouter()

	// 启动服务
	port := ":9010"
	logger.Logger.Infof("服务启动，监听端口 %s", port)
	if err := r.Run(port); err != nil {
		logger.Logger.Fatalf("服务启动失败: %v", err)
	}

	logger.Logger.Info("程序启动完成")
}
