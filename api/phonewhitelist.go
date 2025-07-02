package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"vxmsgpush/config"

	"github.com/gin-gonic/gin"
)

type requestBody struct {
	Mobile string `json:"mobile"`
}

func mobileWhitelistMiddleware() gin.HandlerFunc {
	// 初始化配置中手机号白名单为 map[string]bool
	allowedMobiles := make(map[string]bool)
	for _, mobile := range config.Conf.Security.AllowedMobiles {
		allowedMobiles[mobile] = true
	}

	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var req requestBody
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
			return
		}

		if !allowedMobiles[req.Mobile] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权限给此手机号推送"})
			return
		}

		c.Next()
	}
}
