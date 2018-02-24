package patchitup

import (
	"path"

	"github.com/schollz/utils"
)

var pathToCacheClient, pathToCacheServer string

func init() {
	pathToCacheClient = path.Join(utils.UserHomeDir(), ".patchitup", "client")
	pathToCacheServer = path.Join(utils.UserHomeDir(), ".patchitup", "server")
}
