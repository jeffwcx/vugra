//go:build !js || !wasm

package system

import "errors"

var errBrowserFileSystemUnavailable = errors.New("browser File System Access API is not available")

func RequestDirectory() error {
	return errBrowserFileSystemUnavailable
}

func RequestDirectoryAsync(done func(error)) error {
	return errBrowserFileSystemUnavailable
}

func BrowserFileSystemAvailable() bool {
	return false
}
