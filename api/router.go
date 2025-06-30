// api/router.go
package api

import (
	"github.com/gin-gonic/gin"
	"vxmsgpush/api/handler"
)

// SetupRouter 初始化并返回 Gin Engine
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 注册你的接口路由
	r.POST("/api/push_template", handler.PushTemplateHandler)

	return r
}
