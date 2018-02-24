package utils

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncrypt(t *testing.T) {
	key := GetRandomName()
	encrypted, salt, iv := Encrypt([]byte("hello, world"), key)
	decrypted, err := Decrypt(encrypted, key, salt, iv)
	if err != nil {
		t.Error(err)
	}
	if string(decrypted) != "hello, world" {
		t.Error("problem decrypting")
	}
	_, err = Decrypt(encrypted, "wrong passphrase", salt, iv)
	if err == nil {
		t.Error("should not work!")
	}
}

func TestEncryptFiles(t *testing.T) {
	key := GetRandomName()
	if err := ioutil.WriteFile("temp", []byte("hello, world!"), 0644); err != nil {
		t.Error(err)
	}
	if err := EncryptFile("temp", "temp.enc", key); err != nil {
		t.Error(err)
	}
	if err := DecryptFile("temp.enc", "temp.dec", key); err != nil {
		t.Error(err)
	}
	data, err := ioutil.ReadFile("temp.dec")
	if string(data) != "hello, world!" {
		t.Errorf("Got something weird: " + string(data))
	}
	if err != nil {
		t.Error(err)
	}
	if err := DecryptFile("temp.enc", "temp.dec", key+"wrong password"); err == nil {
		t.Error("should throw error!")
	}
	os.Remove("temp.dec")
	os.Remove("temp.enc")
	os.Remove("temp")
}

func TestBytesToFile(t *testing.T) {
	someBytes := []byte(`Why do we use it?
		It is a long established fact that a reader will be distracted by the readable content of a page when looking at its layout. The point of using Lorem Ipsum is that it has a more-or-less normal distribution of letters, as opposed to using 'Content here, content here', making it look like readable English. Many desktop publishing packages and web page editors now use Lorem Ipsum as their default model text, and a search for 'lorem ipsum' will uncover many web sites still in their infancy. Various versions have evolved over the years, sometimes by accident, sometimes on purpose (injected humour and the like).`)
	err := EncryptBytesToFile(someBytes, "hello.world", "1234")
	assert.Nil(t, err)

	b, err := DecryptBytesFromFile("hello.world", "1234")
	assert.Nil(t, err)
	assert.Equal(t, someBytes, b)
	b, err = DecryptAndDecompressBytesFromFile("hello.world", "124")
	assert.NotNil(t, err)
	assert.NotEqual(t, someBytes, b)
}

func TestBytesToFileWithCompression(t *testing.T) {
	someBytes := []byte(`Why do we use it?
		It is a long established fact that a reader will be distracted by the readable content of a page when looking at its layout. The point of using Lorem Ipsum is that it has a more-or-less normal distribution of letters, as opposed to using 'Content here, content here', making it look like readable English. Many desktop publishing packages and web page editors now use Lorem Ipsum as their default model text, and a search for 'lorem ipsum' will uncover many web sites still in their infancy. Various versions have evolved over the years, sometimes by accident, sometimes on purpose (injected humour and the like).`)
	err := CompressAndEncryptBytesToFile(someBytes, "hello.world2", "1234")
	assert.Nil(t, err)

	b, err := DecryptAndDecompressBytesFromFile("hello.world2", "1234")
	assert.Nil(t, err)
	assert.Equal(t, someBytes, b)
	b, err = DecryptAndDecompressBytesFromFile("hello.world2", "124")
	assert.NotNil(t, err)
	assert.NotEqual(t, someBytes, b)
}
