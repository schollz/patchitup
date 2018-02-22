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
	"github.com/pkg/errors"
)

func init() {
	os.MkdirAll(path.Join(UserHomeDir(), ".patchitup", "client"), 0755)
}

// PatchUp will take a filename and upload it to the server via a patch.
func PatchUp(address, username, pathToFile string) (err error) {
	log.Debugf("check if '%s' exists", pathToFile)
	if !Exists(pathToFile) {
		return fmt.Errorf("'%s' not found", pathToFile)
	}

	// check if cache folder exists
	if !Exists(path.Join(pathToCacheClient, username)) {
		log.Debugf("making cache folder for user '%s'", username)
		os.MkdirAll(path.Join(pathToCacheClient, username), 0755)
	}

	// check if file exists in cache
	_, filename := filepath.Split(pathToFile)
	fileInCache := path.Join(pathToCacheClient, username, filename)
	if !Exists(fileInCache) {
		log.Debug("reconstructing from remote")
		err = reconstructCopyFromRemote(address, username, filename)
		if err != nil {
			return errors.Wrap(err, "problem reconstructing: ")
		}
	}

	// do I have cache?
	return
}

func reconstructCopyFromRemote(address, username, filename string) (err error) {
	// ask for lines from server
	sr := serverRequest{
		Username: username,
		Filename: filename,
	}
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", address+"/lines", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	log.Debug("got response")

	log.Debug("decoding response")
	var target serverResponse
	err = json.NewDecoder(resp.Body).Decode(&target)
	log.Debug("decoded response")
	if err != nil {
		return err
	}
	if !target.Success {
		return errors.New(target.Message)
	}
	log.Debug(target)
	return
}
