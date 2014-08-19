package app

import (
	"container/list"
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"time"
)

type member struct {
	Uid         string
	UserName    string
	NickName    string
	HeadImgUrl  string
	MemberCount int
	MemberList  []member
	PYInitial   string
	PYQuanPin   string
	Status      string
	StarFriend  int
	parentId    string
	sort        int
}

// 客户端设备登录，返回 key 和身份 token.
func (device) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	baseRes := map[string]interface{}{"ret": OK, "errMsg": ""}
	body := ""
	res := map[string]interface{}{"baseResponse": baseRes}
	defer RetPWriteJSON(w, r, res, &body, time.Now())

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)

	var args map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &args); err != nil {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = ParamErr
		return
	}

	baseReq := args["baseRequest"].(map[string]interface{})

	uid := baseReq["uid"]
	deviceId := baseReq["deviceID"]
	userName := args["userName"]
	password := args["password"]

	glog.V(1).Infof("uid [%d], deviceId [%s], userName [%s], password [%s]",
		uid, deviceId, userName, password)

	// TODO: 登录逻辑

	// 返回 key、token
	res["uid"] = "ukey"
	res["token"] = "utoken"

	return
}

// 客户端设备登录，返回 key 和身份 token
func (device) GetOrgInfo(w http.ResponseWriter, r *http.Request) {
	//if r.Method != "POST" {
	//	http.Error(w, "Method Not Allowed", 405)
	//	return
	//}
	baseRes := map[string]interface{}{"Ret": OK, "ErrMsg": ""}
	body := ""
	res := map[string]interface{}{"baseResponse": baseRes}
	defer RetPWrite(w, r, res, &body, time.Now())

	//bodyBytes, err := ioutil.ReadAll(r.Body)
	//if err != nil {
	//	res["ret"] = ParamErr
	//	glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
	//	return
	//}
	//body = string(bodyBytes)

	//var args map[string]interface{}

	//if err := json.Unmarshal(bodyBytes, &args); err != nil {
	//	baseRes["ErrMsg"] = err.Error()
	//	baseRes["Ret"] = ParamErr
	//	return
	//}

	//baseReq := args["baseRequest"].(map[string]interface{})

	//uid := int(baseReq["Uid"].(float64))
	//deviceId := baseReq["DeviceID"]
	//userName := args["userName"]
	//password := args["password"]

	//glog.V(1).Infof("Uid [%d], DeviceId [%s], userName [%s], password [%s]",
	//	uid, deviceId, userName, password)

	//// TODO: 登录逻辑

	//// 返回 key、token
	//res["Uid"] = "ukey"
	//res["Token"] = "utoken"

	smt, err := MySQL.Prepare("select id, name,  parent_id, sort from unit where tenant_id=?")
	if smt != nil {
		defer smt.Close()
	} else {
		return
	}

	if err != nil {
		baseRes["ErrMsg"] = err.Error()
		baseRes["Ret"] = InternalErr
		return
	}

	row, err := smt.Query("testTanantId")
	if row != nil {
		defer row.Close()
	} else {
		return
	}

	data := list.New()
	for row.Next() {
		rec := new(member)
		row.Scan(&rec.Uid, &rec.NickName, &rec.parentId, &rec.sort)
		glog.Errorf("%v", *rec)
		data.PushBack(rec)
	}

	err = row.Err()
	if err != nil {
		baseRes["ErrMsg"] = err.Error()
		baseRes["Ret"] = InternalErr
		return
	}

	return
}
