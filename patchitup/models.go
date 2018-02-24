package patchitup

import (
	"io/ioutil"
	"regexp"
)

type serverRequest struct {
	Username     string              `json:"username" binding:"required"`
	Filename     string              `json:"filename" binding:"required"`
	Data         string              `json:"data"`
	MissingLines map[string]struct{} `json:"missing_lines"`
	Patch        string              `json:"patch"`
}

type serverResponse struct {
	Message         string            `json:"message"`
	Success         bool              `json:"success"`
	HashLinenumbers map[string][]int  `json:"hash_linenumbers"`
	HashLineText    map[string][]byte `json:"hash_linetext"`
}

var convertWindowsLineFeed = regexp.MustCompile(`\r?\n`)

func getFileText(pathToFile string) (fileText string, err error) {
	bFile, err := ioutil.ReadFile(pathToFile)
	bFile = convertWindowsLineFeed.ReplaceAll(bFile, []byte("\n"))
	fileText = string(bFile)
	return
}
