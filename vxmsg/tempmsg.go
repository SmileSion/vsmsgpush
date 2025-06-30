package vxmsg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
		return fmt.Errorf("获取access_token失败: %v", err)
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", accessToken)

	data, _ := json.Marshal(msg)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("请求微信失败: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["errcode"].(float64) != 0 {
		return fmt.Errorf("微信返回错误: %v", result["errmsg"])
	}

	return nil
}
