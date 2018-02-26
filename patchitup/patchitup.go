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

// Patchitup is the main structure to generate and save patches
type Patchitup struct {
	key           keypair.KeyPair
	filename      string
	pathToFile    string
	currentFolder string
	serverAddress string
	signature     string
	syncResponse  serverResponse
	haveSynced    bool
}

// Configuraiton specifies the configuration for generating a New patchitup
type Configuration struct {
	PublicKey     string
	PrivateKey    string
	Signature     string
	PathToFile    string // required
	ServerAddress string
}

// New returns a new patchitup configuration
func New(c Configuration) (p *Patchitup, err error) {
	os.MkdirAll(DataFolder, 0755)

	p = new(Patchitup)
	if c.PublicKey == "" && c.PrivateKey == "" && c.Signature == "" {
		// first check if a key exists
		availableKeys := getKeys()
		log.Debugf("found %d available keys", len(availableKeys))
		if len(availableKeys) > 0 {
			p.key = availableKeys[0]
			log.Debugf("using the first available key: %s", p.key.Public)
			// TODO: allow a user to choose
		} else {
			log.Debug("generate a new key")
			p.key = keypair.New()
			// save it
			err = p.saveKey()
			if err != nil {
				return
			}
		}
	} else if c.PublicKey != "" && c.PrivateKey == "" && c.Signature != "" {
		log.Debug("validate the public key using the signature")
		p.key, err = keypair.FromPublic(c.PublicKey)
		if err != nil {
			return
		}
		err = sharedKey.Validate(c.Signature, p.key)
		if err != nil {
			return
		}
	} else if c.PublicKey != "" && c.PrivateKey != "" && c.Signature == "" {
		// validate the new key
		p.key, err = keypair.FromPair(c.PublicKey, c.PrivateKey)
		if err != nil {
			return
		}
		// save it
		err = p.saveKey()
		if err != nil {
			return
		}
	}

	if c.PathToFile == "" {
		err = errors.New("must provide file")
		return
	}

	p.serverAddress = c.ServerAddress
	if p.key.Private != "" {
		p.signature, err = p.key.Signature(sharedKey)
		if err != nil {
			return
		}
	}

	// generate folder name
	p.pathToFile, p.filename = filepath.Split(c.PathToFile)
	p.currentFolder = path.Join(DataFolder, p.key.Public, p.filename)
	os.MkdirAll(p.currentFolder, 0755)

	p.haveSynced = false
	return
}

// getKeys will return all the public+private key pairs
func getKeys() (keys []keypair.KeyPair) {
	files, errOpen := ioutil.ReadDir(DataFolder)
	if errOpen != nil {
		log.Debug(errOpen)
		return
	}

	for _, f := range files {
		if f.IsDir() {
			keyBytes, errRead := ioutil.ReadFile(path.Join(DataFolder, f.Name(), "key.json"))
			if errRead != nil {
				log.Debug(errRead)
				continue
			}
			var kp keypair.KeyPair
			errMarshal := json.Unmarshal(keyBytes, &kp)
			if errMarshal == nil && kp.Public != "" && kp.Private != "" {
				keys = append(keys, kp)
			} else {
				log.Debug(errMarshal)
			}
		}
	}
	return
}

// saveKey will save the key to a file
func (p *Patchitup) saveKey() (err error) {
	// write to the file
	keyBytes, errMarshal := json.Marshal(p.key)
	if errMarshal != nil {
		err = errors.Wrap(errMarshal, "problem marshaling new key")
		return
	}
	os.MkdirAll(path.Join(DataFolder, p.key.Public), 0755)
	err = ioutil.WriteFile(path.Join(DataFolder, p.key.Public, "key.json"), keyBytes, 0755)
	return
}

// latestHash returns the latest hash
func (p *Patchitup) latestHash() (hash string, err error) {
	patches, err := p.getPatches()
	if err != nil {
		return
	}
	if len(patches) == 0 {
		err = fmt.Errorf("no patches available for '%s'", p.filename)
		return
	}
	hash = patches[len(patches)-1].Hash
	return
}

// Rebuild will rebuild the specified file and return the latest
func (p *Patchitup) Rebuild() (latest string, err error) {
	// flush logs so that they show up
	defer log.Flush()

	log.Debug("rebuilding")

	patches, err := p.getPatches()
	if err != nil {
		log.Debug(err)
		return
	}
	latest = ""
	for _, patch := range patches {
		var patchF patchFile
		var patchString string
		patchF, err = p.loadPatch(patch.EpochTime, patch.Hash)
		if err != nil {
			log.Debug(err)
			return
		}
		patchString, err = p.decode(patchF.Patch)
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

// getPatches will determine determine the latest information for a patch file.
func (p *Patchitup) getPatches() (patchFiles []patchFile, err error) {
	files, err := ioutil.ReadDir(p.currentFolder)
	if err != nil {
		return
	}

	m := make(map[int]patchFile)
	for _, f := range files {
		g := strings.Split(f.Name(), ".")
		if len(g) != 2 {
			continue
		}
		var err2 error
		pf := patchFile{Filename: f.Name()}
		pf.EpochTime, err2 = strconv.Atoi(g[0])
		if err2 != nil {
			err = errors.Wrap(err2, "problem deciphering patch '"+f.Name()+"'")
			return
		}
		pf.Hash = g[1]
		m[pf.EpochTime] = pf
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
func (p *Patchitup) PatchUp() (err error) {
	// flush logs so that they show up
	defer log.Flush()

	err = p.sync()
	if err != nil {
		return
	}

	// first make sure the file to upload exists
	pathToFile := path.Join(p.pathToFile, p.filename)
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
	localCopyOfRemoteText, err := p.Rebuild()
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

	newPatch := patchFile{
		Hash:      hashLocal,
		EpochTime: int(time.Now().UTC().UnixNano() / int64(time.Millisecond)),
		Patch:     encodedPatch,
	}
	p.savePatch(newPatch)
	log.Infof("patched %s (%2.1f%%) to remote '%s' for '%s'",
		humanize.Bytes(uint64(len(patch))),
		100*float64(len(patch))/float64(len(localText)),
		p.filename,
		p.key.Public)
	err = p.sync()
	return
}

// savePatch will save the patch to a file
func (p *Patchitup) savePatch(patch patchFile) (err error) {
	// save to a file
	if patch.Hash == "" {
		err = errors.New("hash cannot be empty")
		return
	}
	filename := path.Join(p.currentFolder, fmt.Sprintf("%d.%s", patch.EpochTime, patch.Hash))
	err = ioutil.WriteFile(filename, []byte(patch.Patch), 0755)
	return
}

// loadPatch will load the specified patch
func (p *Patchitup) loadPatch(epochTime int, hash string) (patch patchFile, err error) {
	filename := path.Join(p.currentFolder, fmt.Sprintf("%d.%s", epochTime, hash))
	patch = patchFile{
		Hash:      hash,
		EpochTime: epochTime,
		Filename:  filename,
	}
	patchBytes, err := ioutil.ReadFile(filename)
	if err == nil {
		patch.Patch = string(patchBytes)
	}
	return
}

// sync will downlaod and upload the specified files
func (p *Patchitup) sync() (err error) {
	localPatches, err := p.getPatches()
	if err != nil {
		return
	}

	var remote serverResponse
	if !p.haveSynced {
		sr := serverRequest{
			Signature: p.signature,
			PublicKey: p.key.Public,
			Filename:  p.filename,
		}
		address := fmt.Sprintf("%s/list", p.serverAddress)
		remote, err = request("GET", address, sr)
		if err != nil {
			err = errors.Wrap(err, "problem with /list")
			log.Debug(err)
			return
		}
		p.syncResponse = remote
	} else {
		remote = p.syncResponse
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
		address := fmt.Sprintf("%s/patch", p.serverAddress)
		patch, err2 := p.loadPatch(localPatch.EpochTime, localPatch.Hash)
		if err2 != nil {
			err = errors.Wrap(err2, "could not load patch while syncing")
			return
		}
		sr := serverRequest{
			Signature: p.signature,
			PublicKey: p.key.Public,
			Filename:  p.filename,
			Patch:     patch,
		}
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

		// download from the server
		log.Debugf("downloading %s from server", remotePatch.Filename)
		address := fmt.Sprintf("%s/patch", p.serverAddress)
		sr := serverRequest{
			Signature: p.signature,
			PublicKey: p.key.Public,
			Filename:  p.filename,
			Patch:     remotePatch,
		}
		response, err2 := request("GET", address, sr)
		if err2 != nil {
			log.Warn(err2)
		}

		// check response
		if response.Success == false {
			log.Warn(response.Message)
		} else {
			log.Debug(response.Message)
			err2 := p.savePatch(response.Patch)
			if err2 != nil {
				err = err2
				return
			}
		}
	}

	p.haveSynced = true
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
	log.Debugf("%s %s: %s", requestType, address, target.Message)
	return
}

// getLatestHash will get latest hash from server
func (p *Patchitup) getLatestHash(filename string) (hashRemote string, err error) {
	sr := serverRequest{
		Signature: p.signature,
		PublicKey: p.key.Public,
		Filename:  p.filename,
	}

	address := fmt.Sprintf("%s/hash", p.serverAddress)
	target, err := request("GET", address, sr)
	hashRemote = target.Message
	return
}
