package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"vxmsgpush/config" // 根据你的模块名和路径调整
)

var (
	token    string
	expireAt time.Time
	mu       sync.Mutex
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// GetAccessToken 返回缓存中的 access_token，过期自动刷新
func GetAccessToken() (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if token != "" && time.Now().Before(expireAt) {
		return token, nil
	}

	appID := config.Conf.VxKey.AppId
	appSecret := config.Conf.VxKey.AppSecret

	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appID, appSecret)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取token失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result tokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析token响应失败: %v", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("微信返回错误: %s", result.ErrMsg)
	}

	token = result.AccessToken
	expireAt = time.Now().Add(time.Duration(result.ExpiresIn-100) * time.Second)
	return token, nil
}
