package mgocmd

import (
	"cmdproto"
	"encoding/json"
	"fmt"
	"gopkg.in/mgo.v2"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"service/cmdlog"
	"service/cmds"
	"sort"
	"sync/atomic"
	"time"
)

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type MgoCmdConfig struct {
	MgoAddrs          []string `toml:"addr"`
	MgoReqPoolSize    int      `toml:"request_pool_size"`
	MgoLRUGate        int      `toml:"lru_percent"`
	DiskCheckInterval duration `toml:"check_interval"`
}

type MgoCmd struct {
	*MgoCmdConfig
	cmdHandlers   map[string]func(mc *MgoCmd, req *cmdproto.MgoRequest) (interface{}, error)
	cmdReqPool    chan *cmdproto.MgoRequest
	mgoSession    *mgo.Session
	diskMonState  int32
	stopDiskMonCh chan bool
	dbMonState    int32
}

func (mc *MgoCmd) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmdlog.Printf("%s\n", string(data))

	defer req.Body.Close()
	mgoReq := mc.getAvalibleReq()

	defer mc.recycle(mgoReq)
	err = json.Unmarshal(data, mgoReq)
	if err != nil {
		cmdlog.EPrintf("%s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if handler, ok := mc.cmdHandlers[mgoReq.DBCmd.DBCmd]; ok {
		res, err := handler(mc, mgoReq)
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
		http.Error(w, fmt.Sprintf("server do not support command %s", mgoReq.DBCmd.DBCmd), http.StatusNotImplemented)
		return
	}
}

func (mc *MgoCmd) ConfigStruct() interface{} {
	return &MgoCmdConfig{
		MgoAddrs:          make([]string, 100), //some risk if config array exceed 100.
		MgoReqPoolSize:    100,
		MgoLRUGate:        20,
		DiskCheckInterval: duration{time.Minute * 30},
	}
}

func (mc *MgoCmd) Init(config interface{}) (err error) {
	mc.MgoCmdConfig = config.(*MgoCmdConfig)
	cmdlog.Printf("MgoCmd Init config :(%+v)\n", mc.MgoCmdConfig)

	mc.cmdHandlers = make(map[string]func(mc *MgoCmd, req *cmdproto.MgoRequest) (interface{}, error))
	mc.cmdReqPool = make(chan *cmdproto.MgoRequest, mc.MgoReqPoolSize)
	//allocates fixed size request pool.
	for i := 0; i < mc.MgoReqPoolSize; i++ {
		mgoReq := &cmdproto.MgoRequest{}
		mc.cmdReqPool <- mgoReq
	}
	dialInfo := &mgo.DialInfo{Addrs: mc.MgoAddrs, Timeout: (500 * time.Millisecond)}
	mc.mgoSession, err = mgo.DialWithInfo(dialInfo)
	if err != nil {
		cmdlog.EPrintln(err.Error())
		return err
	}
	mc.stopDiskMonCh = make(chan bool)
	mc.register("dbStats", statsHandler)
	mc.register("diskMonB", diskMonStartHandler)
	mc.register("diskMonE", diskMonEndHandler)
	mc.register("dbMonB", dbMonStartHandler)
	//mc.register("dbMonE", dbMonEndHandler)

	cmdlog.Printf("MgoCmd Init ok\n")
	return nil
}

func (mc *MgoCmd) register(cmd string, handler func(mc *MgoCmd, req *cmdproto.MgoRequest) (interface{}, error)) {
	if _, ok := mc.cmdHandlers[cmd]; ok {
		cmdlog.EPrintf("duplicate mongo cmd %s handler registered!\n", cmd)
		return
	}

	mc.cmdHandlers[cmd] = handler
}

func (mc *MgoCmd) getAvalibleReq() *cmdproto.MgoRequest {
	mgoReq := <-mc.cmdReqPool
	return mgoReq
}

func (mc *MgoCmd) recycle(mgoReq *cmdproto.MgoRequest) {
	mc.cmdReqPool <- mgoReq
}

func statsHandler(mc *MgoCmd, req *cmdproto.MgoRequest) (res interface{}, err error) {
	result := make(map[string]interface{})
	err = mc.mgoSession.DB(req.DB).Run(req.DBCmd.DBCmd, result)
	if err != nil {
		cmdlog.EPrintln(err.Error())
		return
	}
	res, err = json.Marshal(result)
	if err != nil {
		cmdlog.EPrintln(err.Error())
		return
	}
	return
}

func filterCollNames(collNames []string) map[string][]string {
	reg := regexp.MustCompile(`(?P<hid>.+)_\d+_\d+_\d+`)
	resNames := make(map[string][]string)
	for _, collName := range collNames {
		res := reg.FindStringSubmatch(collName)
		if res != nil {
			hid := res[1]
			if resNames[hid] == nil {
				resNames[hid] = make([]string, 0)
			}
			resNames[hid] = append(resNames[hid], collName)
		}
	}
	for _, v := range resNames {
		sort.Strings(v)
	}
	return resNames
}

func diskMon(mc *MgoCmd, req *cmdproto.MgoRequest) {
	ticker := time.NewTicker(mc.DiskCheckInterval.Duration)
	stop := false
	collDays := 0
	collName := ""
	percent := 100
	var err1 error
	atomic.StoreInt32(&mc.diskMonState, 1)
	for {
		select {
		case <-mc.stopDiskMonCh:
			stop = true
		case <-ticker.C:
			//check if need to lru.
			ds := cmds.DiskUsage("/")
			percent = int(float64(ds.Free) / float64(ds.All) * 100)
			if percent < mc.MgoLRUGate {
				//cmdlog.Printf("mongodb is lru...\n")
				//do delete collections.
				names, err := mc.mgoSession.DB(req.DB).CollectionNames()
				if err != nil {
					cmdlog.EPrintln(err.Error())
					continue
				}
				resNames := filterCollNames(names)
				for _, v := range resNames {
					collDays = len(v)
					for i := 0; i < collDays/3; i++ {
						collName = v[i]
						//cmdlog.Printf("mongodb drop collection %s\n", collName)
						err1 = mc.mgoSession.DB(req.DB).C(collName).DropCollection()
						if err1 != nil {
							cmdlog.EPrintln(err1.Error())
						}
					}
				}
			}
		}
		if stop == true {
			ticker.Stop()
			atomic.StoreInt32(&mc.diskMonState, 0)
			break
		}
	}
}

func diskMonStartHandler(mc *MgoCmd, req *cmdproto.MgoRequest) (res interface{}, err error) {
	monitorState := atomic.LoadInt32(&mc.diskMonState)
	if monitorState > 0 {
		return "disk monitor has started.", nil
	}
	go diskMon(mc, req)
	return "disk monitor started", nil
}

func diskMonEndHandler(mc *MgoCmd, req *cmdproto.MgoRequest) (res interface{}, err error) {
	mc.stopDiskMonCh <- true

	return "disk monitor stopped", nil
}

func dbMon(mc *MgoCmd, req *cmdproto.MgoRequest) {
	atomic.StoreInt32(&mc.dbMonState, 1)
	stop := false

	pingCount := 5
	errCount := 0
	var err error

	ticker := time.NewTicker(time.Second * 2)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&mc.dbMonState, 0)
	}()

	dialInfo := &mgo.DialInfo{Addrs: mc.MgoAddrs, Timeout: (10 * time.Second)}
	mgoSession, err := mgo.DialWithInfo(dialInfo)
	mgoSession.SetSocketTimeout(time.Second * 10)
	addrs, _ := net.InterfaceAddrs()
	if err != nil {
		cmdlog.EPrintln(err.Error())
		cmds.SendMail("mongod", fmt.Sprintf("[%s]---%+v---mongod ping failed, please notice!", time.Now().UTC().String(), addrs))
		return
	}

	for {
		select {
		case <-ticker.C:
			for i := 0; i < pingCount; i++ {
				err = mgoSession.Ping()
				if err != nil {
					cmdlog.EPrintln(err.Error())
					errCount++
				}
			}
			if errCount == pingCount {
				for i := 0; i < 5; i++ {
					cmds.SendMail("mongod", fmt.Sprintf("[%s]---%+v---mongod ping failed, please notice!", time.Now().UTC().String(), addrs))
				}
				errCount = 0
				stop = true
			}
		}
		if stop == true {
			break
		}
	}
}

func dbMonStartHandler(mc *MgoCmd, req *cmdproto.MgoRequest) (res interface{}, err error) {
	monitorState := atomic.LoadInt32(&mc.dbMonState)
	if monitorState > 0 {
		return "db monitor has started.", nil
	}
	go dbMon(mc, req)
	return "db monitor started", nil
}

// func dbMonEndHandler(mc *MgoCmd, req *cmdproto.MgoRequest) (res interface{}, err error) {
// 	mc.stopDbMonCh <- true
// 	return "db  monitor stopped", nil
// }

func init() {
	mgoHandler := &MgoCmd{}
	cmds.RegisterCmd("mongo", mgoHandler)
}
