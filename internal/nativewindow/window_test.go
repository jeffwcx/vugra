//go:build !darwin || !cgo || !vuego_native_window

package nativewindow_test

import (
	"testing"

	"github.com/vugra/vugra/internal/nativewindow"
)

func TestNativeWindowDisabledByDefault(t *testing.T) {
	_, err := nativewindow.New("Vugra", 800, 600)
	if err == nil {
		t.Fatal("expected disabled native window backend by default")
	}
}
