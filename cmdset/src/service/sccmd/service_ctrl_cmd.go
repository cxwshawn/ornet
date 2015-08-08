package sccmd

import (
	"cmdproto"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"service/cmdlog"
	"service/cmds"
)

type ScCmdConfig struct {
	ScRequestPoolSize int `toml:"request_pool_size"`
}

type ServiceCtrlCmd struct {
	*ScCmdConfig
	cmdHandlers map[string]func(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error)
	cmdReqPool  chan *cmdproto.ScRequest
}

func (scc *ServiceCtrlCmd) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmdlog.Printf("%s\n", string(data))

	defer req.Body.Close()
	scReq := scc.getAvalibleReq()

	defer scc.recycle(scReq)
	err = json.Unmarshal(data, scReq)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if handler, ok := scc.cmdHandlers[scReq.Op]; ok {
		res, err := handler(scc, scReq)
		if err != nil {
			cmdlog.EPrintf("%s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		//fmt.Fprintln(w, error)
		switch res.(type) {
		case string:
			fmt.Fprintln(w, res.(string))
		case []byte:
			fmt.Fprintln(w, string(res.([]byte)))
		case []string:
			fmt.Fprintln(w, res.([]string))
		case int:
			fmt.Println(w, res.(int))
		}
	} else {
		cmdlog.EPrintln("method not implemented")
		http.Error(w, fmt.Sprintf("server do not support command %s", scReq.Op), http.StatusNotImplemented)
		return
	}
}

func (scc *ServiceCtrlCmd) ConfigStruct() interface{} {
	return &ScCmdConfig{ScRequestPoolSize: 100}
}

func (scc *ServiceCtrlCmd) Init(config interface{}) (err error) {
	scc.ScCmdConfig = config.(*ScCmdConfig)
	cmdlog.Printf("ServiceCtrlCmd Init config :(%+v)\n", scc.ScCmdConfig)

	scc.cmdHandlers = make(map[string]func(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error))
	scc.cmdReqPool = make(chan *cmdproto.ScRequest, scc.ScRequestPoolSize)
	//allocates fixed size request pool.
	for i := 0; i < scc.ScRequestPoolSize; i++ {
		scc.cmdReqPool <- new(cmdproto.ScRequest)
	}
	//todo:
	scc.register("start", startHandler)
	scc.register("stop", stopHandler)
	scc.register("monitor", monitorHandler)
	cmdlog.Printf("ServiceCtrlCmd Init ok\n")
	return nil
}

func (scc *ServiceCtrlCmd) register(cmd string, handler func(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error)) {
	if _, ok := scc.cmdHandlers[cmd]; ok {
		cmdlog.EPrintf("duplicate serive ctrl cmd %s handler registered!\n", cmd)
		return
	}

	scc.cmdHandlers[cmd] = handler
}

func startHandler(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error) {
	return nil, nil
}

func stopHandler(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error) {
	return nil, nil
}

func monitorHandler(scc *ServiceCtrlCmd, req *cmdproto.ScRequest) (interface{}, error) {
	return nil, nil
}
func (scc *ServiceCtrlCmd) getAvalibleReq() *cmdproto.ScRequest {
	return <-scc.cmdReqPool
}

func (scc *ServiceCtrlCmd) recycle(scReq *cmdproto.ScRequest) {
	scc.cmdReqPool <- scReq
}

func init() {
	scHandler := &ServiceCtrlCmd{}
	cmds.RegisterCmd("sctl", scHandler)
}
