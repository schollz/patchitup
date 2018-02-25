package patchitup

import (
	"github.com/schollz/patchitup/patchitup/keypair"
)

var sharedKey keypair.KeyPair

func init() {
	sharedKey = keypair.NewDeterministic("patchitup")
}
