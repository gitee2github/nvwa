package main

import (
	"strings"
	"github.com/coreos/go-systemd/daemon"
	log "github.com/sirupsen/logrus"
)

func RestoreMainPid(command string) {
	commands := strings.Split(command, "@")
	if (len(commands) != 2) {
		log.Errorf("Invalid command format %s \n", command)
		return
	}
	pid := commands[1]
	ifSupport, err := daemon.SdNotify(false, "READY=1\nMAINPID="+pid)
	if ifSupport == false || err != nil {
		log.Errorf("Unable to notify systemd, support %v err %s \n", ifSupport, err)
	}
}

func NotifySytemd() {
	ifSupport, err := daemon.SdNotify(false, "READY=1")
	if ifSupport == false || err != nil {
		log.Errorf("Unable to notify systemd, support %v err %s \n", ifSupport, err)
	}
}

func SystemdReload() {
	err, _ := runCmd("systemctl", []string{"daemon-reload"}, nil, nil, nil)
	if err != nil {
		log.Errorf("Daemon reload failed, error is %s \n", err)
	}
}
