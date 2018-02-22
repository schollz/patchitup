package patchitup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

func init() {
	os.MkdirAll(path.Join(UserHomeDir(), ".patchitup", "client"), 0755)
}

// PatchUp will take a filename and upload it to the server via a patch.
func PatchUp(address, username, pathToFile string) (err error) {
	defer log.Flush()
	_, filename := filepath.Split(pathToFile)

	log.Debugf("check if '%s' exists", pathToFile)
	if !Exists(pathToFile) {
		return fmt.Errorf("'%s' not found", pathToFile)
	}

	// check if cache folder exists
	if !Exists(path.Join(pathToCacheClient, username)) {
		log.Debugf("making cache folder for user '%s'", username)
		os.MkdirAll(path.Join(pathToCacheClient, username), 0755)
	}
	pathToRemoteCopy := path.Join(pathToCacheClient, username, filename)

	// copy current state of file
	err = CopyFile(pathToFile, filename+".temp")
	if err != nil {
		return
	}

	// reconstruct file from remote
	log.Debug("reconstructing from remote")
	remoteCopyText, err := reconstructCopyFromRemote(address, username, filename)
	if err != nil {
		return errors.Wrap(err, "problem reconstructing: ")
	}
	log.Debugf("reconstructed remote copy text: %s", remoteCopyText)
	err = ioutil.WriteFile(pathToRemoteCopy, []byte(remoteCopyText), 0755)

	// get patches
	localText, err := getFileText(filename + ".temp")
	if err != nil {
		return err
	}
	patch := getPatch(remoteCopyText, string(localText))

	// upload patches
	err = uploadPatches(patch, address, username, pathToFile)
	return
}

func uploadPatches(patch string, address, username, pathToFile string) (err error) {
	_, filename := filepath.Split(pathToFile)

	// ask for lines from server
	sr := serverRequest{
		Username: username,
		Filename: filename,
		Patch:    patch,
	}
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", address+"/patch", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var target serverResponse
	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	if !target.Success {
		err = errors.New(target.Message)
		return
	}
	log.Debugf("POST /patch: %s", target.Message)
	return
}

func getRemoteCopyHashLineNumbers(address, username, pathToFile string) (hashLineNumbers map[string][]int, err error) {
	hashLineNumbers = make(map[string][]int)

	_, filename := filepath.Split(pathToFile)

	// ask for lines from server
	sr := serverRequest{
		Username: username,
		Filename: filename,
	}
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", address+"/lineNumbers", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var target serverResponse
	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	if !target.Success {
		err = errors.New(target.Message)
		return
	}
	hashLineNumbers = target.HashLinenumbers
	return
}

func getRemoteCopyHashLines(remoteHashLineNumbers map[string][]int, address, username, pathToFile string) (hashLines map[string][]byte, err error) {
	hashLines = make(map[string][]byte)

	_, filename := filepath.Split(pathToFile)

	pathToRemoteCopy := path.Join(pathToCacheClient, username, filename)
	if !Exists(pathToRemoteCopy) {
		newFile, err2 := os.Create(pathToRemoteCopy)
		if err2 != nil {
			err = errors.Wrap(err2, "problem creating file")
			return
		}
		newFile.Close()
		if len(remoteHashLineNumbers) == 0 {
			return
		}
	}

	log.Debug("reconstructing, creating local copy of remote")
	file, err := os.Open(pathToRemoteCopy)
	if err != nil {
		return
	}
	defer file.Close()

	log.Debug("determining which lines in current file are in the remote copy")
	hashLines, err = getHashLines(filename + ".temp")
	if err != nil {
		return
	}
	log.Debugf("currentLines: %+v", hashLines)
	missingLines := make(map[string]struct{})
	for h := range remoteHashLineNumbers {
		if _, ok := hashLines[h]; !ok {
			missingLines[h] = struct{}{}
		}
	}

	if len(missingLines) == 0 {
		log.Debug("not missing any lines")
		return
	}
	log.Debugf("requesting missing: %+v", missingLines)

	//
	// MAKE REQUEST FROM SERVER FOR MISSING LINES
	//
	sr := serverRequest{
		Username:     username,
		Filename:     filename,
		MissingLines: missingLines,
	}
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", address+"/lineText", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var target serverResponse
	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	if !target.Success {
		err = errors.New(target.Message)
		return
	}

	log.Debugf("needed lines: %+v", target.HashLineText)
	for line := range target.HashLineText {
		hashLines[line] = target.HashLineText[line]
	}
	return
}

func reconstructCopyFromRemote(address, username, pathToFile string) (reconstructedFile string, err error) {
	remoteHashLineNumbers, err := getRemoteCopyHashLineNumbers(address, username, pathToFile)
	if err != nil {
		return
	}
	log.Debugf("remoteHashLineNumbers: %+v", remoteHashLineNumbers)

	hashLines, err := getRemoteCopyHashLines(remoteHashLineNumbers, address, username, pathToFile)
	if err != nil {
		return
	}
	log.Debug("all lines: %+v", hashLines)

	// reconstruct the file
	numberLines := 0
	for h := range remoteHashLineNumbers {
		for _, lineNum := range remoteHashLineNumbers[h] {
			if lineNum > numberLines {
				numberLines = lineNum
			}
		}
	}
	log.Debugf("# lines: %d", numberLines)
	lines := make([]string, numberLines+1)

	for h := range remoteHashLineNumbers {
		for _, lineNum := range remoteHashLineNumbers[h] {
			lines[lineNum] = string(hashLines[h])
		}
	}

	reconstructedFile = strings.Join(lines, "\n")
	return
}
