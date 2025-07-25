package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"vxmsgpush/logger"
	"vxmsgpush/vxmsg/internal"
)

type MiniProgram struct {
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath"`
}

type TemplateMsg struct {
	ToUser     string                 `json:"touser"`
	TemplateID string                 `json:"template_id"`
	URL        string                 `json:"url,omitempty"`
	Data       map[string]interface{} `json:"data"`
	MiniProgram *MiniProgram          `json:"miniprogram,omitempty"`
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

		// 重新创建 buffer，因为之前那个已经被读完了
		reqBody = bytes.NewBuffer(data)
		resp, err = client.Post(url, "application/json", reqBody)
		if err != nil {
			return fmt.Errorf("请求微信失败: %v", err)
		}
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logger.Errorf("解析微信响应失败: %v", err)
		return fmt.Errorf("解析微信响应失败: %v", err)
	}
	logger.Infof("微信响应: %+v", result)

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg := result["errmsg"]
		logger.Errorf("微信返回错误: %v", errmsg)
		return fmt.Errorf("微信返回错误: %v", errmsg)
	}

	logger.Infof("发送模板消息成功，用户: %s，模板ID: %s", msg.ToUser, msg.TemplateID)
	return nil
}
