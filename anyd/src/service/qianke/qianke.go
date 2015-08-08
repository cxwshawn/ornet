package qianke

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"service/cmds"
	"service/dlog"
	"strings"
)

type qiankeOpConfig struct {
	RequestPoolSize int    `toml:"request_pool_size"`
	MysqlHost       string `toml:"mysql_host"`
	MysqlUsername   string `toml:"mysql_username"`
	MysqlPasswd     string `toml:"mysql_passw"`
	MysqlDbName     string `toml:"mysql_dbname"`
}

type qiankeOp struct {
	*qiankeOpConfig
	opReqPool  chan *qiankeRequest
	opHandlers map[string]HandlerFunc
	dbOp       *QkDBOp
}

type qiankeRequest struct {
	out http.ResponseWriter
	req *http.Request
	op  *qiankeOp
}

type HandlerFunc func(req *qiankeRequest)

func (op *qiankeOp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	pos := strings.LastIndex(req.URL.Path, "/")
	if pos == -1 {
		dlog.EPrintf("invalid request :%s\n", req.RequestURI)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	action := req.URL.Path[pos+1:]
	qkReq := op.getAvalibleReq()
	qkReq.out = w
	qkReq.req = req
	qkReq.op = op
	defer func() {
		op.recycle(qkReq)
		req.Body.Close()
	}()

	if handler, ok := op.opHandlers[action]; ok {
		handler(qkReq)
	} else {
		dlog.EPrintf("path %s handler not found!\n", req.URL.Path)
		http.NotFound(w, req)
	}
}

func (op *qiankeOp) ConfigStruct() interface{} {
	return &qiankeOpConfig{RequestPoolSize: 1000, MysqlHost: "127.0.0.1:3306",
		MysqlUsername: "root", MysqlPasswd: "d4cd390a", MysqlDbName: "qianke"}
}

func (op *qiankeOp) Init(config interface{}) (err error) {
	dlog.Println("qiankeOp init...")
	op.qiankeOpConfig = config.(*qiankeOpConfig)
	op.dbOp = new(QkDBOp)
	err = op.dbOp.Init(op.MysqlHost, op.MysqlUsername, op.MysqlPasswd, op.MysqlDbName)
	if err != nil {
		dlog.EPrintln(err.Error())
		return err
	}

	op.opReqPool = make(chan *qiankeRequest, op.RequestPoolSize)
	op.opHandlers = make(map[string]HandlerFunc, 0)
	for i := 0; i < op.RequestPoolSize; i++ {
		req := &qiankeRequest{}
		op.opReqPool <- req
	}

	op.register("login", loginHandler)
	op.register("register", registerHandler)

	dlog.Printf("qiankeOp init ok, config:%+v\n", op.qiankeOpConfig)
	return nil
}

func (op *qiankeOp) register(opName string, handler HandlerFunc) {
	if _, ok := op.opHandlers[opName]; ok {
		dlog.EPrintf("duplicate files op %s handler registered!\n", opName)
		return
	}
	op.opHandlers[opName] = handler
}

func loginHandler(qkReq *qiankeRequest) {
	req := qkReq.req
	w := qkReq.out
	// op := qkReq.op

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("data:%s\n", string(data))
	fmt.Printf("req:%+v\n", req)
	w.Write([]byte("test response."))
	return
}

func registerHandler(qkReq *qiankeRequest) {
	req := qkReq.req
	w := qkReq.out
	dbOp := qkReq.op.dbOp

	user := &QKUser{}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("data:%s\n", string(data))

	err = json.Unmarshal(data, user)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = dbOp.RegisterUser(user)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (op *qiankeOp) Uninit() {
	//todo: free resources.
	op.dbOp.Uninit()

}

func (op *qiankeOp) getAvalibleReq() *qiankeRequest {
	return <-op.opReqPool
}

func (op *qiankeOp) recycle(req *qiankeRequest) {
	op.opReqPool <- req
}

func init() {
	qiankeOpHandler := &qiankeOp{}
	cmds.RegisterCmd("qianke", true, qiankeOpHandler)
}
