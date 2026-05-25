package main

import (
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/pkg/system"
)

func TestFinderLiteExampleInteractions(t *testing.T) {
	mounted, target, fixture := mountFinderLite(t)
	frame := target.LastFrame()
	if !frameContainsText(frame, fixture.documents) {
		t.Fatalf("initial frame missing path")
	}
	if !frameContainsText(frame, "12 items · Current path: "+fixture.documents) {
		t.Fatalf("initial frame missing status")
	}

	firstRow := findElementWithText(frame, "Design")
	if firstRow.ID == "" {
		t.Fatalf("missing first row")
	}
	if !mounted.DispatchPointerEvent(firstRow.Rect.X+2, firstRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("row click failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "1 items selected") {
		t.Fatalf("selection status not updated")
	}
	if findElementWithText(target.LastFrame(), "Design").Props["class"] != "file-row-selected" {
		t.Fatalf("selected row class not applied")
	}

	thirdRow := findElementWithText(target.LastFrame(), "Roadmap.md")
	if !mounted.DispatchPointerEvent(thirdRow.Rect.X+2, thirdRow.Rect.Y+2, runtime.Modifiers{Meta: true}) {
		t.Fatal("mod-click failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "2 items selected") {
		t.Fatalf("multi-select status not updated")
	}
	fourthRow := findElementWithText(target.LastFrame(), "Budget 2026.xlsx")
	if !mounted.DispatchPointerEvent(fourthRow.Rect.X+2, fourthRow.Rect.Y+2, runtime.Modifiers{Shift: true}) {
		t.Fatal("shift-click failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "9 items selected") {
		t.Fatalf("shift range status not updated")
	}

	if !mounted.DispatchContextMenu(thirdRow.Rect.X+2, thirdRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("context menu failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "Rename") || !frameContainsText(target.LastFrame(), "Duplicate") {
		t.Fatalf("item context menu missing")
	}
	if menu := findElementByClass(target.LastFrame(), "menu"); menu.ID == "" || menu.Rect.Y+menu.Rect.Height > 600 {
		t.Fatalf("context menu is not visible in viewport: %+v", menu)
	}

	if !mounted.DispatchKey("Escape") {
		t.Fatal("escape should dismiss focused row menu")
	}
	mounted.Flush()
	if frameContainsText(target.LastFrame(), "Duplicate") {
		t.Fatalf("escape did not dismiss menu")
	}

	search := findElementByClass(target.LastFrame(), "search")
	if !mounted.DispatchPointer(search.Rect.X+2, search.Rect.Y+2) {
		t.Fatal("search focus failed")
	}
	if !mounted.DispatchTextInput("road") {
		t.Fatal("search text input failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "1 items · Current path: "+fixture.documents) {
		t.Fatalf("search did not filter status")
	}
	if frameContainsText(target.LastFrame(), "Budget 2026.xlsx") {
		t.Fatalf("search did not filter rows")
	}
	pane := findElementByClass(target.LastFrame(), "file-pane")
	if !mounted.DispatchPointer(pane.Rect.X+pane.Rect.Width-4, pane.Rect.Y+pane.Rect.Height-4) {
		t.Fatal("blank click failed")
	}
	mounted.Flush()
	filteredRow := findElementWithText(target.LastFrame(), "Roadmap.md")
	if !mounted.DispatchHover(filteredRow.Rect.X+2, filteredRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("hover failed")
	}
	mounted.Flush()
	if findElementWithText(target.LastFrame(), "Roadmap.md").Props["class"] != "file-row-hover" {
		t.Fatalf("hover class not applied")
	}

	if !mounted.DispatchPointerEvent(filteredRow.Rect.X+2, filteredRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("filtered row click failed")
	}
	mounted.Flush()
	if !mounted.DispatchDoubleClick(filteredRow.Rect.X+2, filteredRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("file double-click failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "System file") {
		t.Fatalf("file preview dialog missing")
	}
	if dialog := findElementByClass(target.LastFrame(), "dialog"); dialog.ID == "" || dialog.Rect.Y+dialog.Rect.Height > 600 {
		t.Fatalf("preview dialog is not visible in viewport: %+v", dialog)
	}
}

func TestFinderLiteNavigationKeyboardAndRename(t *testing.T) {
	mounted, target, fixture := mountFinderLite(t)
	firstRow := findElementWithText(target.LastFrame(), "Design")
	if !mounted.DispatchDoubleClick(firstRow.Rect.X+2, firstRow.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("folder double-click failed")
	}
	mounted.Flush()
	designPath := filepath.Join(fixture.documents, "Design")
	if !frameContainsText(target.LastFrame(), designPath) {
		t.Fatalf("did not navigate into folder")
	}
	if !frameContainsText(target.LastFrame(), "Design System.fig") {
		t.Fatalf("folder contents not rendered")
	}

	list := findElementByClass(target.LastFrame(), "file-list")
	if !mounted.DispatchPointer(list.Rect.X+2, list.Rect.Y+2) {
		t.Fatal("file list focus failed")
	}
	if !mounted.DispatchKey("ArrowDown") {
		t.Fatal("arrow down failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "1 items selected") {
		t.Fatalf("keyboard selection failed")
	}
	if !mounted.DispatchKey("Mod+A") {
		t.Fatal("select all failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "2 items selected") {
		t.Fatalf("select all status wrong")
	}
	if !mounted.DispatchKey("Delete") {
		t.Fatal("delete failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "0 items · Current path: "+designPath) {
		t.Fatalf("delete did not remove rows")
	}

	back := findNthElementByClass(target.LastFrame(), "nav-button", 0)
	if !mounted.DispatchPointer(back.Rect.X+2, back.Rect.Y+2) {
		t.Fatal("back failed")
	}
	mounted.Flush()
	roadmap := findElementWithText(target.LastFrame(), "Roadmap.md")
	if !mounted.DispatchPointerEvent(roadmap.Rect.X+2, roadmap.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("select roadmap failed")
	}
	mounted.Flush()
	if !mounted.DispatchContextMenu(roadmap.Rect.X+2, roadmap.Rect.Y+2, runtime.Modifiers{}) {
		t.Fatal("open menu failed")
	}
	mounted.Flush()
	rename := findElementWithText(target.LastFrame(), "Rename")
	if !mounted.DispatchPointer(rename.Rect.X+2, rename.Rect.Y+2) {
		t.Fatal("begin rename failed")
	}
	mounted.Flush()
	input := findElementByClass(target.LastFrame(), "rename-inline")
	if input.ID == "" {
		t.Fatalf("rename input missing")
	}
	if !mounted.DispatchPointer(input.Rect.X+2, input.Rect.Y+2) || !mounted.DispatchText("Roadmap renamed.md") {
		t.Fatal("rename text failed")
	}
	if !mounted.DispatchKey("Enter") {
		t.Fatal("commit rename failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "Roadmap renamed.md") {
		t.Fatalf("renamed row missing")
	}

	splitter := findElementByClass(target.LastFrame(), "splitter")
	if splitter.ID == "" {
		t.Fatalf("splitter missing")
	}
	if !mounted.DispatchHover(splitter.Rect.X+1, splitter.Rect.Y+10, runtime.Modifiers{}) {
		t.Fatal("splitter hover failed")
	}
	mounted.Flush()
	if findElementByClass(target.LastFrame(), "splitter-hover").ID == "" {
		t.Fatalf("splitter hover class missing")
	}
	if !mounted.DispatchPointer(splitter.Rect.X+1, splitter.Rect.Y+10) {
		t.Fatal("splitter pointer down failed")
	}
	if !mounted.DispatchDrag(splitter.Rect.X+80, splitter.Rect.Y+10, 80, 0, runtime.Modifiers{}) {
		t.Fatal("splitter drag failed")
	}
	mounted.Flush()
	if sidebar := findElementByClass(target.LastFrame(), "sidebar-200"); sidebar.ID == "" {
		if sidebar = findElementByClass(target.LastFrame(), "sidebar-280"); sidebar.ID == "" {
			t.Fatalf("sidebar class did not change after drag")
		}
	}
}

func TestFinderLiteSelectAllCoversRowsOutsideProjectedWindow(t *testing.T) {
	fixture := newFinderTestFixture(t)
	extra := filepath.Join(fixture.documents, "Zebra.md")
	if err := os.WriteFile(extra, []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	mounted, target := mountFinderLiteWithFixture(t, fixture)
	if !frameContainsText(target.LastFrame(), "13 items · Current path: "+fixture.documents) {
		t.Fatalf("fixture did not expose 13 document rows")
	}

	list := findElementByClass(target.LastFrame(), "file-list")
	if !mounted.DispatchPointer(list.Rect.X+2, list.Rect.Y+2) {
		t.Fatal("file list focus failed")
	}
	if !mounted.DispatchKey("Mod+A") {
		t.Fatal("select all failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "13 items selected") {
		t.Fatalf("select all did not cover off-screen row")
	}
	if !mounted.DispatchKey("Delete") {
		t.Fatal("delete failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "0 items · Current path: "+fixture.documents) {
		t.Fatalf("delete did not remove all selected rows")
	}
}

func TestFinderLiteSearchFiltersNamesOnly(t *testing.T) {
	mounted, target, fixture := mountFinderLite(t)
	search := findElementByClass(target.LastFrame(), "search")
	if !mounted.DispatchPointer(search.Rect.X+2, search.Rect.Y+2) {
		t.Fatal("search focus failed")
	}
	if !mounted.DispatchTextInput("folder") {
		t.Fatal("search text input failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "0 items · Current path: "+fixture.documents) {
		t.Fatalf("search matched non-name fields")
	}
	if frameContainsText(target.LastFrame(), "Design") {
		t.Fatalf("search matched folder kind instead of name only")
	}
}

func TestFinderLiteSearchTrimsQuery(t *testing.T) {
	mounted, target, fixture := mountFinderLite(t)
	search := findElementByClass(target.LastFrame(), "search")
	if !mounted.DispatchPointer(search.Rect.X+2, search.Rect.Y+2) {
		t.Fatal("search focus failed")
	}
	if !mounted.DispatchTextInput(" road ") {
		t.Fatal("search text input failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "1 items · Current path: "+fixture.documents) {
		t.Fatalf("search did not trim query")
	}
	if !frameContainsText(target.LastFrame(), "Roadmap.md") {
		t.Fatalf("trimmed search did not match Roadmap.md")
	}
	if frameContainsText(target.LastFrame(), "Budget 2026.xlsx") {
		t.Fatalf("trimmed search did not filter unrelated rows")
	}
}

type finderTestFixture struct {
	root      string
	documents string
	downloads string
	pictures  string
	projectA  string
	projectB  string
}

func mountFinderLite(t *testing.T) (*runtime.App, *renderer.TestRenderer, finderTestFixture) {
	t.Helper()
	fixture := newFinderTestFixture(t)
	mounted, target := mountFinderLiteWithFixture(t, fixture)
	return mounted, target, fixture
}

func mountFinderLiteWithFixture(t *testing.T, fixture finderTestFixture) (*runtime.App, *renderer.TestRenderer) {
	t.Helper()
	_, file, _, ok := stdruntime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "examples", "finder", "FinderLite.vue")
	result, err := compiler.CompileFile(path)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	state := newFinderDemoStateWithFileSystem(scheduler, system.OSFileSystem{}, finderSidebar{
		documents: fixture.documents,
		downloads: fixture.downloads,
		pictures:  fixture.pictures,
		projectA:  fixture.projectA,
		projectB:  fixture.projectB,
	}).state()
	mounted := app.Mount(result, state, target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	return mounted, target
}

func newFinderTestFixture(t *testing.T) finderTestFixture {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	fixture := finderTestFixture{
		root:      root,
		documents: "Documents",
		downloads: "Downloads",
		pictures:  "Pictures",
		projectA:  "Project A",
		projectB:  "Project B",
	}
	for _, dir := range []string{
		fixture.documents,
		filepath.Join(fixture.documents, "Design"),
		fixture.downloads,
		fixture.pictures,
		fixture.projectA,
		fixture.projectB,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{
		"Roadmap.md",
		"Meeting Notes.txt",
		"Budget 2026.xlsx",
		"Client Brief.pdf",
		"Contract Draft.docx",
		"Launch Plan.pages",
		"Research Summary.md",
		"Book Outline.txt",
		"Ideas.txt",
		"Agenda.md",
		"Notes Archive.txt",
	} {
		if err := os.WriteFile(filepath.Join(fixture.documents, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"Design System.fig", "Wireframes"} {
		if err := os.WriteFile(filepath.Join(fixture.documents, "Design", file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func findElementWithText(frame []renderer.Command, text string) renderer.Command {
	for i, command := range frame {
		if command.Kind != "text" || command.Text != text {
			continue
		}
		if element := nearestEventElement(frame, i); element.ID != "" {
			return element
		}
	}
	for i, command := range frame {
		if command.Kind != "text" || !strings.Contains(command.Text, text) {
			continue
		}
		if element := nearestEventElement(frame, i); element.ID != "" {
			return element
		}
	}
	return renderer.Command{}
}

func nearestEventElement(frame []renderer.Command, textIndex int) renderer.Command {
	for j := textIndex - 1; j >= 0; j-- {
		if frame[j].Kind == "element" && frame[j].Tag == "button" {
			return frame[j]
		}
	}
	for j := textIndex - 1; j >= 0; j-- {
		if frame[j].Kind == "element" {
			return frame[j]
		}
	}
	return renderer.Command{}
}

func findElementByClass(frame []renderer.Command, class string) renderer.Command {
	for _, command := range frame {
		if command.Kind == "element" && command.Props["class"] == class {
			return command
		}
	}
	return renderer.Command{}
}

func findNthElementByClass(frame []renderer.Command, class string, index int) renderer.Command {
	seen := 0
	for _, command := range frame {
		if command.Kind != "element" || command.Props["class"] != class {
			continue
		}
		if seen == index {
			return command
		}
		seen++
	}
	return renderer.Command{}
}

func frameContainsText(frame []renderer.Command, text string) bool {
	for _, command := range frame {
		if command.Kind == "text" && (command.Text == text || strings.Contains(command.Text, text)) {
			return true
		}
	}
	return false
}
