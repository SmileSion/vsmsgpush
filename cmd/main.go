package main

import (
	"fmt"
	"log"
	"vxmsgpush/api"
	"vxmsgpush/config"
	"vxmsgpush/logger"
)

func main() {
	config.InitConfig()
	logger.InitLogger()
	r := api.SetupRouter()

	port := ":8080"
	fmt.Printf("服务启动，监听端口 %s\n", port)
	if err := r.Run(port); err != nil {
		fmt.Printf("服务启动失败: %v\n", err)
	}
	
	defer func() {
        if err := logger.CloseAsyncWriters(); err != nil {
            log.Printf("关闭日志写入器失败: %v", err)
        }
    }()

    logger.Logger.Info("程序启动")
}
