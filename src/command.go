package main

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"io"
	"strings"
	"sync"
)

func runCmd(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (error, string) {
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
	retVal, err := command.Output()
	if err != nil {
		log.Errorf("Run command %s failed, error is %s \n", command.Args, err)
		return err, ""
	}
	tmpRet := strings.Trim(string(retVal), "\n")
	tmpRet = strings.Trim(string(retVal), " ")
	return nil, tmpRet
}

func waitCmd(cmd string, args []string, wg *sync.WaitGroup, stdin io.Reader, stdout, stderr io.Writer) (error, string) {
	defer wg.Done()
	return runCmd(cmd, args, stdin, stdout, stderr)
}