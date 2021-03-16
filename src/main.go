package main

import (
	"flag"
)

var socketPath = "/tmp/nvwa.socket"

func main() {
	server := flag.Int("server", 0,
		"set this value to 1 to start a server")
	flag.Parse()
	if *server != 0 {
		startServer(socketPath)
	} else {
		startClient(socketPath)
	}
}
