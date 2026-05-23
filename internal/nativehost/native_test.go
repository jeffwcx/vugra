package nativehost_test

import (
	"os"
	"testing"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/host"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/nativehost"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/runtime"
)

func TestNativeHostWritesPNGAfterEvents(t *testing.T) {
	result, err := compiler.CompileFile("../../examples/counter/Counter.vue")
	if err != nil {
		t.Fatalf("compile counter: %v", err)
	}
	path := t.TempDir() + "/counter.png"
	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	target := nativehost.New(320, 200, path)
	target.Keys = []host.KeyEvent{{Key: "Tab"}, {Key: "Enter"}}
	mounted := app.Mount(result, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Update(func(value int) int { return value + 1 }) },
		},
	}, target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 320, Height: 200},
		Measurer:    layout.FixedMeasurer{},
	})
	if err := target.Run(mounted); err != nil {
		t.Fatalf("run native host: %v", err)
	}
	if target.Frames != 2 {
		t.Fatalf("frames = %d, want 2", target.Frames)
	}
	if len(target.A11y) == 0 {
		t.Fatal("missing accessibility tree")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("missing png: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("png is empty")
	}
}

func TestNativeHostRunsTextEvents(t *testing.T) {
	source := []byte(`<template>
<div>
  <input :value="name">
  <p>{{ name }}</p>
</div>
</template>
<script lang="go">
type State struct {
    Name signal.String ` + "`vugra:\"name\"`" + `
}
</script>
`)
	path := t.TempDir() + "/Input.vue"
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatalf("write input component: %v", err)
	}
	result, err := compiler.CompileFile(path)
	if err != nil {
		t.Fatalf("compile input: %v", err)
	}
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal("", scheduler)
	target := nativehost.New(320, 200, "")
	target.Clicks = []host.PointerEvent{{X: 1, Y: 1}}
	target.Text = []host.TextEvent{{Text: "Grace"}}
	mounted := app.Mount(result, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		Methods: map[string]func(){},
	}, target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 320, Height: 200},
		Measurer:    layout.FixedMeasurer{},
	})
	if err := target.Run(mounted); err != nil {
		t.Fatalf("run native host: %v", err)
	}
	if name.Get() != "Grace" {
		t.Fatalf("name = %q", name.Get())
	}
}
