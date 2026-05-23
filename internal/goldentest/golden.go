package goldentest

import (
	"os"
	"path/filepath"
	"testing"
)

func Assert(t *testing.T, name string, actual []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v", path, err)
	}
	if string(expected) != string(actual) {
		t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", path, expected, actual)
	}
}
