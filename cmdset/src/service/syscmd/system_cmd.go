package syscmd

import (
	"bufio"
	"cmdproto"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"service/cmdlog"
	"service/cmds"
	"strings"
	"sync/atomic"
	"time"
)

type SysCmdConfig struct {
	SysRequestPoolSize int `toml:"request_pool_size"`
	DiskLeftNotify     int `toml:"disk_left_notify"`
}

type SystemCmd struct {
	*SysCmdConfig
	cmdHandlers  map[string]func(sc *SystemCmd, req *cmdproto.SysRequest) (interface{}, error)
	cmdReqPool   chan *cmdproto.SysRequest
	diskMonCh    chan bool
	diskMonState int32
}

func (sc *SystemCmd) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmdlog.Printf("%s\n", string(data))

	defer req.Body.Close()
	sysReq := sc.getAvalibleReq()

	defer sc.recycle(sysReq)
	err = json.Unmarshal(data, sysReq)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	op := strings.ToLower(sysReq.Op)
	if handler, ok := sc.cmdHandlers[op]; ok {
		res, err := handler(sc, sysReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		switch res.(type) {
		case string:
			fmt.Fprintln(w, res.(string))
		case []byte:
			fmt.Fprintln(w, string(res.([]byte)))
		case []string:
			fmt.Fprintln(w, res.([]string))
		case int:
			fmt.Fprintln(w, res.(int))
		case io.ReadCloser:
			stdout := res.(io.ReadCloser)
			defer stdout.Close()
			reader := bufio.NewReader(stdout)
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					cmdlog.EPrintf("err:%s\n", err.Error())
					break
				}
				// fmt.Println("%s is executing...", sysReq.Op)
				fmt.Fprintf(w, "%s", string(line))
				w.(http.Flusher).Flush()
			}
		}
	} else {
		cmdlog.EPrintln("method not implemented")
		http.Error(w, fmt.Sprintf("server do not support command %s", sysReq.Op), http.StatusNotImplemented)
		return
	}
}

func (sc *SystemCmd) ConfigStruct() interface{} {
	return &SysCmdConfig{SysRequestPoolSize: 100,
		DiskLeftNotify: 10}
}

func (sc *SystemCmd) Init(config interface{}) (err error) {
	sc.SysCmdConfig = config.(*SysCmdConfig)
	cmdlog.Printf("SystemCmd Init config :(%+v)\n", sc.SysCmdConfig)

	sc.cmdHandlers = make(map[string]func(sc *SystemCmd, req *cmdproto.SysRequest) (interface{}, error))
	sc.cmdReqPool = make(chan *cmdproto.SysRequest, sc.SysRequestPoolSize)
	//allocates fixed size request pool.
	for i := 0; i < sc.SysRequestPoolSize; i++ {
		sc.cmdReqPool <- new(cmdproto.SysRequest)
	}
	sc.diskMonCh = make(chan bool)
	sc.diskMonState = 0

	//todo:
	sc.register("monitor", monitorHandler)
	sc.register("syscmd", syscmdHandler)
	cmdlog.Printf("SystemCmd Init ok\n")
	return nil
}

func diskMon(sc *SystemCmd) {
	ticker := time.NewTicker(time.Second * 10)
	atomic.StoreInt32(&sc.diskMonState, 1)
	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&sc.diskMonState, 0)
	}()

	stop := false
	addrs, _ := net.InterfaceAddrs()
	for {
		select {
		case <-ticker.C:
			diskInfo := cmds.DiskUsage("/")
			diskLeft := int(float64(diskInfo.Free) / float64(diskInfo.All) * 100)
			if diskLeft < sc.DiskLeftNotify {
				for i := 0; i < 5; i++ {
					cmds.SendMail("Disk Usage", fmt.Sprintf("[%s]---(%+v)---Disk left %d%%, please clear mongo data!", time.Now().UTC().String(), addrs, diskLeft))
				}
				stop = true
			}
		}
		if stop == true {
			break
		}
	}

	return
}

func monitorHandler(sc *SystemCmd, req *cmdproto.SysRequest) (interface{}, error) {
	resourceType := strings.ToLower(req.Args[0])
	if resourceType == "disk" {
		state := atomic.LoadInt32(&sc.diskMonState)
		if state > 0 {
			return "disk monitor has started", nil
		}
		go diskMon(sc)
	} else if resourceType == "mem" {
		//todo
	} else {
		//todo:
	}
	return "monitor start ok.", nil
}

func syscmdHandler(sc *SystemCmd, req *cmdproto.SysRequest) (interface{}, error) {
	cmdlog.Println(req)
	cmdType := strings.ToLower(req.Args[0])
	cmd := exec.Command(cmdType, req.Args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cmdlog.EPrintln(err.Error())
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cmdlog.EPrintln(err.Error())
		return nil, err
	}

	return stdout, nil
}
func (sc *SystemCmd) register(cmd string, handler func(sc *SystemCmd, req *cmdproto.SysRequest) (interface{}, error)) {
	if _, ok := sc.cmdHandlers[cmd]; ok {
		cmdlog.EPrintf("duplicate serive ctrl cmd %s handler registered!\n", cmd)
		return
	}

	sc.cmdHandlers[cmd] = handler
}

func (sc *SystemCmd) getAvalibleReq() *cmdproto.SysRequest {
	return <-sc.cmdReqPool
}

func (sc *SystemCmd) recycle(req *cmdproto.SysRequest) {
	sc.cmdReqPool <- req
}

func init() {
	scHandler := &SystemCmd{}
	cmds.RegisterCmd("syscmd", scHandler)
}
