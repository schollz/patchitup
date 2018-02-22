package patchitup

import (
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/cihub/seelog"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func getPatch(text1, text2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, false)
	patches := dmp.PatchMake(text1, diffs)
	return dmp.PatchToText(patches)
}

func patchFile(pathToFile string, patch string) (err error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		return
	}
	textBase, err := getFileText(pathToFile)
	if err != nil {
		return
	}
	log.Debugf("patches: %+v", patches)
	newText, _ := dmp.PatchApply(patches, textBase)
	log.Debugf("newText: %s", newText)
	err = ioutil.WriteFile(pathToFile, []byte(newText), 0755)
	err = ioutil.WriteFile(fmt.Sprintf("%s.%d", pathToFile, time.Now().UnixNano()/1000000), []byte(patch), 0755)
	return
}
