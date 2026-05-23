package ir_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/goldentest"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/template"
)

func TestBuildCounterIRGolden(t *testing.T) {
	templateSource := `
<div class="counter">
  <p>{{ count }}</p>
  <button @click="Inc" :disabled="disabled">+</button>
</div>
`
	scriptSource := `
type State struct {
    Count signal.Int ` + "`vugra:\"count\"`" + `
    Disabled signal.Bool ` + "`vugra:\"disabled\"`" + `
}

func (s *State) Inc() {}
`
	component := build("Counter", templateSource, scriptSource)
	goldentest.Assert(t, "counter_ir.txt", renderComponent(component))
}

func TestBuildValidationDiagnostics(t *testing.T) {
	templateSource := `<div><p>{{ missing }}</p><button @click="Missing"></button></div>`
	scriptSource := `
type State struct {
    Count signal.Int ` + "`vugra:\"count\"`" + `
}

func (s *State) Inc() {}
`
	component := build("Broken", templateSource, scriptSource)
	if !containsDiag(component.Diagnostics, "ir.unknown_state_binding") {
		t.Fatalf("missing unknown state diagnostic: %+v", component.Diagnostics)
	}
	if !containsDiag(component.Diagnostics, "ir.unknown_event_handler") {
		t.Fatalf("missing unknown event diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildConditionalAndRepeater(t *testing.T) {
	templateSource := `<ul v-if="visible"><li v-for="item in items">{{ item }}</li></ul>`
	scriptSource := `
type State struct {
    Visible signal.Bool ` + "`vugra:\"visible\"`" + `
    Items signal.Signal[[]string] ` + "`vugra:\"items\"`" + `
}
`
	component := build("List", templateSource, scriptSource)
	if len(component.Nodes) != 1 {
		t.Fatalf("nodes = %d", len(component.Nodes))
	}
	cond, ok := component.Nodes[0].(*ir.Conditional)
	if !ok {
		t.Fatalf("expected conditional, got %T", component.Nodes[0])
	}
	if cond.Expression != "visible" {
		t.Fatalf("conditional expression = %q", cond.Expression)
	}
	elem, ok := cond.Child.(*ir.Element)
	if !ok {
		t.Fatalf("expected conditional child element, got %T", cond.Child)
	}
	if _, ok := elem.Children[0].(*ir.Repeater); !ok {
		t.Fatalf("expected repeater child, got %T", elem.Children[0])
	}
}

func TestBuildComponentInstanceFromImport(t *testing.T) {
	templateSource := `<div><Child :label="title" class="parent-class" id="child-id" @click="Save"><span>default</span><template #meta><span>named</span></template></Child></div>`
	scriptSource := `
import Child "./Child.vue"

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}

func (s *State) Save() {}
`
	child := &ir.Component{Name: "Child", PropNames: []string{"label"}, Nodes: []ir.Node{
		&ir.Element{Tag: "button", Props: []ir.Prop{{Name: "class", Value: "child-class"}}, Children: []ir.Node{
			&ir.Interpolation{Binding: "label"},
			&ir.Element{Tag: "slot"},
			&ir.Element{Tag: "slot", Props: []ir.Prop{{Name: "name", Value: "meta"}}, Children: []ir.Node{&ir.Text{Value: "fallback"}}},
		}},
	}}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance, ok := root.Children[0].(*ir.ComponentInstance)
	if !ok {
		t.Fatalf("expected component instance, got %T", root.Children[0])
	}
	if instance.Alias != "Child" || len(instance.Nodes) != 1 {
		t.Fatalf("instance = %+v", instance)
	}
	if len(instance.Props) != 3 || instance.Props[0].Binding != "title" {
		t.Fatalf("props = %+v", instance.Props)
	}
	if len(instance.Events) != 1 || instance.Events[0].Method != "Save" {
		t.Fatalf("events = %+v", instance.Events)
	}
	childButton := instance.Nodes[0].(*ir.Element)
	if propValue(childButton.Props, "class") != "child-class parent-class" {
		t.Fatalf("root class fallthrough = %+v", childButton.Props)
	}
	if propValue(childButton.Props, "id") != "child-id" {
		t.Fatalf("root id fallthrough = %+v", childButton.Props)
	}
	if childButton.Events[0].Method != "Save" {
		t.Fatalf("root events = %+v", childButton.Events)
	}
	if interp := childButton.Children[0].(*ir.Interpolation); interp.Binding != "title" {
		t.Fatalf("prop interpolation = %+v", interp)
	}
	if childButton.Children[1].(*ir.Element).Children[0].(*ir.Text).Value != "default" {
		t.Fatalf("default slot = %+v", childButton.Children[1])
	}
	if childButton.Children[2].(*ir.Element).Children[0].(*ir.Text).Value != "named" {
		t.Fatalf("named slot = %+v", childButton.Children[2])
	}
}

func TestBuildKebabCaseComponentFromImportAlias(t *testing.T) {
	templateSource := `<div><plain-card></plain-card></div>`
	scriptSource := `
import PlainCard "./PlainCard.vue"
type State struct {}
`
	child := &ir.Component{Name: "PlainCard", Nodes: []ir.Node{&ir.Element{Tag: "p", Children: []ir.Node{&ir.Text{Value: "card"}}}}}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "PlainCard", Path: "./PlainCard.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance, ok := root.Children[0].(*ir.ComponentInstance)
	if !ok {
		t.Fatalf("expected component instance, got %T", root.Children[0])
	}
	if instance.Alias != "PlainCard" {
		t.Fatalf("alias = %q", instance.Alias)
	}
}

func TestBuildComponentEmitMapsChildMethodToParentListener(t *testing.T) {
	templateSource := `<div><Child @select="Save"></Child></div>`
	scriptSource := `
import Child "./Child.vue"

type State struct {}

func (s *State) Save() {}
`
	child := &ir.Component{
		Name:      "Child",
		Emits:     []ir.Emit{{Method: "Pressed", Event: "select"}},
		Lifecycle: []ir.Lifecycle{{Hook: "mounted", Method: "Mounted"}},
		Nodes: []ir.Node{&ir.Element{
			Tag:    "button",
			Events: []ir.EventHandler{{Event: "click", Method: "Pressed"}},
		}},
	}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	button := instance.Nodes[0].(*ir.Element)
	if len(button.Events) != 1 {
		t.Fatalf("events = %+v", button.Events)
	}
	if button.Events[0].Event != "click" || button.Events[0].Method != "Pressed" {
		t.Fatalf("child event = %+v", button.Events)
	}
	if len(instance.Lifecycle) != 1 || instance.Lifecycle[0].Hook != "mounted" {
		t.Fatalf("component lifecycle = %+v", instance.Lifecycle)
	}
}

func TestBuildComponentExposesStateAliasesAsPropNames(t *testing.T) {
	component := build("Badge", `<button>{{ label }}</button>`, `
type State struct {
    Label signal.String `+"`vugra:\"label\"`"+`
    Optional signal.String `+"`vugra:\"optional,optional,default=Fallback\"`"+`
}

func (s *State) Mounted() {}
`)
	if len(component.PropNames) != 2 || component.PropNames[0] != "label" || component.PropNames[1] != "optional" {
		t.Fatalf("prop names = %+v", component.PropNames)
	}
	if len(component.Props) != 2 || component.Props[0].Name != "label" || component.Props[0].GoField != "Label" || component.Props[0].Type != "signal.String" || !component.Props[0].Required {
		t.Fatalf("prop defs = %+v", component.Props)
	}
	if component.Props[1].Name != "optional" || component.Props[1].Required || component.Props[1].Default != "Fallback" {
		t.Fatalf("optional prop def = %+v", component.Props[1])
	}
	if len(component.Lifecycle) != 1 || component.Lifecycle[0].Hook != "mounted" || component.Lifecycle[0].Method != "Mounted" {
		t.Fatalf("lifecycle = %+v", component.Lifecycle)
	}
}

func TestBuildComponentModelDirective(t *testing.T) {
	templateSource := `<div><Editor v-model:title="title"></Editor></div>`
	scriptSource := `
import Editor "./Editor.vue"

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}
`
	child := &ir.Component{Name: "Editor", PropNames: []string{"title"}, Nodes: []ir.Node{
		&ir.Element{Tag: "input", Props: []ir.Prop{{Name: "value", Binding: "title", Bound: true}}},
	}}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Editor", Path: "./Editor.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if len(instance.Props) != 1 || instance.Props[0].Name != "title" || instance.Props[0].Binding != "title" {
		t.Fatalf("model props = %+v", instance.Props)
	}
	if len(instance.Events) != 1 || instance.Events[0].Event != "update:title" || instance.Events[0].Method != "title" {
		t.Fatalf("model events = %+v", instance.Events)
	}
	input := instance.Nodes[0].(*ir.Element)
	if len(input.Props) != 1 || input.Props[0].Name != "value" || input.Props[0].Binding != "title" {
		t.Fatalf("model value replacement = %+v", input.Props)
	}
}

func TestBuildScopedSlotProps(t *testing.T) {
	templateSource := `<div><List :label="title"><template #item="item"><span>{{ item.label }}</span></template></List></div>`
	scriptSource := `
import List "./List.vue"

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}
`
	child := &ir.Component{Name: "List", PropNames: []string{"label"}, Nodes: []ir.Node{
		&ir.Element{Tag: "div", Children: []ir.Node{
			&ir.Element{Tag: "slot", Props: []ir.Prop{
				{Name: "name", Value: "item"},
				{Name: "label", Binding: "label", Bound: true},
			}},
		}},
	}}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "List", Path: "./List.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if len(instance.Slots) != 1 || instance.Slots[0].Name != "item" || instance.Slots[0].Scope != "item" {
		t.Fatalf("scoped slot metadata = %+v", instance.Slots)
	}
	listRoot := instance.Nodes[0].(*ir.Element)
	slotSpan := listRoot.Children[0].(*ir.Element)
	interp := slotSpan.Children[0].(*ir.Interpolation)
	if interp.Binding != "title" {
		t.Fatalf("scoped slot binding = %+v", interp)
	}
}

func TestBuildComponentDefaultScopedSlot(t *testing.T) {
	templateSource := `<div><List :label="title" v-slot="item"><span>{{ item.label }}</span></List></div>`
	scriptSource := `
import List "./List.vue"

type State struct {
    Title signal.String ` + "`vugra:\"title\"`" + `
}
`
	child := &ir.Component{Name: "List", PropNames: []string{"label"}, Nodes: []ir.Node{
		&ir.Element{Tag: "div", Children: []ir.Node{
			&ir.Element{Tag: "slot", Props: []ir.Prop{{Name: "label", Binding: "label", Bound: true}}},
		}},
	}}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "List", Path: "./List.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if len(instance.Slots) != 1 || instance.Slots[0].Name != "default" || instance.Slots[0].Scope != "item" {
		t.Fatalf("default scoped slot metadata = %+v", instance.Slots)
	}
	listRoot := instance.Nodes[0].(*ir.Element)
	slotSpan := listRoot.Children[0].(*ir.Element)
	interp := slotSpan.Children[0].(*ir.Interpolation)
	if interp.Binding != "title" {
		t.Fatalf("default scoped slot binding = %+v", interp)
	}
}

func TestBuildComponentProvideInject(t *testing.T) {
	templateSource := `<div><Child></Child></div>`
	scriptSource := `
import Child "./Child.vue"

type State struct {
    Theme signal.String ` + "`vugra:\"theme,provide\"`" + `
}
`
	child := &ir.Component{
		Name:      "Child",
		PropNames: []string{"theme"},
		Props:     []ir.PropDef{{Name: "theme"}},
		Injects:   []ir.InjectDef{{Name: "theme", GoField: "Theme", Type: "signal.String"}},
		Nodes: []ir.Node{&ir.Element{Tag: "span", Children: []ir.Node{
			&ir.Interpolation{Binding: "theme"},
		}}},
	}
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	if len(component.Provides) != 1 || component.Provides[0].Name != "theme" {
		t.Fatalf("provides = %+v", component.Provides)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if len(instance.Props) != 1 || instance.Props[0].Name != "theme" || instance.Props[0].Binding != "theme" {
		t.Fatalf("injected props = %+v", instance.Props)
	}
	childRoot := instance.Nodes[0].(*ir.Element)
	interp := childRoot.Children[0].(*ir.Interpolation)
	if interp.Binding != "theme" {
		t.Fatalf("injected binding = %+v", interp)
	}
}

func TestBuildUnknownComponentDiagnostics(t *testing.T) {
	component := build("Parent", `<div><MissingWidget /></div>`, `type State struct{}`)
	if !containsDiag(component.Diagnostics, "ir.unknown_component") {
		t.Fatalf("missing unknown component diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildDynamicComponent(t *testing.T) {
	templateDoc := template.Parse(`<div><component :is="current"></component></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Card "./Card.vue"
import Badge "./Badge.vue"

type State struct {
    Current signal.String `+"`vugra:\"current\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports: []ir.Import{
			{Alias: "Card", Path: "./Card.vue", Component: &ir.Component{Name: "Card", Nodes: []ir.Node{&ir.Element{Tag: "p", Children: []ir.Node{&ir.Text{Value: "card"}}}}}},
			{Alias: "Badge", Path: "./Badge.vue", Component: &ir.Component{Name: "Badge", Nodes: []ir.Node{&ir.Element{Tag: "span", Children: []ir.Node{&ir.Text{Value: "badge"}}}}}},
		},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	dynamic, ok := root.Children[0].(*ir.DynamicComponent)
	if !ok {
		t.Fatalf("expected dynamic component, got %T", root.Children[0])
	}
	if dynamic.Binding != "current" || len(dynamic.Cases) != 2 {
		t.Fatalf("dynamic component = %+v", dynamic)
	}
}

func TestBuildAsyncComponentDiagnostic(t *testing.T) {
	templateDoc := template.Parse(`<div><Child async></Child><component :is="current" async></component></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct {
    Current signal.String `+"`vugra:\"current\"`"+`
}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: &ir.Component{Name: "Child", Nodes: []ir.Node{&ir.Element{Tag: "span"}}}}},
	})
	if !containsDiag(component.Diagnostics, "ir.unsupported_async_component") {
		t.Fatalf("missing async component diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildUnknownComponentPropDiagnostics(t *testing.T) {
	templateDoc := template.Parse(`<div><Child typo="value" class="ok"></Child></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct{}
`, goanalysis.BasePosition{})
	child := &ir.Component{Name: "Child", PropNames: []string{"label"}, Props: []ir.PropDef{{Name: "label"}}, Nodes: []ir.Node{&ir.Element{Tag: "span"}}}
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if !containsDiag(component.Diagnostics, "ir.unknown_component_prop") {
		t.Fatalf("missing unknown component prop diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildMissingRequiredComponentPropDiagnostics(t *testing.T) {
	templateDoc := template.Parse(`<div><Child></Child></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct{}
`, goanalysis.BasePosition{})
	child := &ir.Component{Name: "Child", PropNames: []string{"label"}, Props: []ir.PropDef{{Name: "label", Required: true}}, Nodes: []ir.Node{&ir.Element{Tag: "span"}}}
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if !containsDiag(component.Diagnostics, "ir.missing_required_component_prop") {
		t.Fatalf("missing required component prop diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildOptionalComponentPropDoesNotRequireValue(t *testing.T) {
	templateDoc := template.Parse(`<div><Child></Child></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct{}
`, goanalysis.BasePosition{})
	child := &ir.Component{Name: "Child", PropNames: []string{"label"}, Props: []ir.PropDef{{Name: "label", Default: "Fallback"}}, Nodes: []ir.Node{&ir.Element{Tag: "span", Children: []ir.Node{&ir.Interpolation{Binding: "label"}}}}}
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if containsDiag(component.Diagnostics, "ir.missing_required_component_prop") {
		t.Fatalf("unexpected required prop diagnostic: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if len(instance.Props) != 1 || instance.Props[0].Name != "label" || instance.Props[0].Value != "Fallback" {
		t.Fatalf("default props = %+v", instance.Props)
	}
	childRoot := instance.Nodes[0].(*ir.Element)
	if childRoot.Children[0].(*ir.Text).Value != "Fallback" {
		t.Fatalf("default prop text = %+v", childRoot.Children)
	}
}

func TestBuildUnboundComponentStateDiagnostic(t *testing.T) {
	templateDoc := template.Parse(`<div><Child></Child></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct{}
`, goanalysis.BasePosition{})
	child := &ir.Component{
		Name:      "Child",
		PropNames: []string{"internal"},
		Props:     []ir.PropDef{{Name: "internal"}},
		Nodes: []ir.Node{&ir.Element{Tag: "span", Children: []ir.Node{
			&ir.Interpolation{Binding: "internal"},
		}}},
	}
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if !containsDiag(component.Diagnostics, "ir.unbound_component_state") {
		t.Fatalf("missing unbound component state diagnostic: %+v", component.Diagnostics)
	}
}

func TestBuildUnresolvedImportedComponentDiagnostics(t *testing.T) {
	templateDoc := template.Parse(`<div><Child /></div>`, 0)
	goMeta := goanalysis.Analyze(`
import Child "./Child.vue"
type State struct{}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: templateDoc,
		Go:       goMeta,
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue"}},
	})
	if !containsDiag(component.Diagnostics, "ir.unresolved_component_import") {
		t.Fatalf("missing unresolved component diagnostic: %+v", component.Diagnostics)
	}
}

func build(name, templateSource, scriptSource string) *ir.Component {
	templateDoc := template.Parse(templateSource, 0)
	goMeta := goanalysis.Analyze(scriptSource, goanalysis.BasePosition{Offset: 1000, Line: 30, Column: 1})
	return ir.Build(ir.BuildInput{Name: name, Template: templateDoc, Go: goMeta})
}

func TestBuildCanonicalizesComponentPropNamesAfterHTMLLowercase(t *testing.T) {
	templateSource := `<div><Child :selectedsummary="selectedSummary"></Child></div>`
	scriptSource := `
import Child "./Child.vue"

type State struct {
    SelectedSummary signal.String ` + "`vugra:\"selectedSummary\"`" + `
}
`
	child := &ir.Component{
		Name:      "Child",
		PropNames: []string{"selectedSummary"},
		Props:     []ir.PropDef{{Name: "selectedSummary", Required: true}},
		Nodes:     []ir.Node{&ir.Element{Tag: "p", Children: []ir.Node{&ir.Interpolation{Binding: "selectedSummary"}}}},
	}
	component := ir.Build(ir.BuildInput{
		Name:     "Parent",
		Template: template.Parse(templateSource, 0),
		Go:       goanalysis.Analyze(scriptSource, goanalysis.BasePosition{}),
		Imports:  []ir.Import{{Alias: "Child", Path: "./Child.vue", Component: child}},
	})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	root := component.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	if instance.Props[0].Name != "selectedSummary" || instance.Props[0].Binding != "selectedSummary" {
		t.Fatalf("props = %+v", instance.Props)
	}
}

func renderComponent(component *ir.Component) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "name=%s\n", component.Name)
	fmt.Fprintf(&b, "diagnostics=%d\n", len(component.Diagnostics))
	for i, diag := range component.Diagnostics {
		fmt.Fprintf(&b, "diagnostic[%d]=%s %s %s %s\n", i, diag.Code, diag.Severity, formatSpan(diag.Span), strconv.Quote(diag.Message))
	}
	fmt.Fprintf(&b, "nodes=%d\n", len(component.Nodes))
	for i, node := range component.Nodes {
		renderNode(&b, fmt.Sprintf("node[%d]", i), node)
	}
	return []byte(b.String())
}

func renderNode(b *strings.Builder, prefix string, node ir.Node) {
	switch n := node.(type) {
	case *ir.Element:
		fmt.Fprintf(b, "%s.kind=element tag=%s span=%s props=%d events=%d children=%d\n", prefix, n.Tag, formatSpan(n.Span), len(n.Props), len(n.Events), len(n.Children))
		for i, prop := range n.Props {
			fmt.Fprintf(b, "%s.prop[%d]=%s value=%s binding=%s bound=%t span=%s\n", prefix, i, prop.Name, strconv.Quote(prop.Value), prop.Binding, prop.Bound, formatSpan(prop.Span))
		}
		for i, event := range n.Events {
			fmt.Fprintf(b, "%s.event[%d]=%s method=%s span=%s\n", prefix, i, event.Event, event.Method, formatSpan(event.Span))
		}
		for i, child := range n.Children {
			renderNode(b, fmt.Sprintf("%s.child[%d]", prefix, i), child)
		}
	case *ir.Text:
		fmt.Fprintf(b, "%s.kind=text span=%s value=%s\n", prefix, formatSpan(n.Span), strconv.Quote(n.Value))
	case *ir.Interpolation:
		fmt.Fprintf(b, "%s.kind=interpolation binding=%s goField=%s span=%s\n", prefix, n.Binding, n.GoField, formatSpan(n.Span))
	case *ir.Conditional:
		fmt.Fprintf(b, "%s.kind=conditional expr=%s span=%s\n", prefix, n.Expression, formatSpan(n.Span))
		renderNode(b, prefix+".child", n.Child)
	case *ir.Repeater:
		fmt.Fprintf(b, "%s.kind=repeater expr=%s span=%s\n", prefix, n.Expression, formatSpan(n.Span))
		renderNode(b, prefix+".child", n.Child)
	case *ir.ComponentInstance:
		fmt.Fprintf(b, "%s.kind=component alias=%s span=%s children=%d\n", prefix, n.Alias, formatSpan(n.Span), len(n.Nodes))
		for i, child := range n.Nodes {
			renderNode(b, fmt.Sprintf("%s.child[%d]", prefix, i), child)
		}
	case *ir.DynamicComponent:
		fmt.Fprintf(b, "%s.kind=dynamicComponent binding=%s span=%s cases=%d\n", prefix, n.Binding, formatSpan(n.Span), len(n.Cases))
		for i, candidate := range n.Cases {
			fmt.Fprintf(b, "%s.case[%d]=%s children=%d\n", prefix, i, candidate.Alias, len(candidate.Nodes))
			for j, child := range candidate.Nodes {
				renderNode(b, fmt.Sprintf("%s.case[%d].child[%d]", prefix, i, j), child)
			}
		}
	default:
		fmt.Fprintf(b, "%s.kind=unknown %T\n", prefix, node)
	}
}

func formatSpan(span ir.Span) string {
	return formatPosition(span.Start) + "-" + formatPosition(span.End)
}

func formatPosition(pos ir.Position) string {
	return fmt.Sprintf("%d:%d@%d", pos.Line, pos.Column, pos.Offset)
}

func containsDiag(diags []ir.Diagnostic, code string) bool {
	for _, diag := range diags {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func propValue(props []ir.Prop, name string) string {
	for _, prop := range props {
		if prop.Name == name {
			return prop.Value
		}
	}
	return ""
}
