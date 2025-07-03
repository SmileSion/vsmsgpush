package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vxmsgpush/vxmsg"
)

type MiniProgram struct {
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath"`
}

type PushTemplateRequest struct {
	Mobile      string                 `json:"mobile" binding:"required"`
	TemplateID  string                 `json:"template_id" binding:"required"`
	URL         string                 `json:"url"`
	Data        map[string]interface{} `json:"data" binding:"required"`
	MiniProgram *MiniProgram           `json:"miniprogram,omitempty"`
}

func PushTemplateHandler(c *gin.Context) {
	var req PushTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 获取微信OpenID
	openid, err := vxmsg.GetUserOpenIDByMobile(req.Mobile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户OpenID失败: " + err.Error()})
		return
	}

	// 构造模板消息
	msg := vxmsg.TemplateMsg{
		ToUser:      openid,
		TemplateID:  req.TemplateID,
		URL:         req.URL,
		Data:        req.Data,
		MiniProgram: nil,
	}

	if req.MiniProgram != nil {
		msg.MiniProgram = &vxmsg.MiniProgram{
			AppID:    req.MiniProgram.AppID,
			PagePath: req.MiniProgram.PagePath,
		}
	}

	// 发送消息
	if err := vxmsg.SendTemplateMsg(msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送模板消息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "消息发送成功"})
}
