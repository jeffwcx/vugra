package app_test

import (
	"testing"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
)

func TestMountCompiledResult(t *testing.T) {
	result, err := compiler.CompileFile("../../examples/counter/Counter.vue")
	if err != nil {
		t.Fatalf("compile counter: %v", err)
	}
	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	target := &renderer.TestRenderer{}
	mounted := app.Mount(result, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Update(func(value int) int { return value + 1 }) },
		},
	}, target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800},
		Measurer:    layout.FixedMeasurer{},
	})
	if mounted == nil {
		t.Fatal("missing app")
	}
	if len(target.Frames) != 1 {
		t.Fatalf("frames = %d", len(target.Frames))
	}
}
