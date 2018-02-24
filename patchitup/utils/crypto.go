package utils

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/mars9/crypt"
	"golang.org/x/crypto/pbkdf2"
)

// Encrypt will encrypt a string
func Encrypt(plaintext []byte, passphrase string, dontencrypt ...bool) (encrypted []byte, salt string, iv string) {
	if len(dontencrypt) > 0 && dontencrypt[0] {
		return plaintext, "salt", "iv"
	}
	key, saltBytes := deriveKey(passphrase, nil)
	ivBytes := make([]byte, 12)
	// http://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf
	// Section 8.2
	rand.Read(ivBytes)
	b, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(b)
	encrypted = aesgcm.Seal(nil, ivBytes, plaintext, nil)
	salt = hex.EncodeToString(saltBytes)
	iv = hex.EncodeToString(ivBytes)
	return
}

func Decrypt(data []byte, passphrase string, salt string, iv string, dontencrypt ...bool) (plaintext []byte, err error) {
	if len(dontencrypt) > 0 && dontencrypt[0] {
		return data, nil
	}
	saltBytes, _ := hex.DecodeString(salt)
	ivBytes, _ := hex.DecodeString(iv)
	key, _ := deriveKey(passphrase, saltBytes)
	b, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(b)
	plaintext, err = aesgcm.Open(nil, ivBytes, data, nil)
	return
}

func deriveKey(passphrase string, salt []byte) ([]byte, []byte) {
	if salt == nil {
		salt = make([]byte, 8)
		// http://www.ietf.org/rfc/rfc2898.txt
		// Salt.
		rand.Read(salt)
	}
	return pbkdf2.Key([]byte(passphrase), salt, 1000, 32, sha256.New), salt
}

func Hash(data string) string {
	return HashBytes([]byte(data))
}

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func EncryptFile(inputFilename string, outputFilename string, password string) error {
	return cryptFile(inputFilename, outputFilename, password, true)
}

func DecryptFile(inputFilename string, outputFilename string, password string) error {
	return cryptFile(inputFilename, outputFilename, password, false)
}

func cryptFile(inputFilename string, outputFilename string, password string, encrypt bool) error {
	in, err := os.Open(inputFilename)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer func() {
		out.Sync()
		out.Close()
	}()
	c := &crypt.Crypter{
		HashFunc: sha1.New,
		HashSize: sha1.Size,
		Key:      crypt.NewPbkdf2Key([]byte(password), 32),
	}
	if encrypt {
		if err := c.Encrypt(out, in); err != nil {
			return err
		}
	} else {
		if err := c.Decrypt(out, in); err != nil {
			return err
		}
	}
	return nil
}

// EncryptBytesToFile will take inputbytes and output them to a file
func EncryptBytesToFile(inputBytes []byte, outputFilename string, password string) (err error) {
	in := bytes.NewReader(inputBytes)
	out, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer func() {
		out.Sync()
		out.Close()
	}()
	c := &crypt.Crypter{
		HashFunc: sha1.New,
		HashSize: sha1.Size,
		Key:      crypt.NewPbkdf2Key([]byte(password), 32),
	}
	if err := c.Encrypt(out, in); err != nil {
		return err
	}
	return nil
}

// DecryptBytesFromFile will take a file and decrypt them.
func DecryptBytesFromFile(inputFilename string, password string) (output []byte, err error) {
	in, err := os.Open(inputFilename)
	if err != nil {
		return
	}
	defer in.Close()
	var b bytes.Buffer
	out := bufio.NewWriter(&b)
	c := &crypt.Crypter{
		HashFunc: sha1.New,
		HashSize: sha1.Size,
		Key:      crypt.NewPbkdf2Key([]byte(password), 32),
	}
	err = c.Decrypt(out, in)
	out.Flush()
	output = b.Bytes()
	return
}

// CompressAndEncryptBytesToFile will compress and then encrypt files to bytes.
func CompressAndEncryptBytesToFile(inputBytes []byte, outputFilename string, password string) (err error) {
	inputBytes = CompressByte(inputBytes)
	return EncryptBytesToFile(inputBytes, outputFilename, password)
}

// DecryptAndDecompressBytesFromFile will take a file and decrypt them.
func DecryptAndDecompressBytesFromFile(inputFilename string, password string) (output []byte, err error) {
	output, err = DecryptBytesFromFile(inputFilename, password)
	if err != nil {
		return
	}
	output = DecompressByte(output)
	return
}
