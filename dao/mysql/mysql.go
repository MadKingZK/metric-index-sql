package mysql

import (
	"fmt"
	"metric-index/conf"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var db *sqlx.DB

// Init 初始化mysql连接
func Init(cfg *conf.MySQLConfig) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)
	// 也可以使用MustConnect连接不成功就panic
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		zap.L().Error("connect DB failed, err:%v\n", zap.Error(err))
		return
	}
	db.SetMaxOpenConns(cfg.MaOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	return
}

// Close 关闭mysql连接
func Close() {
	_ = db.Close()
}

// Exec 执行sql
func Exec(sql string) (err error) {
	_, err = db.Exec(sql)
	return
}

// GetID 获取一条记录ID
func GetID(sql string) (ID int, err error) {
	err = db.Get(&ID, sql)
	return
}
