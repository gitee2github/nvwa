package main

import (
	"net"
	"os"
	"time"

	"github.com/ankur-anand/simple-go-rpc/src/client"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var callFunc func(string) (int, error)

func sendCmdToServer(ip, port, cmd, param string) (int, error) {
	addr := ip + ":" + port
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Errorf("Unable to connect to server, error is %s \n", err)
		return 0, err
	}
	defer conn.Close()
	cli := client.NewClient(conn)
	cli.CallRPC(cmd, &callFunc)
	return callFunc(param)
}

func startClient(ip, port string) {
	log.SetLevel(log.DebugLevel)
	app := &cli.App{
		Name:     "nvwa",
		Usage:    "a tool used for openEuler kernel update.",
		Version:  "0.0.1",
		Compiled: time.Now(),
		Commands: []*cli.Command{
			{
				Name:  "check",
				Usage: "check kexec and criu version",
				Action: func(c *cli.Context) error {
					// When build rpm package, rpmbuild will check requires, no more need here
					log.Debugf("Check satisfied\n")
					return nil
				},
			},
			{
				Name:  "update",
				Usage: "specify kernel version for nvwa to update",
				Action: func(c *cli.Context) error {
					ret, err := sendCmdToServer(ip, port, "update", c.Args().First())
					log.Debugf("Update version to %s \n", c.Args().First())
					if err != nil {
						log.WithFields(log.Fields{
							"ip":        ip,
							"port":      port,
							"parameter": c.Args().First(),
						}).Errorf("Execute update cmd exit with ret %d and error %s \n", ret, err)
					}
					return err
				},
			},
			{
				Name:  "restore",
				Usage: "restore previous running environment",
				Action: func(c *cli.Context) error {
					ret, err := sendCmdToServer(ip, port, "restore", "")
					if err != nil {
						log.WithFields(log.Fields{
							"ip":   ip,
							"port": port,
						}).Errorf("Execute restore cmd exit with ret %d and error %s \n", ret, err)
					}
					return err
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
