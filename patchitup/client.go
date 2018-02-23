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

	"github.com/BurntSushi/toml"
	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

type ClientConfiguration struct {
	ServerAddress string
	Username      string
}

func handleConfiguration(address, username string) (c ClientConfiguration, err error) {
	configFile := path.Join(UserHomeDir(), ".patchitup", "client", "config.toml")
	bConfig, err := ioutil.ReadFile(configFile)
	newConfig := false
	if err == nil {
		err2 := toml.Unmarshal(bConfig, &c)
		if err2 != nil {
			err = err2
			return
		}
	} else {
		newConfig = true
		c = ClientConfiguration{}
	}
	// supplied names always override
	if username != "" {
		c.Username = username
	}
	if address != "" {
		c.ServerAddress = address
	}

	// check that they are not empty
	if c.Username == "" {
		// supply a random username
		c.Username = RandStringBytesMaskImprSrc(10)
		log.Infof("your username is '%s'\n", c.Username)
	}
	if c.ServerAddress == "" {
		err = errors.New("must supply address (-s)")
		return
	}

	// save the configuration
	buf := new(bytes.Buffer)
	err = toml.NewEncoder(buf).Encode(c)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(configFile, buf.Bytes(), 0755)
	if err != nil {
		return
	}
	if newConfig {
		log.Info("configuration file written, next time you do not need to include username (-u) and server (-s)")
	}
	return
}

// PatchUp will take a filename and upload it to the server via a patch.
func PatchUp(address, username, pathToFile string) (err error) {
	// make the directory for the client
	os.MkdirAll(path.Join(UserHomeDir(), ".patchitup", "client"), 0755)

	// flush logs so that they show up
	defer log.Flush()

	// first try to load the configuration file
	c, err := handleConfiguration(address, username)
	if err != nil {
		return
	}
	address = c.ServerAddress
	username = c.Username

	// generate the filename
	_, filename := filepath.Split(pathToFile)

	// first make sure the file to upload exists
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
	defer os.Remove(filename + ".temp")
	if err != nil {
		return
	}

	// get the latest hash from remote
	localHash, err := Filemd5Sum(pathToFile)
	if err != nil {
		return
	}
	remoteHash, err := getLatestHash(address, username, pathToFile)
	if err != nil {
		return
	}
	log.Debugf("local hash: %s", localHash)
	log.Debugf("remote hash: %s", remoteHash)
	if localHash == remoteHash {
		log.Info("remote server is up-to-date")
		return
	}

	// check hash of the cached remote copy and the remote copy
	localRemoteHash, err := Filemd5Sum(pathToRemoteCopy)
	log.Debugf("local remote hash: %s", localRemoteHash)
	if localRemoteHash != remoteHash {
		// local remote copy and remote is out of data
		// reconstruct file from remote
		log.Debug("reconstructing from remote")
		remoteCopyText, err := reconstructCopyFromRemote(address, username, filename)
		if err != nil {
			return errors.Wrap(err, "problem reconstructing: ")
		}
		err = ioutil.WriteFile(pathToRemoteCopy, []byte(remoteCopyText), 0755)
	} else {
		// local remote copy replicate of the remote file, so it can be used to generate diff
		log.Debug("local remote is up-to-date, not reconstructing")
	}

	// get patches between the local version and the local remote version
	localRemoteText, err := getFileText(pathToRemoteCopy)
	if err != nil {
		return err
	}
	localText, err := getFileText(filename + ".temp")
	if err != nil {
		return err
	}
	patch := getPatch(localRemoteText, localText)

	// upload patches
	err = uploadPatches(patch, address, username, pathToFile)
	if err != nil {
		return err
	} else {
		log.Infof("patched %s (%2.1f%%) to remote '%s' for '%s'", humanize.Bytes(uint64(len(patch))), 100*float64(len(patch))/float64(len(localText)), filename, username)
	}

	// update the local remote copy
	err = ioutil.WriteFile(pathToRemoteCopy, convertWindowsLineFeed.ReplaceAll([]byte(localText), []byte("\n")), 0755)
	if err != nil {
		return err
	}

	log.Info("remote server is up-to-date")
	return
}

// postToServer is generic function to post to the server
func postToServer(address string, sr serverRequest) (target serverResponse, err error) {
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", address, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&target)
	if err != nil {
		return
	}
	if !target.Success {
		err = errors.New(target.Message)
	}
	log.Debugf("POST %s: %s", address, target.Message)
	return
}

// getLatestHash will get latest hash from server
func getLatestHash(address, username, pathToFile string) (fileHash string, err error) {
	_, filename := filepath.Split(pathToFile)

	sr := serverRequest{
		Username: username,
		Filename: filename,
	}
	target, err := postToServer(address+"/fileHash", sr)
	fileHash = target.Message
	return
}

// uploadPatches will upload the patch to the server
func uploadPatches(patch string, address, username, pathToFile string) (err error) {
	_, filename := filepath.Split(pathToFile)

	sr := serverRequest{
		Username: username,
		Filename: filename,
		Patch:    patch,
	}
	_, err = postToServer(address+"/patch", sr)
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
	target, err := postToServer(address+"/lineNumbers", sr)
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

	sr := serverRequest{
		Username:     username,
		Filename:     filename,
		MissingLines: missingLines,
	}
	target, err := postToServer(address+"/lineText", sr)

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

	hashLines, err := getRemoteCopyHashLines(remoteHashLineNumbers, address, username, pathToFile)
	if err != nil {
		return
	}

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
			lines[lineNum] = string(convertWindowsLineFeed.ReplaceAll(hashLines[h], []byte("\n")))
		}
	}

	reconstructedFile = strings.Join(lines, "\n")
	return
}
