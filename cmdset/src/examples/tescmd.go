package main

import (
	"fmt"
	//	"os"
	//"io/ioutil"
	"bufio"
	"log"
	"os/exec"
	//"time"
)

func main() {
	cmd := exec.Command("sar", "-n", "DEV", "1", "1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer stdout.Close()
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("err:%s\n", err.Error())
			break
		}
		fmt.Printf("%s", string(line))
	}
}
