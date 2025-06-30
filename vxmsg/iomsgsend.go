package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
		return fmt.Errorf("获取access_token失败: %v", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=%s", accessToken)

	msg := TextMsg{
		ToUser:  toUser,
		MsgType: "text",
	}
	msg.Text.Content = content

	data, _ := json.Marshal(msg)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["errcode"].(float64) != 0 {
		return fmt.Errorf("微信返回错误: %v", result["errmsg"])
	}

	return nil
}
