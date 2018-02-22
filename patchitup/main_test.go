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
	err := os.RemoveAll(path.Join(UserHomeDir(), ".patchitup"))
	assert.Nil(t, err)
	err = CopyFile("client.go", "test1")
	assert.Nil(t, err)

	err = PatchUp("http://localhost:8002", "testuser", "test1")
	assert.Nil(t, err)

	// remove the client folder to see if it reconstructs
	err = os.RemoveAll(path.Join(UserHomeDir(), ".patchitup", "client"))
	assert.Nil(t, err)
	// change the test file
	err = CopyFile("server.go", "test1")
	assert.Nil(t, err)

	err = PatchUp("http://localhost:8002", "testuser", "test1")
	assert.Nil(t, err)

	os.Remove("test1")
}
