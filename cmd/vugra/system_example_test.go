package main

import (
	"os"
	"path/filepath"
	stdruntime "runtime"
	"testing"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/pkg/system"
)

func TestSystemFilesExampleReadsInjectedFileSystem(t *testing.T) {
	original := system.DefaultFileSystem()
	t.Cleanup(func() { system.SetFileSystem(original) })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	system.SetFileSystem(system.OSFileSystem{})

	mounted, target := mountSystemFiles(t)
	load := findElementWithText(target.LastFrame(), "Load")
	if load.ID == "" {
		t.Fatal("missing Load button")
	}
	if !mounted.DispatchPointer(load.Rect.X+2, load.Rect.Y+2) {
		t.Fatal("click Load failed")
	}
	mounted.Flush()

	want := "2 entries in ."
	if !frameContainsText(target.LastFrame(), want) {
		t.Fatalf("missing status %q\n%s", want, frameTexts(target.LastFrame()))
	}
}

func frameTexts(frame []renderer.Command) string {
	var out string
	for _, command := range frame {
		if command.Kind == "text" {
			out += command.Text + "\n"
		}
	}
	return out
}

func mountSystemFiles(t *testing.T) (*runtime.App, *renderer.TestRenderer) {
	t.Helper()
	_, file, _, ok := stdruntime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "examples", "system", "SystemFiles.vue")
	result, err := compiler.CompileFile(path)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	mounted := app.Mount(result, demoStateFor(path, scheduler), target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	return mounted, target
}
