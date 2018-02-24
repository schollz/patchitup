package patchitup

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"

	log "github.com/cihub/seelog"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/schollz/utils"
)

func encode(s string) (encoded string) {
	// compress patch
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := gz.Flush(); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}

	// encrypt patch
	encrypted := utils.Encrypt(b.Bytes(), []byte(`1234`))

	// convert to base64
	encoded = base64.StdEncoding.EncodeToString(encrypted)

	log.Debugf("compressed patch from %s to %s", humanize.Bytes(uint64(len(s))), humanize.Bytes(uint64(len(encoded))))
	return
}

func decode(s string) (decoded string, err error) {
	// convert from base64
	patchBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		err = errors.Wrap(err, "problem converting from base64")
		return
	}

	// decrypt
	decrypted, err := utils.Decrypt(patchBytes, []byte(`1234`))
	if err != nil {
		return
	}

	// decompress
	gr, err := gzip.NewReader(bytes.NewBuffer(decrypted))
	defer gr.Close()
	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return
	}
	decoded = string(data)
	return
}
