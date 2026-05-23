//go:build js && wasm

package system_test

import (
	"testing"

	"github.com/vugra/vugra/pkg/system"
)

func TestDefaultFileSystemUsesWebFileSystemOnWasm(t *testing.T) {
	if _, ok := system.DefaultFileSystem().(*system.WebFileSystem); !ok {
		t.Fatalf("default filesystem = %T, want *system.WebFileSystem", system.DefaultFileSystem())
	}
}

func TestWebFileSystemRequiresSelectedDirectory(t *testing.T) {
	files := system.NewWebFileSystem()
	if _, err := files.ReadDir("."); err == nil {
		t.Fatal("ReadDir succeeded before a browser directory was selected")
	}
	if _, err := files.Stat("."); err == nil {
		t.Fatal("Stat succeeded before a browser directory was selected")
	}
}

func TestRequestDirectoryRequiresBrowserAPI(t *testing.T) {
	if system.BrowserFileSystemAvailable() {
		t.Skip("host provides showDirectoryPicker")
	}
	if err := system.RequestDirectory(); err == nil {
		t.Fatal("RequestDirectory succeeded without showDirectoryPicker")
	}
}
