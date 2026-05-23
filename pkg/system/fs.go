package system

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is the renderer-neutral file metadata exposed to Vugra component code.
type Entry struct {
	Name       string
	Path       string
	Kind       string
	Size       int64
	ModifiedAt time.Time
}

// FileSystem is the system boundary that native runtimes can back with os.* and
// sandboxed runtimes can replace with a mock implementation.
type FileSystem interface {
	ReadDir(path string) ([]Entry, error)
	Stat(path string) (Entry, error)
	Mkdir(path string) error
	Rename(oldPath, newPath string) error
	Remove(path string) error
	Duplicate(srcPath, dstPath string) error
}

var (
	defaultMu sync.RWMutex
	defaultFS FileSystem = initialFileSystem()
)

// SetFileSystem replaces the process-wide system backend used by package-level
// helpers. Passing nil restores the platform default backend.
func SetFileSystem(files FileSystem) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if files == nil {
		defaultFS = initialFileSystem()
		return
	}
	defaultFS = files
}

// DefaultFileSystem returns the active process-wide system backend.
func DefaultFileSystem() FileSystem {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultFS
}

func ReadDir(path string) ([]Entry, error) {
	return DefaultFileSystem().ReadDir(path)
}

func Stat(path string) (Entry, error) {
	return DefaultFileSystem().Stat(path)
}

func Mkdir(path string) error {
	return DefaultFileSystem().Mkdir(path)
}

func Rename(oldPath, newPath string) error {
	return DefaultFileSystem().Rename(oldPath, newPath)
}

func Remove(path string) error {
	return DefaultFileSystem().Remove(path)
}

func Duplicate(srcPath, dstPath string) error {
	return DefaultFileSystem().Duplicate(srcPath, dstPath)
}

type OSFileSystem struct{}

func (OSFileSystem) ReadDir(path string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		info, err := dirEntry.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, entryFromInfo(filepath.Join(path, dirEntry.Name()), info))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "folder"
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (OSFileSystem) Stat(path string) (Entry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	return entryFromInfo(path, info), nil
}

func (OSFileSystem) Mkdir(path string) error {
	return os.Mkdir(path, 0o755)
}

func (OSFileSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (OSFileSystem) Remove(path string) error {
	return os.RemoveAll(path)
}

func (fsys OSFileSystem) Duplicate(srcPath, dstPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(srcPath, dstPath)
	}
	return copyFile(srcPath, dstPath, info.Mode())
}

func entryFromInfo(path string, info os.FileInfo) Entry {
	kind := "file"
	if info.IsDir() {
		kind = "folder"
	}
	return Entry{
		Name:       info.Name(),
		Path:       filepath.Clean(path),
		Kind:       kind,
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}
}

func copyDir(srcPath, dstPath string) error {
	srcClean, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}
	dstClean, err := filepath.Abs(dstPath)
	if err != nil {
		return err
	}
	if dstClean == srcClean || strings.HasPrefix(dstClean, srcClean+string(filepath.Separator)) {
		return fmt.Errorf("duplicate %q: destination cannot be inside source", srcPath)
	}
	return filepath.WalkDir(srcPath, func(path string, dirEntry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstPath, rel)
		info, err := dirEntry.Info()
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(srcPath, dstPath string, mode os.FileMode) error {
	if srcPath == dstPath {
		return fmt.Errorf("duplicate %q: source and destination are the same", srcPath)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	written := false
	defer func() {
		dst.Close()
		if !written {
			os.Remove(dstPath)
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	written = true
	return dst.Close()
}

type MockFileSystem struct {
	mu      sync.RWMutex
	entries map[string]mockEntry
}

// Seed creates or replaces entries in the in-memory filesystem. It is useful
// for demos and tests that need predictable sandbox contents.
func (m *MockFileSystem) Seed(entries []Entry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range entries {
		m.putLocked(entry, nil)
	}
}

type mockEntry struct {
	entry Entry
	data  []byte
}

type mockPathEntry struct {
	path  string
	entry mockEntry
}

func NewMockFileSystem(entries []Entry) *MockFileSystem {
	mock := &MockFileSystem{entries: map[string]mockEntry{}}
	for _, entry := range entries {
		mock.put(entry, nil)
	}
	return mock
}

func (m *MockFileSystem) ReadDir(path string) ([]Entry, error) {
	path = cleanPath(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if path != "." {
		parent, ok := m.entries[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		if parent.entry.Kind != "folder" {
			return nil, fmt.Errorf("read dir %q: not a folder", path)
		}
	}
	var out []Entry
	for childPath, stored := range m.entries {
		if childPath == path {
			continue
		}
		parent := filepath.Dir(childPath)
		if parent == path || (path == "." && !strings.Contains(childPath, string(filepath.Separator))) {
			out = append(out, stored.entry)
			continue
		}
		if path == string(filepath.Separator) && filepath.Dir(childPath) == string(filepath.Separator) {
			out = append(out, stored.entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == "folder"
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (m *MockFileSystem) Stat(path string) (Entry, error) {
	path = cleanPath(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	stored, ok := m.entries[path]
	if !ok {
		return Entry{}, os.ErrNotExist
	}
	return stored.entry, nil
}

func (m *MockFileSystem) Mkdir(path string) error {
	path = cleanPath(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[path]; ok {
		return os.ErrExist
	}
	parent := filepath.Dir(path)
	if parent != "." && parent != path {
		if stored, ok := m.entries[parent]; !ok || stored.entry.Kind != "folder" {
			return os.ErrNotExist
		}
	}
	m.putLocked(Entry{Name: filepath.Base(path), Path: path, Kind: "folder", ModifiedAt: time.Now()}, nil)
	return nil
}

func (m *MockFileSystem) Rename(oldPath, newPath string) error {
	oldPath = cleanPath(oldPath)
	newPath = cleanPath(newPath)
	m.mu.Lock()
	defer m.mu.Unlock()
	stored, ok := m.entries[oldPath]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := m.entries[newPath]; ok {
		return os.ErrExist
	}
	delete(m.entries, oldPath)
	stored.entry.Path = newPath
	stored.entry.Name = filepath.Base(newPath)
	m.entries[newPath] = stored
	if stored.entry.Kind == "folder" {
		m.renameChildrenLocked(oldPath, newPath)
	}
	return nil
}

func (m *MockFileSystem) Remove(path string) error {
	path = cleanPath(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[path]; !ok {
		return os.ErrNotExist
	}
	delete(m.entries, path)
	prefix := path + string(filepath.Separator)
	for childPath := range m.entries {
		if strings.HasPrefix(childPath, prefix) {
			delete(m.entries, childPath)
		}
	}
	return nil
}

func (m *MockFileSystem) Duplicate(srcPath, dstPath string) error {
	srcPath = cleanPath(srcPath)
	dstPath = cleanPath(dstPath)
	m.mu.Lock()
	defer m.mu.Unlock()
	stored, ok := m.entries[srcPath]
	if !ok {
		return os.ErrNotExist
	}
	if _, ok := m.entries[dstPath]; ok {
		return os.ErrExist
	}
	stored.entry.Path = dstPath
	stored.entry.Name = filepath.Base(dstPath)
	m.entries[dstPath] = stored
	if stored.entry.Kind == "folder" {
		var children []mockPathEntry
		prefix := srcPath + string(filepath.Separator)
		for childPath, child := range m.entries {
			if !strings.HasPrefix(childPath, prefix) {
				continue
			}
			children = append(children, mockPathEntry{path: childPath, entry: child})
		}
		for _, child := range children {
			targetPath := dstPath + strings.TrimPrefix(child.path, srcPath)
			child.entry.entry.Path = targetPath
			child.entry.entry.Name = filepath.Base(targetPath)
			m.entries[targetPath] = child.entry
		}
	}
	return nil
}

func (m *MockFileSystem) put(entry Entry, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putLocked(entry, data)
}

func (m *MockFileSystem) putLocked(entry Entry, data []byte) {
	if entry.Path == "" {
		entry.Path = entry.Name
	}
	entry.Path = cleanPath(entry.Path)
	if entry.Name == "" {
		entry.Name = filepath.Base(entry.Path)
	}
	if entry.Kind == "" {
		entry.Kind = "file"
	}
	if entry.ModifiedAt.IsZero() {
		entry.ModifiedAt = time.Now()
	}
	m.entries[entry.Path] = mockEntry{entry: entry, data: append([]byte(nil), data...)}
}

func (m *MockFileSystem) renameChildrenLocked(oldPath, newPath string) {
	prefix := oldPath + string(filepath.Separator)
	renamed := map[string]mockEntry{}
	for childPath, child := range m.entries {
		if !strings.HasPrefix(childPath, prefix) {
			continue
		}
		delete(m.entries, childPath)
		targetPath := newPath + strings.TrimPrefix(childPath, oldPath)
		child.entry.Path = targetPath
		child.entry.Name = filepath.Base(targetPath)
		renamed[targetPath] = child
	}
	for targetPath, child := range renamed {
		m.entries[targetPath] = child
	}
}

func cleanPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "."
	}
	return filepath.Clean(path)
}

var _ FileSystem = OSFileSystem{}
var _ FileSystem = (*MockFileSystem)(nil)

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
