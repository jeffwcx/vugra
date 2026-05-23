package runtime_test

import (
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
)

func TestCSSLayoutEngineFeedsRuntimeAndKeepsEvents(t *testing.T) {
	templateDoc := template.Parse(`
<div class="row">
  <button class="item" @click="Add">Add</button>
  <button class="item">Done</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}
func (s *State) Add() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "CSSRuntime", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}

	css := `
.row {
  display: flex;
  gap: 12px;
  width: 220px;
}
.item {
  height: 32px;
  border: 1px solid #ddd;
  border-radius: 6px;
  overflow: scroll;
}
`
	scheduler := reactivity.NewScheduler()
	calls := 0
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){
			"Add": func() { calls++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      style.Parse(css, style.BasePosition{}),
		StyleCSS:    css,
		Constraints: layout.Constraints{Width: 320, Height: 120},
		Layout:      runtime.LayoutEngineCSS,
	})

	add := nthElement(target.LastFrame(), "button", 0)
	done := nthElement(target.LastFrame(), "button", 1)
	if add.Rect.Width <= 0 || add.Rect.Height != 32 {
		t.Fatalf("add rect = %+v", add.Rect)
	}
	if done.Rect.X <= add.Rect.X {
		t.Fatalf("expected Taffy flex row positions, add=%+v done=%+v", add.Rect, done.Rect)
	}
	if add.Style.BorderWidth != 1 || add.Style.BorderRadius != 6 || add.Style.Overflow != "scroll" {
		t.Fatalf("expected CSS style metadata, got %+v", add.Style)
	}
	if !app.DispatchPointer(add.Rect.X+1, add.Rect.Y+1) {
		t.Fatal("css-layout hit test did not dispatch")
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
}

func TestCSSLayoutEngineFeedsRootHeightToComponent(t *testing.T) {
	templateDoc := template.Parse(`<div class="app"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "CSSRootHeight", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}

	css := `.app { min-height: 100%; }`
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      style.Parse(css, style.BasePosition{}),
		StyleCSS:    css,
		Constraints: layout.Constraints{Width: 320, Height: 180},
		Layout:      runtime.LayoutEngineCSS,
	})

	root := nthElement(target.LastFrame(), "div", 0)
	if root.Rect.Width != 320 || root.Rect.Height != 180 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
}
