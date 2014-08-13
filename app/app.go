package app

import (
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type device struct{}

var Device device

// 客户端设备登录，返回 key 和身份 token
func (device) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	body := ""
	res := map[string]interface{}{"ret": OK, "ErrMsg": ""}
	defer RetPWrite(w, r, res, &body, time.Now())

	// param
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)
	params, err := url.ParseQuery(body)
	if err != nil {
		glog.Errorf("url.ParseQuery(\"%s\") error(%v)", body, err)
		res["ret"] = ParamErr
		return
	}

	uid := params.Get("Uid")
	deviceId := params.Get("DeviceId")
	userName := params.Get("userName")
	password := params.Get("password")

	glog.V(5).Infof("Uid [%s], DeviceId [%s], userName [%s], password [%s]",
		uid, deviceId, userName, password)

	// TODO: 登录逻辑

	// 返回 key、token
	res["Uid"] = "ukey"
	res["Token"] = "utoken"

	return
}
