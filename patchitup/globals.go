package patchitup

import "path"

var pathToCacheClient, pathToCacheServer string

func init() {
	pathToCacheClient = path.Join(UserHomeDir(), ".patchitup", "client")
	pathToCacheServer = path.Join(UserHomeDir(), ".patchitup", "server")
}
