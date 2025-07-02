package config

import (
	"vxmsgpush/utils"
	"log"
	"github.com/BurntSushi/toml"
)

type LogConfig struct {
	Filepath      string `toml:"filepath"`
	MaxSize       int    `toml:"max_size"`
	MaxBackups    int    `toml:"max_backups"`
	MaxAge        int    `toml:"max_age"`
	Level         string `toml:"level"`
	Compress      bool   `toml:"compress"`
	EnableConsole bool   `toml:"enable_console"`
}

type VxConfig struct {
	AppId     string `toml:"appid"`
	AppSecret string `toml:"appsecret"`
}

type SecurityConfig struct {
	EnableMobileWhitelist bool `toml:"enable_mobile_whitelist"`
	AllowedMobiles        []string `toml:"allowed_mobiles"`
}

type RedisConfig struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type Config struct {
	Log   LogConfig `toml:"log"`
	VxKey VxConfig  `toml:"vxkey"`
	Security SecurityConfig `toml:"security"`
	Redis    RedisConfig    `toml:"redis"`
}

var Conf Config

// InitConfig 使用指定路径加载配置文件
func InitConfig() {
	if _, err := toml.DecodeFile("config/config.toml", &Conf); err != nil {
		panic(err)
	}
	log.Println("---配置文件加载成功---")
	if decrypted, err := utils.Decrypt(Conf.VxKey.AppId); err == nil {
		Conf.VxKey.AppId = decrypted
	} else {
		log.Fatalf("AppKey解密失败: %v", err)
	}
	if decrypted, err := utils.Decrypt(Conf.VxKey.AppSecret); err == nil {
		Conf.VxKey.AppSecret = decrypted
	} else {
		log.Fatalf("SecretKey解密失败: %v", err)
	}
}

