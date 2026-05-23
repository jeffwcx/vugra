package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
	"github.com/vugra/vugra/pkg/signal"
)

func TestCounterRenderAndClick(t *testing.T) {
	app, testRenderer := mountCounter(t)

	first := testRenderer.LastFrame()
	if !frameContainsText(first, "0") {
		t.Fatalf("initial frame does not contain count 0: %+v", first)
	}
	root := findElement(first, "div")
	if root.Rect.Width != 200 || root.Rect.Height != 80 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
	button := findElement(first, "button")
	if button.Role != "button" {
		t.Fatalf("button role = %q", button.Role)
	}
	countText := findText(first, "0")
	if countText.Rect.X != 16 || countText.Rect.Y != 16 {
		t.Fatalf("count text rect = %+v", countText.Rect)
	}
	if eventID := findEventID(first, "click"); eventID == "" {
		t.Fatalf("initial frame missing click event: %+v", first)
	}
	if !app.DispatchPointer(20, 50) {
		t.Fatal("pointer dispatch failed")
	}
	app.Flush()

	last := testRenderer.LastFrame()
	if !frameContainsText(app.LastFrame(), "1") {
		t.Fatalf("app last frame missing count 1: %+v", app.LastFrame())
	}
	if len(testRenderer.Frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(testRenderer.Frames))
	}
	if !frameContainsText(last, "1") {
		t.Fatalf("updated frame does not contain count 1: %+v", last)
	}
	if frameContainsText(last, "0") {
		t.Fatalf("updated frame still contains count 0: %+v", last)
	}
}

func TestDefaultSchedulerFlushesPublicSignals(t *testing.T) {
	count := signal.NewInt(0)
	component := &ir.Component{Name: "Counter", Nodes: []ir.Node{
		&ir.Element{
			Tag: "button",
			Events: []ir.EventHandler{
				{Event: "click", Method: "Inc"},
			},
			Children: []ir.Node{&ir.Interpolation{Binding: "count", GoField: "Count"}},
		},
	}}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": &count},
		Methods: map[string]func(){
			"Inc": func() {
				count.Set(count.Get() + 1)
			},
		},
	}, target, runtime.Options{
		Constraints: layout.Constraints{Width: 120, Height: 80},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if !frameContainsText(target.LastFrame(), "0") {
		t.Fatalf("initial frame missing 0: %+v", target.LastFrame())
	}
	button := findElement(target.LastFrame(), "button")
	if !app.DispatchPointerEvent(button.Rect.X+1, button.Rect.Y+1, runtime.Modifiers{}) {
		t.Fatal("click was not handled")
	}
	app.Flush()
	if !frameContainsText(target.LastFrame(), "1") {
		t.Fatalf("default scheduler did not flush public signal update: %+v", target.LastFrame())
	}
}

func TestKeyboardFocusDispatch(t *testing.T) {
	app, testRenderer := mountCounter(t)
	if !app.DispatchKey("Tab") {
		t.Fatal("tab did not focus")
	}
	if app.FocusedEvent() == "" {
		t.Fatal("missing focused event")
	}
	if got, want := app.FocusedID(), strings.TrimSuffix(app.FocusedEvent(), ":click"); got != want {
		t.Fatalf("focused id = %q, want %q from event %q", got, want, app.FocusedEvent())
	}
	if !app.DispatchKey("Enter") {
		t.Fatal("enter did not dispatch focused event")
	}
	app.Flush()
	if !frameContainsText(testRenderer.LastFrame(), "1") {
		t.Fatalf("enter did not update frame: %+v", testRenderer.LastFrame())
	}
}

func TestShiftTabFocusesPreviousElement(t *testing.T) {
	templateDoc := template.Parse(`<div><button @click="First">First</button><button @click="Second">Second</button></div>`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}
func (s *State) First() {}
func (s *State) Second() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "TwoButtons", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Methods: map[string]func(){
			"First":  func() {},
			"Second": func() {},
		},
	}, target, runtime.Options{
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	first := nthElement(target.LastFrame(), "button", 0)
	second := nthElement(target.LastFrame(), "button", 1)
	if !app.DispatchKey("Shift+Tab") || app.FocusedID() != second.ID {
		t.Fatalf("shift tab from empty focus = %q want %q", app.FocusedID(), second.ID)
	}
	if !app.DispatchKey("Shift+Tab") || app.FocusedID() != first.ID {
		t.Fatalf("second shift tab = %q want %q", app.FocusedID(), first.ID)
	}
	if !app.DispatchKey("Tab") || app.FocusedID() != second.ID {
		t.Fatalf("tab after reverse focus = %q want %q", app.FocusedID(), second.ID)
	}
}

func TestFocusedIDForElementFocus(t *testing.T) {
	app, _ := mountTextInput(t, "")
	if app.FocusedEvent() == "" {
		t.Fatal("missing focused input")
	}
	if app.FocusedID() != app.FocusedEvent() {
		t.Fatalf("focused id = %q, event = %q", app.FocusedID(), app.FocusedEvent())
	}
}

func TestFocusIDFocusesElementWithoutActivation(t *testing.T) {
	app, testRenderer := mountCounter(t)
	button := findElement(testRenderer.LastFrame(), "button")
	if !app.FocusID(button.ID) {
		t.Fatalf("focus id %q failed", button.ID)
	}
	if got := app.FocusedID(); got != button.ID {
		t.Fatalf("focused id = %q, want %q", got, button.ID)
	}
	app.Flush()
	if !frameContainsText(testRenderer.LastFrame(), "0") {
		t.Fatalf("focus activated button unexpectedly: %+v", testRenderer.LastFrame())
	}
	if !app.DispatchKey("Enter") {
		t.Fatal("enter did not activate focused button")
	}
	app.Flush()
	if !frameContainsText(testRenderer.LastFrame(), "1") {
		t.Fatalf("enter did not update frame: %+v", testRenderer.LastFrame())
	}
}

func TestDisabledButtonDoesNotFocusOrDispatch(t *testing.T) {
	templateDoc := template.Parse(`<div><p>{{ count }}</p><button disabled @click="Inc">Disabled</button></div>`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Count signal.Int `+"`vugra:\"count\"`"+`
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "DisabledButton", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Set(count.Get() + 1) },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	button := findElement(target.LastFrame(), "button")
	if _, ok := button.Props["disabled"]; !ok {
		t.Fatalf("missing disabled prop: %+v", button)
	}
	if app.FocusID(button.ID) {
		t.Fatal("disabled button should not focus")
	}
	if app.DispatchPointer(button.Rect.X+1, button.Rect.Y+1) {
		t.Fatal("disabled button should not dispatch pointer")
	}
	app.Flush()
	if !frameContainsText(target.LastFrame(), "0") {
		t.Fatalf("disabled button updated count: %+v", target.LastFrame())
	}
}

func TestMultiplePointerEventsBatchIntoOneFrameUntilFlush(t *testing.T) {
	app, testRenderer := mountCounter(t)
	if !app.DispatchPointer(20, 50) || !app.DispatchPointer(20, 50) || !app.DispatchPointer(20, 50) {
		t.Fatal("pointer dispatch failed")
	}
	if len(testRenderer.Frames) != 1 {
		t.Fatalf("event dispatch rendered synchronously: frames=%d", len(testRenderer.Frames))
	}
	app.Flush()
	if len(testRenderer.Frames) != 2 {
		t.Fatalf("batched flush frames=%d", len(testRenderer.Frames))
	}
	if !frameContainsText(testRenderer.LastFrame(), "3") {
		t.Fatalf("batched frame missing count 3: %+v", testRenderer.LastFrame())
	}
}

func TestRenderInlineSVGCommand(t *testing.T) {
	templateDoc := template.Parse(`<svg class="icon" viewBox="0 0 10 10"><circle cx="5" cy="5" r="4" fill="#2563eb"/></svg>`, 0)
	component := ir.Build(ir.BuildInput{Name: "InlineSVG", Template: templateDoc, Go: goanalysis.Metadata{}})
	target := &renderer.TestRenderer{}
	runtime.MountWithOptions(component, runtime.State{}, target, runtime.Options{
		Styles:      style.Parse(`.icon { width: 10px; height: 10px; }`, style.BasePosition{}),
		Constraints: layout.Constraints{Width: 100},
	})
	svg := findCommand(target.LastFrame(), "svg")
	if svg.SVG == "" || !strings.Contains(svg.SVG, `<circle`) {
		t.Fatalf("svg command = %+v", svg)
	}
}

func TestRenderImageSVGCommandFromFile(t *testing.T) {
	dir := t.TempDir()
	iconPath := filepath.Join(dir, "icon.svg")
	if err := os.WriteFile(iconPath, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12"><path d="M0 0h12v12z"/></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}
	templateDoc := template.Parse(`<img class="icon" src="./icon.svg" />`, 0)
	component := ir.Build(ir.BuildInput{Name: filepath.Join(dir, "Icon.vue"), Template: templateDoc, Go: goanalysis.Metadata{}})
	target := &renderer.TestRenderer{}
	runtime.MountWithOptions(component, runtime.State{}, target, runtime.Options{
		AssetBase:   filepath.Join(dir, "Icon.vue"),
		Styles:      style.Parse(`.icon { width: 12px; height: 12px; }`, style.BasePosition{}),
		Constraints: layout.Constraints{Width: 100},
	})
	svg := findCommand(target.LastFrame(), "svg")
	if !strings.Contains(svg.SVG, `<path`) {
		t.Fatalf("svg command from file = %+v", svg)
	}
}

func TestLifecycleHooksRunOnMountAndUpdate(t *testing.T) {
	templateDoc := template.Parse(`<div><p>{{ count }}</p><button @click="Inc">+</button></div>`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Count signal.Int `+"`vugra:\"count\"`"+`
}

func (s *State) BeforeMount() {}
func (s *State) Mounted() {}
func (s *State) BeforeUpdate() {}
func (s *State) Updated() {}
func (s *State) BeforeUnmount() {}
func (s *State) Unmounted() {}
func (s *State) Inc() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Lifecycle", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	var calls []string
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"BeforeMount":   func() { calls = append(calls, "beforeMount") },
			"Mounted":       func() { calls = append(calls, "mounted") },
			"BeforeUpdate":  func() { calls = append(calls, "beforeUpdate") },
			"Updated":       func() { calls = append(calls, "updated") },
			"BeforeUnmount": func() { calls = append(calls, "beforeUnmount") },
			"Unmounted":     func() { calls = append(calls, "unmounted") },
			"Inc":           func() { count.Set(count.Get() + 1) },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if got := strings.Join(calls, ","); got != "beforeMount,mounted" {
		t.Fatalf("mount calls = %s", got)
	}
	button := findElement(target.LastFrame(), "button")
	if !app.DispatchPointer(button.Rect.X+1, button.Rect.Y+1) {
		t.Fatal("pointer dispatch failed")
	}
	app.Flush()
	if got := strings.Join(calls, ","); got != "beforeMount,mounted,beforeUpdate,updated" {
		t.Fatalf("update calls = %s", got)
	}
	app.Unmount()
	if got := strings.Join(calls, ","); got != "beforeMount,mounted,beforeUpdate,updated,beforeUnmount,unmounted" {
		t.Fatalf("unmount calls = %s", got)
	}
}

func TestComponentLifecycleListenersRun(t *testing.T) {
	component := &ir.Component{Name: "Parent", Nodes: []ir.Node{
		&ir.ComponentInstance{
			Alias: "Child",
			Events: []ir.EventHandler{
				{Event: "mounted", Method: "ChildMounted"},
				{Event: "updated", Method: "ChildUpdated"},
				{Event: "beforeUnmount", Method: "ChildBeforeUnmount"},
				{Event: "unmounted", Method: "ChildUnmounted"},
			},
			Lifecycle: []ir.Lifecycle{
				{Hook: "mounted", Method: "Mounted"},
				{Hook: "updated", Method: "Updated"},
				{Hook: "beforeUnmount", Method: "BeforeUnmount"},
				{Hook: "unmounted", Method: "Unmounted"},
			},
			Nodes: []ir.Node{&ir.Element{Tag: "button", Events: []ir.EventHandler{{Event: "click", Method: "Inc"}}, Children: []ir.Node{
				&ir.Interpolation{Binding: "count"},
			}}},
		},
	}}
	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	var calls []string
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"ChildMounted":       func() { calls = append(calls, "childMounted") },
			"ChildUpdated":       func() { calls = append(calls, "childUpdated") },
			"ChildBeforeUnmount": func() { calls = append(calls, "childBeforeUnmount") },
			"ChildUnmounted":     func() { calls = append(calls, "childUnmounted") },
			"Inc":                func() { count.Set(count.Get() + 1) },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if got := strings.Join(calls, ","); got != "childMounted" {
		t.Fatalf("mount calls = %s", got)
	}
	button := findElement(target.LastFrame(), "button")
	if !app.DispatchPointer(button.Rect.X+1, button.Rect.Y+1) {
		t.Fatal("pointer dispatch failed")
	}
	app.Flush()
	if got := strings.Join(calls, ","); got != "childMounted,childUpdated" {
		t.Fatalf("update calls = %s", got)
	}
	app.Unmount()
	if got := strings.Join(calls, ","); got != "childMounted,childUpdated,childBeforeUnmount,childUnmounted" {
		t.Fatalf("unmount calls = %s", got)
	}
}

func TestComponentInstanceOwnsRuntimeStatePropsAndEmit(t *testing.T) {
	scheduler := reactivity.NewScheduler()
	parentTitle := runtime.NewSignal("Parent Label", scheduler)
	childCount := runtime.NewSignal(0, scheduler)
	childLabel := runtime.NewSignal("", scheduler)
	parentSelected := 0
	childComponent := &ir.Component{
		Name:      "Child",
		PropNames: []string{"label", "count"},
		Props: []ir.PropDef{
			{Name: "label"},
			{Name: "count"},
		},
		Emits: []ir.Emit{{Method: "Pressed", Event: "select"}},
		Nodes: []ir.Node{&ir.Element{
			Tag:    "button",
			Events: []ir.EventHandler{{Event: "click", Method: "Pressed"}},
			Children: []ir.Node{
				&ir.Interpolation{Binding: "label"},
				&ir.Interpolation{Binding: "count"},
			},
		}},
		NewState: func() ir.RuntimeState {
			return ir.RuntimeState{
				Signals: map[string]ir.Signal{
					"label": childLabel,
					"count": childCount,
				},
				Methods: map[string]func(){
					"Pressed": func() {
						childCount.Update(func(value int) int { return value + 1 })
					},
				},
			}
		},
	}
	parent := &ir.Component{Name: "Parent", Nodes: []ir.Node{
		&ir.ComponentInstance{
			Alias:     "Child",
			Component: childComponent,
			Props:     []ir.Prop{{Name: "label", Binding: "title", Bound: true}},
			Events:    []ir.EventHandler{{Event: "select", Method: "OnSelect"}},
			Nodes:     childComponent.Nodes,
		},
	}}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(parent, runtime.State{
		Signals: map[string]runtime.Signal{"title": parentTitle},
		Methods: map[string]func(){
			"OnSelect": func() { parentSelected++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if !frameContainsText(target.LastFrame(), "Parent Label") || !frameContainsText(target.LastFrame(), "0") {
		t.Fatalf("initial child frame missing prop or child state: %+v", target.LastFrame())
	}
	if parent.Nodes[0].(*ir.ComponentInstance).Nodes[0].(*ir.Element).Events[0].Method != "Pressed" {
		t.Fatalf("child node event mutated: %+v", parent.Nodes[0].(*ir.ComponentInstance).Nodes[0].(*ir.Element).Events)
	}
	button := findElement(target.LastFrame(), "button")
	if button.Props["on:click"] == "" {
		t.Fatalf("child button missing click handler: %+v", target.LastFrame())
	}
	if !app.DispatchPointer(button.Rect.X+1, button.Rect.Y+1) {
		t.Fatal("child pointer dispatch failed")
	}
	app.Flush()
	if childCount.Get() != 1 {
		t.Fatalf("child count = %d", childCount.Get())
	}
	if parentSelected != 1 {
		t.Fatalf("parent selected calls = %d", parentSelected)
	}
	if !frameContainsText(target.LastFrame(), "1") {
		t.Fatalf("updated child frame missing child state: %+v", target.LastFrame())
	}
}

func TestComponentInputPropSyncsBackToParentSignal(t *testing.T) {
	scheduler := reactivity.NewScheduler()
	parentSearch := runtime.NewSignal("", scheduler)
	childSearch := runtime.NewSignal("", scheduler)
	childComponent := &ir.Component{
		Name:      "SearchBox",
		PropNames: []string{"search"},
		Props:     []ir.PropDef{{Name: "search"}},
		Nodes: []ir.Node{&ir.Element{
			Tag:   "input",
			Props: []ir.Prop{{Name: "value", Binding: "search", Bound: true}},
		}},
		NewState: func() ir.RuntimeState {
			return ir.RuntimeState{Signals: map[string]ir.Signal{"search": childSearch}}
		},
	}
	parent := &ir.Component{Name: "Parent", Nodes: []ir.Node{&ir.ComponentInstance{
		Alias:     "SearchBox",
		Component: childComponent,
		Props:     []ir.Prop{{Name: "search", Binding: "search", Bound: true}},
		Nodes:     childComponent.Nodes,
	}}}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(parent, runtime.State{
		Signals: map[string]runtime.Signal{"search": parentSearch},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 200},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	input := findElement(target.LastFrame(), "input")
	if !app.DispatchPointer(input.Rect.X+1, input.Rect.Y+1) {
		t.Fatal("input focus failed")
	}
	if !app.DispatchTextInput("road") {
		t.Fatal("text input failed")
	}
	if childSearch.Get() != "road" || parentSearch.Get() != "road" {
		t.Fatalf("child=%q parent=%q", childSearch.Get(), parentSearch.Get())
	}
}

func TestResizeUpdatesLayoutConstraintsOnNextFlush(t *testing.T) {
	_, testRenderer := mountCounter(t)
	root := findElement(testRenderer.LastFrame(), "div")
	if root.Rect.Width != 200 {
		t.Fatalf("initial root rect = %+v", root.Rect)
	}
	templateDoc := template.Parse(`<div class="fill"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Resize", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.fill { width: 100%; }`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300, Height: 200},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	app.Resize(540, 200)
	if findElement(target.LastFrame(), "div").Rect.Width != 300 {
		t.Fatalf("resize rendered before flush: %+v", target.LastFrame())
	}
	app.Flush()
	resized := findElement(target.LastFrame(), "div")
	if resized.Rect.Width != 540 || resized.Rect.Height != 200 {
		t.Fatalf("resized root rect = %+v", resized.Rect)
	}
}

func TestRenderCommandsCarryTextLayoutData(t *testing.T) {
	_, testRenderer := mountCounter(t)
	text := findText(testRenderer.LastFrame(), "0")
	if len(text.Lines) != 1 || text.Lines[0].Text != "0" || text.Lines[0].Baseline <= 0 {
		t.Fatalf("text lines = %+v", text.Lines)
	}
	if len(text.Glyphs) != 1 || text.Glyphs[0].Text != "0" || text.Glyphs[0].Advance <= 0 {
		t.Fatalf("text glyph runs = %+v", text.Glyphs)
	}
}

func TestDocumentTextSelectionSelectsRenderedText(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <p>Alpha</p>
  <p>Beta</p>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "SelectableText", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	alpha := findText(target.LastFrame(), "Alpha")
	if alpha.ID == "" {
		t.Fatalf("missing text command: alpha=%+v", alpha)
	}
	if !app.DispatchPointer(alpha.Rect.X+1, alpha.Rect.Y+1) {
		t.Fatal("document text selection start failed")
	}
	if !app.DispatchDrag(alpha.Rect.X+alpha.Rect.Width+1, alpha.Rect.Y+alpha.Rect.Height/2, 0, 0, runtime.Modifiers{}) {
		t.Fatal("document text selection drag failed")
	}
	app.Flush()
	selected, ok := app.SelectedDocumentText()
	if !ok || selected != "Alpha" {
		t.Fatalf("selected document text = %q, %v", selected, ok)
	}
	selection, ok := app.DocumentTextSelection()
	if !ok || selection.Start != 0 || selection.End != 5 {
		t.Fatalf("document selection = %+v, %v", selection, ok)
	}
	if !frameContainsKind(target.LastFrame(), "selection") {
		t.Fatalf("selection highlight command missing: %+v", target.LastFrame())
	}
	if !app.ClearDocumentTextSelection() {
		t.Fatal("clear document selection failed")
	}
	app.Flush()
	if _, ok := app.SelectedDocumentText(); ok {
		t.Fatal("document selection still present after clear")
	}
}

func TestRendererSystemTokensAffectLayout(t *testing.T) {
	templateDoc := template.Parse(`<div class="toolbar"><button>Back</button></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Toolbar", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.toolbar {
  display: flex;
  padding: 10px;
  padding-left: env(vugra-window-controls-left, 10px);
}
`, style.BasePosition{})
	target := &systemTokenRenderer{
		tokens: style.SystemTokens{"vugra-window-controls-left": 72},
	}
	runtime.MountWithOptions(component, runtime.State{}, target, runtime.Options{
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	button := findElement(target.LastFrame(), "button")
	if button.Rect.X != 72 {
		t.Fatalf("button rect should consume renderer system token: %+v", button)
	}
}

func TestSetSystemTokensSchedulesRelayout(t *testing.T) {
	templateDoc := template.Parse(`<div class="toolbar"><button>Back</button></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Toolbar", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.toolbar {
  display: flex;
  padding-left: env(vugra-window-controls-left, 10px);
}
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{}, target, runtime.Options{
		Scheduler:    scheduler,
		Styles:       sheet,
		Constraints:  layout.Constraints{Width: 240},
		Measurer:     layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		SystemTokens: style.SystemTokens{"vugra-window-controls-left": 40},
	})
	app.Flush()
	button := findElement(target.LastFrame(), "button")
	if button.Rect.X != 40 {
		t.Fatalf("button rect should consume initial system token: %+v", button)
	}
	app.SetSystemTokens(style.SystemTokens{"vugra-window-controls-left": 84})
	app.Flush()
	button = findElement(target.LastFrame(), "button")
	if button.Rect.X != 84 {
		t.Fatalf("button rect should consume updated system token: %+v", button)
	}
}

type systemTokenRenderer struct {
	renderer.TestRenderer
	tokens style.SystemTokens
}

func (r *systemTokenRenderer) SystemTokens() style.SystemTokens {
	return r.tokens
}

func TestInputTextDispatchUpdatesBoundSignal(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <input :value="name">
  <p>{{ name }}</p>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Name signal.String `+"`vugra:\"name\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Input", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal("Ada", scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	input := findElement(target.LastFrame(), "input")
	if input.Role != "textbox" || input.Props["value"] != "Ada" || input.Bindings["value"] != "name" {
		t.Fatalf("input command = %+v", input)
	}
	if !app.DispatchPointer(input.Rect.X+1, input.Rect.Y+1) {
		t.Fatal("input pointer focus failed")
	}
	if !app.DispatchText("Grace") {
		t.Fatal("input text dispatch failed")
	}
	app.Flush()
	if name.Get() != "Grace" {
		t.Fatalf("signal value = %q", name.Get())
	}
	if !frameContainsText(target.LastFrame(), "Grace") {
		t.Fatalf("updated frame missing Grace: %+v", target.LastFrame())
	}
}

func TestInputTextEntryAppendsAndDeletes(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <input :value="name">
  <p>{{ name }}</p>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Name signal.String `+"`vugra:\"name\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Input", Template: templateDoc, Go: goMeta})
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal("", scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	input := findElement(target.LastFrame(), "input")
	if !app.DispatchPointer(input.Rect.X+1, input.Rect.Y+1) {
		t.Fatal("input pointer focus failed")
	}
	if !app.DispatchTextInput("A") || !app.DispatchTextInput("d") || !app.DispatchTextInput("a") {
		t.Fatal("text input dispatch failed")
	}
	app.Flush()
	if name.Get() != "Ada" {
		t.Fatalf("name = %q", name.Get())
	}
	if !app.DispatchKey("Backspace") {
		t.Fatal("backspace dispatch failed")
	}
	app.Flush()
	if name.Get() != "Ad" {
		t.Fatalf("name after backspace = %q", name.Get())
	}
}

func TestInputTextCaretInsertionAndDeletion(t *testing.T) {
	app, name := mountTextInput(t, "abc")
	if !app.DispatchKey("ArrowLeft") {
		t.Fatal("arrow left dispatch failed")
	}
	if !app.DispatchTextInput("X") {
		t.Fatal("text insertion failed")
	}
	if name.Get() != "abXc" {
		t.Fatalf("name after caret insertion = %q", name.Get())
	}
	if !app.DispatchKey("Backspace") {
		t.Fatal("backspace dispatch failed")
	}
	if name.Get() != "abc" {
		t.Fatalf("name after backspace = %q", name.Get())
	}
	if !app.DispatchKey("ArrowLeft") || !app.DispatchKey("Delete") {
		t.Fatal("forward delete dispatch failed")
	}
	if name.Get() != "ac" {
		t.Fatalf("name after delete = %q", name.Get())
	}
}

func TestInputTextSelectAllReplacesSelection(t *testing.T) {
	app, name := mountTextInput(t, "Draft")
	if !app.DispatchKey("Mod+A") {
		t.Fatal("select all dispatch failed")
	}
	if !app.DispatchTextInput("Final") {
		t.Fatal("replacement text input failed")
	}
	if name.Get() != "Final" {
		t.Fatalf("name after select all replacement = %q", name.Get())
	}
}

func TestTextSelectionAPIReadsAndReplacesFocusedSelection(t *testing.T) {
	app, name := mountTextInput(t, "hello")
	selection, ok := app.TextSelection()
	if !ok {
		t.Fatal("missing focused text selection")
	}
	if selection.ID == "" || selection.Binding != "name" || selection.Start != 5 || selection.End != 5 {
		t.Fatalf("initial selection = %+v", selection)
	}
	if !selection.Collapsed() || selection.Caret() != 5 {
		t.Fatalf("initial selection helpers = collapsed %v caret %d", selection.Collapsed(), selection.Caret())
	}
	if !app.SetTextSelection(runtime.TextSelection{Start: 1, End: 4}) {
		t.Fatal("set focused selection failed")
	}
	selected, ok := app.SelectedText()
	if !ok || selected != "ell" {
		t.Fatalf("selected text = %q, %v", selected, ok)
	}
	if !app.DispatchTextInput("i") {
		t.Fatal("replace selection failed")
	}
	if name.Get() != "hio" {
		t.Fatalf("name after selection replacement = %q", name.Get())
	}
	selection, ok = app.TextSelection()
	if !ok || selection.Start != 2 || selection.End != 2 {
		t.Fatalf("selection after replacement = %+v, %v", selection, ok)
	}
}

func TestTextSelectionAPIUsesRuneOffsetsAndClamps(t *testing.T) {
	app, name := mountTextInput(t, "你好吗")
	if !app.SetTextSelection(runtime.TextSelection{Start: 2, End: 1}) {
		t.Fatal("set reversed rune selection failed")
	}
	selected, ok := app.SelectedText()
	if !ok || selected != "好" {
		t.Fatalf("selected text = %q, %v", selected, ok)
	}
	if !app.SetTextSelection(runtime.TextSelection{Start: -10, End: 20}) {
		t.Fatal("set clamped selection failed")
	}
	selection, ok := app.TextSelection()
	if !ok || selection.Start != 0 || selection.End != 3 {
		t.Fatalf("clamped selection = %+v, %v", selection, ok)
	}
	if !app.DispatchTextInput("哈") {
		t.Fatal("replace clamped selection failed")
	}
	if name.Get() != "哈" {
		t.Fatalf("name after clamped replacement = %q", name.Get())
	}
}

func TestTextSelectionAPICollapseAndRejectsInvalidTargets(t *testing.T) {
	app, name := mountTextInput(t, "abcd")
	if !app.CollapseTextSelection(2) {
		t.Fatal("collapse selection failed")
	}
	selection, ok := app.TextSelection()
	if !ok || selection.Start != 2 || selection.End != 2 {
		t.Fatalf("collapsed selection = %+v, %v", selection, ok)
	}
	if !app.DispatchTextInput("X") {
		t.Fatal("insert at collapsed selection failed")
	}
	if name.Get() != "abXcd" {
		t.Fatalf("name after collapsed insertion = %q", name.Get())
	}
	if app.SetTextSelection(runtime.TextSelection{ID: "missing", Start: 0, End: 1}) {
		t.Fatal("set selection on missing input succeeded")
	}
	if app.SetTextSelection(runtime.TextSelection{Binding: "other", Start: 0, End: 1}) {
		t.Fatal("set selection with wrong binding succeeded")
	}
}

func TestTextSelectionAPIReadsSelectionByInputID(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <input :value="first">
  <input :value="second">
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    First signal.String `+"`vugra:\"first\"`"+`
    Second signal.String `+"`vugra:\"second\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Inputs", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	first := runtime.NewSignal("alpha", scheduler)
	second := runtime.NewSignal("bravo", scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"first": first, "second": second},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	firstInput := nthElement(target.LastFrame(), "input", 0)
	secondInput := nthElement(target.LastFrame(), "input", 1)
	if !app.DispatchPointer(firstInput.Rect.X+1, firstInput.Rect.Y+1) {
		t.Fatal("first input focus failed")
	}
	if !app.SetTextSelection(runtime.TextSelection{ID: secondInput.ID, Start: 1, End: 4}) {
		t.Fatal("set second input selection failed")
	}
	selection, ok := app.TextSelectionFor(secondInput.ID)
	if !ok || selection.ID != secondInput.ID || selection.Binding != "second" || selection.Start != 1 || selection.End != 4 {
		t.Fatalf("second selection = %+v, %v", selection, ok)
	}
	if focused := app.FocusedID(); focused != secondInput.ID {
		t.Fatalf("focused after setting second selection = %q want %q", focused, secondInput.ID)
	}
	selected, ok := app.SelectedText()
	if !ok || selected != "rav" {
		t.Fatalf("selected second text = %q, %v", selected, ok)
	}
	selected, ok = app.SelectedTextFor(secondInput.ID)
	if !ok || selected != "rav" {
		t.Fatalf("selected second text by id = %q, %v", selected, ok)
	}
	if !app.CollapseTextSelectionFor(firstInput.ID, 2) {
		t.Fatal("collapse first input selection by id failed")
	}
	selection, ok = app.TextSelectionFor(firstInput.ID)
	if !ok || selection.Start != 2 || selection.End != 2 {
		t.Fatalf("collapsed first selection = %+v, %v", selection, ok)
	}
	if focused := app.FocusedID(); focused != firstInput.ID {
		t.Fatalf("focused after collapsing first selection = %q want %q", focused, firstInput.ID)
	}
}

func TestInputTextCaretUsesRuneOffsets(t *testing.T) {
	app, name := mountTextInput(t, "你a")
	if !app.DispatchKey("ArrowLeft") {
		t.Fatal("arrow left dispatch failed")
	}
	if !app.DispatchTextInput("好") {
		t.Fatal("unicode insertion failed")
	}
	if name.Get() != "你好a" {
		t.Fatalf("name after unicode insertion = %q", name.Get())
	}
	if !app.DispatchKey("Backspace") {
		t.Fatal("unicode backspace failed")
	}
	if name.Get() != "你a" {
		t.Fatalf("name after unicode backspace = %q", name.Get())
	}
}

func TestInputTextHomeEndMovesCaret(t *testing.T) {
	app, name := mountTextInput(t, "abc")
	if !app.DispatchKey("Home") {
		t.Fatal("home dispatch failed")
	}
	if !app.DispatchTextInput("X") {
		t.Fatal("home text insertion failed")
	}
	if name.Get() != "Xabc" {
		t.Fatalf("name after home insertion = %q", name.Get())
	}
	if !app.DispatchKey("Home") || !app.DispatchKey("End") {
		t.Fatal("home/end dispatch failed")
	}
	if !app.DispatchTextInput("Y") {
		t.Fatal("end text insertion failed")
	}
	if name.Get() != "XabcY" {
		t.Fatalf("name after end insertion = %q", name.Get())
	}
	if !app.DispatchKey("Mod+A") {
		t.Fatal("select all dispatch failed")
	}
	if !app.DispatchKey("Home") {
		t.Fatal("home collapsed selection failed")
	}
	if !app.DispatchTextInput("Z") {
		t.Fatal("collapsed selection insertion failed")
	}
	if name.Get() != "ZXabcY" {
		t.Fatalf("name after collapsed home insertion = %q", name.Get())
	}
}

func TestCheckboxToggleDispatchUpdatesBoundSignal(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <input type="checkbox" :checked="enabled">
  <p>{{ enabled }}</p>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Enabled signal.Bool `+"`vugra:\"enabled\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Checkbox", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	enabled := runtime.NewSignal(false, scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"enabled": enabled},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	checkbox := findElement(target.LastFrame(), "input")
	if checkbox.Role != "checkbox" || checkbox.Props["checked"] != "false" || checkbox.Bindings["checked"] != "enabled" {
		t.Fatalf("checkbox command = %+v", checkbox)
	}
	if !app.DispatchPointer(checkbox.Rect.X+1, checkbox.Rect.Y+1) {
		t.Fatal("checkbox pointer toggle failed")
	}
	app.Flush()
	if !enabled.Get() {
		t.Fatal("checkbox did not toggle true")
	}
	if !app.DispatchKey(" ") {
		t.Fatal("checkbox keyboard toggle failed")
	}
	app.Flush()
	if enabled.Get() {
		t.Fatal("checkbox did not toggle false")
	}
}

func TestRepeaterRendersSignalList(t *testing.T) {
	templateDoc := template.Parse(`
<ul>
  <li v-for="item in items">{{ item }}</li>
</ul>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Items signal.Signal[[]string] `+"`vugra:\"items\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "List", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	items := runtime.NewSignal([]string{"alpha", "beta", "gamma"}, scheduler)
	target := &renderer.TestRenderer{}
	runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"items": items},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	frame := target.LastFrame()
	for _, text := range []string{"alpha", "beta", "gamma"} {
		if !frameContainsText(frame, text) {
			t.Fatalf("frame missing %q: %+v", text, frame)
		}
	}
}

func TestOverflowScrollOffsetsChildrenAndHitTesting(t *testing.T) {
	templateDoc := template.Parse(`
<div class="viewport">
  <button @click="First">First</button>
  <button @click="Second">Second</button>
  <button @click="Third">Third</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}

func (s *State) First() {}
func (s *State) Second() {}
func (s *State) Third() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Scroll", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.viewport {
  width: 160px;
  height: 30px;
  overflow: scroll;
}
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	calls := map[string]int{}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){
			"First":  func() { calls["first"]++ },
			"Second": func() { calls["second"]++ },
			"Third":  func() { calls["third"]++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 220},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	initialSecond := nthElement(target.LastFrame(), "button", 1)
	if initialSecond.Rect.Y != 20 {
		t.Fatalf("initial second button rect = %+v", initialSecond.Rect)
	}
	if !app.DispatchScroll(10, 10, 30) {
		t.Fatal("scroll dispatch failed")
	}
	app.Flush()
	scrolledSecond := nthElement(target.LastFrame(), "button", 1)
	if scrolledSecond.Rect.Y != -10 {
		t.Fatalf("scrolled second button rect = %+v", scrolledSecond.Rect)
	}
	scrolledSecondText := findText(target.LastFrame(), "Second")
	if scrolledSecondText.Rect.Y != -10 || scrolledSecondText.Lines[0].Y != -10 || scrolledSecondText.Glyphs[0].Y != -10 {
		t.Fatalf("scrolled second text was not translated consistently: rect=%+v lines=%+v glyphs=%+v", scrolledSecondText.Rect, scrolledSecondText.Lines, scrolledSecondText.Glyphs)
	}
	if app.DispatchPointer(10, 35) {
		t.Fatal("click outside clipped third button should not dispatch")
	}
	if !app.DispatchPointer(10, 5) {
		t.Fatal("click inside scrolled second button did not dispatch")
	}
	if calls["second"] != 1 {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestNestedOverflowDoesNotCreateViewportScroll(t *testing.T) {
	templateDoc := template.Parse(`
<div class="app">
  <div class="viewport">
    <button @click="First">First</button>
    <button @click="Second">Second</button>
    <button @click="Third">Third</button>
  </div>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}

func (s *State) First() {}
func (s *State) Second() {}
func (s *State) Third() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "NestedScroll", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.app {
  width: 160px;
  height: 30px;
  overflow: hidden;
}
.viewport {
  width: 160px;
  height: 30px;
  overflow: scroll;
}
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){
			"First":  func() {},
			"Second": func() {},
			"Third":  func() {},
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 220, Height: 30},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	initialRoot := findElement(target.LastFrame(), "div")
	if initialRoot.Rect.Y != 0 {
		t.Fatalf("initial root rect = %+v", initialRoot.Rect)
	}
	if !app.DispatchScroll(10, 10, 30) {
		t.Fatal("nested scroll dispatch failed")
	}
	app.Flush()
	scrolledRoot := findElement(target.LastFrame(), "div")
	if scrolledRoot.Rect.Y != 0 {
		t.Fatalf("viewport scrolled the whole app instead of inner overflow: %+v", scrolledRoot.Rect)
	}
	scrolledSecond := nthElement(target.LastFrame(), "button", 1)
	if scrolledSecond.Rect.Y != -10 {
		t.Fatalf("inner scrolled second button rect = %+v", scrolledSecond.Rect)
	}
}

func TestViewportScrollOffsetsRootContentAndHitTesting(t *testing.T) {
	templateDoc := template.Parse(`
<div class="app">
  <button @click="First">First</button>
  <button @click="Second">Second</button>
  <button @click="Third">Third</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}

func (s *State) First() {}
func (s *State) Second() {}
func (s *State) Third() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "ViewportScroll", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.app {
  width: 160px;
  min-height: 100%;
}
button {
  height: 20px;
}
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	calls := map[string]int{}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){
			"First":  func() { calls["first"]++ },
			"Second": func() { calls["second"]++ },
			"Third":  func() { calls["third"]++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 220, Height: 30},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})

	initialThird := nthElement(target.LastFrame(), "button", 2)
	if initialThird.Rect.Y != 40 {
		t.Fatalf("initial third button rect = %+v", initialThird.Rect)
	}
	if app.DispatchPointer(10, 45) {
		t.Fatal("click outside viewport should not dispatch")
	}
	if !app.DispatchScroll(10, 10, 30) {
		t.Fatal("viewport scroll dispatch failed")
	}
	app.Flush()
	scrolledThird := nthElement(target.LastFrame(), "button", 2)
	if scrolledThird.Rect.Y != 10 {
		t.Fatalf("scrolled third button rect = %+v", scrolledThird.Rect)
	}
	scrolledThirdText := findText(target.LastFrame(), "Third")
	if scrolledThirdText.Rect.Y != 10 || scrolledThirdText.Lines[0].Y != 10 || scrolledThirdText.Glyphs[0].Y != 10 {
		t.Fatalf("viewport scrolled third text was not translated consistently: rect=%+v lines=%+v glyphs=%+v", scrolledThirdText.Rect, scrolledThirdText.Lines, scrolledThirdText.Glyphs)
	}
	if !app.DispatchPointer(10, 15) {
		t.Fatal("click inside scrolled third button did not dispatch")
	}
	if calls["third"] != 1 {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestHitTestTreeDispatchesTopmostOverlappingElement(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <button class="under" @click="Under">Under</button>
  <button class="over" @click="Over">Over</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}
func (s *State) Under() {}
func (s *State) Over() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Overlap", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.root {
  display: grid;
  grid-template-columns: 80px;
  grid-template-rows: 30px;
}
.under {
  grid-column: 1;
  grid-row: 1;
  width: 80px;
  height: 30px;
}
.over {
  grid-column: 1;
  grid-row: 1;
  width: 80px;
  height: 30px;
}
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	calls := map[string]int{}
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){
			"Under": func() { calls["under"]++ },
			"Over":  func() { calls["over"]++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 200},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if id, ok := app.HitTest(1, 1); !ok || id == "" {
		t.Fatalf("hit test = %q %v", id, ok)
	}
	if !app.DispatchPointer(1, 1) {
		t.Fatal("overlapping pointer dispatch failed")
	}
	if calls["over"] != 1 || calls["under"] != 0 {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestPointerHoverDragAndAutofocusKeyDispatch(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <button class="splitter" @hover="Hover" @drag="Drag">Split</button>
  <input class="rename" :value="name" autofocus="true" @keydown="Commit">
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Name signal.String `+"`vugra:\"name\"`"+`
}
func (s *State) Hover() {}
func (s *State) Drag() {}
func (s *State) Commit() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Pointer", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.splitter { width: 20px; height: 40px; }
.rename { width: 160px; height: 30px; }
`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal("Draft", scheduler)
	target := &renderer.TestRenderer{}
	var hover runtime.Event
	var drag runtime.Event
	commits := 0
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		EventMethods: map[string]func(runtime.Event){
			"Hover":  func(event runtime.Event) { hover = event },
			"Drag":   func(event runtime.Event) { drag = event },
			"Commit": func(event runtime.Event) { commits++ },
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240, Height: 100},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if !app.DispatchText("Final") {
		t.Fatal("autofocused input did not accept text")
	}
	if name.Get() != "Final" {
		t.Fatalf("name = %q", name.Get())
	}
	if !app.DispatchKey("Enter") {
		t.Fatal("enter did not dispatch input keydown")
	}
	if commits != 1 {
		t.Fatalf("commits = %d", commits)
	}
	splitter := findElement(target.LastFrame(), "button")
	if !app.DispatchHover(splitter.Rect.X+1, splitter.Rect.Y+1, runtime.Modifiers{Shift: true}) {
		t.Fatal("hover dispatch failed")
	}
	if hover.Type != "hover" || !hover.Modifiers.Shift {
		t.Fatalf("hover event = %+v", hover)
	}
	if !app.DispatchPointer(splitter.Rect.X+1, splitter.Rect.Y+1) {
		t.Fatal("pointer down failed")
	}
	if !app.DispatchDrag(splitter.Rect.X+21, splitter.Rect.Y+1, 20, 0, runtime.Modifiers{}) {
		t.Fatal("drag dispatch failed")
	}
	if drag.Type != "drag" || drag.DeltaX != 20 {
		t.Fatalf("drag event = %+v", drag)
	}
}

func TestModalFocusScopeTrapsTabNavigation(t *testing.T) {
	templateDoc := template.Parse(`
<div class="root">
  <button @click="Background">Background</button>
  <div class="modal" focus-scope="modal">
    <button @click="Cancel">Cancel</button>
    <button @click="Done">Done</button>
  </div>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {}
func (s *State) Background() {}
func (s *State) Cancel() {}
func (s *State) Done() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Modal", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Methods: map[string]func(){
			"Background": func() {},
			"Cancel":     func() {},
			"Done":       func() {},
		},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240, Height: 120},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if !app.DispatchKey("Tab") {
		t.Fatal("first tab failed")
	}
	first := app.FocusedEvent()
	if first == "" || !strings.HasSuffix(first, ":click") {
		t.Fatalf("focused event = %q", first)
	}
	if first == findEventID(target.LastFrame(), "click") {
		t.Fatalf("focus escaped to background: %q", first)
	}
	if !app.DispatchKey("Tab") || app.FocusedEvent() == first {
		t.Fatalf("second tab did not cycle inside modal: %q", app.FocusedEvent())
	}
	second := app.FocusedEvent()
	if !app.DispatchKey("Tab") || app.FocusedEvent() != first {
		t.Fatalf("third tab did not wrap inside modal: %q want %q", app.FocusedEvent(), first)
	}
	if !app.DispatchKey("Shift+Tab") || app.FocusedEvent() != second {
		t.Fatalf("shift tab did not reverse inside modal: got %q want %q", app.FocusedEvent(), second)
	}
}

func mountCounter(t *testing.T) (*runtime.App, *renderer.TestRenderer) {
	t.Helper()
	templateDoc := template.Parse(`
<div class="counter">
  <p>{{ count }}</p>
  <button @click="Inc">+</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Count signal.Int `+"`vugra:\"count\"`"+`
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}
`, goanalysis.BasePosition{Offset: 100, Line: 10, Column: 1})
	component := ir.Build(ir.BuildInput{
		Name:     "Counter",
		Template: templateDoc,
		Go:       goMeta,
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}

	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	testRenderer := &renderer.TestRenderer{}
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 200px;
}
`, style.BasePosition{})
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{
			"count": count,
		},
		Methods: map[string]func(){
			"Inc": func() {
				count.Update(func(value int) int { return value + 1 })
			},
		},
	}, testRenderer, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	return app, testRenderer
}

func mountTextInput(t *testing.T, initial string) (*runtime.App, *runtime.SignalValue[string]) {
	t.Helper()
	templateDoc := template.Parse(`
<div class="root">
  <input :value="name">
  <p>{{ name }}</p>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Name signal.String `+"`vugra:\"name\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Input", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("unexpected IR diagnostics: %+v", component.Diagnostics)
	}
	scheduler := reactivity.NewScheduler()
	name := runtime.NewSignal(initial, scheduler)
	target := &renderer.TestRenderer{}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"name": name},
		Methods: map[string]func(){},
	}, target, runtime.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	input := findElement(target.LastFrame(), "input")
	if !app.DispatchPointer(input.Rect.X+1, input.Rect.Y+1) {
		t.Fatal("input pointer focus failed")
	}
	return app, name
}

func findElement(frame []renderer.Command, tag string) renderer.Command {
	for _, command := range frame {
		if command.Kind == "element" && command.Tag == tag {
			return command
		}
	}
	return renderer.Command{}
}

func nthElement(frame []renderer.Command, tag string, index int) renderer.Command {
	seen := 0
	for _, command := range frame {
		if command.Kind == "element" && command.Tag == tag {
			if seen == index {
				return command
			}
			seen++
		}
	}
	return renderer.Command{}
}

func findText(frame []renderer.Command, text string) renderer.Command {
	for _, command := range frame {
		if command.Kind == "text" && command.Text == text {
			return command
		}
	}
	return renderer.Command{}
}

func findCommand(frame []renderer.Command, kind string) renderer.Command {
	for _, command := range frame {
		if command.Kind == kind {
			return command
		}
	}
	return renderer.Command{}
}

func frameContainsKind(frame []renderer.Command, kind string) bool {
	for _, command := range frame {
		if command.Kind == kind {
			return true
		}
	}
	return false
}

func frameContainsText(frame []renderer.Command, text string) bool {
	for _, command := range frame {
		if command.Kind == "text" && command.Text == text {
			return true
		}
	}
	return false
}

func findEventID(frame []renderer.Command, event string) string {
	for _, command := range frame {
		if command.Kind != "element" {
			continue
		}
		if command.Props == nil {
			continue
		}
		if id := command.Props["on:"+event]; id != "" {
			return id
		}
	}
	return ""
}
