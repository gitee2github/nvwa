package main

import (
	"net"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func sendCmdToServer(path, cmd, param string) int {
	c, err := net.Dial("unix", path)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	msg := "nvwa: " + cmd + " " + param
	_, err = c.Write([]byte(msg))
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 1024)
	n, err := c.Read(buf[:])
	if err != nil {
		log.Fatal(err)
	}
	ret, err := strconv.Atoi(string(buf[0:n]))
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func handleRet(cmd string, ret int) {
	log.Fatalf("Execute %s with ret %d, "+
		"Please check nvwa service log \n", cmd, ret)
}

func startClient(path string) {
	log.SetLevel(log.DebugLevel)
	app := &cli.App{
		Name:     "nvwa",
		Usage:    "a tool used for openEuler kernel update.",
		Compiled: time.Now(),
		Commands: []*cli.Command{
			{
				Name:   "check",
				Usage:  "check kexec and criu version",
				Hidden: true,
				Action: func(c *cli.Context) error {
					// When build rpm package, rpmbuild will check requires, no more need here
					log.Debugf("Check satisfied\n")
					return nil
				},
			},
			{
				Name:   "update",
				Usage:  "specify kernel version for nvwa to update",
				Hidden: false,
				Action: func(c *cli.Context) error {
					ret := sendCmdToServer(path, "update", c.Args().First())
					log.Debugf("Update version to %s \n", c.Args().First())
					if ret != 0 {
						handleRet("update", ret)
					}
					return nil
				},
			},
			{
				Name:   "restore",
				Usage:  "restore service",
				Hidden: true,
				Action: func(c *cli.Context) error {
					ret := sendCmdToServer(path, "restore", c.Args().First())
					log.Debugf("Resore service %s \n", c.Args().First())
					if ret != 0 {
						handleRet("restore", ret)
					}
					RestoreMainPid(c.Args().First())
					NotifySytemd()
					SystemdReload()
					return nil
				},
			},
			{
				Name:   "init",
				Usage:  "init nvwa running environment",
				Hidden: false,
				Action: func(c *cli.Context) error {
					ret := sendCmdToServer(path, "init", "")
					if ret != 0 {
						handleRet("init", ret)
					}
					return nil
				},
			},
			{
				Name:   "exit",
				Usage:  "exit nvwa service",
				Hidden: true,
				Action: func(c *cli.Context) error {
					ret := sendCmdToServer(path, "exit", "")
					if ret != 0 {
						handleRet("exit", ret)
					}
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
