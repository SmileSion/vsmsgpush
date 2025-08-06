package api

import (
	"vxmsgpush/api/handler"
	"vxmsgpush/api/whitelist"
	"vxmsgpush/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter 初始化并返回 Gin Engine
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	proAllowedIPs := []string{"127.0.0.1", "192.169.132.223", "192.169.132.224"}
	// Prometheus 监控指标暴露接口
	proGroup := r.Group("/prometheus", whitelist.AllowProthemeus(proAllowedIPs...))
	{
		proGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	outGroup := r.Group("/out", whitelist.AllowOutSystem(config.Conf.Security.AllowedIPs...))
	{
		outGroup.POST("/template", handler.PushTemplateHandlerRedis)
	}

	// WeChat 路由组
	{
		wechatServer := handler.NewWechatServer("SmileSion")
		wechatGroup := r.Group("/wechat")
		{
			wechatServer.RegisterRoutes(wechatGroup)
		}
	}

	// push 路由组（含中间件）
	{
		middlewares := []gin.HandlerFunc{
			whitelist.OnlyAllowLocalhost(),
		}

		if config.Conf.Security.EnableMobileWhitelist {
			middlewares = append(middlewares, whitelist.MobileWhitelistMiddleware())
		}

		localGroup := r.Group("/local", middlewares...)
		{
			localGroup.POST("/template", handler.PushTemplateHandler)
		}
	}

	return r
}
