package api

import (
	"vxmsgpush/api/handler"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化并返回 Gin Engine
func SetupRouter() *gin.Engine {
	r := gin.Default()

	wechatServer := handler.NewWechatServer("SmileSion")

	wechatGroup := r.Group("/wechat")
	{
		wechatServer.RegisterRoutes(wechatGroup)
	}

	// 注册你的接口路由
	r.POST("/api/push_template", handler.PushTemplateHandler)

	return r
}
