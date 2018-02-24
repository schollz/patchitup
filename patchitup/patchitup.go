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
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/schollz/utils"
)

type Patchitup struct {
	username      string
	serverAddress string
	filename      string
	cacheFolder   string
	hashLocal     string
}

func New(address, username string) (p *Patchitup) {
	p = new(Patchitup)
	p.cacheFolder = path.Join(utils.UserHomeDir(), ".patchitup", username)
	os.MkdirAll(p.cacheFolder, 0755)
	p.username = username
	p.serverAddress = address
	return p
}

type patchFile struct {
	Filename  string
	Hash      string
	Timestamp int
}

func (p *Patchitup) Rebuild(pathToFile string) (latest string, err error) {
	// flush logs so that they show up
	defer log.Flush()

	log.Debug("rebuilding")

	// generate the filename
	_, p.filename = filepath.Split(pathToFile)

	patches, err := p.getPatches()
	if err != nil {
		return
	}
	latest = ""
	for _, patch := range patches {
		var patchBytes []byte
		var patchString string
		patchBytes, err = ioutil.ReadFile(patch.Filename)
		if err != nil {
			return
		}
		patchString, err = decode(string(patchBytes))
		if err != nil {
			return
		}
		latest, err = patchText(latest, patchString)
		if err != nil {
			return
		}
		if utils.Md5Sum(latest) != patch.Hash {
			log.Warnf("rebuilt(%s) != supposed(%s)", utils.Md5Sum(latest), patch.Hash)
			err = errors.New("hashes do not match")
		}
	}
	return
}

// getPatches will determine the hash of the latest file
func (p *Patchitup) getPatches() (patchFiles []patchFile, err error) {
	files, err := ioutil.ReadDir(p.cacheFolder)
	if err != nil {
		return
	}

	m := make(map[int]patchFile)
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), p.filename+".") {
			continue
		}
		g := strings.Split(f.Name(), ".")
		if len(g) < 4 {
			continue
		}
		var err2 error
		pf := patchFile{Filename: path.Join(p.cacheFolder, f.Name())}
		pf.Timestamp, err2 = strconv.Atoi(g[len(g)-1])
		if err2 != nil {
			continue
		}
		pf.Hash = g[len(g)-2]
		m[pf.Timestamp] = pf
	}
	if len(m) == 0 {
		return
	}

	keys := make([]int, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	sort.Ints(keys)

	patchFiles = make([]patchFile, len(keys))
	for i, key := range keys {
		patchFiles[i] = m[key]
	}
	return
}

// PatchUp will take a filename and upload it to the server via a patch using the specified user.
func (p *Patchitup) PatchUp(pathToFile string) (err error) {
	// flush logs so that they show up
	defer log.Flush()

	// generate the filename
	_, p.filename = filepath.Split(pathToFile)

	// first make sure the file to upload exists
	if !utils.Exists(pathToFile) {
		return fmt.Errorf("'%s' not found", pathToFile)
	}

	// copy current state of file
	err = utils.CopyFile(pathToFile, pathToFile+".current")
	if err != nil {
		return
	}
	defer os.Remove(pathToFile + ".current")

	// get current text
	localText, err := getFileText(pathToFile + ".current")
	if err != nil {
		return
	}
	localText = string(utils.Dos2Unix([]byte(localText)))

	// get current hash
	p.hashLocal = utils.Md5Sum(localText)
	if err != nil {
		return
	}

	// get current hash of the remote file and compare
	localCopyOfRemoteText, err := p.Rebuild(pathToFile)
	if err != nil {
		return
	}
	hashLocalCopyOfRemote := utils.Md5Sum(localCopyOfRemoteText)

	if hashLocalCopyOfRemote == p.hashLocal {
		log.Debug("hashes match, not doing anything")
		return
	}

	// upload patches
	patch := getPatch(localCopyOfRemoteText, localText)
	err = p.uploadPatches(encode(patch))
	if err != nil {
		return err
	} else {
		log.Infof("patched %s (%2.1f%%) to remote '%s' for '%s'",
			humanize.Bytes(uint64(len(patch))),
			100*float64(len(patch))/float64(len(localText)),
			p.filename,
			p.username)
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
func (p *Patchitup) getLatestHash() (hashRemote string, err error) {
	sr := serverRequest{
		Username: p.username,
		Filename: p.filename,
	}
	target, err := postToServer(p.serverAddress+"/hash", sr)
	hashRemote = target.Message
	return
}

// uploadPatches will upload the patch to the server
func (p *Patchitup) uploadPatches(patch string) (err error) {
	filename := fmt.Sprintf("%s.%s.%d",
		p.filename,
		p.hashLocal,
		time.Now().UTC().UnixNano()/int64(time.Millisecond),
	)
	err = ioutil.WriteFile(path.Join(p.cacheFolder, filename), []byte(patch), 0755)
	if err != nil {
		return
	}
	sr := serverRequest{
		Username: p.username,
		Filename: filename,
		Patch:    patch,
	}
	_, err = postToServer(p.serverAddress+"/patch", sr)

	return
}
