package app

import (
	"encoding/json"
	myrpc "github.com/EPICPaaS/gopush-cluster/rpc"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
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

	baseRes := map[string]interface{}{"Ret": OK, "ErrMsg": ""}
	body := ""
	res := map[string]interface{}{"BaseResponse": baseRes}
	defer RetPWrite(w, r, res, &body, time.Now())

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)

	var args map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &args); err != nil {
		baseRes["ErrMsg"] = err.Error()
		baseRes["Ret"] = ParamErr
		return
	}

	baseReq := args["BaseRequest"].(map[string]interface{})

	uid := int(baseReq["Uid"].(float64))
	deviceId := baseReq["DeviceID"]
	userName := args["userName"]
	password := args["password"]

	glog.V(1).Infof("Uid [%d], DeviceId [%s], userName [%s], password [%s]",
		uid, deviceId, userName, password)

	// TODO: 登录逻辑

	// 返回 key、token
	res["Uid"] = "ukey"
	res["Token"] = "utoken"

	return
}

// 客户端设备发送消息
func (device) Push(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	baseRes := map[string]interface{}{"Ret": OK, "ErrMsg": ""}
	body := ""
	res := map[string]interface{}{"BaseResponse": baseRes}
	defer RetPWrite(w, r, res, &body, time.Now())

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		baseRes["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)

	var args map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &args); err != nil {
		baseRes["ErrMsg"] = err.Error()
		baseRes["Ret"] = ParamErr
		return
	}

	// TODO: token 校验

	//baseReq := args["BaseRequest"].(map[string]interface{})
	msg := args["Msg"].(map[string]interface{})

	// TODO: 发送信息验证（发送人是否合法、消息内容是否合法）
	// TODO: 好友关系校验（不是好友不能发等业务校验）

	// TODO: 根据目标用户名获取 uid
	uid := "uid"

	// uid 就是 gopush 的 key
	key := uid

	node := myrpc.GetComet(key)
	if node == nil || node.CometRPC == nil {
		baseRes["Ret"] = NotFoundServer
		return
	}
	client := node.CometRPC.Get()
	if client == nil {
		baseRes["Ret"] = NotFoundServer
		return
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		baseRes["Ret"] = ParamErr
		glog.Error(err)
		return
	}

	// TODO: 消息过期时间由客户端指定？
	expire := 600 // 600s

	pushArgs := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msgBytes), Expire: uint(expire), Key: key}
	ret := 0
	if err := client.Call(myrpc.CometServicePushPrivate, pushArgs, &ret); err != nil {
		glog.Errorf("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivate, args, err)
		baseRes["Ret"] = InternalErr
		return
	}

	// 返回 key、token
	res["MsgID"] = "msgid"
	res["LocalID"] = msg["LocalID"]

	return
}
