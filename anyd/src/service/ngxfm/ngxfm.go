package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/signal"
	"runtime"
	"service/cmds"
	"service/dlog"
	_ "service/files"
	_ "service/qianke"
	_ "service/sandbox"
	_ "service/url2name"
	// "sync"
)

var configFileName *string = flag.String("config", "fmdconf.toml", "nginx files manager server configuration file name.")
var fmConf *FmdConfig

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
}

func analyzeFmdConf(config map[string]toml.Primitive, md *toml.MetaData) error {
	if section, ok := config["ngxfmd"]; ok {
		err := md.PrimitiveDecode(section, fmConf)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("(%+v)\n", *fmConf)
		return nil
	} else {
		fmt.Println("analyze nginx files manager configuration failed")
		return errors.New("does not specify fmd section configuration")
	}
}
func main() {
	if configFileName == nil {
		fmt.Println("without specifiy config file.")
		os.Exit(1)
	}
	config, md, err := LoadFmdConfig(*configFileName)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmConf = &FmdConfig{BasicSvrConfig: BasicSvrConfig{ErrorLog: true, AccessLog: false},
		FastcgiListenAddr: ":11000", HttpListenAddr: ":11001"}
	err = analyzeFmdConf(config, md)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = dlog.InitLog("ngxfmd", fmConf.ErrorLog, fmConf.AccessLog)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = cmds.InitHandlerConf(config, md)
	if err != nil {
		log.Fatal(err.Error())
	}
	cmdServer := &http.Server{Addr: fmConf.HttpListenAddr, Handler: cmds.CmdServerMux}
	err = cmdServer.ListenAndServe()
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	// var wg sync.WaitGroup
	// wg.Add(1)
	go func() {
		cmdServer := &http.Server{Addr: fmConf.HttpListenAddr, Handler: cmds.CmdServerMux}
		err = cmdServer.ListenAndServe()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
		// wg.Done()
	}()
	// wg.Add(1)
	go func() {
		ln, err := net.Listen("tcp", fmConf.FastcgiListenAddr)
		if err != nil {
			log.Fatal(err.Error())
		}

		err = fcgi.Serve(ln, cmds.CmdServerMux)
		if err != nil {
			log.Fatal(err.Error())
		}
		// wg.Done()
	}()
	// wg.Wait()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	cmds.Uninit()
	dlog.Printf("Accept %s signal, quit server...\n", s.String())
}
