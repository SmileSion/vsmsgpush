package db

import (
	"database/sql"
	"fmt"
	"time"
	"vxmsgpush/config"
	"vxmsgpush/logger"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

const (
	maxOpenConns = 15
	maxIdleConns = 8
)

//初始化数据库连接（显式调用）
func Init() error {
	cfg := config.Conf.MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		logger.Errorf("[mysql] 打开连接失败: %v", err)
		return err
	}

	if err := DB.Ping(); err != nil {
		logger.Errorf("[mysql] Ping 失败: %v", err)
		return err
	}

	DB.SetMaxOpenConns(maxOpenConns)
	DB.SetMaxIdleConns(maxIdleConns)

	logger.Info("[mysql] 数据库连接成功")
	return nil
}

//初始化所有数据库表
func InitMySQL() error {
	createStatTable := `
	CREATE TABLE IF NOT EXISTS push_stat (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		stat_time DATETIME NOT NULL,
		success_count BIGINT NOT NULL,
		fail_count BIGINT NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`
	createReasonTable := `
	CREATE TABLE IF NOT EXISTS push_fail_reason (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		stat_time DATETIME NOT NULL,
		reason VARCHAR(255) NOT NULL,
		count BIGINT NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	// 用户发送统计表，唯一索引防止重复发送计数
	createUserStatTable := `
	CREATE TABLE IF NOT EXISTS push_user_stat (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		mobile VARCHAR(20) NOT NULL,
		openid VARCHAR(100) NOT NULL DEFAULT '',
		success_count BIGINT NOT NULL DEFAULT 0,
		fail_count BIGINT NOT NULL DEFAULT 0,
		UNIQUE KEY uniq_mobile_openid (mobile, openid)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	tables := []string{createStatTable, createReasonTable, createUserStatTable}
	for _, sqlStmt := range tables {
		if _, err := DB.Exec(sqlStmt); err != nil {
			logger.Errorf("[mysql] 创建表失败: %v", err)
			return err
		}
	}

	logger.Info("[mysql] 所有数据库表初始化成功")
	return nil
}

// 保存分钟统计数据
func StoreStat(ts time.Time, succ, fail int64, reasons map[string]int64) error {
	tx, err := DB.Begin()
	if err != nil {
		logger.Errorf("[mysql] 开启事务失败: %v", err)
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO push_stat (stat_time, success_count, fail_count) VALUES (?, ?, ?)`,
		ts, succ, fail,
	); err != nil {
		logger.Errorf("[mysql] 插入 push_stat 失败: %v", err)
		return err
	}

	for reason, count := range reasons {
		if _, err := tx.Exec(
			`INSERT INTO push_fail_reason (stat_time, reason, count) VALUES (?, ?, ?)`,
			ts, reason, count,
		); err != nil {
			logger.Errorf("[mysql] 插入 push_fail_reason 失败, reason=%s: %v", reason, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Errorf("[mysql] 提交事务失败: %v", err)
		return err
	}

	logger.Infof("[mysql] 成功保存统计数据，时间: %s", ts.Format("2006-01-02 15:04"))
	return nil
}

// UpdateUserSendStat 根据手机号和openid更新发送成功/失败统计（不计重复发送）
func UpdateUserSendStatWithAppID(mobile, openid, appid string, success bool) error {
	if mobile == "" {
		return fmt.Errorf("mobile 不能为空")
	}

	table := "push_user_stat" // 默认表
	if appid != "" {
		table = fmt.Sprintf("push_user_stat_%s", appid)
		// 可选：提前建表逻辑，如果不存在则创建
		if err := InitAppIDTable(table); err != nil {
			return err
		}
	}

	var openidVal interface{}
	if openid == "" {
		openidVal = ""
	} else {
		openidVal = openid
	}

	var sqlStr string
	if success {
		sqlStr = fmt.Sprintf(`
			INSERT INTO %s (mobile, openid, success_count, fail_count)
			VALUES (?, ?, 1, 0)
			ON DUPLICATE KEY UPDATE success_count = success_count + 1
		`, table)
	} else {
		sqlStr = fmt.Sprintf(`
			INSERT INTO %s (mobile, openid, success_count, fail_count)
			VALUES (?, ?, 0, 1)
			ON DUPLICATE KEY UPDATE fail_count = fail_count + 1
		`, table)
	}

	_, err := DB.Exec(sqlStr, mobile, openidVal)
	if err != nil {
		logger.Errorf("[mysql] 更新用户发送统计失败: table=%s mobile=%s openid=%v err=%v", table, mobile, openidVal, err)
		return err
	}

	logger.Infof("[mysql] 更新用户发送统计成功: table=%s mobile=%s openid=%v success=%v", table, mobile, openidVal, success)
	return nil
}

func UpdateUserOpenID(mobile, openid string) error {
	if mobile == "" || openid == "" {
		return fmt.Errorf("mobile 和 openid 不能为空")
	}

	sqlStr := `
	UPDATE push_user_stat 
	SET openid = ? 
	WHERE mobile = ? AND (openid IS NULL OR openid = '')
	`
	res, err := DB.Exec(sqlStr, openid, mobile)
	if err != nil {
		logger.Errorf("[mysql] 更新openid失败: mobile=%s openid=%s err=%v", mobile, openid, err)
		return err
	}

	n, _ := res.RowsAffected()
	if n > 0 {
		logger.Infof("[mysql] 成功更新openid: mobile=%s openid=%s", mobile, openid)
	}
	return nil
}

func InitAppIDTable(table string) error {
	createSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		mobile VARCHAR(20) NOT NULL,
		openid VARCHAR(100) NOT NULL DEFAULT '',
		success_count BIGINT NOT NULL DEFAULT 0,
		fail_count BIGINT NOT NULL DEFAULT 0,
		UNIQUE KEY uniq_mobile_openid (mobile, openid)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`, table)

	_, err := DB.Exec(createSQL)
	if err != nil {
		logger.Errorf("[mysql] 创建 appid 表失败: %s, err=%v", table, err)
	}
	return err
}

func StoreFailReasonWithAppID(ts time.Time, reason string, count int64, appid string) error {
	table := "push_fail_reason"
	if appid != "" {
		table = fmt.Sprintf("push_fail_reason_%s", appid)
		// 确保表存在
		if err := InitAppIDFailReasonTable(table); err != nil {
			return err
		}
	}

	_, err := DB.Exec(
		fmt.Sprintf(`INSERT INTO %s (stat_time, reason, count) VALUES (?, ?, ?)`, table),
		ts, reason, count,
	)
	if err != nil {
		logger.Errorf("[mysql] 插入失败原因失败: table=%s reason=%s count=%d err=%v", table, reason, count, err)
	}
	return err
}

func InitAppIDFailReasonTable(table string) error {
	createSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		stat_time DATETIME NOT NULL,
		reason VARCHAR(255) NOT NULL,
		count BIGINT NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`, table)
	_, err := DB.Exec(createSQL)
	if err != nil {
		logger.Errorf("[mysql] 创建失败原因表失败: table=%s err=%v", table, err)
	}
	return err
}

// UpdateUserOpenIDWithAppID 支持 AppID 的 openid 更新
func UpdateUserOpenIDWithAppID(mobile, openid, appid string) error {
	if mobile == "" || openid == "" {
		return fmt.Errorf("mobile 和 openid 不能为空")
	}

	table := "push_user_stat"
	if appid != "" {
		table = fmt.Sprintf("push_user_stat_%s", appid)
		if err := InitAppIDTable(table); err != nil {
			return err
		}
	}

	sqlStr := fmt.Sprintf(`
	UPDATE %s
	SET openid = ?
	WHERE mobile = ? AND (openid IS NULL OR openid = '')
	`, table)

	res, err := DB.Exec(sqlStr, openid, mobile)
	if err != nil {
		logger.Errorf("[mysql] 更新openid失败: table=%s mobile=%s openid=%s err=%v", table, mobile, openid, err)
		return err
	}

	n, _ := res.RowsAffected()
	if n > 0 {
		logger.Infof("[mysql] 成功更新openid: table=%s mobile=%s openid=%s", table, mobile, openid)
	}
	return nil
}

func InitAppIDStatTable(appid string) error {
	table := fmt.Sprintf("push_stat_%s", appid)
	createSQL := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		stat_time DATETIME NOT NULL,
		success_count BIGINT NOT NULL,
		fail_count BIGINT NOT NULL,
		UNIQUE KEY uniq_stat_time (stat_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`, table)

	_, err := DB.Exec(createSQL)
	if err != nil {
		logger.Errorf("[mysql] 创建 AppID 统计表失败: %s, err=%v", table, err)
	}
	return err
}

// UpdatePushStatWithAppID 按分钟统计消息成功/失败数，支持 AppID
func UpdatePushStatWithAppID(ts time.Time, success bool, appid string) error {
	// 时间按分钟截断
	minute := ts.Truncate(time.Minute)

	table := "push_stat"
	if appid != "" {
		table = fmt.Sprintf("push_stat_%s", appid)
		if err := InitAppIDStatTable(appid); err != nil {
			return err
		}
	}

	var sqlStr string
	if success {
		sqlStr = fmt.Sprintf(`
			INSERT INTO %s (stat_time, success_count, fail_count)
			VALUES (?, 1, 0)
			ON DUPLICATE KEY UPDATE success_count = success_count + 1
		`, table)
	} else {
		sqlStr = fmt.Sprintf(`
			INSERT INTO %s (stat_time, success_count, fail_count)
			VALUES (?, 0, 1)
			ON DUPLICATE KEY UPDATE fail_count = fail_count + 1
		`, table)
	}

	_, err := DB.Exec(sqlStr, minute)
	if err != nil {
		logger.Errorf("[mysql] 更新 push_stat 失败: table=%s time=%s success=%v err=%v",
			table, minute.Format("2006-01-02 15:04"), success, err)
		return err
	}

	logger.Infof("[mysql] 更新 push_stat 成功: table=%s time=%s success=%v",
		table, minute.Format("2006-01-02 15:04"), success)
	return nil
}
