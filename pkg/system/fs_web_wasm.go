//go:build js && wasm

package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

var (
	errWebFileSystemUnavailable = errors.New("browser File System Access API is not available")
	errWebDirectoryNotSelected  = errors.New("no browser directory has been selected")
)

func initialFileSystem() FileSystem {
	return NewWebFileSystem()
}

// WebFileSystem exposes a user-selected browser directory through the FileSystem
// interface. It is backed by the File System Access API; it does not fabricate
// files or read arbitrary host paths.
type WebFileSystem struct {
	mu      sync.RWMutex
	root    js.Value
	entries map[string]Entry
}

func NewWebFileSystem() *WebFileSystem {
	return &WebFileSystem{entries: map[string]Entry{}}
}

func RequestDirectory() error {
	files, ok := DefaultFileSystem().(*WebFileSystem)
	if !ok {
		return nil
	}
	return files.RequestDirectory()
}

func RequestDirectoryAsync(done func(error)) error {
	files, ok := DefaultFileSystem().(*WebFileSystem)
	if !ok {
		if done != nil {
			done(nil)
		}
		return nil
	}
	return files.RequestDirectoryAsync(done)
}

func BrowserFileSystemAvailable() bool {
	return js.Global().Get("showDirectoryPicker").Type() == js.TypeFunction
}

func (fsys *WebFileSystem) RequestDirectory() error {
	if !BrowserFileSystemAvailable() {
		return errWebFileSystemUnavailable
	}
	handle, err := await(js.Global().Call("showDirectoryPicker"))
	if err != nil {
		return err
	}
	if err := fsys.loadRoot(handle); err != nil {
		return err
	}
	return nil
}

func (fsys *WebFileSystem) RequestDirectoryAsync(done func(error)) error {
	if !BrowserFileSystemAvailable() {
		return errWebFileSystemUnavailable
	}
	promise := js.Global().Call("showDirectoryPicker")
	go func() {
		handle, err := await(promise)
		if err == nil {
			err = fsys.loadRoot(handle)
		}
		if done != nil {
			done(err)
		}
	}()
	return nil
}

func (fsys *WebFileSystem) ReadDir(path string) ([]Entry, error) {
	path = webCleanPath(path)
	fsys.mu.RLock()
	defer fsys.mu.RUnlock()
	if !fsys.hasRootLocked() {
		return nil, errWebDirectoryNotSelected
	}
	if path != "." {
		parent, ok := fsys.entries[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		if parent.Kind != "folder" {
			return nil, fmt.Errorf("read dir %q: not a folder", path)
		}
	}
	var out []Entry
	for childPath, entry := range fsys.entries {
		if childPath == path {
			continue
		}
		parent := filepath.Dir(childPath)
		if parent == path || (path == "." && !strings.Contains(childPath, string(filepath.Separator))) {
			out = append(out, entry)
		}
	}
	sortEntries(out)
	return out, nil
}

func (fsys *WebFileSystem) Stat(path string) (Entry, error) {
	path = webCleanPath(path)
	fsys.mu.RLock()
	defer fsys.mu.RUnlock()
	if !fsys.hasRootLocked() {
		return Entry{}, errWebDirectoryNotSelected
	}
	if path == "." {
		return Entry{Name: ".", Path: ".", Kind: "folder", ModifiedAt: time.Now()}, nil
	}
	entry, ok := fsys.entries[path]
	if !ok {
		return Entry{}, os.ErrNotExist
	}
	return entry, nil
}

func (fsys *WebFileSystem) Mkdir(path string) error {
	return errWebWriteUnsupported("mkdir", path)
}

func (fsys *WebFileSystem) Rename(oldPath, newPath string) error {
	return fmt.Errorf("rename %q to %q: browser write operations are not supported yet", oldPath, newPath)
}

func (fsys *WebFileSystem) Remove(path string) error {
	return errWebWriteUnsupported("remove", path)
}

func (fsys *WebFileSystem) Duplicate(srcPath, dstPath string) error {
	return fmt.Errorf("duplicate %q to %q: browser write operations are not supported yet", srcPath, dstPath)
}

func (fsys *WebFileSystem) loadRoot(root js.Value) error {
	entries := map[string]Entry{}
	if err := readDirectoryHandle(root, ".", entries); err != nil {
		return err
	}
	fsys.mu.Lock()
	defer fsys.mu.Unlock()
	fsys.root = root
	fsys.entries = entries
	return nil
}

func (fsys *WebFileSystem) hasRootLocked() bool {
	return fsys.root.Truthy()
}

func readDirectoryHandle(handle js.Value, path string, entries map[string]Entry) error {
	iterator := handle.Call("entries")
	for {
		next, err := await(iterator.Call("next"))
		if err != nil {
			return err
		}
		if next.Get("done").Bool() {
			break
		}
		value := next.Get("value")
		name := value.Index(0).String()
		child := value.Index(1)
		childPath := filepath.Join(path, name)
		if child.Get("kind").String() == "directory" {
			entries[childPath] = Entry{Name: name, Path: childPath, Kind: "folder", ModifiedAt: time.Now()}
			if err := readDirectoryHandle(child, childPath, entries); err != nil {
				return err
			}
			continue
		}
		file, err := await(child.Call("getFile"))
		if err != nil {
			return err
		}
		entries[childPath] = Entry{
			Name:       name,
			Path:       childPath,
			Kind:       "file",
			Size:       int64(file.Get("size").Int()),
			ModifiedAt: time.UnixMilli(int64(file.Get("lastModified").Float())),
		}
	}
	return nil
}

func await(promise js.Value) (js.Value, error) {
	done := make(chan struct{})
	var result js.Value
	var rejection js.Value
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		result = args[0]
		close(done)
		return nil
	})
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		rejection = args[0]
		close(done)
		return nil
	})
	defer then.Release()
	defer catch.Release()
	promise.Call("then", then).Call("catch", catch)
	<-done
	if rejection.Truthy() {
		return js.Value{}, jsError(rejection)
	}
	return result, nil
}

func jsError(value js.Value) error {
	if value.Get("message").Truthy() {
		return errors.New(value.Get("message").String())
	}
	return errors.New(value.String())
}

func errWebWriteUnsupported(op, path string) error {
	return fmt.Errorf("%s %q: browser write operations are not supported yet", op, path)
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "folder"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func webCleanPath(path string) string {
	path = cleanPath(path)
	if path == string(filepath.Separator) {
		return "."
	}
	return strings.TrimPrefix(path, string(filepath.Separator))
}

var _ FileSystem = (*WebFileSystem)(nil)
