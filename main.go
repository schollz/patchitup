package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/schollz/patchitup/patchitup"
)

func main() {
	var (
		doDebug    bool
		port       string
		dataFolder string
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
	flag.StringVar(&dataFolder, "data", "", "folder to data (default $HOME/.patchitup)")
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
		p := patchitup.New(username)
		p.SetPassphrase("1234")
		p.SetServerAddress(address)
		if dataFolder != "" {
			p.SetDataFolder(dataFolder)
		}
		var latest string
		_, filename := filepath.Split(pathToFile)
		p.Sync(filename)
		latest, err = p.Rebuild(filename)
		fmt.Println(latest)
	} else {
		p := patchitup.New(username)
		p.SetPassphrase("1234")
		p.SetServerAddress(address)
		if dataFolder != "" {
			p.SetDataFolder(dataFolder)
		}
		err = p.Register()
		if err != nil {
			fmt.Println(err)
		}
		err = p.PatchUp(pathToFile)
	}
	if err != nil {
		fmt.Println(err)
	}
}
