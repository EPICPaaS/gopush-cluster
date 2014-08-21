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

// 客户端设备推送消息.
// 1. 单推
// 2. 群推
// 3. 组织机构推
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

	// 获取推送目标用户 id 集
	toUserIds := getToUserIds(toUserName)

	// 多推时接收端看到的发送人应该是 XXX 群/组织机构
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
			glog.Errorf("Get comet node failed [key=%s]", key)
			baseRes["ret"] = NotFoundServer

			// 推送分发过程中失败不立即返回，继续下一个推送，下同
		}

		client := node.CometRPC.Get()
		if client == nil {
			glog.Errorf("Get comet node RPC client failed [key=%s]", key)
			baseRes["ret"] = NotFoundServer
		}

		pushArgs := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msgBytes), Expire: uint(expire), Key: key}
		ret := 0
		if err := client.Call(myrpc.CometServicePushPrivate, pushArgs, &ret); err != nil {
			glog.Errorf("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivate, args, err)
			baseRes["ret"] = InternalErr
		}
	}

	// 返回 key、token
	res["msgID"] = "msgid"
	res["clientMsgId"] = msg["clientMsgId"]

	return
}

// 根据 toUserName 获得最终推送的 uid 集.
func getToUserIds(toUserName string) []string {
	if strings.HasSuffix(toUserName, QUN_SUFFIX) { // 群推
		userIds, err := getUserIdsInQun(toUserName)
		if nil != err {
			return []string{}
		}

		return userIds
	} else if strings.HasSuffix(toUserName, ORG_SUFFIX) { // 组织机构全部门推
		users, err := GetUserListByOrgId(toUserName)
		if nil != err {
			return []string{}
		}

		userIds := []string{}
		for _, user := range users {
			userIds = append(userIds, user.Uid)
		}

		return userIds
	} else if strings.HasSuffix(toUserName, TENANT_SUFFIX) { // 组织机构全单位推
		users, err := GetUserListByTenantId(toUserName)
		if nil != err {
			return []string{}
		}

		userIds := []string{}
		for _, user := range users {
			userIds = append(userIds, user.Uid)
		}

		return userIds
	} else { // 单推
		return []string{toUserName}
	}
}
