package patchitup

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	math_rand "math/rand"
	"os"
	"runtime"
	"time"
)

// Filemd5Sum returns the md5 sum of a file
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

func HashSHA256(s []byte) string {
	h := sha256.New()
	h.Write(s)
	return string(base64.StdEncoding.EncodeToString(h.Sum(nil)))[:8]
}

// UserHomeDir returns the user home directory
// taken from go1.8c2
// https://stackoverflow.com/a/41786440
func UserHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

// Exists returns whether the given file or directory exists or not
// from http://stackoverflow.com/questions/10510691/how-to-check-whether-a-file-or-directory-denoted-by-a-path-exists-in-golang
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, ~~attempt to create a hard link
// between the two files. If that fail,~~ copy the file contents from src to dst.
// from http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	// if err = os.Link(src, dst); err == nil {
	// 	return
	// }
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
// from http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// src is seeds the random generator for generating random strings
var src = math_rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandStringBytesMaskImprSrc prints a random string
func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func GzipFile(filename string) (err error) {
	copiedBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	compressedBytes, err := GzipBytes(convertWindowsLineFeed.ReplaceAll(copiedBytes, []byte("\n")))
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filename+".gz", compressedBytes, 0755)
	return
}

func GunzipBytes(compressed []byte) (uncompressed []byte, err error) {
	if len(compressed) == 0 {
		uncompressed = compressed
		return
	}
	gr, err := gzip.NewReader(bytes.NewBuffer(compressed))
	if err != nil {
		return
	}
	defer gr.Close()
	uncompressed, err = ioutil.ReadAll(gr)
	return
}

func GzipBytes(uncompressed []byte) (compressed []byte, err error) {
	if len(uncompressed) == 0 {
		compressed = uncompressed
		return
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err = gz.Write(uncompressed); err != nil {
		return
	}
	if err = gz.Flush(); err != nil {
		return
	}
	if err = gz.Close(); err != nil {
		return
	}
	compressed = b.Bytes()
	return
}
