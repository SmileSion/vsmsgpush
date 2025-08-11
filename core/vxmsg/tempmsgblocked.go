package vxmsg

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"vxmsgpush/logger"
	"vxmsgpush/core/vxmsg/internal"
)

// QueryBlockedTemplateMsg 根据消息 ID 查询是否被拦截
func QueryBlockedTemplateMsg(msgID string) error {
	accessToken, err := internal.GetAccessToken()
	if err != nil {
		logger.Errorf("获取 access_token 失败: %v", err)
		return fmt.Errorf("access_token 获取失败: %v", err)
	}
	logger.Infof("access_token 获取成功")

	url := fmt.Sprintf("http://192.170.144.52:9010/weixin_api/wxa/sec/queryblocktmplmsg?access_token=%s", accessToken)
	reqURL := fmt.Sprintf("%s&msgid=%s", url, msgID)

	resp, err := http.Get(reqURL)
	if err != nil {
		logger.Errorf("请求微信接口失败: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	logger.Infof("查询结果: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("查询失败，状态码: %d，响应: %s", resp.StatusCode, string(body))
	}

	return nil
}
