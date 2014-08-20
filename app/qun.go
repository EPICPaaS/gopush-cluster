package app

import (
	"encoding/json"
	"github.com/EPICPaaS/go-uuid/uuid"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// 群插入 SQL
	QunInsertSQL = "INSERT INTO `qun` (`id`, `name`, `description`, `max_member`, `avatar`, `created`, `updated`) VALUES " +
		"(?, ?, ?, ?, ?, ?, ?)"
	// 群-用户关联插入 SQL
	QunUserInsertSQL = "INSERT INTO `qun_user` (`id`, `qun_id`, `user_id`, `sort`, `role`, `created`, `updated`) VALUES " +
		"(?, ?, ?, ?, ?, ?, ?)"
)

// 群结构
type Qun struct {
	Id          string
	CreatorId   string
	Name        string
	Description string
	MaxMember   int
	Avatar      string
	Created     time.Time
	Updated     time.Time
}

// 群-用户关联结构
type QunUser struct {
	Id      string
	QunId   string
	UserId  string
	Sort    int
	Role    int
	Created time.Time
	Updated time.Time
}

// 创建群
func (device) CreateQun(w http.ResponseWriter, r *http.Request) {
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

	now := time.Now()

	creatorId := args["creatorId"].(string)
	topic := args["topic"].(string)

	qid := uuid.New() + "@qun"
	qun := Qun{Id: qid, CreatorId: creatorId, Name: topic, Description: "", MaxMember: 100, Avatar: "", Created: now, Updated: now}

	memberList := args["memberList"].([]map[string]interface{})
	qunUsers := []QunUser{}
	for _, member := range memberList {
		memberId := member["uid"].(string)

		qunUser := QunUser{Id: uuid.New(), QunId: qid, UserId: memberId, Sort: 0, Role: 0, Created: now, Updated: now}

		qunUsers = append(qunUsers, qunUser)
	}

	if createQun(qun, qunUsers) {
		glog.Infof("Created Qun [id=%s]", qid)
	} else {
		glog.Error("Create Qun faild")
	}

	res["ChatRoomName"] = qid

	return
}

func createQun(qun *Qun, users *[]QunUser) bool {
	tx, err := MySQL.Begin()

	if err != nil {
		glog.Error(err)

		return false
	}

	stmt, err := tx.Prepare(QunInsertSQL)

	if err != nil {
		glog.Error(err)

		if err := tx.Rollback(); err != nil {
			glog.Error(err)
		}

		return false
	}

	defer stmt.Close()

	// 创建群记录
	_, err = stmt.Exec(qun.Id, qun.CreatorId, qun.Name, qun.Description, qun.MaxMember, qun.Avatar, qun.Created, qun.Updated)
	if err != nil {
		glog.Error(err)

		if err := tx.Rollback(); err != nil {
			glog.Error(err)
		}

		return false
	}

	// 创建群成员关联

	if err := tx.Commit(); err != nil {
		glog.Error(err)

		return false
	}

	return true
}
