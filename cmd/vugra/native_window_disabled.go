//go:build !darwin || !cgo || !vuego_native_window

package main

import (
	"fmt"
)

func runNativeWindow(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	if canAutoRelaunchNative() {
		return relaunchWithNativeWindowTag(append([]string{"native-window"}, args...))
	}
	return fmt.Errorf("native-window is not enabled; build on darwin with cgo and -tags vuego_native_window")
}

func runProject(args []string) error {
	if len(args) > 1 {
		return usage()
	}
	if canAutoRelaunchNative() {
		return relaunchWithNativeWindowTag(append([]string{"run"}, args...))
	}
	return fmt.Errorf("run is not enabled; build on darwin with cgo and -tags vuego_native_window")
}

func runGoFinderLiteNativeWindowSmoke() error {
	if canAutoRelaunchNative() {
		return relaunchWithNativeWindowTag([]string{"go-finder-lite", "native-window-smoke"})
	}
	return fmt.Errorf("go-finder-lite native-window-smoke is not enabled; build on darwin with cgo and -tags vuego_native_window")
}
