package app

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"os"
)

var MySQL *sql.DB

// 初始化数据库连接.
func InitDB() {
	glog.Info("Connecting DB....")

	var err error
	MySQL, err = sql.Open("mysql", Conf.AppDBURL)

	if nil != err {
		glog.Error(err)
		os.Exit(-1)
	}

	MySQL.SetMaxIdleConns(100)
	MySQL.SetMaxOpenConns(500)

	glog.Info("DB connected")
}

// 关闭数据库连接.
func CloseDB() {
	MySQL.Close()
}
