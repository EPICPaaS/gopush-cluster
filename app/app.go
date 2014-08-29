package app

import (
	"fmt"
	"net/http"
)

// 定义移动端操作结构
type device struct{}

// 声明移动端操作接口
var Device = device{}

func UserErWeiMa(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	uid := ""
	if len(r.Form) > 0 {
		uid = r.Form.Get("id")
	}

	user := getUserByUid(uid)
	if nil == user {
		fmt.Fprintln(w, "")
	} else {
		fmt.Fprintln(w, user.NickName)
	}
}
