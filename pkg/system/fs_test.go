package system_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vugra/vugra/pkg/system"
)

func TestOSFileSystemReadRenameDuplicateRemove(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := system.OSFileSystem{}
	entries, err := files.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "docs" || entries[0].Kind != "folder" || entries[1].Name != "README.md" {
		t.Fatalf("entries = %+v", entries)
	}

	renamed := filepath.Join(root, "Guide.md")
	if err := files.Rename(readme, renamed); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	duplicate := filepath.Join(root, "Guide copy.md")
	if err := files.Duplicate(renamed, duplicate); err != nil {
		t.Fatalf("Duplicate: %v", err)
	}
	if got, err := os.ReadFile(duplicate); err != nil || string(got) != "hello" {
		t.Fatalf("duplicate content = %q, %v", got, err)
	}
	if err := files.Remove(duplicate); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(duplicate); !os.IsNotExist(err) {
		t.Fatalf("duplicate still exists: %v", err)
	}
}

func TestMockFileSystemOperations(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	files := system.NewMockFileSystem([]system.Entry{
		{Name: "Documents", Path: "/Documents", Kind: "folder", ModifiedAt: now},
		{Name: "Notes.txt", Path: "/Documents/Notes.txt", Kind: "file", Size: 12, ModifiedAt: now},
		{Name: "Archive", Path: "/Documents/Archive", Kind: "folder", ModifiedAt: now},
		{Name: "Old.txt", Path: "/Documents/Archive/Old.txt", Kind: "file", Size: 8, ModifiedAt: now},
	})

	entries, err := files.ReadDir("/Documents")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "Archive" || entries[1].Name != "Notes.txt" {
		t.Fatalf("entries = %+v", entries)
	}

	if err := files.Rename("/Documents/Archive", "/Documents/History"); err != nil {
		t.Fatalf("Rename folder: %v", err)
	}
	if entry, err := files.Stat("/Documents/History/Old.txt"); err != nil || entry.Name != "Old.txt" {
		t.Fatalf("renamed child = %+v, %v", entry, err)
	}

	if err := files.Duplicate("/Documents/History", "/Documents/History Copy"); err != nil {
		t.Fatalf("Duplicate folder: %v", err)
	}
	if _, err := files.Stat("/Documents/History Copy/Old.txt"); err != nil {
		t.Fatalf("duplicated child missing: %v", err)
	}

	if err := files.Remove("/Documents/History"); err != nil {
		t.Fatalf("Remove folder: %v", err)
	}
	if _, err := files.Stat("/Documents/History/Old.txt"); !system.IsNotExist(err) {
		t.Fatalf("removed child still exists: %v", err)
	}
}

func TestDefaultFileSystemCanBeReplaced(t *testing.T) {
	original := system.DefaultFileSystem()
	t.Cleanup(func() { system.SetFileSystem(original) })

	mock := system.NewMockFileSystem([]system.Entry{
		{Name: "tmp", Path: "/tmp", Kind: "folder"},
	})
	system.SetFileSystem(mock)
	if entries, err := system.ReadDir("/tmp"); err != nil || len(entries) != 0 {
		t.Fatalf("package ReadDir used wrong backend: entries=%+v err=%v", entries, err)
	}

	system.SetFileSystem(nil)
	if system.DefaultFileSystem() == nil {
		t.Fatal("default backend was not restored")
	}
}
