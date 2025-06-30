package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type userIdRequest struct {
	TxnBodyCom struct {
		Mobile string `json:"mobile"`
	} `json:"txnBodyCom"`
	TxnCommCom struct {
		TRecInPage        string `json:"tRecInPage"`
		TxnIttChnlCgyCode string `json:"txnIttChnlCgyCode"`
		TStsTraceId       string `json:"tStsTraceId"`
		TPageJump         string `json:"tPageJump"`
		TxnIttChnlId      string `json:"txnIttChnlId"`
	} `json:"txnCommCom"`
}

type rawResponse struct {
	CResponseBody string `json:"C-Response-Body"`
}

type parsedBody struct {
	ID string `json:"id"`
}

// GetUserOpenIDByMobile 根据手机号查询微信 openid
func GetUserOpenIDByMobile(mobile string) (string, error) {
	url := "https://zwfwxtzx.shaanxi.gov.cn:8202/gsp/uc10051"

	reqBody := userIdRequest{}
	reqBody.TxnBodyCom.Mobile = mobile
	reqBody.TxnCommCom = struct {
		TRecInPage        string `json:"tRecInPage"`
		TxnIttChnlCgyCode string `json:"txnIttChnlCgyCode"`
		TStsTraceId       string `json:"tStsTraceId"`
		TPageJump         string `json:"tPageJump"`
		TxnIttChnlId      string `json:"txnIttChnlId"`
	}{
		TRecInPage:        "10",
		TxnIttChnlCgyCode: "D001C004",
		TStsTraceId:       "110567980",
		TPageJump:         "1",
		TxnIttChnlId:      "99990001000000000000000",
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("C-Business-Id", "5d6b66525611473f900c2a9d053227e8")
	req.Header.Set("C-Tenancy-Id", "610000000000")
	req.Header.Set("Referer", "https://zwfwxtzx.shaanxi.gov.cn:8202")
	req.Header.Set("C-App-Id", "201199_app_17272556872746481")
	req.Header.Set("apppwd", "6e32c1771d6148fba60f07c96d0d2791")
	req.Header.Set("Cookie", "JSESSIONID=18870768FA07370079EAAA435A90C453")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	var raw rawResponse
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return "", fmt.Errorf("解析外层 JSON 失败: %v", err)
	}

	var parsed parsedBody
	if err := json.Unmarshal([]byte(raw.CResponseBody), &parsed); err != nil {
		return "", fmt.Errorf("解析内层 JSON 失败: %v", err)
	}

	if parsed.ID == "" {
		return "", fmt.Errorf("未找到 ID，响应可能异常: %s", raw.CResponseBody)
	}

	return parsed.ID, nil
}
