package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"vxmsgpush/logger"
	"vxmsgpush/core/vxmsg/internal"
)

type MiniProgram struct {
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath"`
}

type TemplateMsg struct {
	ToUser      string                 `json:"touser"`
	TemplateID  string                 `json:"template_id"`
	URL         string                 `json:"url,omitempty"`
	Data        map[string]interface{} `json:"data"`
	MiniProgram *MiniProgram           `json:"miniprogram,omitempty"`
}

// WechatError 用于结构化微信返回错误
type WechatError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (e *WechatError) Error() string {
	return fmt.Sprintf("微信返回错误: %d - %s", e.ErrCode, e.ErrMsg)
}

// SendTemplateMsg 发送模板消息
func SendTemplateMsg(msg TemplateMsg) error {
	accessToken, err := internal.GetAccessToken()
	if err != nil {
		logger.Errorf("获取access_token失败: %v", err)
		return fmt.Errorf("获取access_token失败: %v", err)
	}
	logger.Infof("获取access_token成功: %s", accessToken)

	url := fmt.Sprintf("http://192.170.144.52:9010/weixin_api/cgi-bin/message/template/send?access_token=%s", accessToken)
	logger.Infof("发送模板消息，URL: %s", url)

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Errorf("模板消息序列化失败: %v", err)
		return fmt.Errorf("模板消息序列化失败: %v", err)
	}
	logger.Debugf("模板消息JSON: %s", string(data))

	client := &http.Client{Timeout: 5 * time.Second}
	reqBody := bytes.NewBuffer(data)

	resp, err := client.Post(url, "application/json", reqBody)
	if err != nil {
		logger.Warnf("第一次发送失败，准备重试: %v", err)
		time.Sleep(500 * time.Millisecond)

		reqBody = bytes.NewBuffer(data)
		resp, err = client.Post(url, "application/json", reqBody)
		if err != nil {
			return fmt.Errorf("请求微信失败: %v", err)
		}
	}
	defer resp.Body.Close()

	// 响应结果结构体
	var result WechatError
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Errorf("解析微信响应失败: %v", err)
		return fmt.Errorf("解析微信响应失败: %v", err)
	}
	logger.Infof("微信响应: %+v", result)

	if result.ErrCode != 0 {
		logger.Errorf("微信返回错误: %d - %s", result.ErrCode, result.ErrMsg)
		return &result // 返回结构化错误
	}

	logger.Infof("发送模板消息成功，用户: %s，模板ID: %s", msg.ToUser, msg.TemplateID)
	return nil
}
