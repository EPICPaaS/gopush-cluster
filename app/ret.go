package app

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"time"
)

const (
	OK             = 0
	NotFoundServer = 1001
	ParamErr       = 65534
	InternalErr    = 65535
)

// retWrite marshal the result and write to client(get).
func RetWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, callback string, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		glog.Errorf("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := ""
	if callback == "" {
		// Normal json
		dataStr = string(data)
	} else {
		// Jsonp
		dataStr = fmt.Sprintf("%s(%s)", callback, string(data))
	}
	if n, err := w.Write([]byte(dataStr)); err != nil {
		glog.Errorf("w.Write(\"%s\") error(%v)", dataStr, err)
	} else {
		glog.V(1).Infof("w.Write(\"%s\") write %d bytes", dataStr, n)
	}
	glog.Infof("req: \"%s\", res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}

// retPWrite marshal the result and write to client(post).
func RetPWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, body *string, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		glog.Errorf("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := string(data)
	if n, err := w.Write([]byte(dataStr)); err != nil {
		glog.Errorf("w.Write(\"%s\") error(%v)", dataStr, err)
	} else {
		glog.V(1).Infof("w.Write(\"%s\") write %d bytes", dataStr, n)
	}
	glog.Infof("req: \"%s\", post: \"%s\", res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), *body, dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}
