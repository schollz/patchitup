package patchitup

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/cihub/seelog"
	"github.com/dustin/go-humanize"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func getPatch(text1, text2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, false)
	patches := dmp.PatchMake(text1, diffs)
	patchUncompressed := dmp.PatchToText(patches)

	// compress patch
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(patchUncompressed)); err != nil {
		panic(err)
	}
	if err := gz.Flush(); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	compressedPatch := base64.StdEncoding.EncodeToString(b.Bytes())

	log.Debugf("compressed patch from %s to %s", humanize.Bytes(uint64(len(patchUncompressed))), humanize.Bytes(uint64(len(compressedPatch))))
	return compressedPatch
}

func patchFile(pathToFile string, compressedPatch string) (err error) {
	// decompress patch
	compressedPatchBytes, err := base64.StdEncoding.DecodeString(compressedPatch)
	if err != nil {
		return
	}
	gr, err := gzip.NewReader(bytes.NewBuffer(compressedPatchBytes))
	defer gr.Close()
	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	patch := string(data)

	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		return
	}
	textBase, err := getFileText(pathToFile)
	if err != nil {
		return
	}
	newText, _ := dmp.PatchApply(patches, textBase)
	err = ioutil.WriteFile(pathToFile, []byte(newText), 0755)
	err = ioutil.WriteFile(fmt.Sprintf("%s.%d", pathToFile, time.Now().UnixNano()/1000000), []byte(compressedPatch), 0755)
	return
}
