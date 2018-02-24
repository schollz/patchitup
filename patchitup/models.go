package patchitup

import (
	"io/ioutil"
	"regexp"
)

type serverRequest struct {
	Username string `json:"username" binding:"required"`
	Filename string `json:"filename" binding:"required"`
	Patch    string `json:"patch"`
}

type serverResponse struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Patches []patchFile `json:"patches" omitempty:"true"`
}

var convertWindowsLineFeed = regexp.MustCompile(`\r?\n`)

func getFileText(pathToFile string) (fileText string, err error) {
	bFile, err := ioutil.ReadFile(pathToFile)
	bFile = convertWindowsLineFeed.ReplaceAll(bFile, []byte("\n"))
	fileText = string(bFile)
	return
}
