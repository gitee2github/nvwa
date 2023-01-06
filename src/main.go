package main

import (
	"flag"
)

var socketPath = "/run/nvwa/nvwa.socket"

func main() {
	server := flag.Int("server", 0,
		"set this value to 1 to start a server")
	_ = flag.Bool("h", false, "use nvwa help to see help text")
	_ = flag.Bool("help", false, "use nvwa help to see help text")
	flag.Parse()
	if *server != 0 {
		startServer(socketPath)
	} else {
		startClient(socketPath)
	}
}
