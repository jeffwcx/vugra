//go:build darwin && cgo && vuego_native_window

package nativewindow

import "runtime"

func init() {
	runtime.LockOSThread()
}
