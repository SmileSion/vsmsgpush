package test

import (
	"testing"
	"vxmsgpush/vxmsg"
	"vxmsgpush/logger"
)

func TestGetUserOpenIDByMobile(t *testing.T) {
	logger.InitLogger()
	mobile := "18918399562" // 测试手机号

	openid, err := vxmsg.GetUserOpenIDByMobile(mobile)
	if err != nil {
		t.Fatalf("调用失败: %v", err)
	}

	if openid == "" {
		t.Fatal("返回的openid为空")
	}

	t.Logf("手机号 %s 对应的openid: %s", mobile, openid)
}
