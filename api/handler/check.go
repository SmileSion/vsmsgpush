package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"crypto/sha1"
	"fmt"
	"strings"
	"io/ioutil"

	"vxmsgpush/logger"  // 引入你的日志模块
)

// 微信公众号验证结构体
type WechatServer struct {
	Token string
}

func NewWechatServer(token string) *WechatServer {
	return &WechatServer{Token: token}
}

func (s *WechatServer) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", s.handleGet)
	rg.POST("", s.handlePost)
}

// GET 用于微信服务器验证
func (s *WechatServer) handleGet(c *gin.Context) {
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	logger.Logger.Infof("请求来源 IP: %s", c.ClientIP())
	logger.Logger.Infof("请求参数: %v", c.Request.URL.Query())

	if s.checkSignature(signature, timestamp, nonce) {
		c.String(http.StatusOK, echostr)
	} else {
		c.String(http.StatusForbidden, "验证失败")
	}
}

// POST 接收微信消息推送
func (s *WechatServer) handlePost(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		logger.Logger.Errorf("读取请求体失败: %v", err)
		c.String(http.StatusBadRequest, "读取请求失败")
		return
	}
	logger.Logger.Infof("收到微信推送消息: %s", string(body))

	// 这里可以扩展消息处理逻辑，暂时回复固定内容
	c.String(http.StatusOK, "收到消息")
}

func (s *WechatServer) checkSignature(signature, timestamp, nonce string) bool {
	tmpList := []string{s.Token, timestamp, nonce}
	sort.Strings(tmpList)
	tmpStr := strings.Join(tmpList, "")

	sha1Hash := sha1.New()
	sha1Hash.Write([]byte(tmpStr))
	hashStr := fmt.Sprintf("%x", sha1Hash.Sum(nil))

	return hashStr == signature
}
