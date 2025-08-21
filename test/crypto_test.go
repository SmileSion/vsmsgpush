package test

import (
	"vxmsgpush/utils"
	"testing"
)

const testPlainText = "v7j1X8xFk+lktzgDl8qA9G7V6z+wUgSBC9wlveFVX/Y3oYmccldtyiQVXG7xGsZt"

func TestEncryptDecrypt(t *testing.T) {
	// 加密
	encrypted, err := utils.Encrypt(testPlainText)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	t.Log("加密结果：",encrypted)

	// 解密
	decrypted, err := utils.Decrypt(testPlainText)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	t.Log("解密结果：",decrypted)

	// 校验解密结果
	if decrypted != testPlainText {
		t.Errorf("解密结果与原文不一致:\n原文: %s\n解密后: %s", testPlainText, decrypted)
	}
}