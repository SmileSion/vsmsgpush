package api

import (
	"vxmsgpush/api/handler"
	"vxmsgpush/config"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter 初始化并返回 Gin Engine
func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Prometheus 监控指标暴露接口
	{
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
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
			onlyAllowLocalhost(),
		}

		if config.Conf.Security.EnableMobileWhitelist {
			middlewares = append(middlewares, mobileWhitelistMiddleware())
		}

		apiGroup := r.Group("/push", middlewares...)
		{
			apiGroup.POST("/template", handler.PushTemplateHandler)
		}
	}

	return r
}
