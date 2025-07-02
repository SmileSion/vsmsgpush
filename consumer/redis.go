package consumer

import (
	"github.com/go-redis/redis/v8"
	"vxmsgpush/config"
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

	return RDB
}
