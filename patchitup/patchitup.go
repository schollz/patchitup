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
	"github.com/schollz/patchitup/patchitup/keypair"
	"github.com/schollz/utils"
)

type Patchitup struct {
	username      string
	serverAddress string
	cacheFolder   string
	passphrase    string
	key           keypair.KeyPair
}

func New(username string) (p *Patchitup) {
	p = new(Patchitup)
	p.cacheFolder = path.Join(utils.UserHomeDir(), ".patchitup", username)
	os.MkdirAll(p.cacheFolder, 0755)
	p.username = username
	return p
}

func (p *Patchitup) SetPassphrase(passphrase string) {
	p.passphrase = passphrase
	p.key = keypair.NewDeterministic(p.username + ":" + p.passphrase)
}

type patchFile struct {
	Filename  string
	Hash      string
	Timestamp int
}

// SetDataFolder will specify where to store the data
func (p *Patchitup) SetDataFolder(folder string) {
	os.MkdirAll(folder, 0755)
	p.cacheFolder = folder
}

func (p *Patchitup) SetServerAddress(address string) {
	p.serverAddress = address
}

// LatestHash returns the latest hash
func (p *Patchitup) LatestHash(filename string) (hash string, err error) {
	patches, err := p.getPatches(filename)
	if err != nil {
		return
	}
	if len(patches) == 0 {
		err = fmt.Errorf("no patches available for '%s'", filename)
		return
	}
	hash = patches[len(patches)-1].Hash
	return
}

func (p *Patchitup) Rebuild(filename string) (latest string, err error) {
	// flush logs so that they show up
	defer log.Flush()

	log.Debug("rebuilding")

	patches, err := p.getPatches(filename)
	if err != nil {
		log.Debug(err)
		return
	}
	latest = ""
	for _, patch := range patches {
		var patchBytes []byte
		var patchString string
		patchBytes, err = ioutil.ReadFile(path.Join(p.cacheFolder, patch.Filename))
		if err != nil {
			log.Debug(err)
			return
		}
		patchString, err = p.decode(string(patchBytes))
		if err != nil {
			log.Debug(err)
			return
		}
		latest, err = patchText(latest, patchString)
		if err != nil {
			log.Debug(err)
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
func (p *Patchitup) getPatches(filename string) (patchFiles []patchFile, err error) {
	files, err := ioutil.ReadDir(p.cacheFolder)
	if err != nil {
		return
	}

	m := make(map[int]patchFile)
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), filename+".patchitupv1.") {
			continue
		}
		g := strings.Split(f.Name(), ".")
		if len(g) < 4 {
			continue
		}
		var err2 error
		pf := patchFile{Filename: f.Name()}
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
	_, filename := filepath.Split(pathToFile)

	err = p.Sync(filename)
	if err != nil {
		return
	}

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
	hashLocal := utils.Md5Sum(localText)
	if err != nil {
		return
	}

	// get current hash of the remote file and compare
	localCopyOfRemoteText, err := p.Rebuild(filename)
	if err != nil {
		return
	}
	hashLocalCopyOfRemote := utils.Md5Sum(localCopyOfRemoteText)

	log.Debug(hashLocal)
	log.Debug(hashLocalCopyOfRemote)
	if hashLocalCopyOfRemote == hashLocal {
		log.Debug("hashes match, not doing anything")
		return
	}

	// upload patches
	patch := getPatch(localCopyOfRemoteText, localText)
	encodedPatch, err := p.encode(patch)
	if err != nil {
		return
	}

	// filename.patchitupv1.HASH.TIMESTAMP
	saveFilename := fmt.Sprintf("%s.patchitupv1.%s.%d",
		filename,
		hashLocal,
		time.Now().UTC().UnixNano()/int64(time.Millisecond),
	)
	p.SavePatch(saveFilename, encodedPatch)
	log.Infof("patched %s (%2.1f%%) to remote '%s' for '%s'",
		humanize.Bytes(uint64(len(patch))),
		100*float64(len(patch))/float64(len(localText)),
		filename,
		p.username)
	err = p.Sync(filename)
	return
}

func (p *Patchitup) SavePatch(filename, patch string) (err error) {
	// save to a file
	err = ioutil.WriteFile(path.Join(p.cacheFolder, filename), []byte(patch), 0755)
	return
}

func (p *Patchitup) LoadPatch(patchFilename string) (patch string, err error) {
	// save to a file
	patchBytes, err := ioutil.ReadFile(path.Join(p.cacheFolder, patchFilename))
	if err == nil {
		patch = string(patchBytes)
	}
	return
}

func (p *Patchitup) Sync(filename string) (err error) {
	localPatches, err := p.getPatches(filename)
	if err != nil {
		return
	}

	signature, err := p.key.Signature(sharedKey)
	if err != nil {
		return
	}

	address := fmt.Sprintf("%s/list/%s/%s", p.serverAddress, p.username, filename)
	remote, err := request("GET", address, serverRequest{Authentication: signature})
	if err != nil {
		return
	}

	signature, err := p.key.Signature(sharedKey)
	if err != nil {
		return
	}

	// upload to server
	remoteHas := make(map[string]struct{})
	for _, patch := range remote.Patches {
		remoteHas[patch.Hash] = struct{}{}
	}
	for _, localPatch := range localPatches {
		if _, ok := remoteHas[localPatch.Hash]; ok {
			continue
		}
		// server doesn't have
		log.Debugf("uploading %s to server", localPatch.Filename)
		address = fmt.Sprintf("%s/patch/%s/%s", p.serverAddress, p.username, localPatch.Filename)
		patch, err2 := p.LoadPatch(localPatch.Filename)
		if err2 != nil {
			err = err2
			return
		}
		sr := serverRequest{Authentication: signature, Patch: patch}
		response, err2 := request("POST", address, sr)
		if err2 != nil {
			log.Warn(err2)
		}
		if response.Success == false {
			log.Warn(response.Message)
		} else {
			log.Debug(response.Message)
		}
	}

	// download from server
	localHas := make(map[string]struct{})
	for _, patch := range localPatches {
		localHas[patch.Hash] = struct{}{}
	}
	for _, remotePatch := range remote.Patches {
		if _, ok := localHas[remotePatch.Hash]; ok {
			continue
		}
		// server doesn't have
		log.Debugf("downloading %s from server", remotePatch.Filename)
		address = fmt.Sprintf("%s/patch/%s/%s", p.serverAddress, p.username, remotePatch.Filename)
		sr := serverRequest{}
		response, err2 := request("GET", address, sr)
		if err2 != nil {
			log.Warn(err2)
		}
		if response.Success == false {
			log.Warn(response.Message)
		} else {
			log.Debug(response.Message)
			err2 := p.SavePatch(remotePatch.Filename, response.Patch)
			if err2 != nil {
				err = err2
				return
			}
		}
	}
	return
}

func (p *Patchitup) Register() (err error) {
	address := fmt.Sprintf("%s/register/%s", p.serverAddress, p.username)
	signature, err := p.key.Signature(sharedKey)
	if err != nil {
		return
	}
	response, err := request("POST", address,
		serverRequest{
			Authentication: signature,
			PublicKey:      p.key.Public,
		})

	if err != nil {
		return
	}
	if response.Success == false {
		err = errors.New(response.Message)
	}
	return
}

// request is generic function to post to the server
func request(requestType string, address string, sr serverRequest) (target serverResponse, err error) {
	payloadBytes, err := json.Marshal(sr)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest(requestType, address, body)
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
		err = errors.Wrap(err, "could not unmarshal server request")
		return
	}
	if !target.Success {
		err = errors.New(target.Message)
	}
	log.Debugf("POST %s: %s", address, target.Message)
	return
}

// getLatestHash will get latest hash from server
func (p *Patchitup) getLatestHash(filename string) (hashRemote string, err error) {
	signature, err := p.key.Signature(sharedKey)
	if err != nil {
		return
	}

	sr := serverRequest{
		Authentication: signature,
	}

	address := fmt.Sprintf("%s/hash/%s/%s", p.serverAddress, p.username, filename)
	target, err := request("GET", address, sr)
	hashRemote = target.Message
	return
}
