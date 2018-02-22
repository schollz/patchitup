package patchitup

import (
	"bufio"
	"bytes"
	"os"
	"strings"
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

func getFileText(pathToFile string) (fileText string, err error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
	}
	file.Close()

	file, err = os.Open(pathToFile)
	if err != nil {
		return
	}
	defer file.Close()
	lines := make([]string, lineNumber)
	scanner = bufio.NewScanner(file)
	lineNumber = 0
	for scanner.Scan() {
		lines[lineNumber] = strings.TrimRight(scanner.Text())
		lineNumber++
	}
	fileText = strings.Join(lines, "\n")
	return
}

func getHashLineNumbers(pathToFile string) (lines map[string][]int, err error) {
	lines = make(map[string][]int)
	file, err := os.Open(pathToFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		h := HashSHA256(bytes.TrimRight(scanner.Bytes()))
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

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimRight(scanner.Bytes())
		h := HashSHA256(line)
		lines[h] = line
	}
	return
}
