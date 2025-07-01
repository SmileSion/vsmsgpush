package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"vxmsgpush/logger"
	"vxmsgpush/vxmsg/internal"
)

type TextMsg struct {
	ToUser  string `json:"touser"`
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// SendTextMessage 发送文本客服消息
func SendTextMessage(toUser string, content string) error {
	accessToken, err := internal.GetAccessToken()
	if err != nil {
		logger.Logger.Errorf("获取access_token失败: %v", err)
		return fmt.Errorf("获取access_token失败: %v", err)
	}
	logger.Logger.Infof("获取access_token成功: %s", accessToken)

	url := fmt.Sprintf("http://192.170.144.52:9010/weixin_api/cgi-bin/message/custom/send?access_token=%s", accessToken)
	logger.Logger.Infof("发送客服消息，URL: %s", url)

	msg := TextMsg{
		ToUser:  toUser,
		MsgType: "text",
	}
	msg.Text.Content = content

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Errorf("消息内容序列化失败: %v", err)
		return fmt.Errorf("消息内容序列化失败: %v", err)
	}
	logger.Logger.Debugf("消息内容JSON: %s", string(data))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		logger.Logger.Errorf("发送消息失败: %v", err)
		return fmt.Errorf("发送消息失败: %v", err)
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

	logger.Logger.Infof("发送消息成功，用户: %s，内容: %s", toUser, content)
	return nil
}
