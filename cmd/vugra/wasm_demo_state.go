package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const wasmRefreshHook = "vugraRefresh"

func wasmDemoStateSource(path string) string {
	if !hasComponentBase(path, "FinderLite") {
		return ""
	}
	return finderWasmDemoStateSource()
}

func finderWasmDemoStateSource() string {
	return `package component

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall/js"
	"time"

	"github.com/vugra/vugra/pkg/system"
	"github.com/vugra/vugra/pkg/vugra"
)

func NewDemoState() vugra.State {
	state := &State{}
	model := newFinderModel(state)
	model.render()
	runtimeState := NewRuntimeState(state)
	runtimeState.Methods["OpenDocuments"] = model.requestDirectory
	runtimeState.Methods["OpenDownloads"] = model.requestDirectory
	runtimeState.Methods["OpenPictures"] = model.requestDirectory
	runtimeState.Methods["OpenProjectA"] = model.requestDirectory
	runtimeState.Methods["OpenProjectB"] = model.requestDirectory
	return runtimeState
}

type finderModel struct {
	state       *State
	sidebar     finderSidebar
	currentPath string
	entries     []system.Entry
	status      string
}

func newFinderModel(state *State) *finderModel {
	return &finderModel{
		state: state,
		status: "Select a browser directory to inspect files.",
		sidebar: finderSidebar{
			documents: ".",
			downloads: ".",
			pictures:  ".",
			projectA:  ".",
			projectB:  ".",
		},
	}
}

func (m *finderModel) requestDirectory() {
	m.state.Status.Set("Waiting for browser directory permission...")
	if err := system.RequestDirectoryAsync(func(err error) {
		if err != nil {
			m.status = err.Error()
			m.render()
			refreshVugra()
			return
		}
		m.currentPath = "."
		m.entries = nil
		m.status = ""
		m.load()
		m.render()
		refreshVugra()
	}); err != nil {
		m.status = err.Error()
		m.render()
	}
}

func refreshVugra() {
	hook := js.Global().Get("window").Get("` + wasmRefreshHook + `")
	if hook.Type() == js.TypeFunction {
		hook.Invoke()
	}
}

func (m *finderModel) load() {
	currentPath := strings.TrimSpace(m.currentPath)
	if currentPath == "" {
		currentPath = "."
	}
	entries, err := system.ReadDir(currentPath)
	if err != nil {
		m.entries = nil
		m.status = err.Error()
		return
	}
	m.currentPath = currentPath
	m.entries = entries
	m.status = fmt.Sprintf("%d items · Current path: %s", len(entries), currentPath)
}

func (m *finderModel) render() {
	state := m.state
	if strings.TrimSpace(m.currentPath) == "" {
		m.currentPath = "."
	}
	if m.status == "" {
		m.status = fmt.Sprintf("%d items · Current path: %s", len(m.entries), m.currentPath)
	}
	sidebar := m.sidebar
	currentPath := m.currentPath
	state.Status.Set(m.status)
	state.Path.Set(currentPath)
	state.Search.Set("")
	state.SelectedSummary.Set("0 items selected")
	state.FavoritesOpen.Set(true)
	state.WorkspaceOpen.Set(true)
	state.FavoritesClosed.Set(false)
	state.WorkspaceClosed.Set(false)
	state.FavoritesChevron.Set("▾")
	state.WorkspaceChevron.Set("▾")
	state.SidebarClass.Set("sidebar-200")
	state.SplitterClass.Set("splitter")
	state.DocumentsClass.Set(mapFinderClass(pathWithin(currentPath, sidebar.documents)))
	state.DownloadsClass.Set(mapFinderClass(pathWithin(currentPath, sidebar.downloads)))
	state.PicturesClass.Set(mapFinderClass(pathWithin(currentPath, sidebar.pictures)))
	state.ProjectAClass.Set(mapFinderClass(pathWithin(currentPath, sidebar.projectA)))
	state.ProjectBClass.Set(mapFinderClass(pathWithin(currentPath, sidebar.projectB)))
	state.FilePaneVisible.Set(true)
	state.ItemMenuOpen.Set(false)
	state.BlankMenuOpen.Set(false)
	state.PreviewOpen.Set(false)
	state.RenameText.Set("")
	state.PreviewTitle.Set("")
	state.PreviewBody.Set("")
	setFinderRows(state, m.entries)
}

func unusedFinderSidebarDefaults() finderSidebar {
	return finderSidebar{
		documents: filepath.Join(".", "Documents"),
		downloads: filepath.Join(".", "Downloads"),
		pictures:  filepath.Join(".", "Pictures"),
		projectA:  filepath.Join("Workspace", "Vugra"),
		projectB:  "Workspace",
	}
}

type finderSidebar struct {
	documents string
	downloads string
	pictures  string
	projectA  string
	projectB  string
}

func firstExistingFinderPath(files system.FileSystem, sidebar finderSidebar) string {
	for _, path := range []string{sidebar.documents, sidebar.downloads, sidebar.pictures, sidebar.projectA, sidebar.projectB, "."} {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if entry, err := files.Stat(path); err == nil && entry.Kind == "folder" {
			return filepath.Clean(path)
		}
	}
	return "."
}

func setFinderRows(state *State, entries []system.Entry) {
	for i := 0; i < 12; i++ {
		visible := i < len(entries)
		row := finderRowSignals(state, i+1)
		row.visible.Set(visible)
		row.class.Set("file-row")
		row.editing.Set(false)
		if !visible {
			row.text.Set("")
			row.name.Set("")
			row.modified.Set("")
			row.size.Set("")
			row.folder.Set(false)
			row.file.Set(false)
			continue
		}
		entry := entries[i]
		kind := entry.Kind
		if kind == "" {
			kind = "file"
		}
		folder := kind == "folder"
		row.text.Set(fmt.Sprintf("%-28s  %-12s  %8s", entry.Name, formatFinderModified(entry.ModifiedAt), formatFinderSize(entry)))
		row.name.Set(entry.Name)
		row.modified.Set(formatFinderModified(entry.ModifiedAt))
		row.size.Set(formatFinderSize(entry))
		row.folder.Set(folder)
		row.file.Set(!folder)
	}
}

type finderRow struct {
	text     interface{ Set(string) }
	name     interface{ Set(string) }
	modified interface{ Set(string) }
	size     interface{ Set(string) }
	class    interface{ Set(string) }
	visible  interface{ Set(bool) }
	editing  interface{ Set(bool) }
	folder   interface{ Set(bool) }
	file     interface{ Set(bool) }
}

func finderRowSignals(state *State, row int) finderRow {
	switch row {
	case 1:
		return finderRow{&state.Row1, &state.Row1Name, &state.Row1Modified, &state.Row1Size, &state.Row1Class, &state.Row1Visible, &state.Row1Editing, &state.Row1Folder, &state.Row1File}
	case 2:
		return finderRow{&state.Row2, &state.Row2Name, &state.Row2Modified, &state.Row2Size, &state.Row2Class, &state.Row2Visible, &state.Row2Editing, &state.Row2Folder, &state.Row2File}
	case 3:
		return finderRow{&state.Row3, &state.Row3Name, &state.Row3Modified, &state.Row3Size, &state.Row3Class, &state.Row3Visible, &state.Row3Editing, &state.Row3Folder, &state.Row3File}
	case 4:
		return finderRow{&state.Row4, &state.Row4Name, &state.Row4Modified, &state.Row4Size, &state.Row4Class, &state.Row4Visible, &state.Row4Editing, &state.Row4Folder, &state.Row4File}
	case 5:
		return finderRow{&state.Row5, &state.Row5Name, &state.Row5Modified, &state.Row5Size, &state.Row5Class, &state.Row5Visible, &state.Row5Editing, &state.Row5Folder, &state.Row5File}
	case 6:
		return finderRow{&state.Row6, &state.Row6Name, &state.Row6Modified, &state.Row6Size, &state.Row6Class, &state.Row6Visible, &state.Row6Editing, &state.Row6Folder, &state.Row6File}
	case 7:
		return finderRow{&state.Row7, &state.Row7Name, &state.Row7Modified, &state.Row7Size, &state.Row7Class, &state.Row7Visible, &state.Row7Editing, &state.Row7Folder, &state.Row7File}
	case 8:
		return finderRow{&state.Row8, &state.Row8Name, &state.Row8Modified, &state.Row8Size, &state.Row8Class, &state.Row8Visible, &state.Row8Editing, &state.Row8Folder, &state.Row8File}
	case 9:
		return finderRow{&state.Row9, &state.Row9Name, &state.Row9Modified, &state.Row9Size, &state.Row9Class, &state.Row9Visible, &state.Row9Editing, &state.Row9Folder, &state.Row9File}
	case 10:
		return finderRow{&state.Row10, &state.Row10Name, &state.Row10Modified, &state.Row10Size, &state.Row10Class, &state.Row10Visible, &state.Row10Editing, &state.Row10Folder, &state.Row10File}
	case 11:
		return finderRow{&state.Row11, &state.Row11Name, &state.Row11Modified, &state.Row11Size, &state.Row11Class, &state.Row11Visible, &state.Row11Editing, &state.Row11Folder, &state.Row11File}
	default:
		return finderRow{&state.Row12, &state.Row12Name, &state.Row12Modified, &state.Row12Size, &state.Row12Class, &state.Row12Visible, &state.Row12Editing, &state.Row12Folder, &state.Row12File}
	}
}

func mapFinderClass(selected bool) string {
	if selected {
		return "tree-item-selected"
	}
	return "tree-item"
}

func pathWithin(path, base string) bool {
	if path == "" || base == "" {
		return false
	}
	path = filepath.Clean(path)
	base = filepath.Clean(base)
	if path == base {
		return true
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

func formatFinderSize(entry system.Entry) string {
	if entry.Kind == "folder" {
		return "--"
	}
	if entry.Size >= 1000000 {
		return fmt.Sprintf("%.1f MB", float64(entry.Size)/1000000)
	}
	if entry.Size >= 1000 {
		return fmt.Sprintf("%.0f KB", float64(entry.Size)/1000)
	}
	return fmt.Sprintf("%d B", entry.Size)
}

func formatFinderModified(value time.Time) string {
	if value.IsZero() {
		return "--"
	}
	now := time.Now()
	if sameFinderDay(value, now) {
		return "Today " + value.Format("15:04")
	}
	if sameFinderDay(value, now.AddDate(0, 0, -1)) {
		return "Yesterday"
	}
	if value.Year() == now.Year() {
		return value.Format("Jan 02")
	}
	return value.Format("2006-01-02")
}

func sameFinderDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
`
}

func writeWasmDemoState(componentDir, entryPath string) error {
	source := wasmDemoStateSource(entryPath)
	if strings.TrimSpace(source) == "" {
		return nil
	}
	if err := os.WriteFile(filepath.Join(componentDir, "demo_state.go"), []byte(source), 0o644); err != nil {
		return fmt.Errorf("write generated wasm demo state: %w", err)
	}
	return nil
}
