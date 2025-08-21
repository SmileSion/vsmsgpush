package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	pb "vxmsgpush/core/grpc/token"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	token    string
	expireAt time.Time
	mu       sync.Mutex
)

// 微信接口返回结构
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type server struct {
	pb.UnimplementedTokenServiceServer
}

// gRPC 方法：返回缓存的 AccessToken
func (s *server) GetAccessToken(ctx context.Context, req *pb.TokenRequest) (*pb.TokenReply, error) {
	mu.Lock()
	defer mu.Unlock()
	return &pb.TokenReply{
		AccessToken: token,
		ExpireAt:    expireAt.Unix(),
	}, nil
}

// 定时获取 token
func fetchTokenLoop(appID, appSecret string, logger *log.Logger) {
	for {
		//本地测试用
		// url := fmt.Sprintf("http://127.0.0.1:9011/weixin_api/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appID, appSecret)

		// 生产环境用
		url := fmt.Sprintf("http://192.170.144.52:9010/weixin_api/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appID, appSecret)

		resp, err := http.Get(url)
		if err != nil {
			logger.Printf("请求微信接口失败: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result tokenResponse
		if err := json.Unmarshal(body, &result); err != nil {
			logger.Printf("解析响应失败: %v, 原始响应: %s", err, string(body))
			time.Sleep(10 * time.Second)
			continue
		}

		if result.ErrCode != 0 {
			logger.Printf("微信返回错误: %s，原始响应: %s", result.ErrMsg, string(body))
			time.Sleep(10 * time.Second)
			continue
		}

		mu.Lock()
		token = result.AccessToken
		expireAt = time.Now().Add(time.Duration(result.ExpiresIn-100) * time.Second)
		mu.Unlock()
		logger.Printf("成功刷新 access_token: %s，有效期 %ds", result.AccessToken, result.ExpiresIn)
		logger.Printf("成功刷新 access_token，有效期 %ds", result.ExpiresIn)

		// 休眠到过期前 1 分钟
		sleepDuration := time.Duration(result.ExpiresIn-60) * time.Second
		time.Sleep(sleepDuration)
	}
}

func main() {
	// 创建日志文件
	logOutput := &lumberjack.Logger{
		Filename:   "log/tokenservice.log", // 日志文件路径
		MaxSize:    50,                     // 每个日志文件最大 50MB
		MaxBackups: 0,                      // 最多保留 5 个旧日志
		MaxAge:     0,                      // 最多保留 30 天
		Compress:   true,                   // 压缩旧日志
	}
	logger := log.New(logOutput, "", log.LstdFlags)

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("加载 .env 文件失败: %v", err)
	}
	// 微信参数（写死，也可以改成读取配置）
	appID := os.Getenv("WX_APPID")
	appSecret := os.Getenv("WX_APPSECRET")
	if appID == "" || appSecret == "" {
		log.Fatal("WX_APPID 或 WX_APPSECRET 未配置")
	}

	// 启动定时刷新协程
	go fetchTokenLoop(appID, appSecret, logger)

	// 启动 gRPC 服务
	lis, err := net.Listen("tcp", ":51001")
	if err != nil {
		logger.Fatalf("监听端口失败: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterTokenServiceServer(s, &server{})
	reflection.Register(s)

	logger.Println("TokenService gRPC 服务启动，监听 :51001")
	if err := s.Serve(lis); err != nil {
		logger.Fatalf("gRPC 服务启动失败: %v", err)
	}
}
