//go:build !js || !wasm

package system

func initialFileSystem() FileSystem {
	return OSFileSystem{}
}
