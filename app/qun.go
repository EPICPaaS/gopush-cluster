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
	// 群插入 SQL.
	InsertQunSQL = "INSERT INTO `qun` (`id`, `creator_id`, `name`, `description`, `max_member`, `avatar`, `created`, `updated`) VALUES " +
		"(?, ?, ?, ?, ?, ?, ?, ?)"
	// 群-用户关联插入 SQL.
	InsertQunUserSQL = "INSERT INTO `qun_user` (`id`, `qun_id`, `user_id`, `sort`, `role`, `created`, `updated`) VALUES " +
		"(?, ?, ?, ?, ?, ?, ?)"
	// 根据群 id 查询群内用户.
	SelectQunUserSQL = "SELECT `id`, `nickname`, `avatar`, `status` FROM `user` where `id` in (SELECT `user_id` FROM `qun_user` where `qun_id` = ?)"
	// 根据群 id 查询群内用户 id.
	SelectQunUserIdSQL = "SELECT `user_id` FROM `qun_user` where `qun_id` = ?"
)

// 群结构.
type Qun struct {
	Id          string    `json:"id"`
	CreatorId   string    `json:"creatorId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MaxMember   int       `json:"maxMember"`
	Avatar      string    `json:"avatar"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// 群-用户关联结构.
type QunUser struct {
	Id      string
	QunId   string
	UserId  string
	Sort    int
	Role    int
	Created time.Time
	Updated time.Time
}

// 创建群.
func (device) CreateQun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	baseRes := baseResponse{OK, ""}
	body := ""
	res := map[string]interface{}{"baseResponse": baseRes}
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

	now := time.Now()

	creatorId := args["creatorId"].(string)
	topic := args["topic"].(string)

	qid := uuid.New()
	qun := Qun{Id: qid, CreatorId: creatorId, Name: topic, Description: "", MaxMember: 100, Avatar: "", Created: now, Updated: now}

	memberList := args["memberList"].([]interface{})
	qunUsers := []QunUser{}
	for _, m := range memberList {
		member := m.(map[string]interface{})
		memberId := member["uid"].(string)

		qunUser := QunUser{Id: uuid.New(), QunId: qid, UserId: memberId, Sort: 0, Role: 0, Created: now, Updated: now}

		qunUsers = append(qunUsers, qunUser)
	}

	if createQun(&qun, qunUsers) {
		glog.Infof("Created Qun [id=%s]", qid)
	} else {
		glog.Error("Create Qun faild")
		baseRes.ErrMsg = "Create Qun faild"
		baseRes.Ret = InternalErr
	}

	res["ChatRoomName"] = qid + QUN_SUFFIX

	return
}

// 获取群成员.
func (device) GetUsersInQun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	baseRes := baseResponse{OK, ""}
	body := ""
	res := map[string]interface{}{"baseResponse": baseRes}
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

	qid := args["qid"].(string)

	members, err := getUsersInQun(qid)
	if err != nil {
		baseRes.ErrMsg = err.Error()
		baseRes.Ret = InternalErr
		return
	}

	// TODO: MemberList

	res["memberList"] = members

	return
}

// 数据库中插入群记录、群-用户关联记录.
func createQun(qun *Qun, qunUsers []QunUser) bool {
	tx, err := MySQL.Begin()

	if err != nil {
		glog.Error(err)

		return false
	}

	// 创建群记录
	_, err = tx.Exec(InsertQunSQL, qun.Id, qun.CreatorId, qun.Name, qun.Description, qun.MaxMember, qun.Avatar, qun.Created, qun.Updated)
	if err != nil {
		glog.Error(err)

		if err := tx.Rollback(); err != nil {
			glog.Error(err)
		}

		return false
	}

	// 创建群成员关联
	for _, qunUser := range qunUsers {
		_, err = tx.Exec(InsertQunUserSQL, qunUser.Id, qunUser.QunId, qunUser.UserId, qunUser.Sort, qunUser.Role, qunUser.Created, qunUser.Updated)

		if err != nil {
			glog.Error(err)

			if err := tx.Rollback(); err != nil {
				glog.Error(err)
			}

			return false
		}
	}

	if err := tx.Commit(); err != nil {
		glog.Error(err)

		return false
	}

	return true
}

// 在数据库中查询群内用户.
func getUsersInQun(qunId string) ([]member, error) {
	ret := []member{}

	rows, err := MySQL.Query(SelectQunUserSQL, qunId)
	if err != nil {
		glog.Error(err)

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		m := member{}

		if err := rows.Scan(&m.Uid, &m.NickName, &m.HeadImgUrl, &m.Status); err != nil {
			glog.Error(err)

			return nil, err
		}

		ret = append(ret, m)
	}

	if err := rows.Err(); err != nil {
		glog.Error(err)

		return nil, err
	}

	return ret, nil
}

// 在数据库中查询群内用户 id.
func getUserIdsInQun(qunId string) ([]string, error) {
	ret := []string{}

	rows, err := MySQL.Query(SelectQunUserIdSQL, qunId)
	if err != nil {
		glog.Error(err)

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var uid string

		if err := rows.Scan(&uid); err != nil {
			glog.Error(err)

			return nil, err
		}

		ret = append(ret, uid)
	}

	if err := rows.Err(); err != nil {
		glog.Error(err)

		return nil, err
	}

	return ret, nil
}
