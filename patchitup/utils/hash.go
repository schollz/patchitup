package utils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"regexp"
)

var convertWindowsLineFeed = regexp.MustCompile(`\r?\n`)

// Filemd5Sum returns the md5 sum of a file and produces the same
// hash for both Windows and Unix systems.
func Filemd5Sum(pathToFile string) (result string, err error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		return
	}
	defer file.Close()
	hash := md5.New()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := convertWindowsLineFeed.ReplaceAll(scanner.Bytes(), []byte("\n"))
		hash.Write(line)
	}
	result = hex.EncodeToString(hash.Sum(nil))
	return
}

// HashSHA256 returns a string hash of a byte
func HashSHA256(s []byte) string {
	h := sha256.New()
	h.Write(s)
	return string(base64.StdEncoding.EncodeToString(h.Sum(nil)))[:8]
}
