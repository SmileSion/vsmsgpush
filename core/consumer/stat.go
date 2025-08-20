package consumer

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"vxmsgpush/core/db"
	"vxmsgpush/logger"

	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus 指标（持续累加）
var (
	successCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "push_success_total",
		Help: "Total number of successful push messages",
	})
	failCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "push_fail_total",
		Help: "Total number of failed push messages",
	})
	failByReasonCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "push_fail_reason_total",
			Help: "Total number of failed pushes by reason code",
		},
		[]string{"reason"},
	)
)

// 日志用的每分钟计数（独立于 Prometheus）
var (
	successCount int64
	failCount    int64

	failReasonLogCounter = struct {
		sync.Mutex
		m map[string]int64
	}{m: make(map[string]int64)}
)

// 文件切割相关
var (
	mu          sync.Mutex
	currentFile *os.File
	curMonth    string // 格式 "2006-01"
)

// AddSuccess 成功计数（Prometheus + 日志）
func AddSuccess() {
	atomic.AddInt64(&successCount, 1)
	successCounter.Inc()
}

// AddFail 失败计数（Prometheus + 日志）
func AddFail() {
	atomic.AddInt64(&failCount, 1)
	failCounter.Inc()
}

// AddFailWithReason 增加失败原因（Prometheus + 日志），支持 AppID
func AddFailWithReason(reason string, appid string) {
	AddFail() // 累加总失败
	label := reason
	if appid != "" {
		label = fmt.Sprintf("%s|%s", reason, appid)
	}
	failByReasonCounter.WithLabelValues(label).Inc()

	// 日志单独统计
	failReasonLogCounter.Lock()
	defer failReasonLogCounter.Unlock()
	failReasonLogCounter.m[label]++
}

func init() {
	// 注册 Prometheus 指标
	prometheus.MustRegister(successCounter)
	prometheus.MustRegister(failCounter)
	prometheus.MustRegister(failByReasonCounter)
}

// StartStatRecorder 启动统计协程，每分钟写一次日志
func StartStatRecorder() {
	go func() {
		for {
			now := time.Now()
			next := now.Truncate(time.Minute).Add(time.Minute)
			time.Sleep(time.Until(next))

			// 原子交换统计值
			succ := atomic.SwapInt64(&successCount, 0)
			fail := atomic.SwapInt64(&failCount, 0)
			timestamp := next.Format("2006-01-02 15:04")

			// 拼日志（保留原格式）
			line := fmt.Sprintf("%s 成功: %d, 失败: %d\n", timestamp, succ, fail)

			// 处理失败原因
			failReasonLogCounter.Lock()
			reasonCopy := make(map[string]int64, len(failReasonLogCounter.m))
			for label, count := range failReasonLogCounter.m {
				if count > 0 {
					line += fmt.Sprintf("%s 原因[%s]: %d\n", timestamp, label, count)
				}
				reasonCopy[label] = count
			}
			// 清空 map
			failReasonLogCounter.m = make(map[string]int64)
			failReasonLogCounter.Unlock()

			// 写日志
			if err := writeLogLine(line); err != nil {
				logger.Infof("[stat] 写入统计日志失败: %v\n", err)
			} else {
				logger.Infof("[stat] 写入统计日志成功: %s", line)
			}

			// 写 MySQL
			for label, count := range reasonCopy {
				reasonPart := label
				appid := ""
				if idx := strings.Index(label, "|"); idx != -1 {
					reasonPart = label[:idx]
					appid = label[idx+1:]
				}
				if err := db.StoreFailReasonWithAppID(next, reasonPart, count, appid); err != nil {
					logger.Infof("[stat] 写入 MySQL 失败: %v\n", err)
				}
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
		if currentFile != nil {
			currentFile.Close()
		}

		dir := "stat"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

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
