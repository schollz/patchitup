package patchitup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/schollz/utils"
)

type Patchitup struct {
	username         string
	serverAddress    string
	pathToFile       string
	pathToCachedFile string
	filename         string
	hashOfFile       string
	hashOfRemoteFile string
	cacheFolder      string
}

func New(address, username string) (p *Patchitup) {
	p = new(Patchitup)
	p.cacheFolder = path.Join(utils.UserHomeDir(), ".patchitup", "client", username)
	os.MkdirAll(p.cacheFolder, 0755)
	p.username = username
	p.serverAddress = address
	return p
}

// PatchUp will take a filename and upload it to the server via a patch using the specified user.
func (p *Patchitup) PatchUp(pathToFile string) (err error) {
	// flush logs so that they show up
	defer log.Flush()

	// generate the filename
	p.pathToFile = pathToFile
	_, p.filename = filepath.Split(pathToFile)

	// first make sure the file to upload exists
	if !utils.Exists(pathToFile) {
		return fmt.Errorf("'%s' not found", pathToFile)
	}

	// copy current state of file
	err = utils.CopyFile(pathToFile, pathToFile+".temp")
	if err != nil {
		return
	}
	pathToFile = pathToFile + ".temp"

	// get hash of file
	err = p.getLatestHash()
	if err != nil {
		return
	}
	log.Debugf("local hash: %s", p.hashOfFile)
	log.Debugf("remote hash: %s", p.hashOfRemoteFile)
	if p.hashOfFile == p.hashOfRemoteFile {
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
func (p *Patchitup) getLatestHash() (err error) {
	sr := serverRequest{
		Username: p.username,
		Filename: p.filename,
	}
	target, err := postToServer(p.serverAddress+"/hash", sr)
	if err == nil {
		p.hashOfRemoteFile = target.Message
	}
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
