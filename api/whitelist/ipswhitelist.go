package whitelist

import (
	"vxmsgpush/logger"
	"github.com/gin-gonic/gin"
)

func AllowProthemeus(allowedIPs ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedIPs))
	for _, ip := range allowedIPs {
		allowed[ip] = struct{}{}
	}
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if _, ok := allowed[clientIP]; !ok {
			logger.Warnf("Prothemeus 拒绝访问，IP: %s", clientIP)
			c.AbortWithStatusJSON(403, gin.H{
				"error": "Forbidden: IP not allowed (Prothemeus)",
			})
			return
		}
		c.Next()
	}
}

func AllowOutSystem(allowedIPs ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedIPs))
	for _, ip := range allowedIPs {
		allowed[ip] = struct{}{}
	}
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if _, ok := allowed[clientIP]; !ok {
			logger.Warnf("OutSystem 拒绝访问，IP: %s", clientIP)
			c.AbortWithStatusJSON(403, gin.H{
				"error": "Forbidden: IP not allowed (OutSystem)",
			})
			return
		}
		c.Next()
	}
}

func OnlyAllowLocalhost() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP != "127.0.0.1" && clientIP != "::1" {
			c.AbortWithStatusJSON(403, gin.H{
				"error": "Forbidden: only localhost can access this endpoint",
			})
			return
		}
		c.Next()
	}
}
