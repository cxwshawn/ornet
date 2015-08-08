package sandbox

/*
#cgo CFLAGS: -I/usr/local/openresty/luajit/include/luajit-2.1/
#cgo LDFLAGS: -L/root/myopensrc/ornet/anyd/src/service/sandbox -L/usr/local/openresty/luajit/lib/ -lluajit-5.1 -lsandbox
#include "sandbox.h"
#include <stdlib.h>
*/
import "C"
import (
	"io/ioutil"
	"net/http"
	"service/cmds"
	"service/dlog"
	"strings"
	"unsafe"
)

type SandboxOpConfig struct {
	RequestPoolSize int    `toml:"request_pool_size"`
	LuaFilename     string `toml:"lua_filename"`
}

type SandboxOp struct {
	*SandboxOpConfig
	opReqPool chan *SandBoxRequest
	luaCtx    unsafe.Pointer
}

type SandBoxRequest struct {
	w   http.ResponseWriter
	req *http.Request
	op  *SandboxOp
}

//export GetURIPath
//returned result need to be freed after no use.
func GetURIPath(ptr unsafe.Pointer) *C.char {
	reqCtx := (*SandBoxRequest)(ptr)
	req := reqCtx.req
	return C.CString(req.URL.Path)
}

//export ReadBodyData
//returned data need to be freed after no use.
func ReadBodyData(ptr unsafe.Pointer) (body *C.char, n int) {
	reqCtx := (*SandBoxRequest)(ptr)
	req := reqCtx.req
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		return C.CString("nil"), -1
	}

	return C.CString(string(data)), len(data)
}

//export WriteData
func WriteData(ptr unsafe.Pointer, data *C.char, n C.int) C.int {
	reqCtx := (*SandBoxRequest)(ptr)
	// req := reqCtx.req
	if reqCtx == nil {
		dlog.EPrintln("invalid request context pointer!")
		return -1
	}

	w := reqCtx.w
	written, err := w.Write([]byte(C.GoStringN(data, n)))
	if err != nil {
		dlog.EPrintln(err.Error())
		return -1
	}
	return C.int(written)
}

type HandlerFunc func(req *SandBoxRequest)

func (op *SandboxOp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	dlog.Printf("(%+v)\n", *req)
	pos := strings.LastIndex(req.URL.Path, "/")
	if pos == -1 {
		dlog.EPrintf("invalid request :%s\n", req.RequestURI)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	// action := req.URL.Path[pos+1:]
	unReq := op.getAvalibleReq()
	unReq.w = w
	unReq.req = req
	unReq.op = op
	defer func() {
		op.recycle(unReq)
		req.Body.Close()
	}()
	_, err := C.process_request(op.luaCtx, unsafe.Pointer(unReq))
	if err != nil {
		dlog.EPrintln(err.Error())
		return
	}
}

func (op *SandboxOp) ConfigStruct() interface{} {
	return &SandboxOpConfig{RequestPoolSize: 100}
}

func (op *SandboxOp) Init(config interface{}) (err error) {
	dlog.Printf("SandboxOp init...\n")
	op.SandboxOpConfig = config.(*SandboxOpConfig)
	op.opReqPool = make(chan *SandBoxRequest, op.RequestPoolSize)
	for i := 0; i < op.RequestPoolSize; i++ {
		req := &SandBoxRequest{}
		op.opReqPool <- req
	}
	// http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	op.luaCtx = C.init_lua()
	luaFilename := C.CString(op.LuaFilename)
	defer C.free(unsafe.Pointer(luaFilename))
	_, err = C.load_lua_file(op.luaCtx, luaFilename)
	if err != nil {
		dlog.EPrintln(err.Error())
		return err
	}
	dlog.Printf("SandboxOp init ok, config:(%+v)\n", op.SandboxOpConfig)
	return nil
}

func (op *SandboxOp) Uninit() {
	C.uninit(op.luaCtx)
}

func (op *SandboxOp) getAvalibleReq() *SandBoxRequest {
	return <-op.opReqPool
}

func (op *SandboxOp) recycle(req *SandBoxRequest) {
	op.opReqPool <- req
}

func init() {
	sandboxHandler := &SandboxOp{}
	cmds.RegisterCmd("sandbox", true, sandboxHandler)
}
