//go:build darwin && cgo && vuego_native_window

package main

import (
	"os"
	"testing"
)

func TestGoFinderLiteNativeWindowSmokeDefaultsToVelloNative(t *testing.T) {
	os.Unsetenv("VUGRA_NATIVE_RENDERER")
	os.Unsetenv("VUEGO_NATIVE_RENDERER")
	if got := nativeRendererModeName(); got != "vello-native" {
		t.Fatalf("default native-window smoke renderer = %q, want vello-native", got)
	}
}
