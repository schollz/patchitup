package patchitup

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatchUp(t *testing.T) {
	SetLogLevel("debug")
	go func() {
		err := Run("8002")
		assert.Nil(t, err)
	}()

	// test on clean directory
	os.RemoveAll(path.Join(UserHomeDir(), ".patchitup"))
	CopyFile("client.go", "test1")

	err := PatchUp("http://localhost:8002", "testuser", "test1")
	assert.Nil(t, err)

	// remove the client folder to see if it reconstructs
	os.RemoveAll(path.Join(UserHomeDir(), ".patchitup", "client"))
	// change the test file
	CopyFile("server.go", "test1")

	err = PatchUp("http://localhost:8002", "testuser", "test1")
	assert.Nil(t, err)

	os.Remove("test1")
}
