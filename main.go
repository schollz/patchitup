package main

import (
	"flag"
	"fmt"

	"github.com/schollz/patchitup/patchitup"
)

var (
	doDebug bool
	port    string
	server  bool
)

func main() {
	flag.StringVar(&port, "port", "8002", "port to run server")
	flag.BoolVar(&doDebug, "debug", false, "enable debugging")
	flag.BoolVar(&server, "server", false, "enable server")
	flag.Parse()

	patchitup.SetLogLevel("debug")
	var err error
	if !server {
		err = patchitup.Run(port)
	} else {
		err = patchitup.PatchUp("http://localhost:8002", "zack", "README.md")
	}
	if err != nil {
		fmt.Println(err)
	}
}
