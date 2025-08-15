package test

import (
	"testing"
	"vxmsgpush/config"
	"vxmsgpush/core/vxmsg"
)
func TestSendTextMessage(t *testing.T) {
	config.InitConfig()
	err := vxmsg.SendTextMessage("oWQD47IblPwb8VdueJygyGByDl9M", "测试内容")
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
}

