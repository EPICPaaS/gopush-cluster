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
// 2. 群推（@qun）
// 3. 组织机构推（部门 @org，单位 @tenant）
func (device) Push(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	baseRes := baseResponse{OK, ""}
	body := ""
	res := map[string]interface{}{"baseResponse": &baseRes}
	defer RetPWriteJSON(w, r, res, &body, time.Now())

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		baseRes.Ret = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)

	var args map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &args); err != nil {
		baseRes.ErrMsg = err.Error()
		baseRes.Ret = ParamErr
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
	toUserIds, pushType := getToUserIds(toUserName)

	// 多推时接收端看到的发送人应该是 XXX 群/组织机构
	if pushType == QUN_SUFFIX || pushType == TENANT_SUFFIX || pushType == ORG_SUFFIX {
		msg["fromUserName"] = toUserName
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		baseRes.Ret = ParamErr
		glog.Error(err)

		return
	}

	// 推送分发
	for _, uid := range toUserIds {
		// uid 就是 gopush 的 key
		key := uid

		result := push(key, msgBytes, expire)
		if OK != result {
			baseRes.Ret = result

			// 推送分发过程中失败不立即返回，继续下一个推送
		}
	}

	res["msgID"] = "msgid"
	res["clientMsgId"] = msg["clientMsgId"]

	return
}

func push(key string, msgBytes []byte, expire int) int {
	node := myrpc.GetComet(key)

	if node == nil || node.CometRPC == nil {
		glog.Errorf("Get comet node failed [key=%s]", key)

		return NotFoundServer
	}

	client := node.CometRPC.Get()
	if client == nil {
		glog.Errorf("Get comet node RPC client failed [key=%s]", key)

		return NotFoundServer
	}

	pushArgs := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msgBytes), Expire: uint(expire), Key: key}

	ret := OK
	if err := client.Call(myrpc.CometServicePushPrivate, pushArgs, &ret); err != nil {
		glog.Errorf("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivate, string(msgBytes), err)

		return InternalErr
	}

	return ret
}

// 根据 toUserName 获得最终推送的 uid 集.
func getToUserIds(toUserName string) (userIds []string, pushType string) {
	if strings.HasSuffix(toUserName, QUN_SUFFIX) { // 群推
		userIds, err := getUserIdsInQun(toUserName)
		if nil != err {
			return []string{}, QUN_SUFFIX
		}

		return userIds, QUN_SUFFIX
	} else if strings.HasSuffix(toUserName, ORG_SUFFIX) { // 组织机构部门推
		users := getUserListByOrgId(toUserName)
		if nil == users {
			return []string{}, ORG_SUFFIX
		}

		userIds := []string{}
		for _, user := range users {
			userIds = append(userIds, user.Uid)
		}

		return userIds, ORG_SUFFIX
	} else if strings.HasSuffix(toUserName, TENANT_SUFFIX) { // 组织机构单位推
		users := getUserListByTenantId(toUserName)
		if nil == users {
			return []string{}, TENANT_SUFFIX
		}

		userIds := []string{}
		for _, user := range users {
			userIds = append(userIds, user.Uid)
		}

		return userIds, TENANT_SUFFIX
	} else { // 单推
		return []string{toUserName}, "@user"
	}
}
