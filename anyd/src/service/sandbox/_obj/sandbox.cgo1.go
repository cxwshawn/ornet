// Created by cgo - DO NOT EDIT

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:1
package sandbox

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:4

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:3
import (
//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:5
	"net"
	"net/http"
	"service/cmds"
	"service/dlog"
)

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:12

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:11
type SandboxOpConfig struct {
	RequestPoolSize `toml:"request_pool_size"`
	LuaFilename     `toml:"lua_filename"`
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:17

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:16
type SandboxOp struct {
	*SandboxOpConfig
	opReqPool chan *SandBoxRequest
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:22

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:21
type SandBoxRequest struct {
	w   http.ResponseWriter
	req *http.Request
	op  *SandboxOp
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:30

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:29
func GetURIPath(ptr unsafe.Pointer) *_Ctype_char {
	reqCtx := *SandBoxRequest(ptr)
	req := reqCtx.req
	return _Cfunc_CString(req.URL.Path)
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:38

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:37
func ReadBodyData(ptr unsafe.Pointer) (body *_Ctype_char, n int) {
	reqCtx := *SandBoxRequest(ptr)
	req := reqCtx.req
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		return _Cfunc_CString("nil"), -1
	}
//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:47

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:46
	return _Cfunc_CString(string(data)), len(data)
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:51

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:50
func WriteData(ptr unsafe.Pointer, data *_Ctype_char, n int) _Ctype_int {
	reqCtx := *SandBoxRequest(ptr)
	req := reqCtx.req
	w := reqCtx.w
	n, err := w.Write(_Cfunc_GoStringN(data, n))
	if err != nil {
		dlog.EPrintln(err.Error())
		return -1
	}
	return n
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:63

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:62
type HandlerFunc func(req *SandBoxRequest)

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:65

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:64
func (op *SandboxOp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	dlog.Printf("(%+v)\n", *req)
	pos := strings.LastIndex(req.URL.Path, "/")
	if pos == -1 {
		dlog.EPrintf("invalid request :%s\n", req.RequestURI)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	action := req.URL.Path[pos+1:]
	unReq := op.getAvalibleReq()
	unReq.out = w
	unReq.req = req
	unReq.op = op
	defer func() {
		op.recycle(unReq)
		req.Body.Close()
	}()
//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:88

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:87
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:90

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:89
func (op *SandboxOp) ConfigStruct() interface{} {
	return &SandboxOpConfig{RequestPoolSize: 100}
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:94

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:93
func (op *SandboxOp) Init(config interface{}) (err error) {
	dlog.Printf("SandboxOp init...\n")
	op.SandboxOpConfig = config.(*SandboxOpConfig)
	op.opReqPool = make(chan *SandBoxRequest, op.RequestPoolSize)
	for i := 0; i < op.RequestPoolSize; i++ {
		req := &SandBoxRequest{}
		op.opReqPool <- req
	}
//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:104

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:103
	dlog.Printf("SandboxOp init ok, config:(%+v)\n", op.SandboxOpConfig)
	return nil
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:108

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:107
func (op *SandboxOp) Uninit() {
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:111

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:110
func (op *SandboxOp) getAvalibleReq() *SandBoxRequest {
	return <-op.opReqPool
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:115

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:114
func (op *SandboxOp) recycle(req *SandBoxRequest) {
	op.opReqPool <- req
}

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:119

//line /root/myopensrc/ornet/anyd/src/service/sandbox/sandbox.go:118
func init() {
	sandboxHandler := &SandboxOp{}
	cmds.RegisterCmd("sandbox", true, sandboxHandler)
}
