package main

import (
	"vxmsgpush/api"
	"vxmsgpush/config"
	"vxmsgpush/core/consumer"
	"vxmsgpush/core/db"
	"vxmsgpush/logger"
)

const (
	mainQueue  = "wx_template_msg_queue"
	delayQueue = "wx_template_msg_delay"
	dispatcherCount = 10  // dispatcher并发BRPop（根据CPU核数调整）
    workerCount = 50     // worker并发处理（根据业务耗时调整）
    chanBuffer = 2000     // chan缓冲大小，防止瞬时阻塞
)

func main() {
	// 初始化配置和日志
	config.InitConfig()
	config.InitMobileWhitelist()
	config.InitMobileBlacklist()
	logger.InitLogger()

	// 初始化数据库连接
	if err := db.Init(); err != nil {
		logger.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化数据库表
	if err := db.InitMySQL(); err != nil {
		logger.Fatalf("数据库表初始化失败: %v", err)
	}

	defer func() {
		if err := logger.CloseAsyncWriters(); err != nil {
			logger.Errorf("关闭日志写入器失败: %v", err)
		}
	}()

	// 初始化 Redis 客户端并启动消费者
	rdb := consumer.InitRedis()
	consumer.StartStatRecorder()
	consumer.StartStatWriter()
	consumer.StartRedisConsumers(rdb, mainQueue, dispatcherCount,workerCount,chanBuffer)
	consumer.StartRetryScheduler(rdb, delayQueue, mainQueue,30)

	// 初始化 Gin 路由
	r := api.SetupRouter()

	// 启动服务
	port := ":9010"
	logger.Infof("服务启动，监听端口 %s", port)
	if err := r.Run(port); err != nil {
		logger.Fatalf("服务启动失败: %v", err)
	}

	logger.Info("程序启动完成")
}
