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

type MessageContentWrapper struct {
	UserName    string
	Name        string
	NickName    string
	ContentBody string
}

func NewMessageContentWrapper() *MessageContentWrapper {
	return &MessageContentWrapper{}
}

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

	baseReq := args["baseRequest"].(map[string]interface{})

	// Token 校验
	token := baseReq["token"].(string)

	user := getUserByToken(token)

	if nil == user {
		baseRes.Ret = AuthErr

		return
	}

	//baseReq := args["baseRequest"].(map[string]interface{})
	msg := args["msg"].(map[string]interface{})
	fromUserName := msg["fromUserName"].(string)
	fromUserID := fromUserName[:strings.Index(fromUserName, "@")]
	toUserName := msg["toUserName"].(string)
	toUserID := toUserName[:strings.Index(toUserName, "@")]

	if strings.HasSuffix(toUserName, USER_SUFFIX) { // 如果是推人
		m := getUserByUid(fromUserID)

		msg["fromDisplayName"] = m.NickName
	} else if strings.HasSuffix(toUserName, QUN_SUFFIX) { // 如果是推群
		m := getUserByUid(fromUserID)

		qun, err := getQunById(toUserID)

		if nil != err {
			baseRes.Ret = InternalErr

			return
		}

		msg["content"] = fromUserName + "|" + m.Name + "|" + m.NickName + "&&" + msg["content"].(string)
		msg["fromDisplayName"] = qun.Name
		msg["fromUserName"] = toUserName
	} // TODO: 组织机构（部门/单位）推送消息体处理

	// TODO: 发送信息验证（发送人是否合法、消息内容是否合法）
	// TODO: 好友关系校验（不是好友不能发等业务校验）

	// 消息过期时间（单位：秒）
	exp := msg["expire"]
	expire := 600
	if nil != exp {
		expire = int(exp.(float64))
	}

	// 获取推送目标用户 id 集
	toUserNames, _ := getToUserNames(toUserName)

	// 推送分发
	for _, userName := range toUserNames {
		// userName 就是 gopush 的 key
		key := userName

		// 看到的接收人应该是具体的目标接收者
		msg["toUserName"] = userName

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			baseRes.Ret = ParamErr
			glog.Error(err)

			return
		}

		result := push(key, msgBytes, expire)
		if OK != result {
			baseRes.Ret = result

			glog.Errorf("Push message failed [%v]", msg)

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

	glog.V(3).Infof("Pushed a message to [key=%s]", key)

	return ret
}

// 根据 toUserName 获得最终推送的 name 集.
func getToUserNames(toUserName string) (userNames []string, pushType string) {
	if strings.HasSuffix(toUserName, QUN_SUFFIX) { // 群推
		qunId := toUserName[:len(toUserName)-len(QUN_SUFFIX)]

		glog.Info(qunId)
		userNames, err := getUserNamesInQun(qunId)

		if nil != err {
			return []string{}, QUN_SUFFIX
		}

		return userNames, QUN_SUFFIX
	} else if strings.HasSuffix(toUserName, ORG_SUFFIX) { // 组织机构部门推
		orgId := toUserName[:len(toUserName)-len(ORG_SUFFIX)]

		users := getUserListByOrgId(orgId)

		if nil == users {
			return []string{}, ORG_SUFFIX
		}

		userNames := []string{}
		for _, user := range users {
			userNames = append(userNames, user.UserName)
		}

		return userNames, ORG_SUFFIX
	} else if strings.HasSuffix(toUserName, TENANT_SUFFIX) { // 组织机构单位推
		tenantId := toUserName[:len(toUserName)-len(TENANT_SUFFIX)]

		users := getUserListByTenantId(tenantId)

		if nil == users {
			return []string{}, TENANT_SUFFIX
		}

		userNames := []string{}
		for _, user := range users {
			userNames = append(userNames, user.UserName)
		}

		return userNames, TENANT_SUFFIX
	} else if strings.HasSuffix(toUserName, USER_SUFFIX) { // 用户推
		return []string{toUserName}, USER_SUFFIX
	} else if strings.HasSuffix(toUserName, APP_SUFFIX) { // 应用推
		// TODO: 应用推
		return []string{}, APP_SUFFIX
	} else {
		return []string{}, "@UNDEFINDED"
	}
}
