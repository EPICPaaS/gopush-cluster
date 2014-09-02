package app

import (
	"github.com/golang/glog"
	"time"
)

const (
	// 根据 id 查询应用记录.
	SelectApplicationById = "SELECT * FROM `app` WHERE id = ?"
)

// 应用结构.
type application struct {
	Id      string    `json:"id"`
	Name    string    `json:"name"`
	Token   string    `json:"token"`
	Type    int       `json:"type"`
	Status  int       `json:"status"`
	Sort    int       `json:"sort"`
	Level   int       `json:"level"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// 数据库中根据 id 查询应用记录.
func getApplication(appId string) (*application, error) {
	row := MySQL.QueryRow(SelectApplicationById, appId)

	application := application{}
	if err := row.Scan(&application.Id, &application.Name, &application.Token, &application.Type, &application.Status,
		&application.Sort, &application.Level, &application.Created, &application.Updated); err != nil {
		glog.Error(err)

		return nil, err
	}

	return &application, nil
}
