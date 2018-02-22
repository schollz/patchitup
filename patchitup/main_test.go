package patchitup

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPatchUp(t *testing.T) {
	SetLogLevel("debug")
	go func() {
		err := Run("8002")
		assert.Nil(t, err)
	}()
	time.Sleep(100 * time.Millisecond)

	//
	// test on clean directory
	//
	err := os.RemoveAll(path.Join(UserHomeDir(), ".patchitup"))
	assert.Nil(t, err)
	err = CopyFile("client.go", "../test1")
	assert.Nil(t, err)

	err = PatchUp("http://localhost:8002", "testuser", "../test1")
	assert.Nil(t, err)
	// check that it copied correctly
	originalHash, err := Filemd5Sum("../test1")
	assert.Nil(t, err)
	serverHash, err := Filemd5Sum(path.Join(UserHomeDir(), ".patchitup", "server", "testuser", "test1"))
	assert.Nil(t, err)
	assert.Equal(t, originalHash, serverHash)

	//
	// remove the client folder to see if it reconstructs
	//
	err = os.RemoveAll(path.Join(UserHomeDir(), ".patchitup", "client"))
	assert.Nil(t, err)
	// change the test file
	os.Remove("../test1")
	err = CopyFile("server.go", "../test1")
	assert.Nil(t, err)

	err = PatchUp("http://localhost:8002", "testuser", "../test1")
	assert.Nil(t, err)
	// check that it copied correctly
	originalHash, err = Filemd5Sum("../test1")
	assert.Nil(t, err)
	serverHash, err = Filemd5Sum(path.Join(UserHomeDir(), ".patchitup", "server", "testuser", "test1"))
	assert.Nil(t, err)
	assert.Equal(t, originalHash, serverHash)

}
