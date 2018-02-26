package patchitup

import (
	"io/ioutil"
	"regexp"
)

type serverRequest struct {
	Signature string    `json:"signature" binding:"required"`
	PublicKey string    `json:"public_key" binding:"required"`
	Filename  string    `json:"filename"`
	Patch     patchFile `json:"patch"`
}

type patchFile struct {
	Filename  string
	Patch     string
	Hash      string
	EpochTime int
}

type serverResponse struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Patch   patchFile   `json:"patch,omitempty"`
	Patches []patchFile `json:"patches,omitempty"`
}

var convertWindowsLineFeed = regexp.MustCompile(`\r?\n`)

func getFileText(pathToFile string) (fileText string, err error) {
	bFile, err := ioutil.ReadFile(pathToFile)
	bFile = convertWindowsLineFeed.ReplaceAll(bFile, []byte("\n"))
	fileText = string(bFile)
	return
}
