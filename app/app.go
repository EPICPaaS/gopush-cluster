package app

import (
	"fmt"
	"net/http"
)

// 定义应用端操作结构
type app struct{}

// 声明应用端操作接口
var App = app{}

// 用户二维码处理，返回用户信息 HTML.
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
		// TODO: 完善显示用户信息
		fmt.Fprintln(w, user.NickName)
	}
}
