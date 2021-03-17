package main

import (
	"github.com/coreos/go-systemd/daemon"
	log "github.com/sirupsen/logrus"
)

func RestoreMainPid(pid string) {
	ifSupport, err := daemon.SdNotify(false, "READY=1\nMAINPID="+pid)
	if ifSupport == false || err != nil {
		log.Errorf("Unable to notify systemd, support %d err %s \n", ifSupport, err)
	}
}

func NotifySytemd() {
	ifSupport, err := daemon.SdNotify(false, "READY=1")
	if ifSupport == false || err != nil {
		log.Errorf("Unable to notify systemd, support %d err %s \n", ifSupport, err)
	}
}

func SystemdReload() {
	err, _ := runCmd("systemctl", []string{"daemon-reload"}, nil, nil, nil)
	if err != nil {
		log.Errorf("Daemon reload failed, error is %s \n", err)
	}
}
