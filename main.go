package main

import (
	"flag"
	"fmt"

	"github.com/schollz/patchitup/patchitup"
)

var (
	doDebug    bool
	port       string
	dataFolder string
	server     bool
	rebuild    bool
	pathToFile string
	username   string
	address    string
	passphrase string
)

func main() {

	flag.StringVar(&port, "port", "8002", "port to run server")
	flag.StringVar(&pathToFile, "f", "", "path to the file to patch")
	flag.StringVar(&username, "u", "", "username on the cloud")
	flag.StringVar(&passphrase, "p", "", "passphrase to use")
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

	if dataFolder != "" {
		patchitup.DataFolder = dataFolder
	}

	err := run()
	if err != nil {
		fmt.Println(err)
	}
}

func run() error {
	if server {
		patchitup.SetLogLevel("info")
		err := patchitup.Run(port)
		if err != nil {
			return err
		}
	} else if rebuild {
		p, err := patchitup.New(patchitup.Configuration{
			PathToFile:    pathToFile,
			ServerAddress: address,
		})
		if err != nil {
			return err
		}
		latest, err := p.Rebuild()
		if err != nil {
			return err
		}
		fmt.Println(latest)
	} else {
		p, err := patchitup.New(patchitup.Configuration{
			PathToFile:    pathToFile,
			ServerAddress: address,
		})
		if err != nil {
			return err
		}
		err = p.PatchUp()
		if err != nil {
			return err
		}
	}
	return nil
}
