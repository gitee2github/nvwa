package main

import (
	"flag"
)

func main() {
	mode := flag.Int("mode", 0,
		"set this value to 1 to start a server")
	ipAddr := flag.String("ip", "localhost",
		"specify server ip")
	port := flag.String("port", "3232",
		"specify server port")
	flag.Parse()
	if *mode != 0 {
		startServer(*ipAddr, *port, *mode)
	} else {
		startClient(*ipAddr, *port)
	}
}