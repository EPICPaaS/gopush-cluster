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

// 客户端设备发送消息.
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

	// 多推时接收端看到的发送人应该是XXX群/组织机构
	if len(toUserIds) > 1 {
		msg["fromUserName"] = toUserName
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		baseRes["ret"] = ParamErr
		glog.Error(err)
		return
	}

	// 推送分发
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

// 根据 toUserName 获得最终推送的 uid 集.
func getToUserIds(toUserName string) []string {
	ret := []string{""}

	if strings.HasSuffix(toUserName, "@qun") { // 群推
		// TODO: 群推
	} else if strings.HasSuffix(toUserName, "@org") { // 组织机构推
		// TODO: 组织机构推
	} else { // 单推
		ret = append(ret, toUserName)
	}

	return ret
}
