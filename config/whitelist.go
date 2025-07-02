package config

import (
	"sync"
)

var (
	once         sync.Once
	mobileMap    map[string]struct{}
	enableCheck  bool
)

// InitMobileWhitelist 初始化白名单（只调用一次）
func InitMobileWhitelist() {
	once.Do(func() {
		mobileMap = make(map[string]struct{})
		for _, m := range Conf.Security.AllowedMobiles {
			mobileMap[m] = struct{}{}
		}
		enableCheck = Conf.Security.EnableMobileWhitelist
	})
}

// IsMobileAllowed 检查是否在白名单中
func IsMobileAllowed(mobile string) bool {
	if !enableCheck {
		return true
	}
	_, ok := mobileMap[mobile]
	return ok
}
