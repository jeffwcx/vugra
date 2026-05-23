package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/pkg/system"
)

func demoStateFor(path string, scheduler *reactivity.Scheduler) runtime.State {
	if hasComponentBase(path, "FinderLite") {
		return newFinderDemoState(scheduler).state()
	}
	if hasComponentBase(path, "SystemFiles") {
		return newSystemFilesDemoState(scheduler)
	}
	if hasComponentBase(path, "Parent") {
		title := runtime.NewSignal("Parent using Go import", scheduler)
		return runtime.State{
			Signals: map[string]runtime.Signal{"title": title},
			Methods: map[string]func(){
				"OnBadgeClick": func() {
					title.Set("Badge clicked")
				},
			},
		}
	}
	return newCounterDemoState(scheduler)
}

func hasComponentBase(path, name string) bool {
	base := filepath.Base(path)
	return strings.EqualFold(base, name+".vue") || strings.EqualFold(base, name+".vugra")
}

func newCounterDemoState(scheduler *reactivity.Scheduler) runtime.State {
	count := runtime.NewSignal(0, scheduler)
	return runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() {
				count.Update(func(value int) int { return value + 1 })
				fmt.Fprintf(os.Stderr, "vugra native Inc count=%d\n", count.Get())
			},
		},
	}
}

func newSystemFilesDemoState(scheduler *reactivity.Scheduler) runtime.State {
	path := runtime.NewSignal(".", scheduler)
	status := runtime.NewSignal("Ready", scheduler)
	load := func() {
		current := strings.TrimSpace(path.Get())
		if current == "" {
			current = "."
			path.Set(current)
		}
		entries, err := system.ReadDir(current)
		if err != nil {
			status.Set(err.Error())
			return
		}
		status.Set(fmt.Sprintf("%d entries in %s", len(entries), current))
	}
	return runtime.State{
		Signals: map[string]runtime.Signal{
			"path":   path,
			"status": status,
		},
		Methods: map[string]func(){
			"Load": load,
		},
	}
}

type finderNode struct {
	id         string
	name       string
	kind       string
	size       int64
	modifiedAt string
	path       string
}

type finderDemoState struct {
	scheduler *reactivity.Scheduler
	signals   map[string]runtime.Signal
	strings   map[string]*runtime.SignalValue[string]
	bools     map[string]*runtime.SignalValue[bool]

	files       system.FileSystem
	sidebar     finderSidebar
	rows        []*finderNode
	currentPath string
	lastError   string
	history     []string
	forward     []string
	selected    map[string]bool
	anchor      int
	focus       int
	hover       int
	menuIndex   int
	renameID    string
	renameIndex int
	sidebarMode int
}

type finderSidebar struct {
	documents string
	downloads string
	pictures  string
	projectA  string
	projectB  string
}

func newFinderDemoState(scheduler *reactivity.Scheduler) *finderDemoState {
	sidebar := defaultFinderSidebar()
	f := &finderDemoState{
		scheduler:   scheduler,
		signals:     map[string]runtime.Signal{},
		strings:     map[string]*runtime.SignalValue[string]{},
		bools:       map[string]*runtime.SignalValue[bool]{},
		files:       system.DefaultFileSystem(),
		sidebar:     sidebar,
		currentPath: firstExistingFinderPath(system.DefaultFileSystem(), sidebar),
		selected:    map[string]bool{},
		anchor:      -1,
		focus:       -1,
		hover:       -1,
		menuIndex:   -1,
		renameIndex: -1,
		sidebarMode: 0,
	}
	for _, name := range []string{
		"path", "search", "status", "selectedSummary",
		"favoritesChevron", "workspaceChevron", "sidebarClass", "splitterClass",
		"documentsClass", "downloadsClass", "picturesClass", "projectAClass", "projectBClass",
		"renameText", "previewTitle", "previewBody",
	} {
		f.stringSig(name, "")
	}
	for i := 1; i <= 12; i++ {
		f.stringSig(fmt.Sprintf("row%d", i), "")
		f.stringSig(fmt.Sprintf("row%dName", i), "")
		f.stringSig(fmt.Sprintf("row%dModified", i), "")
		f.stringSig(fmt.Sprintf("row%dSize", i), "")
		f.stringSig(fmt.Sprintf("row%dClass", i), "file-row")
		f.boolSig(fmt.Sprintf("row%dVisible", i), false)
		f.boolSig(fmt.Sprintf("row%dEditing", i), false)
		f.boolSig(fmt.Sprintf("row%dFolder", i), false)
		f.boolSig(fmt.Sprintf("row%dFile", i), false)
	}
	for _, name := range []string{
		"favoritesOpen", "workspaceOpen", "favoritesClosed", "workspaceClosed", "filePaneVisible", "itemMenuOpen", "blankMenuOpen", "previewOpen",
	} {
		f.boolSig(name, false)
	}
	f.bools["favoritesOpen"].Set(true)
	f.bools["workspaceOpen"].Set(true)
	f.bools["filePaneVisible"].Set(true)
	f.strings["search"].Subscribe(func() { f.recompute() })
	f.recompute()
	return f
}

func newFinderDemoStateWithFileSystem(scheduler *reactivity.Scheduler, files system.FileSystem, sidebar finderSidebar) *finderDemoState {
	if files == nil {
		files = system.OSFileSystem{}
	}
	if sidebar.documents == "" && sidebar.downloads == "" && sidebar.pictures == "" && sidebar.projectA == "" && sidebar.projectB == "" {
		sidebar = defaultFinderSidebar()
	}
	return newFinderDemoStateConfigured(scheduler, files, sidebar)
}

func newFinderDemoStateConfigured(scheduler *reactivity.Scheduler, files system.FileSystem, sidebar finderSidebar) *finderDemoState {
	f := &finderDemoState{
		scheduler:   scheduler,
		signals:     map[string]runtime.Signal{},
		strings:     map[string]*runtime.SignalValue[string]{},
		bools:       map[string]*runtime.SignalValue[bool]{},
		files:       files,
		sidebar:     sidebar,
		currentPath: firstExistingFinderPath(files, sidebar),
		selected:    map[string]bool{},
		anchor:      -1,
		focus:       -1,
		hover:       -1,
		menuIndex:   -1,
		renameIndex: -1,
		sidebarMode: 0,
	}
	for _, name := range []string{
		"path", "search", "status", "selectedSummary",
		"favoritesChevron", "workspaceChevron", "sidebarClass", "splitterClass",
		"documentsClass", "downloadsClass", "picturesClass", "projectAClass", "projectBClass",
		"renameText", "previewTitle", "previewBody",
	} {
		f.stringSig(name, "")
	}
	for i := 1; i <= 12; i++ {
		f.stringSig(fmt.Sprintf("row%d", i), "")
		f.stringSig(fmt.Sprintf("row%dName", i), "")
		f.stringSig(fmt.Sprintf("row%dModified", i), "")
		f.stringSig(fmt.Sprintf("row%dSize", i), "")
		f.stringSig(fmt.Sprintf("row%dClass", i), "file-row")
		f.boolSig(fmt.Sprintf("row%dVisible", i), false)
		f.boolSig(fmt.Sprintf("row%dEditing", i), false)
		f.boolSig(fmt.Sprintf("row%dFolder", i), false)
		f.boolSig(fmt.Sprintf("row%dFile", i), false)
	}
	for _, name := range []string{
		"favoritesOpen", "workspaceOpen", "favoritesClosed", "workspaceClosed", "filePaneVisible", "itemMenuOpen", "blankMenuOpen", "previewOpen",
	} {
		f.boolSig(name, false)
	}
	f.bools["favoritesOpen"].Set(true)
	f.bools["workspaceOpen"].Set(true)
	f.bools["filePaneVisible"].Set(true)
	f.strings["search"].Subscribe(func() { f.recompute() })
	f.recompute()
	return f
}

func (f *finderDemoState) stringSig(name, initial string) *runtime.SignalValue[string] {
	signal := runtime.NewSignal(initial, f.scheduler)
	f.signals[name] = signal
	f.strings[name] = signal
	return signal
}

func (f *finderDemoState) boolSig(name string, initial bool) *runtime.SignalValue[bool] {
	signal := runtime.NewSignal(initial, f.scheduler)
	f.signals[name] = signal
	f.bools[name] = signal
	return signal
}

func (f *finderDemoState) state() runtime.State {
	methods := map[string]func(){
		"Back":              f.back,
		"Forward":           f.forwardPath,
		"ToggleFavorites":   f.toggleFavorites,
		"ToggleWorkspace":   f.toggleWorkspace,
		"OpenDocuments":     func() { f.navigate(f.sidebar.documents, true) },
		"OpenDownloads":     func() { f.navigate(f.sidebar.downloads, true) },
		"OpenPictures":      func() { f.navigate(f.sidebar.pictures, true) },
		"OpenProjectA":      func() { f.navigate(f.sidebar.projectA, true) },
		"OpenProjectB":      func() { f.navigate(f.sidebar.projectB, true) },
		"ClearSelection":    f.clearSelection,
		"DismissOverlay":    f.dismissOverlay,
		"OpenSelected":      f.openSelected,
		"BeginRename":       f.beginRename,
		"CancelRename":      f.cancelRename,
		"CommitRename":      f.commitRename,
		"DeleteSelected":    f.deleteSelected,
		"DuplicateSelected": f.duplicateSelected,
		"NewFolder":         f.newFolder,
		"Paste":             f.dismissOverlay,
		"Refresh":           f.refresh,
		"ClosePreview":      f.closePreview,
	}
	events := map[string]func(runtime.Event){
		"FocusList":      f.handleListEvent,
		"FocusRename":    f.handleRenameEvent,
		"RenameKey":      f.handleRenameEvent,
		"HoverSplitter":  f.hoverSplitter,
		"ResizeSidebar":  f.resizeSidebar,
		"ClearSelection": func(event runtime.Event) { f.clearSelection() },
		"DismissOverlay": func(event runtime.Event) { f.dismissOverlay() },
		"ShowBlankMenu":  func(event runtime.Event) { f.showBlankMenu() },
		"Escape":         f.handleListEvent,
	}
	for i := 1; i <= 12; i++ {
		index := i - 1
		events[fmt.Sprintf("SelectRow%d", i)] = func(event runtime.Event) {
			if event.Type == "key" {
				f.handleListEvent(event)
				return
			}
			f.selectVisibleIndex(index, event.Modifiers)
		}
		events[fmt.Sprintf("OpenRow%d", i)] = func(event runtime.Event) {
			if event.Type == "key" {
				f.handleListEvent(event)
				return
			}
			f.openVisibleIndex(index)
		}
		events[fmt.Sprintf("ShowRow%dMenu", i)] = func(event runtime.Event) {
			if event.Type == "key" {
				f.handleListEvent(event)
				return
			}
			f.showRowMenu(index, event.Modifiers)
		}
		events[fmt.Sprintf("HoverRow%d", i)] = func(event runtime.Event) {
			f.hoverVisibleIndex(index)
		}
	}
	return runtime.State{Signals: f.signals, Methods: methods, EventMethods: events}
}

func (f *finderDemoState) visibleRows() []*finderNode {
	nodes := append([]*finderNode(nil), f.rows...)
	query := strings.ToLower(strings.TrimSpace(f.strings["search"].Get()))
	if query == "" {
		return nodes
	}
	out := make([]*finderNode, 0, len(nodes))
	for _, node := range nodes {
		if strings.Contains(strings.ToLower(node.name), query) {
			out = append(out, node)
		}
	}
	return out
}

func (f *finderDemoState) recompute() {
	f.loadCurrentDirectory()
	f.strings["path"].Set(f.currentPath)
	f.strings["sidebarClass"].Set([]string{"sidebar", "sidebar-200", "sidebar-280", "sidebar-320"}[f.sidebarMode])
	f.strings["splitterClass"].Set("splitter")
	f.strings["favoritesChevron"].Set(mapBool(f.bools["favoritesOpen"].Get(), "▾", "▸"))
	f.strings["workspaceChevron"].Set(mapBool(f.bools["workspaceOpen"].Get(), "▾", "▸"))
	f.bools["favoritesClosed"].Set(!f.bools["favoritesOpen"].Get())
	f.bools["workspaceClosed"].Set(!f.bools["workspaceOpen"].Get())
	f.updateSidebarClasses()
	rows := f.visibleRows()
	for i := 0; i < 12; i++ {
		row := i + 1
		visible := i < len(rows)
		f.bools[fmt.Sprintf("row%dVisible", row)].Set(visible)
		if !visible {
			f.strings[fmt.Sprintf("row%d", row)].Set("")
			f.strings[fmt.Sprintf("row%dName", row)].Set("")
			f.strings[fmt.Sprintf("row%dModified", row)].Set("")
			f.strings[fmt.Sprintf("row%dSize", row)].Set("")
			f.strings[fmt.Sprintf("row%dClass", row)].Set("file-row")
			f.bools[fmt.Sprintf("row%dEditing", row)].Set(false)
			f.bools[fmt.Sprintf("row%dFolder", row)].Set(false)
			f.bools[fmt.Sprintf("row%dFile", row)].Set(false)
			continue
		}
		node := rows[i]
		editing := f.renameID == node.id
		f.bools[fmt.Sprintf("row%dEditing", row)].Set(editing)
		isFolder := node.kind == "folder"
		f.bools[fmt.Sprintf("row%dFolder", row)].Set(isFolder)
		f.bools[fmt.Sprintf("row%dFile", row)].Set(!isFolder)
		f.strings[fmt.Sprintf("row%d", row)].Set(f.rowText(node))
		f.strings[fmt.Sprintf("row%dName", row)].Set(node.name)
		f.strings[fmt.Sprintf("row%dModified", row)].Set(node.modifiedAt)
		f.strings[fmt.Sprintf("row%dSize", row)].Set(formatSize(node))
		class := "file-row"
		if editing {
			class = "file-row-editing"
		} else if f.selected[node.id] {
			class = "file-row-selected"
		} else if i == f.focus {
			class = "file-row-focus"
		} else if i == f.hover {
			class = "file-row-hover"
		}
		f.strings[fmt.Sprintf("row%dClass", row)].Set(class)
	}
	selected := f.selectedCount()
	total := len(rows)
	f.bools["filePaneVisible"].Set(!f.bools["itemMenuOpen"].Get() && !f.bools["blankMenuOpen"].Get() && !f.bools["previewOpen"].Get())
	status := fmt.Sprintf("%d items · Current path: %s", total, f.currentPath)
	if f.lastError != "" {
		status = f.lastError
	}
	f.strings["status"].Set(status)
	f.strings["selectedSummary"].Set(fmt.Sprintf("%d items selected", selected))
}

func (f *finderDemoState) updateSidebarClasses() {
	selected := map[string]bool{
		f.sidebar.documents: pathWithin(f.currentPath, f.sidebar.documents),
		f.sidebar.downloads: pathWithin(f.currentPath, f.sidebar.downloads),
		f.sidebar.pictures:  pathWithin(f.currentPath, f.sidebar.pictures),
		f.sidebar.projectA:  pathWithin(f.currentPath, f.sidebar.projectA),
		f.sidebar.projectB:  pathWithin(f.currentPath, f.sidebar.projectB),
	}
	f.strings["documentsClass"].Set(mapBool(selected[f.sidebar.documents], "tree-item-selected", "tree-item"))
	f.strings["downloadsClass"].Set(mapBool(selected[f.sidebar.downloads], "tree-item-selected", "tree-item"))
	f.strings["picturesClass"].Set(mapBool(selected[f.sidebar.pictures], "tree-item-selected", "tree-item"))
	f.strings["projectAClass"].Set(mapBool(selected[f.sidebar.projectA], "tree-item-selected", "tree-item"))
	f.strings["projectBClass"].Set(mapBool(selected[f.sidebar.projectB], "tree-item-selected", "tree-item"))
}

func (f *finderDemoState) rowText(node *finderNode) string {
	return fmt.Sprintf("%-28s  %-12s  %8s", node.name, node.modifiedAt, formatSize(node))
}

func formatSize(node *finderNode) string {
	if node.kind == "folder" {
		return "--"
	}
	if node.size >= 1000000 {
		return fmt.Sprintf("%.1f MB", float64(node.size)/1000000)
	}
	if node.size >= 1000 {
		return fmt.Sprintf("%.0f KB", float64(node.size)/1000)
	}
	return fmt.Sprintf("%d B", node.size)
}

func mapBool(value bool, yes, no string) string {
	if value {
		return yes
	}
	return no
}

func (f *finderDemoState) navigate(path string, remember bool) {
	if strings.TrimSpace(path) == "" {
		return
	}
	path = filepath.Clean(path)
	if entry, err := f.files.Stat(path); err != nil || entry.Kind != "folder" {
		if err != nil {
			f.lastError = err.Error()
		} else {
			f.lastError = fmt.Sprintf("%s is not a folder", path)
		}
		f.recompute()
		return
	}
	if remember && path != f.currentPath {
		f.history = append(f.history, f.currentPath)
		f.forward = nil
	}
	f.currentPath = path
	f.selected = map[string]bool{}
	f.anchor = -1
	f.focus = -1
	f.hover = -1
	f.renameID = ""
	f.renameIndex = -1
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) back() {
	if len(f.history) == 0 {
		return
	}
	last := f.history[len(f.history)-1]
	f.history = f.history[:len(f.history)-1]
	f.forward = append(f.forward, f.currentPath)
	f.navigate(last, false)
}

func (f *finderDemoState) forwardPath() {
	if len(f.forward) == 0 {
		return
	}
	next := f.forward[len(f.forward)-1]
	f.forward = f.forward[:len(f.forward)-1]
	f.history = append(f.history, f.currentPath)
	f.navigate(next, false)
}

func (f *finderDemoState) toggleFavorites() {
	f.bools["favoritesOpen"].Set(!f.bools["favoritesOpen"].Get())
	f.recompute()
}

func (f *finderDemoState) toggleWorkspace() {
	f.bools["workspaceOpen"].Set(!f.bools["workspaceOpen"].Get())
	f.recompute()
}

func (f *finderDemoState) hoverSplitter(event runtime.Event) {
	f.strings["splitterClass"].Set("splitter-hover")
}

func (f *finderDemoState) resizeSidebar(event runtime.Event) {
	if event.DeltaX < -8 && f.sidebarMode > 1 {
		f.sidebarMode--
	} else if event.DeltaX < -8 && f.sidebarMode == 1 {
		f.sidebarMode = 0
	} else if event.DeltaX > 8 && f.sidebarMode < 3 {
		f.sidebarMode++
	}
	f.recompute()
}

func (f *finderDemoState) selectVisibleIndex(index int, modifiers runtime.Modifiers) {
	rows := f.visibleRows()
	if index < 0 || index >= len(rows) {
		return
	}
	node := rows[index]
	if modifiers.Shift && f.anchor >= 0 {
		start, end := f.anchor, index
		if start > end {
			start, end = end, start
		}
		f.selected = map[string]bool{}
		for i := start; i <= end && i < len(rows); i++ {
			f.selected[rows[i].id] = true
		}
	} else if modifiers.Command() {
		if f.selected[node.id] {
			delete(f.selected, node.id)
		} else {
			f.selected[node.id] = true
		}
		f.anchor = index
	} else {
		f.selected = map[string]bool{node.id: true}
		f.anchor = index
	}
	f.focus = index
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) hoverVisibleIndex(index int) {
	if index == f.hover {
		return
	}
	f.hover = index
	f.recompute()
}

func (f *finderDemoState) openVisibleIndex(index int) {
	rows := f.visibleRows()
	if index < 0 || index >= len(rows) {
		return
	}
	node := rows[index]
	if node.kind == "folder" {
		f.navigate(node.path, true)
		return
	}
	f.showPreview(node)
}

func (f *finderDemoState) showRowMenu(index int, modifiers runtime.Modifiers) {
	rows := f.visibleRows()
	if index < 0 || index >= len(rows) {
		return
	}
	if !f.selected[rows[index].id] {
		f.selectVisibleIndex(index, modifiers)
	}
	f.menuIndex = index
	f.bools["blankMenuOpen"].Set(false)
	f.bools["itemMenuOpen"].Set(true)
	f.bools["previewOpen"].Set(false)
	f.recompute()
}

func (f *finderDemoState) showBlankMenu() {
	f.bools["itemMenuOpen"].Set(false)
	f.bools["blankMenuOpen"].Set(true)
	f.bools["previewOpen"].Set(false)
	f.recompute()
}

func (f *finderDemoState) dismissOverlay() {
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) dismissOverlayOnly() {
	f.bools["itemMenuOpen"].Set(false)
	f.bools["blankMenuOpen"].Set(false)
}

func (f *finderDemoState) clearSelection() {
	f.selected = map[string]bool{}
	f.anchor = -1
	f.focus = -1
	f.renameID = ""
	f.renameIndex = -1
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) handleListEvent(event runtime.Event) {
	if event.Type == "click" {
		f.clearSelection()
		return
	}
	switch event.Key {
	case "ArrowUp":
		f.moveFocus(-1)
	case "ArrowDown":
		f.moveFocus(1)
	case "Enter":
		f.openFocused()
	case "Delete":
		f.deleteSelected()
	case "Escape":
		f.cancelRename()
		f.dismissOverlayOnly()
		f.bools["previewOpen"].Set(false)
		f.recompute()
	case "Mod+A":
		f.selectAll()
	}
}

func (f *finderDemoState) handleRenameEvent(event runtime.Event) {
	switch event.Key {
	case "Enter":
		f.commitRename()
	case "Escape":
		f.cancelRename()
	}
}

func (f *finderDemoState) moveFocus(delta int) {
	rows := f.visibleRows()
	if len(rows) == 0 {
		return
	}
	next := f.focus
	if next < 0 {
		next = 0
	} else {
		next += delta
	}
	if next < 0 {
		next = 0
	}
	if next >= len(rows) {
		next = len(rows) - 1
	}
	f.focus = next
	f.selected = map[string]bool{rows[next].id: true}
	f.anchor = next
	f.recompute()
}

func (f *finderDemoState) openFocused() {
	if f.focus >= 0 {
		f.openVisibleIndex(f.focus)
		return
	}
	f.openSelected()
}

func (f *finderDemoState) openSelected() {
	rows := f.visibleRows()
	for i, node := range rows {
		if f.selected[node.id] {
			f.openVisibleIndex(i)
			return
		}
	}
}

func (f *finderDemoState) showPreview(node *finderNode) {
	f.strings["previewTitle"].Set(node.name)
	f.strings["previewBody"].Set(fmt.Sprintf("System file · %s · %s", node.modifiedAt, formatSize(node)))
	f.bools["previewOpen"].Set(true)
	f.bools["itemMenuOpen"].Set(false)
	f.bools["blankMenuOpen"].Set(false)
	f.recompute()
}

func (f *finderDemoState) closePreview() {
	f.bools["previewOpen"].Set(false)
	f.recompute()
}

func (f *finderDemoState) beginRename() {
	node := f.firstSelectedNode()
	if node == nil {
		return
	}
	f.renameID = node.id
	f.renameIndex = f.visibleIndexByID(node.id)
	f.strings["renameText"].Set(node.name)
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) cancelRename() {
	f.renameID = ""
	f.renameIndex = -1
	f.recompute()
}

func (f *finderDemoState) commitRename() {
	if f.renameID == "" {
		return
	}
	name := strings.TrimSpace(f.strings["renameText"].Get())
	if name == "" {
		return
	}
	node := f.nodeByID(f.renameID)
	if node == nil {
		return
	}
	target := filepath.Join(filepath.Dir(node.path), name)
	if err := f.files.Rename(node.path, target); err != nil {
		f.lastError = err.Error()
		f.recompute()
		return
	}
	f.renameID = ""
	f.renameIndex = -1
	f.selected = map[string]bool{nodeID(target): true}
	f.recompute()
}

func (f *finderDemoState) deleteSelected() {
	if len(f.selected) == 0 {
		return
	}
	for _, node := range f.rows {
		if f.selected[node.id] {
			if err := f.files.Remove(node.path); err != nil {
				f.lastError = err.Error()
				break
			}
		}
	}
	f.selected = map[string]bool{}
	f.anchor = -1
	f.focus = -1
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) duplicateSelected() {
	for _, node := range f.rows {
		if f.selected[node.id] {
			target := f.availableSiblingPath(node.path, node.name+" copy")
			if err := f.files.Duplicate(node.path, target); err != nil {
				f.lastError = err.Error()
				break
			}
		}
	}
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) newFolder() {
	name := "Untitled Folder"
	path := f.availableChildPath(name)
	if err := f.files.Mkdir(path); err != nil {
		f.lastError = err.Error()
		f.recompute()
		return
	}
	f.selected = map[string]bool{nodeID(path): true}
	f.anchor = 0
	f.focus = 0
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) refresh() {
	f.dismissOverlayOnly()
	f.recompute()
}

func (f *finderDemoState) selectAll() {
	f.selected = map[string]bool{}
	for _, node := range f.visibleRows() {
		f.selected[node.id] = true
	}
	f.recompute()
}

func (f *finderDemoState) firstSelectedNode() *finderNode {
	for _, node := range f.rows {
		if f.selected[node.id] {
			return node
		}
	}
	return nil
}

func (f *finderDemoState) visibleIndexByID(id string) int {
	for i, node := range f.visibleRows() {
		if node.id == id {
			return i
		}
	}
	return -1
}

func (f *finderDemoState) selectedCount() int {
	return len(f.selected)
}

func (f *finderDemoState) nodeByID(id string) *finderNode {
	for _, node := range f.rows {
		if node.id == id {
			return node
		}
	}
	return nil
}

func (f *finderDemoState) loadCurrentDirectory() {
	entries, err := f.files.ReadDir(f.currentPath)
	if err != nil {
		f.rows = nil
		f.lastError = err.Error()
		return
	}
	f.lastError = ""
	f.rows = make([]*finderNode, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		kind := entry.Kind
		if kind == "" {
			kind = "file"
		}
		path := entry.Path
		if path == "" {
			path = filepath.Join(f.currentPath, entry.Name)
		}
		node := &finderNode{
			id:         nodeID(path),
			name:       entry.Name,
			kind:       kind,
			size:       entry.Size,
			modifiedAt: formatModified(entry.ModifiedAt),
			path:       filepath.Clean(path),
		}
		seen[node.id] = true
		f.rows = append(f.rows, node)
	}
	for id := range f.selected {
		if !seen[id] {
			delete(f.selected, id)
		}
	}
}

func (f *finderDemoState) availableSiblingPath(path string, fallbackName string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(fallbackName)
	stem := strings.TrimSuffix(fallbackName, ext)
	candidate := filepath.Join(dir, fallbackName)
	for i := 2; ; i++ {
		if _, err := f.files.Stat(candidate); system.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s %d%s", stem, i, ext))
	}
}

func (f *finderDemoState) availableChildPath(name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	candidate := filepath.Join(f.currentPath, name)
	for i := 2; ; i++ {
		if _, err := f.files.Stat(candidate); system.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(f.currentPath, fmt.Sprintf("%s %d%s", stem, i, ext))
	}
}

func nodeID(path string) string {
	return filepath.Clean(path)
}

func defaultFinderSidebar() finderSidebar {
	home := defaultFinderHomeDir()
	wd := defaultFinderWorkDir(home)
	return finderSidebar{
		documents: filepath.Join(home, "Documents"),
		downloads: filepath.Join(home, "Downloads"),
		pictures:  filepath.Join(home, "Pictures"),
		projectA:  wd,
		projectB:  filepath.Dir(wd),
	}
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

func formatModified(value time.Time) string {
	if value.IsZero() {
		return "--"
	}
	now := time.Now()
	if sameDay(value, now) {
		return "Today " + value.Format("15:04")
	}
	if sameDay(value, now.AddDate(0, 0, -1)) {
		return "Yesterday"
	}
	if value.Year() == now.Year() {
		return value.Format("Jan 02")
	}
	return value.Format("2006-01-02")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
