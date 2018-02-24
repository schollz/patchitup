package main

import (
	"flag"
	"fmt"

	"github.com/schollz/patchitup/patchitup"
)

func main() {
	var (
		doDebug    bool
		port       string
		server     bool
		rebuild    bool
		pathToFile string
		username   string
		address    string
	)

	flag.StringVar(&port, "port", "8002", "port to run server")
	flag.StringVar(&pathToFile, "f", "", "path to the file to patch")
	flag.StringVar(&username, "u", "", "username on the cloud")
	flag.StringVar(&address, "s", "", "server name")
	flag.BoolVar(&doDebug, "debug", false, "enable debugging")
	flag.BoolVar(&server, "host", false, "enable hosting")
	flag.BoolVar(&rebuild, "rebuild", false, "rebuild file")
	flag.Parse()

	if doDebug {
		patchitup.SetLogLevel("debug")
	} else {
		patchitup.SetLogLevel("info")
	}
	var err error
	if server {
		patchitup.SetLogLevel("info")
		err = patchitup.Run(port)
	} else if rebuild {
		p := patchitup.New(address, username, true)
		err = p.Rebuild(pathToFile)
	} else {
		p := patchitup.New(address, username)
		err = p.PatchUp(pathToFile)
	}
	if err != nil {
		fmt.Println(err)
	}
}
