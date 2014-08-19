package app

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"os"
)

var MySQL *sql.DB

func InitDB() {
	glog.Info("Connecting DB....")

	var err error
	// TODO: 数据库连接配置
	MySQL, err = sql.Open("mysql", Conf.AppDBURL)

	if nil != err {
		glog.Error(err)
		os.Exit(-1)
	}

	MySQL.SetMaxIdleConns(100)
	MySQL.SetMaxOpenConns(500)

	glog.Info("DB connected")
}

func CloseDB() {
	MySQL.Close()
}
