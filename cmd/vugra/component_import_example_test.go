package main

import (
	"path/filepath"
	stdruntime "runtime"
	"testing"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
)

func TestComponentImportExampleRendersChild(t *testing.T) {
	_, file, _, ok := stdruntime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "examples", "components", "Parent.vue")
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
	if !frameContainsText(target.LastFrame(), "Parent using Go import") {
		t.Fatal("missing parent state text")
	}
	if !frameContainsText(target.LastFrame(), "default slot") {
		t.Fatal("missing default slot text")
	}
	if !frameContainsText(target.LastFrame(), "named slot") {
		t.Fatal("missing named slot text")
	}
	badge := findElementByClass(target.LastFrame(), "badge")
	if badge.ID == "" {
		t.Fatal("missing badge root element")
	}
	if !mounted.DispatchPointer(badge.Rect.X+1, badge.Rect.Y+1) {
		t.Fatal("component event dispatch failed")
	}
	mounted.Flush()
	if !frameContainsText(target.LastFrame(), "Badge clicked") {
		t.Fatal("component event did not reach parent")
	}
}
