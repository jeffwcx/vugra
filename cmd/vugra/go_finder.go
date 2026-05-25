package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/pkg/system"
)

var goFinderNativeWindow = runNativeWindow
var goFinderProjectRun = runProject
var goFinderNativeWindowSmoke = runGoFinderLiteNativeWindowSmoke

func runGoFinderLite(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("go-finder-lite variant must be smoke, native, run, or native-window-smoke")
	}
	switch args[0] {
	case "smoke":
		return runGoFinderLiteSmoke()
	case "native":
		path, err := goFinderLiteComponentPath(".")
		if err != nil {
			return err
		}
		return goFinderNativeWindow([]string{path})
	case "run":
		path, err := goFinderLiteProjectPath(".")
		if err != nil {
			return err
		}
		return goFinderProjectRun([]string{path})
	case "native-window-smoke":
		return goFinderNativeWindowSmoke()
	default:
		return fmt.Errorf("go-finder-lite variant must be smoke, native, run, or native-window-smoke")
	}
}

func runGUIRuntimeSmoke(args []string) error {
	includeWindow := false
	switch {
	case len(args) == 0:
	case len(args) == 1 && args[0] == "window":
		includeWindow = true
	default:
		return usage()
	}
	if err := runGoFinderLiteSmoke(); err != nil {
		return err
	}
	if err := runRustSFCSmoke(nil); err != nil {
		return err
	}
	if err := runRustFinderLite([]string{"native-smoke"}); err != nil {
		return err
	}
	if err := runRustFinderLite([]string{"generated-adapter-smoke"}); err != nil {
		return err
	}
	if err := runRustFinderLite([]string{"wgpu-device-smoke"}); err != nil {
		return err
	}
	if includeWindow {
		if err := runGoFinderLite([]string{"native-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderSFC([]string{"native-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderSFC([]string{"native-software-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderSFC([]string{"native-wgpu-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"native-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"native-software-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"native-wgpu-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"abi-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"abi-software-window-smoke"}); err != nil {
			return err
		}
		if err := runRustFinderLite([]string{"abi-wgpu-window-smoke"}); err != nil {
			return err
		}
	}
	fmt.Println("gui-runtime-smoke ok")
	return nil
}

func runGoFinderLiteSmoke() error {
	previousWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	root, err := os.MkdirTemp("", "vugra-go-finder-smoke-*")
	if err != nil {
		return fmt.Errorf("create smoke fixture: %w", err)
	}
	defer os.RemoveAll(root)
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("enter smoke fixture: %w", err)
	}
	defer os.Chdir(previousWD)

	fixture, err := createGoFinderSmokeFixture(root)
	if err != nil {
		return err
	}
	componentPath, err := goFinderLiteComponentPath(previousWD)
	if err != nil {
		return err
	}
	result, err := compiler.CompileFile(componentPath)
	if err != nil {
		return fmt.Errorf("compile FinderLite: %w", err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		return fmt.Errorf("compile FinderLite produced %d diagnostic(s)", len(diagnostics))
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
		Layout:      layoutEngineFromEnv(),
	})

	frame := target.LastFrame()
	if err := validateGoFinderFrame(frame, "initial", fixture.documents, "Design", "12 items · Current path: "+fixture.documents); err != nil {
		return err
	}
	firstRow := goFinderElementWithText(frame, "Design")
	if firstRow.ID == "" {
		return fmt.Errorf("go-finder-lite smoke failed: missing Design row hit target")
	}
	if !mounted.DispatchPointerEvent(firstRow.Rect.X+2, firstRow.Rect.Y+2, runtime.Modifiers{}) {
		return fmt.Errorf("go-finder-lite smoke failed: row click did not dispatch")
	}
	mounted.Flush()
	frame = target.LastFrame()
	if err := validateGoFinderFrame(frame, "selection", fixture.documents, "Design", "1 items selected"); err != nil {
		return err
	}

	search := goFinderElementByClass(frame, "search")
	if search.ID == "" {
		return fmt.Errorf("go-finder-lite smoke failed: missing search textbox")
	}
	if !mounted.DispatchPointer(search.Rect.X+2, search.Rect.Y+2) {
		return fmt.Errorf("go-finder-lite smoke failed: search focus did not dispatch")
	}
	if !mounted.DispatchTextInput("road") {
		return fmt.Errorf("go-finder-lite smoke failed: search text input did not dispatch")
	}
	mounted.Flush()
	frame = target.LastFrame()
	if err := validateGoFinderFrame(frame, "search", fixture.documents, "Roadmap.md", "1 items · Current path: "+fixture.documents); err != nil {
		return err
	}
	if goFinderFrameContainsText(frame, "Budget 2026.xlsx") {
		return fmt.Errorf("go-finder-lite smoke failed: search frame still contains Budget 2026.xlsx")
	}

	software := renderer.NewSoftware(800, 600)
	software.Render(frame)
	if software.Image == nil {
		return fmt.Errorf("go-finder-lite smoke failed: software renderer produced no image")
	}
	drawn := goFinderDrawnPixelCount(software)
	if drawn == 0 {
		return fmt.Errorf("go-finder-lite smoke failed: software renderer produced only background pixels")
	}
	fmt.Printf("go software commands=%d pixels=%d drawn=%d\n", len(frame), software.Width*software.Height, drawn)
	fmt.Printf("go runtime path=%s row=Roadmap.md status=%q\n", fixture.documents, "1 items · Current path: "+fixture.documents)
	fmt.Println("go-finder-lite smoke ok")
	return nil
}

func goFinderLiteComponentPath(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "examples", "finder", "FinderLite.vue")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("locate examples/finder/FinderLite.vue")
}

func goFinderLiteProjectPath(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "examples", "finder")
		if _, err := os.Stat(filepath.Join(candidate, "vugra.config.json")); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("locate examples/finder")
}

type goFinderSmokeFixture struct {
	documents string
	downloads string
	pictures  string
	projectA  string
	projectB  string
}

func createGoFinderSmokeFixture(root string) (goFinderSmokeFixture, error) {
	fixture := goFinderSmokeFixture{
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
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return fixture, fmt.Errorf("create smoke dir %s: %w", dir, err)
		}
	}
	for _, name := range []string{
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
		if err := os.WriteFile(filepath.Join(root, fixture.documents, name), []byte(name), 0o644); err != nil {
			return fixture, fmt.Errorf("create smoke file %s: %w", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, fixture.documents, "Design", "Design System.fig"), []byte("design"), 0o644); err != nil {
		return fixture, fmt.Errorf("create smoke design file: %w", err)
	}
	return fixture, nil
}

func validateGoFinderFrame(frame []renderer.Command, phase, path, row, status string) error {
	if len(frame) == 0 {
		return fmt.Errorf("go-finder-lite smoke failed: %s frame has no commands", phase)
	}
	for _, want := range []string{path, row, status} {
		if !goFinderFrameContainsText(frame, want) {
			return fmt.Errorf("go-finder-lite smoke failed: %s frame missing %q", phase, want)
		}
	}
	if goFinderElementWithText(frame, row).ID == "" {
		return fmt.Errorf("go-finder-lite smoke failed: %s frame missing hit target for %q", phase, row)
	}
	return nil
}

func goFinderElementWithText(frame []renderer.Command, text string) renderer.Command {
	for i, command := range frame {
		if command.Kind != "text" || command.Text != text {
			continue
		}
		if element := goFinderNearestEventElement(frame, i); element.ID != "" {
			return element
		}
	}
	for i, command := range frame {
		if command.Kind != "text" || !strings.Contains(command.Text, text) {
			continue
		}
		if element := goFinderNearestEventElement(frame, i); element.ID != "" {
			return element
		}
	}
	return renderer.Command{}
}

func goFinderNearestEventElement(frame []renderer.Command, textIndex int) renderer.Command {
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

func goFinderElementByClass(frame []renderer.Command, class string) renderer.Command {
	for _, command := range frame {
		if command.Kind == "element" && command.Props["class"] == class {
			return command
		}
	}
	return renderer.Command{}
}

func goFinderFrameContainsText(frame []renderer.Command, text string) bool {
	for _, command := range frame {
		if command.Kind == "text" && (command.Text == text || strings.Contains(command.Text, text)) {
			return true
		}
	}
	return false
}

func goFinderDrawnPixelCount(r *renderer.SoftwareRenderer) int {
	if r.Image == nil {
		return 0
	}
	background := color.RGBA{R: 250, G: 250, B: 250, A: 255}
	count := 0
	for y := 0; y < r.Image.Bounds().Dy(); y++ {
		for x := 0; x < r.Image.Bounds().Dx(); x++ {
			if r.Image.RGBAAt(x, y) != background {
				count++
			}
		}
	}
	return count
}
