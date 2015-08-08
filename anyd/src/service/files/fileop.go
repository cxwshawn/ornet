package files

import (
	"errors"
	"fmt"
	"gopkg.in/redis.v2"
	"io"
	// "mime"
	// "mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"service/cmds"
	"service/dlog"
	"strconv"
	"strings"
	"time"
)

const (
	ngx_fastcgi   = 1
	upload_direct = 2
)

type FileOpConfig struct {
	UploadPath      string `toml:"upload_path"`
	StorePath       string `toml:"store_path"`
	RequestPoolSize int    `toml:"request_pool_size"`
	UploadType      int    `toml:"upload_type"`
	RedisAddr       string `toml:"redis_addr"`
}

type HandlerFunc func(req *FileRequest)

type FileOp struct {
	*FileOpConfig
	opReqPool  chan *FileRequest
	opHandlers map[string]HandlerFunc
	client     *redis.Client
}

type FileRequest struct {
	out http.ResponseWriter
	req *http.Request
	op  *FileOp
}

type UploadFileInfo struct {
	FileName    string `name`
	UploadPath  string `path`
	FileSize    int    `size`
	FileMd5     string `md5`
	ContentType string `content_type`
}

func (fi *UploadFileInfo) setInfo(field string, val string) {
	if strings.HasSuffix(field, "md5") {
		fi.FileMd5 = val
	} else if strings.HasSuffix(field, "size") {
		filesize, err := strconv.Atoi(val)
		if err != nil {
			dlog.EPrintln(err.Error())
		}
		fi.FileSize = filesize
	} else if strings.HasSuffix(field, "path") {
		fi.UploadPath = val
	} else if strings.HasSuffix(field, "content_type") {
		fi.ContentType = val
	} else if strings.HasSuffix(field, "name") {
		fi.FileName = val
	} else {
		dlog.EPrintln("invalid SaveInfo filed type.")
	}
}

func exists(filepath string) error {
	_, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	return nil
}

func (fo *FileOp) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	dlog.Printf("(%+v)\n", *req)
	pos := strings.LastIndex(req.URL.Path, "/")
	if pos == -1 {
		dlog.EPrintf("invalid request :%s\n", req.RequestURI)
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	action := req.URL.Path[pos+1:]
	foReq := fo.getAvalibleReq()
	foReq.out = w
	foReq.req = req
	foReq.op = fo
	defer func() {
		fo.recycle(foReq)
		req.Body.Close()
	}()

	if handler, ok := fo.opHandlers[action]; ok {
		handler(foReq)
	} else {
		dlog.EPrintln("bad request!")
		http.NotFound(w, req)
	}
}

func (fo *FileOp) ConfigStruct() interface{} {
	return &FileOpConfig{UploadPath: "/data/tmp", StorePath: "/data/store", RequestPoolSize: 10,
		UploadType: ngx_fastcgi, RedisAddr: "127.0.0.1:6379"}
}

func (fo *FileOp) Init(config interface{}) (err error) {
	dlog.Printf("FileOp init...\n")
	fo.FileOpConfig = config.(*FileOpConfig)
	fo.opHandlers = make(map[string]HandlerFunc, 0)
	fo.opReqPool = make(chan *FileRequest, fo.RequestPoolSize)
	for i := 0; i < fo.RequestPoolSize; i++ {
		req := &FileRequest{}
		fo.opReqPool <- req
	}
	fo.client = redis.NewTCPClient(&redis.Options{Addr: fo.RedisAddr,
		DialTimeout: time.Millisecond * 100})
	val, err := fo.client.Ping().Result()
	if err != nil {
		dlog.EPrintln(err.Error())
		return err
	}
	if val != "PONG" {
		dlog.EPrintln("redis returned ping result invalid!")
		return errors.New("redis returned ping result invalid!")
	}

	fo.register("upload", uploadHandler)
	fo.register("download", downloadHandler)
	fo.register("ping", pingHandler)

	dlog.Printf("FileOp init ok, config:(%+v)\n", fo.FileOpConfig)
	return nil
}

func (fo *FileOp) Uninit() {
	//todo: free all the resources.
	fo.client.Close()
}

func (fo *FileOp) register(op string, handler HandlerFunc) {
	if _, ok := fo.opHandlers[op]; ok {
		dlog.EPrintf("duplicate files op %s handler registered!\n", op)
		return
	}
	fo.opHandlers[op] = handler
}

func (fo *FileOp) getAvalibleReq() *FileRequest {
	return <-fo.opReqPool
}

func (fo *FileOp) recycle(req *FileRequest) {
	fo.opReqPool <- req
}

func uploadHandler(foReq *FileRequest) {
	if foReq.op.UploadType == ngx_fastcgi {
		fastcgiUploadHandler(foReq)
	} else if foReq.op.UploadType == upload_direct {
		directUploadHandler(foReq)
	} else {
		dlog.Printf("invalid upload type!\n")
		http.Error(foReq.out, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

func directUploadHandler(foReq *FileRequest) {
	req := foReq.req
	w := foReq.out
	var fileSize int64
	if contentLen, ok := req.Header["Content-Length"]; ok {
		contentLength, err := strconv.Atoi(contentLen[0])
		if err != nil {
			dlog.EPrintln(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileSize = int64(contentLength)
	} else {
		dlog.EPrintf("invalid request header\n")
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}
	if fileSize <= 0 {
		dlog.EPrintf("invalid request header\n")
		http.Error(w, http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if filename, ok := values["filename"]; ok {
		tmpFilePath := path.Join(foReq.op.UploadPath, filename[0])
		localWriter, err := os.Create(tmpFilePath)
		if err != nil {
			dlog.EPrintln(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}
		// defer localWriter.Close()
		writeLen, err := io.CopyN(localWriter, req.Body, fileSize)
		if writeLen != fileSize {
			localWriter.Close()
			os.Remove(tmpFilePath)
			dlog.EPrintf("content size is not equal to content-length field value.")
			http.Error(w, "body size not equal to content-length field value.",
				http.StatusBadRequest)
			return
		}
		localWriter.Close()
		fullFilePath := path.Join(foReq.op.StorePath, filename[0])
		err = os.Rename(tmpFilePath, fullFilePath)
		if err != nil {
			os.Remove(tmpFilePath)
			dlog.EPrintf("rename file %s to file %s failed.", tmpFilePath, fullFilePath)
			http.Error(w, "rename file failed.", http.StatusInternalServerError)
			return
		}

		//todo:
	} else {
		dlog.EPrintf("request bad parameter\n")
		http.Error(w, "bad parameter", http.StatusBadRequest)
	}
}

func fastcgiUploadHandler(foReq *FileRequest) {
	req := foReq.req
	w := foReq.out
	// contentType, ret := req.Header["Content-Type"]
	// if ret == false {
	// 	dlog.EPrintf()
	// 	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	// 	return
	// }

	// mediaType, params, err := mime.ParseMediaType(contentType[0])
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// fullFilePath := ""
	// if strings.HasPrefix(mediaType, "multipart/") {
	// reader, err := req.MultipartReader()
	// if err != nil {
	// 	dlog.EPrintln(err.Error())
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// // mr := multipart.NewReader(req.Body, params["boundary"])
	// mf, err := reader.ReadForm(0)
	// if err != nil {
	// 	dlog.EPrintln(err.Error())
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filename, ok := values["filename"]
	if !ok {
		dlog.EPrintf("request without specify filename paramter!\n")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = req.ParseMultipartForm(32 * 1024)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var infoList []UploadFileInfo
	dlog.Printf("(%+v)\n", req.Form)
	for k, v := range req.Form {
		if infoList == nil {
			infoList = make([]UploadFileInfo, len(v))
		}
		for i := 0; i < len(v); i++ {
			infoList[i].setInfo(k, v[i])
		}
	}
	dlog.Printf("(%+v)\n", infoList)
	// fmt.Println(infoList)
	for _, fi := range infoList {
		fullFilePath := path.Join(foReq.op.StorePath, filename[0])
		err = os.Rename(fi.UploadPath, fullFilePath)
		dlog.Printf("rename file %s to file %s\n", fi.UploadPath, fullFilePath)
		if err != nil {
			dlog.EPrintf(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// }
	w.WriteHeader(http.StatusOK)
}

const (
	Url2NameKey = "url2name"
	UrlTimeKey  = "urltime"
)

func pingHandler(foReq *FileRequest) {
	w := foReq.out
	w.WriteHeader(http.StatusOK)
}

func downloadHandler(foReq *FileRequest) {
	dlog.Println("downloadHandler in")

	w := foReq.out
	req := foReq.req
	fo := foReq.op
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		dlog.EPrintln(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if reqUrl, ok := values["link"]; ok {
		if reqUrl[0] == "" {
			dlog.EPrintf("invalid link parameter,from remote %s, url:%s\n",
				req.RemoteAddr, req.URL.Path)
			http.Error(w, "invalid link parameter", http.StatusBadRequest)
			return
		}
		// realUrl, err := url.QueryUnescape(reqUrl[0])
		// if err != nil {
		// 	dlog.EPrintln(err.Error())
		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		hmResponse := fo.client.HMGet(Url2NameKey, reqUrl[0])
		result, err := hmResponse.Result()
		if err != nil {
			dlog.EPrintln(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(result) == 0 || result[0] == nil {
			dlog.EPrintf("url %s not found!\n", reqUrl[0])
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		//just update url visit time.
		fo.client.ZAdd(UrlTimeKey, redis.Z{Score: float64(time.Now().Unix()), Member: reqUrl[0]})
		if len(result) == 0 || result[0] == nil {
			dlog.EPrintf("url2name failed, url:%s\n", reqUrl[0])
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		filename := result[0].(string)
		// fmt.Println(filename)
		urlRes, err := url.Parse(reqUrl[0])
		if err != nil {
			dlog.EPrintln(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		realFilename := ""
		if path.Ext(filename) == "" {
			realFilename = fmt.Sprintf("%s%s", filename, path.Ext(urlRes.Path))
		} else {
			realFilename = filename
		}
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", realFilename))
		w.Header().Add("Content-Type", "application/octet-stream")
		fullFilePath := path.Join(foReq.op.StorePath, filename)
		if err = exists(fullFilePath); err != nil {
			fullFilePath = path.Join(foReq.op.StorePath, realFilename)
			if err = exists(fullFilePath); err != nil {
				dlog.EPrintf("file %s not found\n", fullFilePath)
				http.NotFound(w, req)
				return
			}
		}
		dlog.Printf("download file %s\n", fullFilePath)
		http.ServeFile(w, req, fullFilePath)
		//todo:
	} else {
		dlog.EPrintf("request bad parameter\n")
		http.Error(w, "bad parameter", http.StatusBadRequest)
	}
}

func init() {
	filesHandler := &FileOp{}

	cmds.RegisterCmd("files", true, filesHandler)
}
