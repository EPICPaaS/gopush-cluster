// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"github.com/EPICPaaS/gopush-cluster/app"
	"github.com/golang/glog"
	"net"
	"net/http"
	"time"
)

const (
	httpReadTimeout = 30 //seconds
)

func init() {
	glog.Info("init http")
}

// StartHTTP start listen http.
func StartHTTP() {
	// external
	httpServeMux := http.NewServeMux()
	// 1.0
	httpServeMux.HandleFunc("/1/server/get", GetServer)
	httpServeMux.HandleFunc("/1/msg/get", GetOfflineMsg)
	httpServeMux.HandleFunc("/1/time/get", GetTime)
	// old
	httpServeMux.HandleFunc("/server/get", GetServer0)
	httpServeMux.HandleFunc("/msg/get", GetOfflineMsg0)
	httpServeMux.HandleFunc("/time/get", GetTime0)
	// internal
	httpAdminServeMux := http.NewServeMux()
	// 1.0
	httpAdminServeMux.HandleFunc("/1/admin/push/private", PushPrivate)
	httpAdminServeMux.HandleFunc("/1/admin/msg/del", DelPrivate)
	// old
	httpAdminServeMux.HandleFunc("/admin/push", PushPrivate)
	httpAdminServeMux.HandleFunc("/admin/msg/clean", DelPrivate)

	// 应用消息服务
	appAppServeMux := http.NewServeMux()
	appAppServeMux.HandleFunc("/app/client/device/login", app.Device.Login)
	appAppServeMux.HandleFunc("/app/client/device/push", app.Device.Push)
	appAppServeMux.HandleFunc("/app/client/device/create-qun", app.Device.CreateQun)

	for _, bind := range app.Conf.HttpBind {
		glog.Infof("start http listen addr:\"%s\"", bind)
		go httpListen(httpServeMux, bind)
	}

	for _, bind := range app.Conf.AdminBind {
		glog.Infof("start admin http listen addr:\"%s\"", bind)
		go httpListen(httpAdminServeMux, bind)
	}

	for _, bind := range app.Conf.AppBind {
		glog.Infof("start app http listen addr:\"%s\"", bind)
		go httpListen(appAppServeMux, bind)
	}
}

func httpListen(mux *http.ServeMux, bind string) {
	server := &http.Server{Handler: mux, ReadTimeout: httpReadTimeout * time.Second}
	l, err := net.Listen("tcp", bind)
	if err != nil {
		glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	if err := server.Serve(l); err != nil {
		glog.Errorf("server.Serve() error(%v)", err)
		panic(err)
	}
}
