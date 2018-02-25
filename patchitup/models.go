package patchitup

import (
	"io/ioutil"
	"regexp"
)

type serverRequest struct {
	Authentication string `json:"authentication" binding:"required"`
	PublicKey      string `json:"public_key"`
	Patch          string `json:"patch"`
}

type serverResponse struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Patch   string      `json:"patch,omitempty"`
	Patches []patchFile `json:"patches,omitempty"`
}

var convertWindowsLineFeed = regexp.MustCompile(`\r?\n`)

func getFileText(pathToFile string) (fileText string, err error) {
	bFile, err := ioutil.ReadFile(pathToFile)
	bFile = convertWindowsLineFeed.ReplaceAll(bFile, []byte("\n"))
	fileText = string(bFile)
	return
}
