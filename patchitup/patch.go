package patchitup

import (
	"github.com/sergi/go-diff/diffmatchpatch"
)

func getPatch(text1, text2 string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(text1, text2, false)
	patches := dmp.PatchMake(text1, diffs)
	return dmp.PatchToText(patches)
}

func patchText(textBase string, patchText string) (newText string, err error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		return
	}
	newText, _ = dmp.PatchApply(patches, textBase)
	return
}
