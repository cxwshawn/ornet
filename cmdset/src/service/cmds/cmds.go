package cmds

import (
	"github.com/BurntSushi/toml"
	"net/http"
	"net/smtp"
	"runtime/debug"
	"service/cmdlog"
	"strings"
	"syscall"
)

type CmdHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	Init(config interface{}) error
	ConfigStruct() interface{}
}

var CmdServerMux = http.NewServeMux()
var CmdHandlers map[string]CmdHandler

func init() {
	CmdHandlers = make(map[string]CmdHandler)
}

func RegisterCmd(name string, handler CmdHandler) {
	CmdHandlers[name] = handler
	pattern := "/" + name
	CmdServerMux.Handle(pattern, handler)
}

func InitHandlerConf(confs map[string]toml.Primitive, md *toml.MetaData) error {
	var err error
	for k, handler := range CmdHandlers {
		if conf, ok := confs[k]; ok {
			handlerConf := handler.ConfigStruct()
			err = md.PrimitiveDecode(conf, handlerConf)
			if err != nil {
				cmdlog.EPrintln(err.Error())
				return err
			}

			err = handler.Init(handlerConf)
			if err != nil {
				cmdlog.EPrintln(err.Error())
				return err
			}
		}
	}
	return err
}

func SafeHandler(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err, ok := recover().(error); ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				cmdlog.EPrintln("WARN: panic in %v - %v", handler, err)
				cmdlog.EPrintln(string(debug.Stack()))
			}
		}()
		handler.ServeHTTP(w, req)
	}
}

type DiskStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
}

// disk usage of path/disk
func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}

func SendMail(subject, body string) error {
	user := "cxw_eric@126.com"
	password := "abc123"
	host := "smtp.126.com:25"
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	content_type := "Content-Type: text/plain" + "; charset=UTF-8"
	to := "cxw_eric@126.com"
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	return err
}
