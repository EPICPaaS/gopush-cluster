package app

import (
	"encoding/json"
	myrpc "github.com/EPICPaaS/gopush-cluster/rpc"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"strings"
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

// 客户端设备发送消息
func (device) Push(w http.ResponseWriter, r *http.Request) {
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
		baseRes["ret"] = ParamErr
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

	// TODO: token 校验

	//baseReq := args["baseRequest"].(map[string]interface{})
	msg := args["msg"].(map[string]interface{})

	// TODO: 发送信息验证（发送人是否合法、消息内容是否合法）
	// TODO: 好友关系校验（不是好友不能发等业务校验）

	// 消息过期时间（单位：秒）
	exp := msg["expire"]
	expire := 600
	if nil != exp {
		expire = int(exp.(float64))
	}

	toUserName := msg["toUserName"].(string)

	toUserIds := getToUserIds(toUserName)

	// 群组发送时接收端看到的发送人应该是XXX群组
	if len(toUserIds) > 1 {
		msg["fromUserName"] = toUserName
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		baseRes["ret"] = ParamErr
		glog.Error(err)
		return
	}

	for _, uid := range toUserIds {
		// uid 就是 gopush 的 key
		key := uid

		node := myrpc.GetComet(key)
		if node == nil || node.CometRPC == nil {
			baseRes["ret"] = NotFoundServer
			return
		}

		client := node.CometRPC.Get()
		if client == nil {
			baseRes["ret"] = NotFoundServer
			return
		}

		pushArgs := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msgBytes), Expire: uint(expire), Key: key}
		ret := 0
		if err := client.Call(myrpc.CometServicePushPrivate, pushArgs, &ret); err != nil {
			glog.Errorf("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivate, args, err)
			baseRes["ret"] = InternalErr

			// 失败不立即返回，继续下一个推送
		}
	}

	// 返回 key、token
	res["msgID"] = "msgid"
	res["clientMsgId"] = msg["clientMsgId"]

	return
}

func getToUserIds(toUserName string) []string {
	// TODO: 如果 toUserName 是群组，则进行群组->用户解析

	ret := []string{"toUserId1"}
	if strings.Contains(toUserName, "@room") {
		ret = append(ret, "toUserId2")
	}

	return ret
}
