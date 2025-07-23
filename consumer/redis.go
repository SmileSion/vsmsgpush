package consumer

import (
	"context"
	"vxmsgpush/config"
	"vxmsgpush/logger"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

// InitRedis 从 config.Conf 初始化 Redis 客户端
func InitRedis() *redis.Client {
	conf := config.Conf.Redis

	RDB = redis.NewClient(&redis.Options{
		Addr:     conf.Addr,
		Password: conf.Password,
		DB:       conf.DB,
	})

	if err := RDB.Ping(context.Background()).Err(); err != nil {
		logger.Fatalf("Redis 初始化失败: %v", err)
	} else {
		logger.Info("Redis 初始化成功")
	}

	return RDB
}
