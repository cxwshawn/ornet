package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type BasicSvrConfig struct {
	ErrorLog  bool `toml:"error_log"`
	AccessLog bool `toml:"access_log"`
}
type CmddConfig struct {
	BasicSvrConfig
	ListenAddr    string   `toml:"listen_addr"`
	ConfigServers []string `toml:"config_addrs"`
	ConfigDir     string   `toml:"config_dir"`
}

func LoadCmdConfig(configFileName string) (config map[string]toml.Primitive, md *toml.MetaData, err error) {
	exePath, err1 := exec.LookPath(os.Args[0])
	if err1 != nil {
		fmt.Println(err1.Error())
		return nil, nil, err1
	}
	exeFullPath, err1 := filepath.Abs(exePath)
	if err1 != nil {
		fmt.Println(err1.Error())
		return nil, nil, err1
	}
	configFullPath := fmt.Sprintf("%s/%s", filepath.Dir(exeFullPath), configFileName)

	file, err1 := os.Open(configFullPath)
	defer file.Close()

	if err1 != nil {
		fmt.Println(err1.Error())
		return nil, nil, err1
	}
	data, err1 := ioutil.ReadAll(file)
	if err1 != nil {
		fmt.Println(err1.Error())
		return nil, nil, err1
	}
	config = make(map[string]toml.Primitive)
	md1, err1 := toml.Decode(string(data), config)
	if err1 != nil {
		fmt.Println(err1.Error())
		return nil, nil, err1
	}
	md = &md1
	return config, md, nil
}

// func LoadCmdConfig(configFileName string) (config *CmdSrvConfig, err error) {
// 	exePath, err1 := exec.LookPath(os.Args[0])
// 	if err1 != nil {
// 		fmt.Println(err1.Error())
// 		return nil, err1
// 	}
// 	exeFullPath, err1 := filepath.Abs(exePath)
// 	if err1 != nil {
// 		fmt.Println(err1.Error())
// 		return nil, err1
// 	}
// 	configFullPath := fmt.Sprintf("%s/%s", filepath.Dir(exeFullPath), configFileName)

// 	file, err1 := os.Open(configFullPath)
// 	defer file.Close()

// 	if err1 != nil {
// 		fmt.Println(err1.Error())
// 		return nil, err1
// 	}
// 	data, err1 := ioutil.ReadAll(file)
// 	if err1 != nil {
// 		fmt.Println(err1.Error())
// 		return nil, err1
// 	}
// 	config = &CmdSrvConfig{BasicSvrConfig{true, false}, ":9000"}
// 	if config == nil {
// 		return config, nil
// 	}
// 	err1 = json.Unmarshal(data, config)
// 	if err1 != nil {
// 		fmt.Println(err1.Error())
// 		return nil, err1
// 	}
// 	return config, nil
// }
