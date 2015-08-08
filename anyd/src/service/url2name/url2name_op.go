package url2name

import (
	"encoding/json"
	"fmt"
	"gopkg.in/redis.v2"
	// "io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	// "path"
	"service/cmds"
	"service/dlog"
	"service/types"
	"strings"
	"time"
)

type Url2NameOpConfig struct {
	DownPath        string         `toml:"down_path"`
	StorePath       string         `toml:"store_path"`
	RequestPoolSize int            `toml:"request_pool_size"`
	EntryPoolSize   int            `toml:"url_pool_size"`
	RedisWriteCount int            `toml:"write_count"`
	RedisAddr       string         `toml:"redis_addr"`
	MaxDownloadTime types.Duration `toml:"max_download_time"`
}

type Url2NameOp struct {
	*Url2NameOpConfig
	opReqPool  chan *Url2NameRequest
	opHandlers map[string]HandlerFunc
	entryPool  chan *Url2NameEntry
}

type Url2NameEntry struct {
	AppName string `json:"name"`
	Url     string `json:"url"`
}
type Url2NameRequest struct {
	out http.ResponseWriter
	req *http.Request
	op  *Url2NameOp
}

type HandlerFunc func(req *Url2NameRequest)

const (
	Url2NameKey = "url2name"
	UrlTimeKey  = "urltime"
)

func (op *Url2NameOp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	if handler, ok := op.opHandlers[action]; ok {
		handler(unReq)
	} else {
		http.NotFound(w, req)
	}
}

func (op *Url2NameOp) ConfigStruct() interface{} {
	return &Url2NameOpConfig{DownPath: "/data/tmp", StorePath: "/data/store",
		RequestPoolSize: 100, EntryPoolSize: 100, RedisWriteCount: 4,
		RedisAddr: "127.0.0.1:6379", MaxDownloadTime: types.Duration{time.Hour * 2}}
}

func (op *Url2NameOp) Init(config interface{}) (err error) {
	dlog.Printf("Url2NameOp init...\n")
	op.Url2NameOpConfig = config.(*Url2NameOpConfig)
	op.opHandlers = make(map[string]HandlerFunc, 0)
	op.opReqPool = make(chan *Url2NameRequest, op.RequestPoolSize)
	for i := 0; i < op.RequestPoolSize; i++ {
		req := &Url2NameRequest{}
		op.opReqPool <- req
	}
	op.entryPool = make(chan *Url2NameEntry, op.EntryPoolSize)
	for i := 0; i < op.RedisWriteCount; i++ {
		go writeToRedis(op)
	}
	err = os.MkdirAll(op.DownPath, 0755)
	if err != nil {
		dlog.EPrintln(err.Error())
		return err
	}
	err = os.MkdirAll(op.StorePath, 0755)
	if err != nil {
		dlog.EPrintln(err.Error())
		return err
	}

	op.register("add", addHandler)
	op.register("rm", removeHandler)
	op.register("help", helpHandler)
	// http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	dlog.Printf("Url2NameOp init ok, config:(%+v)\n", op.Url2NameOpConfig)
	return nil
}

func (op *Url2NameOp) register(opName string, handler HandlerFunc) {
	if _, ok := op.opHandlers[opName]; ok {
		dlog.EPrintf("duplicate files op %s handler registered!\n", opName)
		return
	}
	op.opHandlers[opName] = handler
}

func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

func writeToRedis(op *Url2NameOp) {
	redisClient := redis.NewTCPClient(&redis.Options{Addr: op.RedisAddr,
		DialTimeout: time.Millisecond * 200})
	defer redisClient.Close()
	// var tr http.RoundTripper = &http.Transport{
	// 	Dial: TimeoutDialer(time.Second*5, op.MaxDownloadTime.Duration)}

	// httpClient := &http.Client{Transport: tr}
	for entry := range op.entryPool {
		urlWithoutScheme := strings.TrimLeft(entry.Url, "http://")

		getResponse := redisClient.HMGet(Url2NameKey, urlWithoutScheme)
		result, err := getResponse.Result()
		if err != nil {
			dlog.EPrintln(err.Error())
			continue
		}
		if len(result) != 0 && result[0] != nil {
			dlog.EPrintf("url %s duplicate! value:%s\n", urlWithoutScheme, result[0].(string))
			continue
		}

		// res, err := httpClient.Get(entry.Url)
		// if err != nil {
		// 	dlog.EPrintln(err.Error())
		// 	continue
		// }
		// if res.StatusCode != 200 {
		// 	dlog.EPrintf("download data from %s failed, error code :%d\n",
		// 		entry.Url, res.StatusCode)
		// 	continue
		// }
		// defer res.Body.Close()
		// downloadFileName := path.Join(op.DownPath, entry.AppName)
		// file, err := os.Create(downloadFileName)
		// if err != nil {
		// 	dlog.EPrintln(err.Error())
		// 	continue
		// }
		// defer file.Close()
		// written, err := io.Copy(file, res.Body)
		// if err != nil {
		// 	dlog.EPrintln(err.Error())
		// 	continue
		// }
		// if res.ContentLength >= 0 && res.ContentLength != written {
		// 	dlog.EPrintf("url %s body data not equal to content-length\n", entry.Url)
		// 	continue
		// }
		zaddResponse := redisClient.ZAdd(UrlTimeKey, redis.Z{Score: float64(time.Now().Unix()), Member: urlWithoutScheme})
		_, err = zaddResponse.Result()
		if err != nil {
			dlog.EPrintln(err.Error())
			continue
		}

		setResponse := redisClient.HMSet(Url2NameKey, urlWithoutScheme, entry.AppName)
		_, err = setResponse.Result()
		if err != nil {
			dlog.EPrintln(err.Error())
			continue
		}
		// storeFileName := path.Join(op.StorePath, entry.AppName)
		// err = os.Rename(downloadFileName, storeFileName)
		// if err != nil {
		// 	dlog.EPrintln(err.Error())
		// }
		redisClient.Save()
	}
}

func removeHandler(unReq *Url2NameRequest) {
	req := unReq.req
	w := unReq.out
	op := unReq.op
	if req.Method != "POST" {
		dlog.EPrintf("invalid request method.\n")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entry := &Url2NameEntry{}
	err = json.Unmarshal(data, entry)
	if err != nil {
		dlog.EPrintf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	urlWithoutScheme := strings.TrimLeft(entry.Url, "http://")
	redisClient := redis.NewTCPClient(&redis.Options{Addr: op.RedisAddr,
		DialTimeout: time.Millisecond * 200})
	defer redisClient.Close()

	// getResponse := redisClient.HMGet(Url2NameKey, urlWithoutScheme)
	// result, err := getResponse.Result()
	// if err != nil {
	// 	dlog.EPrintln(err.Error())
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// if len(result) == 0 || result[0] == nil {
	// 	dlog.EPrintf("url %s not found!\n", urlWithoutScheme)
	// 	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	// 	return
	// }
	// filename := result[0].(string)
	delResponse := redisClient.HDel(Url2NameKey, urlWithoutScheme)
	_, err = delResponse.Result()
	if err != nil {
		dlog.EPrintf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	zRemResponse := redisClient.ZRem(UrlTimeKey, urlWithoutScheme)
	_, err = zRemResponse.Result()
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redisClient.Save()
	// delFileName := path.Join(op.StorePath, filename)
	// err = os.Remove(delFileName)
	// if err != nil {
	// 	dlog.EPrintln(err.Error())
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	w.WriteHeader(http.StatusOK)
}

func addHandler(unReq *Url2NameRequest) {
	req := unReq.req
	w := unReq.out
	op := unReq.op
	if req.Method != "POST" {
		dlog.EPrintf("invalid request method.\n")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entry := &Url2NameEntry{}
	err = json.Unmarshal(data, entry)
	if err != nil {
		dlog.EPrintf(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	op.entryPool <- entry
	w.WriteHeader(http.StatusOK)
}

func helpHandler(unReq *Url2NameRequest) {
	w := unReq.out
	fmt.Fprintf(w, "/url2name/add")
}
func (op *Url2NameOp) Uninit() {
	close(op.entryPool)
}

func (op *Url2NameOp) getAvalibleReq() *Url2NameRequest {
	return <-op.opReqPool
}

func (op *Url2NameOp) recycle(req *Url2NameRequest) {
	op.opReqPool <- req
}

func init() {
	url2nameHandler := &Url2NameOp{}
	cmds.RegisterCmd("url2name", true, url2nameHandler)
}
