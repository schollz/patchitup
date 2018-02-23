package patchitup

import (
	"bufio"
	"compress/gzip"
	"io/ioutil"
	"os"
	"regexp"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
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
var convertWindowsLineFeed2 = regexp.MustCompile(`\r`)

func getFileText(pathToFile string) (fileText string, err error) {
	bFile, err := ioutil.ReadFile(pathToFile)
	if err != nil {
		return
	}
	bFile, err = GunzipBytes(bFile)
	if err != nil {
		return
	}
	bFile = convertWindowsLineFeed.ReplaceAll(bFile, []byte("\n"))
	bFile = convertWindowsLineFeed2.ReplaceAll(bFile, []byte(""))
	fileText = string(bFile)
	return
}

func getHashLineNumbers(pathToFile string) (lines map[string][]int, err error) {
	lines = make(map[string][]int)
	file, err := os.Open(pathToFile)
	if err != nil {
		err = errors.New("problem opening file")
		return
	}
	defer file.Close()

	fz, err := gzip.NewReader(file)
	if err != nil {
		err = nil
		return
	}
	scanner := bufio.NewScanner(fz)
	lineNumber := 0
	for scanner.Scan() {
		line := convertWindowsLineFeed.ReplaceAll(scanner.Bytes(), []byte("\n"))
		h := HashSHA256(line)
		log.Debug(line, h)
		if _, ok := lines[h]; !ok {
			lines[h] = []int{}
		}
		lines[h] = append(lines[h], lineNumber)
		lineNumber++
	}
	return
}

func getHashLines(pathToFile string) (lines map[string][]byte, err error) {
	lines = make(map[string][]byte)
	file, err := os.Open(pathToFile)
	if err != nil {
		return
	}
	defer file.Close()

	fz, err := gzip.NewReader(file)
	scanner := bufio.NewScanner(fz)
	for scanner.Scan() {
		line := convertWindowsLineFeed.ReplaceAll(scanner.Bytes(), []byte("\n"))
		h := HashSHA256(line)
		lines[h] = line
	}
	return
}
