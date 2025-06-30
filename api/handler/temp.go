package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vxmsgpush/vxmsg"
)

type PushTemplateRequest struct {
	Mobile     string                 `json:"mobile" binding:"required"`
	TemplateID string                 `json:"template_id" binding:"required"`
	URL        string                 `json:"url"`
	Data       map[string]interface{} `json:"data" binding:"required"`
}

func PushTemplateHandler(c *gin.Context) {
	var req PushTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 根据手机号获取微信OpenID
	openid, err := vxmsg.GetUserOpenIDByMobile(req.Mobile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户OpenID失败: " + err.Error()})
		return
	}

	// 组装模板消息结构
	msg := vxmsg.TemplateMsg{
		ToUser:     openid,
		TemplateID: req.TemplateID,
		URL:        req.URL,
		Data:       req.Data,
	}

	// 调用发送模板消息函数
	if err := vxmsg.SendTemplateMsg(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送模板消息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "消息发送成功"})
}
