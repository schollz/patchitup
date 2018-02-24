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

	"github.com/BurntSushi/toml"
	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/schollz/utils"
)

type clientConfiguration struct {
	ServerAddress string
	Username      string
}

func handleConfiguration(address, username string) (c clientConfiguration, err error) {
	configFile := path.Join(utils.UserHomeDir(), ".patchitup", "client", "config.toml")
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
		c = clientConfiguration{}
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
		c.Username = utils.RandStringBytesMaskImprSrc(10)
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

// PatchUp will take a filename and upload it to the server via a patch using the specified user.
func PatchUp(address, username, pathToFile string) (err error) {
	// make the directory for the client
	os.MkdirAll(path.Join(utils.UserHomeDir(), ".patchitup", "client"), 0755)

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
	if !utils.Exists(pathToFile) {
		return fmt.Errorf("'%s' not found", pathToFile)
	}

	// check if cache folder exists
	if !utils.Exists(path.Join(pathToCacheClient, username)) {
		log.Debugf("making cache folder for user '%s'", username)
		os.MkdirAll(path.Join(pathToCacheClient, username), 0755)
	}
	pathToRemoteCopy := path.Join(pathToCacheClient, username, filename)

	// copy current state of file
	err = utils.CopyFile(pathToFile, filename+".temp")
	defer os.Remove(filename + ".temp")
	if err != nil {
		return
	}

	// get the latest hash from remote
	localHash, err := utils.Filemd5Sum(pathToFile)
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
	localRemoteHash, err := utils.Filemd5Sum(pathToRemoteCopy)
	log.Debugf("local remote hash: %s", localRemoteHash)
	if localRemoteHash != remoteHash {
		// TODO
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
	// TODO COPY the temp file

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
	target, err := postToServer(address+"/hash", sr)
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
