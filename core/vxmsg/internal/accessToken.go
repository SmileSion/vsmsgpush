package internal

import (
	"context"
	"sync"
	"time"

	pb "vxmsgpush/core/grpc/token"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcClient pb.TokenServiceClient
	conn       *grpc.ClientConn
	mu         sync.Mutex
)

func initGRPCClient() {
	mu.Lock()
	defer mu.Unlock()

	if grpcClient != nil {
		return
	}

	var err error
	conn, err = grpc.NewClient(
		"127.0.0.1:51001",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	grpcClient = pb.NewTokenServiceClient(conn)
}

func GetAccessToken() (string, error) {
	if grpcClient == nil {
		initGRPCClient()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := grpcClient.GetAccessToken(ctx, &pb.TokenRequest{})
	if err != nil {
		// 失败时重建连接
		mu.Lock()
		grpcClient = nil
		if conn != nil {
			_ = conn.Close()
			conn = nil
		}
		mu.Unlock()
		return "", err
	}
	return resp.AccessToken, nil
}
