package main

import (
	log "github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"strings"
	"sync"
)

func runCmd(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (error, string) {
	var retVal []byte
	var err error

	command := exec.Command(cmd, args...)
	if stdin != nil {
		command.Stdin = stdin
	}
	if stdout != nil {
		command.Stdout = stdout
	}
	if stderr != nil {
		command.Stderr = stderr
	}

	if stdout != nil {
		err = command.Run()
	} else {
		retVal, err = command.Output()
	}
	if err != nil {
		log.Errorf("Run command %s failed, error is %s \n", command.Args, err)
		return err, ""
	}
	tmpRet := strings.Trim(string(retVal), "\n")
	tmpRet = strings.Trim(string(retVal), " ")
	return nil, tmpRet
}

func waitCmd(cmd string, args []string, wg *sync.WaitGroup, stdin io.Reader, stdout, stderr io.Writer, count *int) {
	defer wg.Done()
	err, _ := runCmd(cmd, args, stdin, stdout, stderr)
	if err == nil {
		*count ++
	}
}
