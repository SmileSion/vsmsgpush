package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"vxmsgpush/consumer"
	"vxmsgpush/logger"
)

// 定义结构体用于校验 JSON 格式
type RedisTemplateMessage struct {
	Mobile      string                 `json:"mobile" binding:"required"`
	TemplateID  string                 `json:"template_id" binding:"required"`
	URL         string                 `json:"url"`
	Data        map[string]interface{} `json:"data" binding:"required"`
	MiniProgram *MiniProgram           `json:"miniprogram,omitempty"`
}

// PushTemplateHandlerRedis 将校验通过的请求存入 Redis 队列
func PushTemplateHandlerRedis(c *gin.Context) {
	clientIP := c.ClientIP()
	var req RedisTemplateMessage

	// 参数格式校验
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf("请求参数格式错误，IP: %s，错误: %v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误: " + err.Error()})
		return
	}

	// 原始 JSON 数据转字符串
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		logger.Errorf("请求序列化失败，IP: %s，错误: %v", clientIP, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "序列化失败: " + err.Error()})
		return
	}

	logger.Infof("接收到推送请求，IP: %s，内容: %s", clientIP, string(jsonBytes))
	
	// 存入 Redis list
	err = consumer.RDB.RPush(context.Background(), "wx_template_msg_queue", jsonBytes).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入 Redis 失败: " + err.Error()})
		return
	}

	logger.Infof("消息成功入队，IP: %s", clientIP)
	c.JSON(http.StatusOK, gin.H{"message": "消息入队成功"})
}
