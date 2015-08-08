package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/go-etcd/etcd"
	"net"
	"net/http"
	"os"
	"service/cmdlog"
	"service/cmds"
	_ "service/mgocmd"
	_ "service/sccmd"
	_ "service/syscmd"
	"strings"
)

var configFileName *string = flag.String("config", "cmdconf.toml", "cmdset server configuration file name.")

var cmddConfig *CmddConfig
var configClient *etcd.Client

func init() {
	flag.Parse()
}

func analyzeCmddConf(config map[string]toml.Primitive, md *toml.MetaData) error {
	if section, ok := config["cmdd"]; ok {
		err := md.PrimitiveDecode(section, cmddConfig)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
		fmt.Printf("(%+v)\n", *cmddConfig)
		return nil
	} else {
		fmt.Println("analyze cmddConfig failed")
		return errors.New("does not specify cmmd section configuration")
	}
}

func registerToConfigServer() error {
	substrs := strings.SplitN(cmddConfig.ListenAddr, ":", 2)
	ip, port := substrs[0], substrs[1]
	if ip == "" {
		conn, err := net.Dial("udp", "baidu.com:80")
		if err != nil {
			cmdlog.EPrintln(err.Error())
			return err
		}
		//fmt.Printf("client addr :%s\n", conn.LocalAddr().String())
		ip = strings.Split(conn.LocalAddr().String(), ":")[0] + ":" + port
		conn.Close()
	} else {
		ip = ip + ":" + port
	}
	//todo:register to config server.
	_, err := configClient.CreateDir(cmddConfig.ConfigDir, 0)
	if err != nil {
		cmdlog.EPrintln(err.Error())
	}
	_, err = configClient.Set((cmddConfig.ConfigDir + "/" + ip), ip, 0)
	if err != nil {
		cmdlog.EPrintln(err.Error())
		return err
	}

	return nil
}

func main() {
	if configFileName == nil {
		fmt.Println("without specifiy config file.")
		os.Exit(1)
	}
	config, md, err := LoadCmdConfig(*configFileName)
	if err != nil {
		os.Exit(1)
	}
	cmddConfig = &CmddConfig{BasicSvrConfig: BasicSvrConfig{ErrorLog: true, AccessLog: false},
		ListenAddr: ":9000", ConfigDir: "/cmds"}
	err = analyzeCmddConf(config, md)
	if err != nil {
		os.Exit(1)
	}

	err = cmdlog.InitLog("cmdServer", cmddConfig.ErrorLog, cmddConfig.AccessLog)
	if err != nil {
		os.Exit(1)
	}
	configClient = etcd.NewClient(cmddConfig.ConfigServers)
	err = registerToConfigServer()
	if err != nil {
		os.Exit(1)
	}

	err = cmds.InitHandlerConf(config, md)
	if err != nil {
		os.Exit(1)
	}

	cmdServer := &http.Server{Addr: cmddConfig.ListenAddr, Handler: cmds.CmdServerMux}
	err = cmdServer.ListenAndServe()
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
}
