package config

import (
    "sync"
)

var (
    onceBlacklist     sync.Once
    mobileBlacklist   map[string]struct{}
    enableBlacklist   bool
)

// InitMobileBlacklist 初始化黑名单（只调用一次）
func InitMobileBlacklist() {
    onceBlacklist.Do(func() {
        mobileBlacklist = make(map[string]struct{})
        for _, m := range Conf.Security.BlockedMobiles {
            mobileBlacklist[m] = struct{}{}
        }
        enableBlacklist = Conf.Security.EnableMobileBlacklist
    })
}

// IsMobileBlocked 判断手机号是否在黑名单中
func IsMobileBlocked(mobile string) bool {
    if !enableBlacklist {
        return false
    }
    _, ok := mobileBlacklist[mobile]
    return ok
}
