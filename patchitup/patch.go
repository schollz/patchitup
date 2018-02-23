package patchitup

import (
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
	uncompressedPatch := dmp.PatchToText(patches)

	// compress patch
	gzipped, err := GzipBytes([]byte(uncompressedPatch))
	if err != nil {
		panic(err)
	}
	compressedPatch := base64.StdEncoding.EncodeToString(gzipped)

	log.Debugf("compressed patch from %s to %s", humanize.Bytes(uint64(len(uncompressedPatch))), humanize.Bytes(uint64(len(compressedPatch))))
	return compressedPatch
}

func patchFile(pathToFile string, compressedPatch string) (err error) {
	// decompress patch
	compressedPatchBytes, err := base64.StdEncoding.DecodeString(compressedPatch)
	if err != nil {
		return
	}

	uncompressed, err := GunzipBytes(compressedPatchBytes)
	patch := string(uncompressed)

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
	compressedText, err := GzipBytes([]byte(newText))
	if err != nil {
		return
	}
	err = ioutil.WriteFile(pathToFile, compressedText, 0755)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s.%d", pathToFile, time.Now().UnixNano()/1000000), []byte(compressedPatch), 0755)
	if err != nil {
		return
	}
	return
}
