package api

import "github.com/gin-gonic/gin"

func onlyAllowLocalhost() gin.HandlerFunc {
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
