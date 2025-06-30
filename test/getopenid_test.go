package test

import (
	"testing"
	"vxmsgpush/utils"
)

func TestGetUserOpenIDByMobile(t *testing.T) {
	mobile := "" // 测试手机号

	openid, err := utils.GetUserOpenIDByMobile(mobile)
	if err != nil {
		t.Fatalf("调用失败: %v", err)
	}

	if openid == "" {
		t.Fatal("返回的openid为空")
	}

	t.Logf("手机号 %s 对应的openid: %s", mobile, openid)
}
