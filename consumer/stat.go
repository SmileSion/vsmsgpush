package consumer

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"vxmsgpush/logger"
)

// 统计计数
var (
	successCount int64
	failCount    int64
)

// 文件切割相关
var (
	mu          sync.Mutex
	currentFile *os.File
	curMonth    string // 格式 "2006-01"
)

// AddSuccess 计数成功
func AddSuccess() {
	atomic.AddInt64(&successCount, 1)
}

// AddFail 计数失败
func AddFail() {
	atomic.AddInt64(&failCount, 1)
}

// StartStatRecorder 启动统计协程，每10分钟写一次日志文件
func StartStatRecorder() {
	go func() {
		for {
			now := time.Now()
			next := now.Truncate(time.Minute).Add(time.Minute)
			time.Sleep(time.Until(next))

			succ := atomic.SwapInt64(&successCount, 0)
			fail := atomic.SwapInt64(&failCount, 0)

			timestamp := next.Format("2006-01-02 15:04")
			line := fmt.Sprintf("%s 成功: %d, 失败: %d\n", timestamp, succ, fail)

			err := writeLogLine(line)
			if err != nil {
				logger.Infof("[stat] 写入统计日志失败: %v\n", err)
			} else {
				logger.Infof("[stat] 写入统计日志成功: %s", line)
			}
		}
	}()
}

// writeLogLine 写入日志，按月份切割
func writeLogLine(line string) error {
	mu.Lock()
	defer mu.Unlock()

	nowMonth := time.Now().Format("2006-01")
	if curMonth != nowMonth {
		// 关闭旧文件
		if currentFile != nil {
			currentFile.Close()
		}

		// 创建目录
		dir := "stat"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		// 打开新文件，文件名带年月
		filepath := fmt.Sprintf("stat/statistics-%s.log", nowMonth)
		f, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		currentFile = f
		curMonth = nowMonth
	}

	_, err := currentFile.WriteString(line)
	return err
}
