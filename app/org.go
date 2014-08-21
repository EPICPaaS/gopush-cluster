package app

import (
	"container/list"
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

type member struct {
	Uid         string    `json:"uid"`
	UserName    string    `json:"userName"`
	NickName    string    `json:"nickName"`
	HeadImgUrl  string    `json:"headImgUrl"`
	MemberCount int       `json:"memberCount"`
	MemberList  []*member `json:"memberList"`
	PYInitial   string    `json:"pYInitial"`
	PYQuanPin   string    `json:"pYQuanPin"`
	Status      string    `json:"status"`
	StarFriend  int       `json:"starFriend"`
	parentId    string    `json:"parentId"`
	sort        int       `json:"sort"`
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

type members []*member

type BySort struct {
	memberList members
}

func (s BySort) Len() int { return len(s.memberList) }
func (s BySort) Swap(i, j int) {
	s.memberList[i], s.memberList[j] = s.memberList[j], s.memberList[i]
}

func (s BySort) Less(i, j int) bool {
	return s.memberList[i].sort < s.memberList[j].sort
}

func sortMemberList(lst []*member) {
	sort.Sort(BySort{lst})

	for _, rec := range lst {
		sort.Sort(BySort{rec.MemberList})
	}
}

func (device) GetUserListByTenantId(id string) members {
	smt, err := MySQL.Prepare("select id, name, nickname, status user where tenant_id=?")
	if smt != nil {
		defer smt.Close()
	} else {
		return nil
	}

	if err != nil {
		return nil
	}

	row, err := smt.Query(id)
	if row != nil {
		defer row.Close()
	} else {
		return nil
	}
	ret := members{}
	for row.Next() {
		rec := new(member)
		row.Scan(&rec.Uid, &rec.UserName, &rec.NickName, &rec.Status)
		ret = append(ret, rec)
	}

	return ret
}

func (device) GetUserListByOrgId(id string) members {
	smt, err := MySQL.Prepare("select id, name, nickname, status user where user_id in (select user_id from org_user where org_id=?)")
	if smt != nil {
		defer smt.Close()
	} else {
		return nil
	}

	if err != nil {
		return nil
	}

	row, err := smt.Query(id)
	if row != nil {
		defer row.Close()
	} else {
		return nil
	}
	ret := members{}
	for row.Next() {
		rec := new(member)
		row.Scan(&rec.Uid, &rec.UserName, &rec.NickName, &rec.Status)
		ret = append(ret, rec)
	}

	return ret
}

// 客户端设备登录，返回 key 和身份 token
func (device) GetOrgInfo(w http.ResponseWriter, r *http.Request) {
	//if r.Method != "POST" {
	//	http.Error(w, "Method Not Allowed", 405)
	//	return
	//}
	baseRes := map[string]interface{}{"ret": OK, "errMsg": ""}
	body := ""
	res := map[string]interface{}{"baseResponse": baseRes}
	defer RetPWriteJSON(w, r, res, &body, time.Now())

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

	smt, err := MySQL.Prepare("select id, name,  parent_id, sort from org where tenant_id=?")
	if smt != nil {
		defer smt.Close()
	} else {
		return
	}

	if err != nil {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
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
		rec.Uid = rec.Uid
		rec.UserName = rec.Uid + ORG_SUFFIX
		data.PushBack(rec)
	}
	err = row.Err()
	if err != nil {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	unitMap := map[string]*member{}
	for ele := data.Front(); ele != nil; ele = ele.Next() {
		rec := ele.Value.(*member)
		unitMap[rec.Uid] = rec
	}

	rootList := []*member{}
	for _, val := range unitMap {
		if val.parentId == "" {
			rootList = append(rootList, val)
		} else {
			parent := unitMap[val.parentId]
			if parent == nil {
				continue
			}
			parent.MemberList = append(parent.MemberList, val)
			parent.MemberCount++
		}
	}

	tenant := new(member)
	res["ognizationMemberList"] = tenant
	sortMemberList(rootList)
	tenant.MemberList = rootList
	smt, err = MySQL.Prepare("select id, code, name from tenant where id=?")
	if smt != nil {
		defer smt.Close()
	} else {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	if err != nil {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	row, err = smt.Query("testTanantId")
	if row != nil {
		defer row.Close()
	} else {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	data = list.New()
	for row.Next() {
		row.Scan(&tenant.Uid, &tenant.UserName, &tenant.NickName)
		tenant.UserName = tenant.Uid + TENANT_SUFFIX
		break
	}
	smt, err = MySQL.Prepare("select org_id from org_user where user_id=?")
	if smt != nil {
		defer smt.Close()
	} else {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	if err != nil {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	row, err = smt.Query("testuser")
	if row != nil {
		defer row.Close()
	} else {
		baseRes["errMsg"] = err.Error()
		baseRes["ret"] = InternalErr
		return
	}

	data = list.New()
	for row.Next() {
		userOgnization := ""
		row.Scan(&userOgnization)
		res["userOgnization"] = userOgnization
		break
	}

	res["starMemberCount"] = 2
	starMembers := make(members, 2)

	starMembers[0] = &member{Uid: "11222", UserName: "11222@USER", NickName: "hehe"}
	starMembers[1] = &member{Uid: "22233", UserName: "22233@USER", NickName: "haha"}
	res["starMemberList"] = starMembers
	return
}
