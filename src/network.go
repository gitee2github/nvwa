package main

import (
	"os"
	"path"
	log "github.com/sirupsen/logrus"
)

// fix: ipv6
func dumpNet(cmd, path string) {
	file, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, error is %s \n", path, err)
		return
	}
	defer file.Close()
	runCmd("ip", []string{cmd, "dump"}, nil, file, nil)
}

func restoreNet(cmd, path string) {
	file, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, error is %s \n", path, err)
		return
	}
	defer file.Close()
	runCmd("ip", []string{cmd, "restore"}, file, nil, nil)
}

func DumpIfaddr(path string) {
	dumpNet("addr", path)
}

func RestoreIfaddr(path string) {
	restoreNet("addr", path)
}

func DumpRoute(path string) {
	dumpNet("route", path)
}

func RestoreRoute(path string) {
	restoreNet("route", path)
}

func DumpRule(path string) {
	dumpNet("rule", path)
}

func RestoreRule(path string) {
	restoreNet("rule", path)
}

func DumpIptables(path string) {
	file, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, error is %s \n", path, err)
		return
	}
	defer file.Close()
	runCmd("iptables-save", []string{}, nil, file, nil)
}

func RestoreIptables(path string) {
	file, err := os.Create(path)
	if err != nil {
		log.Errorf("Unable to create file %s, error is %s \n", path, err)
		return
	}
	defer file.Close()
	runCmd("iptables-restore", []string{"-c", "--noflush"}, nil, file, nil)
}

func DumpAllNet(dirPath string) {
	DumpIfaddr(path.Join(dirPath, "addr"))
	DumpRoute(path.Join(dirPath, "route"))
	DumpRule(path.Join(dirPath, "rule"))
	DumpIptables(path.Join(dirPath, "iptables"))
}

func RestoreAllNet(dirPath string) {
	RestoreIfaddr(path.Join(dirPath, "addr"))
	RestoreRoute(path.Join(dirPath, "route"))
	RestoreRule(path.Join(dirPath, "rule"))
	RestoreIptables(path.Join(dirPath, "iptables"))
}

