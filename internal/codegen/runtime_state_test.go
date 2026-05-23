package codegen_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/codegen"
	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/style"
)

func TestGenerateRuntimeState(t *testing.T) {
	script := `
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Count signal.Int ` + "`vugra:\"count\"`" + `
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}

func (s *State) Select(event vugra.Event) {}
`
	meta := goanalysis.Analyze(script, goanalysis.BasePosition{})
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "counter",
		Script:      script,
		Metadata:    meta,
		Component: &ir.Component{Name: "Counter", Nodes: []ir.Node{
			&ir.Element{
				Tag: "button",
				Events: []ir.EventHandler{
					{Event: "click", Method: "Inc"},
				},
				Children: []ir.Node{&ir.Interpolation{Binding: "count", GoField: "Count"}},
			},
		}},
		Styles: &style.Stylesheet{Rules: []style.Rule{
			{
				Selector:  ".counter",
				ClassName: "counter",
				Declarations: []style.Declaration{
					{Name: "padding", Value: "16px"},
				},
			},
		}},
	})
	if !strings.Contains(generated, `"count": &state.Count`) {
		t.Fatalf("missing signal binding:\n%s", generated)
	}
	if !strings.Contains(generated, `"Inc": state.Inc`) {
		t.Fatalf("missing method binding:\n%s", generated)
	}
	if strings.Contains(generated, `"Select": state.Select,`) && strings.Contains(generated, "Methods: map[string]func(){\n\t\t\t\"Select\"") {
		t.Fatalf("event method registered as simple method:\n%s", generated)
	}
	if !strings.Contains(generated, `EventMethods: map[string]func(vugra.Event)`) || !strings.Contains(generated, `"Select": state.Select`) {
		t.Fatalf("missing event method binding:\n%s", generated)
	}
	if !strings.Contains(generated, "func Component() *vugra.Component") {
		t.Fatalf("missing static component constructor:\n%s", generated)
	}
	if !strings.Contains(generated, "func Styles() *vugra.Stylesheet") {
		t.Fatalf("missing static style constructor:\n%s", generated)
	}
	if strings.Contains(generated, "import import") {
		t.Fatalf("malformed import block:\n%s", generated)
	}
	if strings.Contains(generated, "github.com/vugra/vugra/internal/") {
		t.Fatalf("generated component should not import internal packages:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "counter_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStatePreservesSystemAPICalls(t *testing.T) {
	script := `
import (
    "github.com/vugra/vugra/pkg/signal"
    "github.com/vugra/vugra/pkg/system"
)

type State struct {
    Status signal.String ` + "`vugra:\"status\"`" + `
}

func (s *State) LoadHome() {
    entries, err := system.ReadDir(".")
    if err != nil {
        s.Status.Set(err.Error())
        return
    }
    s.Status.Set(entries[0].Name)
}
`
	meta := goanalysis.Analyze(script, goanalysis.BasePosition{})
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "systemdemo",
		Script:      script,
		Metadata:    meta,
		Component: &ir.Component{Name: "SystemDemo", Nodes: []ir.Node{
			&ir.Element{
				Tag:    "button",
				Events: []ir.EventHandler{{Event: "click", Method: "LoadHome"}},
				Children: []ir.Node{
					&ir.Interpolation{Binding: "status", GoField: "Status"},
				},
			},
		}},
	})
	if !strings.Contains(generated, `"github.com/vugra/vugra/pkg/system"`) {
		t.Fatalf("missing system import:\n%s", generated)
	}
	if !strings.Contains(generated, "system.ReadDir") {
		t.Fatalf("system call was not preserved:\n%s", generated)
	}
	if !strings.Contains(generated, `"LoadHome": state.LoadHome`) {
		t.Fatalf("missing method binding:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "systemdemo_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStateStripsVugraImportsAndWritesComponentInstances(t *testing.T) {
	script := `
import (
    Badge "./Badge.vue"
    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}
`
	meta := goanalysis.Analyze(script, goanalysis.BasePosition{})
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "parent",
		Script:      script,
		Metadata:    meta,
		Component: &ir.Component{
			Name:      "Parent",
			PropNames: []string{"title"},
			Props:     []ir.PropDef{{Name: "title", GoField: "Title", Type: "signal.String", Required: true, Default: "Untitled"}},
			Provides:  []ir.ProvideDef{{Name: "theme", Binding: "theme"}},
			Injects:   []ir.InjectDef{{Name: "locale", GoField: "Locale", Type: "signal.String", Default: "en"}},
			Emits:     []ir.Emit{{Method: "Pressed", Event: "select"}},
			Lifecycle: []ir.Lifecycle{{Hook: "mounted", Method: "Mounted"}},
			Nodes: []ir.Node{
				&ir.ComponentInstance{
					Alias:     "Badge",
					Props:     []ir.Prop{{Name: "label", Binding: "title", Bound: true}},
					Events:    []ir.EventHandler{{Event: "click", Method: "Save"}},
					Slots:     []ir.Slot{{Name: "default", Scope: "slot", Nodes: []ir.Node{&ir.Text{Value: "slot"}}}},
					Lifecycle: []ir.Lifecycle{{Hook: "mounted", Method: "Mounted"}},
					Nodes: []ir.Node{&ir.Element{
						Tag:      "span",
						Children: []ir.Node{&ir.Text{Value: "Imported"}},
					}},
				},
			},
		},
	})
	if strings.Contains(generated, ".vugra") {
		t.Fatalf("generated code kept .vue import:\n%s", generated)
	}
	if !strings.Contains(generated, "vugra.ComponentInstance") {
		t.Fatalf("missing component instance output:\n%s", generated)
	}
	if !strings.Contains(generated, `PropNames: []string{"title"`) {
		t.Fatalf("missing component prop names output:\n%s", generated)
	}
	if !strings.Contains(generated, `Props: []vugra.PropDef`) || !strings.Contains(generated, `GoField: "Title"`) || !strings.Contains(generated, `Required: true`) || !strings.Contains(generated, `Default: "Untitled"`) {
		t.Fatalf("missing component prop defs output:\n%s", generated)
	}
	if !strings.Contains(generated, `Emits: []vugra.EmitDef`) || !strings.Contains(generated, `Event: "select"`) {
		t.Fatalf("missing component emits output:\n%s", generated)
	}
	if !strings.Contains(generated, `Provides: []vugra.ProvideDef`) || !strings.Contains(generated, `Injects: []vugra.InjectDef`) || !strings.Contains(generated, `Default: "en"`) {
		t.Fatalf("missing provide/inject output:\n%s", generated)
	}
	if !strings.Contains(generated, `Lifecycle: []vugra.Lifecycle`) || !strings.Contains(generated, `Hook: "mounted"`) {
		t.Fatalf("missing lifecycle output:\n%s", generated)
	}
	if !strings.Contains(generated, `Props: []vugra.Prop`) || !strings.Contains(generated, `Slots: []vugra.Slot`) || !strings.Contains(generated, `Events: []vugra.EventHandler`) {
		t.Fatalf("missing component metadata output:\n%s", generated)
	}
	if !strings.Contains(generated, `Scope: "slot"`) {
		t.Fatalf("missing slot scope output:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "parent_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStateWritesImportedComponentStateFactories(t *testing.T) {
	parentScript := `
import (
    Badge "./Badge.vue"
    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}

func (s *State) OnSelect() {}
`
	childScript := `
import (
    "github.com/vugra/vugra/pkg/signal"
    "github.com/vugra/vugra/pkg/vugra"
)

type State struct {
    Label signal.String ` + "`vugra:\"label\"`" + `
}

func (s *State) Pressed() {
    vugra.Emit("select")
}
`
	childMeta := goanalysis.Analyze(childScript, goanalysis.BasePosition{})
	child := &ir.Component{
		Name:      "Badge",
		PropNames: []string{"label"},
		Props:     []ir.PropDef{{Name: "label", GoField: "Label", Type: "signal.String", Required: true}},
		Emits:     []ir.Emit{{Method: "Pressed", Event: "select"}},
		Nodes: []ir.Node{&ir.Element{
			Tag:    "button",
			Events: []ir.EventHandler{{Event: "click", Method: "Pressed"}},
			Children: []ir.Node{
				&ir.Interpolation{Binding: "label", GoField: "Label"},
			},
		}},
	}
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "parent",
		Script:      parentScript,
		Metadata:    goanalysis.Analyze(parentScript, goanalysis.BasePosition{}),
		Component: &ir.Component{Name: "Parent", Nodes: []ir.Node{&ir.ComponentInstance{
			Alias:     "Badge",
			Component: child,
			Props:     []ir.Prop{{Name: "label", Binding: "title", Bound: true}},
			Events:    []ir.EventHandler{{Event: "select", Method: "OnSelect"}},
			Nodes:     child.Nodes,
		}}},
		Imports: []codegen.ImportedComponentInput{{
			Alias:     "Badge",
			Script:    childScript,
			Metadata:  childMeta,
			Component: child,
		}},
	})
	if !strings.Contains(generated, "type BadgeState struct") || strings.Contains(generated, "type State struct {\n\tLabel") {
		t.Fatalf("missing renamed child state:\n%s", generated)
	}
	if !strings.Contains(generated, "func NewBadgeStateRuntimeState() vugra.State") {
		t.Fatalf("missing child runtime state factory:\n%s", generated)
	}
	if !strings.Contains(generated, `NewState: NewBadgeStateRuntimeState`) {
		t.Fatalf("missing component NewState attachment:\n%s", generated)
	}
	if strings.Contains(generated, ".vugra") {
		t.Fatalf("generated code kept .vue import:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "parent_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStateStripsChildPackageClauses(t *testing.T) {
	parentScript := `
import Badge "./Badge.vue"

type State struct{}
`
	childScript := `
package finder

type State struct{}
`
	child := &ir.Component{Name: "Badge", Nodes: []ir.Node{&ir.Element{Tag: "span"}}}
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "parent",
		Script:      parentScript,
		Metadata:    goanalysis.Analyze(parentScript, goanalysis.BasePosition{}),
		Component: &ir.Component{Name: "Parent", Nodes: []ir.Node{&ir.ComponentInstance{
			Alias:     "Badge",
			Component: child,
			Nodes:     child.Nodes,
		}}},
		Imports: []codegen.ImportedComponentInput{{
			Alias:     "Badge",
			Script:    childScript,
			Metadata:  goanalysis.Analyze(childScript, goanalysis.BasePosition{}),
			Component: child,
		}},
	})
	if strings.Contains(generated, "package finder") {
		t.Fatalf("generated code kept child package clause:\n%s", generated)
	}
	if !strings.Contains(generated, "type BadgeState struct{}") {
		t.Fatalf("missing renamed child state:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "parent_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStateDeduplicatesRepeatedImportedComponents(t *testing.T) {
	parentScript := `
import (
    One "./Icon.vue"
    Two "./Icon.vue"
)

type State struct{}
`
	childScript := `type State struct{}`
	childOne := &ir.Component{Name: "Icon", Nodes: []ir.Node{&ir.Element{Tag: "span"}}}
	childTwo := &ir.Component{Name: "Icon", Nodes: []ir.Node{&ir.Element{Tag: "span"}}}
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "parent",
		Script:      parentScript,
		Metadata:    goanalysis.Analyze(parentScript, goanalysis.BasePosition{}),
		Component: &ir.Component{Name: "Parent", Nodes: []ir.Node{
			&ir.ComponentInstance{Alias: "One", Component: childOne, Nodes: childOne.Nodes},
			&ir.ComponentInstance{Alias: "Two", Component: childTwo, Nodes: childTwo.Nodes},
		}},
		Imports: []codegen.ImportedComponentInput{
			{Alias: "One", Path: "/tmp/Icon.vue", Script: childScript, Metadata: goanalysis.Analyze(childScript, goanalysis.BasePosition{}), Component: childOne},
			{Alias: "Two", Path: "/tmp/Icon.vue", Script: childScript, Metadata: goanalysis.Analyze(childScript, goanalysis.BasePosition{}), Component: childTwo},
		},
	})
	if strings.Count(generated, "type OneState struct{}") != 1 {
		t.Fatalf("expected one child state declaration:\n%s", generated)
	}
	if strings.Count(generated, "func NewOneStateRuntimeState() vugra.State") != 1 {
		t.Fatalf("expected one child runtime factory:\n%s", generated)
	}
	if strings.Contains(generated, "\tstate := &OneState{}") {
		t.Fatalf("empty child state factory should not declare unused state:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "parent_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateRuntimeStateWritesDynamicComponent(t *testing.T) {
	script := `
type State struct {
    Current signal.String ` + "`vugra:\"current\"`" + `
}
`
	meta := goanalysis.Analyze(script, goanalysis.BasePosition{})
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "dynamic",
		Script:      script,
		Metadata:    meta,
		Component: &ir.Component{Name: "Dynamic", Nodes: []ir.Node{
			&ir.DynamicComponent{
				Binding: "current",
				Cases: []ir.DynamicComponentCase{
					{Alias: "Card", Nodes: []ir.Node{&ir.Element{Tag: "p", Children: []ir.Node{&ir.Text{Value: "card"}}}}},
				},
			},
		}},
	})
	if !strings.Contains(generated, "vugra.DynamicComponent") || !strings.Contains(generated, "vugra.DynamicComponentCase") {
		t.Fatalf("missing dynamic component output:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "dynamic_vuego.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateSoftwareMain(t *testing.T) {
	generated := codegen.GenerateSoftwareMain(codegen.SoftwareMainInput{
		PackageImport: "github.com/example/counter/component",
		OutputPath:    "counter.png",
	})
	if !strings.Contains(generated, `component "github.com/example/counter/component"`) {
		t.Fatalf("missing component import:\n%s", generated)
	}
	if strings.Contains(generated, "internal/compiler") {
		t.Fatalf("generated software main should not compile .vue at runtime:\n%s", generated)
	}
	if strings.Contains(generated, "github.com/vugra/vugra/internal/") {
		t.Fatalf("generated software main should not import internal packages:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateWasmMain(t *testing.T) {
	generated := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: "github.com/example/counter/component",
		CanvasID:      "counter-canvas",
	})
	if !strings.Contains(generated, "//go:build js && wasm") {
		t.Fatalf("missing wasm build tag:\n%s", generated)
	}
	if !strings.Contains(generated, `document.Call("getElementById", "counter-canvas")`) {
		t.Fatalf("missing canvas lookup:\n%s", generated)
	}
	if !strings.Contains(generated, `if value := canvas.Get("clientWidth"); value.Truthy()`) ||
		!strings.Contains(generated, `if value := canvas.Get("clientHeight"); value.Truthy()`) {
		t.Fatalf("missing viewport canvas sizing:\n%s", generated)
	}
	if !strings.Contains(generated, "vugra.NewCanvasRenderer(canvas, width, height)") {
		t.Fatalf("missing viewport-sized canvas renderer:\n%s", generated)
	}
	if !strings.Contains(generated, `window.Call("addEventListener", "resize", resize)`) ||
		!strings.Contains(generated, `window.Get("addEventListener").Truthy()`) ||
		!strings.Contains(generated, "vugra.ResizeCanvasRenderer(target, width, height)") ||
		!strings.Contains(generated, "mounted.Resize(float32(width), float32(height))") {
		t.Fatalf("missing viewport resize bridge:\n%s", generated)
	}
	if !strings.Contains(generated, "measurer := vugra.NewCanvasMeasurer(canvas)") || !strings.Contains(generated, "Measurer:    measurer") {
		t.Fatalf("missing canvas measurer:\n%s", generated)
	}
	if !strings.Contains(generated, "mounted.Flush()") {
		t.Fatalf("missing lifecycle initialization flush:\n%s", generated)
	}
	if !strings.Contains(generated, "vugra.InstallTextEvents") {
		t.Fatalf("missing text event bridge:\n%s", generated)
	}
	if !strings.Contains(generated, `vugra.SyncAccessibility("vugra-a11y", mounted.LastFrame(), mounted.FocusedID())`) {
		t.Fatalf("missing accessibility sync:\n%s", generated)
	}
	if !strings.Contains(generated, `vugra.InstallAccessibilityEventHandlers("vugra-a11y"`) ||
		!strings.Contains(generated, "vugra.AccessibilityEvents") ||
		!strings.Contains(generated, "mounted.FocusID(id)") ||
		!strings.Contains(generated, "mounted.DispatchKey(key)") ||
		!strings.Contains(generated, "mounted.DispatchTextInput(text)") {
		t.Fatalf("missing accessibility action bridge:\n%s", generated)
	}
	if !strings.Contains(generated, "mounted.DispatchScroll(float32(x), float32(y), float32(deltaY))") {
		t.Fatalf("scroll event bridge should match runtime float32 API:\n%s", generated)
	}
	if strings.Contains(generated, "internal/compiler") {
		t.Fatalf("generated wasm main should not compile .vue at runtime:\n%s", generated)
	}
	if strings.Contains(generated, "github.com/vugra/vugra/internal/") {
		t.Fatalf("generated wasm main should not import internal packages:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "wasm_main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateWasmMainUsesConfiguredSize(t *testing.T) {
	generated := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: "github.com/example/counter/component",
		CanvasID:      "counter-canvas",
		Width:         1024,
		Height:        720,
	})
	if !strings.Contains(generated, "width = 1024") || !strings.Contains(generated, "height = 720") {
		t.Fatalf("missing configured fallback canvas size:\n%s", generated)
	}
	if !strings.Contains(generated, "vugra.Constraints{Width: float32(width), Height: float32(height)}") {
		t.Fatalf("missing viewport layout constraints:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "wasm_main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateWasmMainUsesAssetBase(t *testing.T) {
	generated := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: "github.com/example/counter/component",
		CanvasID:      "counter-canvas",
		AssetBase:     "/project/examples/svg/SVGDemo.vue",
	})
	if !strings.Contains(generated, `AssetBase:   "/project/examples/svg/SVGDemo.vue"`) {
		t.Fatalf("missing asset base:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "wasm_main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateWasmMainUsesConfiguredState(t *testing.T) {
	generated := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: "github.com/example/finder/component",
		CanvasID:      "finder-canvas",
		State:         "component.NewDemoState()",
		RefreshHook:   "vugraRefresh",
	})
	if !strings.Contains(generated, "vugra.Mount(component.Component(), component.NewDemoState(), target") {
		t.Fatalf("missing configured state expression:\n%s", generated)
	}
	if !strings.Contains(generated, `refreshTarget.Set("vugraRefresh", refresh)`) {
		t.Fatalf("missing refresh hook:\n%s", generated)
	}
	if strings.Contains(generated, "component.NewRuntimeState(nil)") {
		t.Fatalf("custom wasm state should replace default state:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "wasm_main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}

func TestGenerateWasmMainUsesCSSLayout(t *testing.T) {
	generated := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: "github.com/example/counter/component",
		CanvasID:      "counter-canvas",
		Layout:        "css",
	})
	if !strings.Contains(generated, "Layout:      vugra.LayoutEngineCSS") {
		t.Fatalf("missing CSS layout option:\n%s", generated)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "wasm_main.go", generated, parser.AllErrors); err != nil {
		t.Fatalf("generated invalid Go: %v\n%s", err, generated)
	}
}
