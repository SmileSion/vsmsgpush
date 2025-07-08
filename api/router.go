package api

import (
	"vxmsgpush/api/handler"
	"vxmsgpush/config"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化并返回 Gin Engine
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	wechatServer := handler.NewWechatServer("SmileSion")

	wechatGroup := r.Group("/wechat")
	{
		wechatServer.RegisterRoutes(wechatGroup)
	}

	// 根据配置决定是否启用手机号白名单中间件
	middlewares := []gin.HandlerFunc{
		onlyAllowLocalhost(),
	}

	if config.Conf.Security.EnableMobileWhitelist {
		middlewares = append(middlewares, mobileWhitelistMiddleware())
	}

	apiGroup := r.Group("/push", middlewares...)
	{
		apiGroup.POST("/template", handler.PushTemplateHandler)
	}

	return r
}
