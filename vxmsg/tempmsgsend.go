package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"vxmsgpush/logger"
	"vxmsgpush/vxmsg/internal"
)

type TemplateMsg struct {
	ToUser     string                 `json:"touser"`
	TemplateID string                 `json:"template_id"`
	URL        string                 `json:"url,omitempty"`
	Data       map[string]interface{} `json:"data"`
}

// SendTemplateMsg 发送模板消息
func SendTemplateMsg(msg TemplateMsg) error {
	accessToken, err := internal.GetAccessToken()
	if err != nil {
		logger.Logger.Errorf("获取access_token失败: %v", err)
		return fmt.Errorf("获取access_token失败: %v", err)
	}
	logger.Logger.Infof("获取access_token成功: %s", accessToken)

	url := fmt.Sprintf("http://192.170.144.52:9010/weixin_api/cgi-bin/message/template/send?access_token=%s", accessToken)
	logger.Logger.Infof("发送模板消息，URL: %s", url)

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Errorf("模板消息序列化失败: %v", err)
		return fmt.Errorf("模板消息序列化失败: %v", err)
	}
	logger.Logger.Debugf("模板消息JSON: %s", string(data))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		logger.Logger.Errorf("请求微信失败: %v", err)
		return fmt.Errorf("请求微信失败: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Logger.Errorf("解析微信响应失败: %v", err)
		return fmt.Errorf("解析微信响应失败: %v", err)
	}
	logger.Logger.Debugf("微信响应: %+v", result)

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg := result["errmsg"]
		logger.Logger.Errorf("微信返回错误: %v", errmsg)
		return fmt.Errorf("微信返回错误: %v", errmsg)
	}

	logger.Logger.Infof("发送模板消息成功，用户: %s，模板ID: %s", msg.ToUser, msg.TemplateID)
	return nil
}
