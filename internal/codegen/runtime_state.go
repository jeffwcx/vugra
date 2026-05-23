package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"
	"unicode"

	"github.com/vugra/vugra/internal/componentfile"
	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/style"
)

type RuntimeStateInput struct {
	PackageName string
	Script      string
	Metadata    goanalysis.Metadata
	Component   *ir.Component
	Styles      *style.Stylesheet
	Imports     []ImportedComponentInput
}

type ImportedComponentInput struct {
	Alias     string
	Path      string
	Script    string
	Metadata  goanalysis.Metadata
	Component *ir.Component
	Imports   []ImportedComponentInput
}

type SoftwareMainInput struct {
	PackageImport string
	OutputPath    string
}

type WasmMainInput struct {
	PackageImport string
	CanvasID      string
	Width         int
	Height        int
	Layout        string
	AssetBase     string
	State         string
	RefreshHook   string
}

const generatedPackagePrefix = "package component\n\n"

func GenerateRuntimeState(input RuntimeStateInput) string {
	packageName := input.PackageName
	if packageName == "" {
		packageName = "component"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", packageName)
	childDecls, childFactories, childImports := generatedChildren(input.Imports)
	script := stripPackageClause(stripVugraImports(strings.TrimSpace(input.Script)))
	requiredImports := append([]string{
		`"github.com/vugra/vugra/pkg/vugra"`,
	}, childImports...)
	fmt.Fprintln(&b, withImports(script, uniqueStrings(requiredImports)))
	if strings.TrimSpace(childDecls) != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, childDecls)
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "func NewRuntimeState(state *State) vugra.State {")
	fmt.Fprintln(&b, "\tif state == nil {")
	fmt.Fprintln(&b, "\t\tstate = &State{}")
	fmt.Fprintln(&b, "\t}")
	fmt.Fprintln(&b, "\treturn vugra.State{")
	fmt.Fprintln(&b, "\t\tSignals: map[string]vugra.Signal{")
	for _, field := range sortedFields(input.Metadata) {
		if !field.IsSignal {
			continue
		}
		fmt.Fprintf(&b, "\t\t\t%q: &state.%s,\n", field.Alias, field.Name)
	}
	fmt.Fprintln(&b, "\t\t},")
	fmt.Fprintln(&b, "\t\tMethods: map[string]func(){")
	for _, method := range sortedMethods(input.Metadata) {
		if method.EventArg {
			continue
		}
		fmt.Fprintf(&b, "\t\t\t%q: state.%s,\n", method.Name, method.Name)
	}
	fmt.Fprintln(&b, "\t\t},")
	fmt.Fprintln(&b, "\t\tEventMethods: map[string]func(vugra.Event){")
	for _, method := range sortedMethods(input.Metadata) {
		if !method.EventArg {
			continue
		}
		fmt.Fprintf(&b, "\t\t\t%q: state.%s,\n", method.Name, method.Name)
	}
	fmt.Fprintln(&b, "\t\t},")
	fmt.Fprintln(&b, "\t}")
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	writtenFactories := map[string]bool{}
	for _, child := range flattenImports(input.Imports) {
		typeName := componentStateTypeName(child.Alias)
		factoryName := childFactories[child.Component]
		if factoryName == "" || writtenFactories[factoryName] {
			continue
		}
		writtenFactories[factoryName] = true
		writeRuntimeStateFactory(&b, factoryName, typeName, child.Metadata)
		fmt.Fprintln(&b)
	}
	fmt.Fprintln(&b, "func Component() *vugra.Component {")
	fmt.Fprint(&b, "\treturn ")
	writeComponent(&b, input.Component, childFactories)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "func Styles() *vugra.Stylesheet {")
	if input.Styles == nil {
		fmt.Fprintln(&b, "\treturn nil")
	} else {
		fmt.Fprint(&b, "\treturn ")
		writeStylesheet(&b, input.Styles)
		fmt.Fprintln(&b)
	}
	fmt.Fprintln(&b, "}")
	return b.String()
}

func generatedChildren(imports []ImportedComponentInput) (string, map[*ir.Component]string, []string) {
	factories := map[*ir.Component]string{}
	var decls []string
	var importsOut []string
	seen := map[*ir.Component]bool{}
	factoryByKey := map[string]string{}
	for _, child := range flattenImports(imports) {
		if child.Component == nil || child.Script == "" {
			continue
		}
		key := importedComponentKey(child)
		if factory := factoryByKey[key]; factory != "" {
			factories[child.Component] = factory
			continue
		}
		if factories[child.Component] != "" || seen[child.Component] {
			continue
		}
		seen[child.Component] = true
		typeName := componentStateTypeName(child.Alias)
		factoryName := "New" + typeName + "RuntimeState"
		factories[child.Component] = factoryName
		factoryByKey[key] = factoryName
		decl := transformStateType(stripImports(child.Script), typeName)
		if strings.TrimSpace(decl) != "" {
			decls = append(decls, strings.TrimSpace(decl))
		}
		importsOut = append(importsOut, nonVugraImports(child.Script)...)
	}
	return strings.Join(decls, "\n\n"), factories, uniqueStrings(importsOut)
}

func importedComponentKey(child ImportedComponentInput) string {
	if child.Path != "" {
		return child.Path
	}
	if child.Component != nil && child.Component.Name != "" {
		return child.Component.Name
	}
	return child.Alias
}

func flattenImports(imports []ImportedComponentInput) []ImportedComponentInput {
	var out []ImportedComponentInput
	for _, child := range imports {
		out = append(out, child)
		out = append(out, flattenImports(child.Imports)...)
	}
	return out
}

func componentStateTypeName(alias string) string {
	name := exportedIdentifier(alias)
	if name == "" {
		name = "Component"
	}
	return name + "State"
}

func exportedIdentifier(value string) string {
	var b strings.Builder
	upperNext := true
	for _, r := range value {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			upperNext = true
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		if b.Len() == 0 && unicode.IsDigit(r) {
			b.WriteRune('C')
		}
		if upperNext {
			b.WriteRune(unicode.ToUpper(r))
			upperNext = false
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func GenerateSoftwareMain(input SoftwareMainInput) string {
	outputPath := input.OutputPath
	if outputPath == "" {
		outputPath = "vugra-output.png"
	}
	componentImport := input.PackageImport
	if componentImport == "" {
		componentImport = "component"
	}
	return fmt.Sprintf(`package main

import (
	"log"

	component %q

	"github.com/vugra/vugra/pkg/vugra"
)

func main() {
	target := vugra.NewSoftware(800, 600)
	vugra.Mount(component.Component(), component.NewRuntimeState(nil), target, vugra.Options{
		Styles:      component.Styles(),
		Constraints: vugra.Constraints{Width: 800, Height: 600},
		Measurer:    vugra.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	if err := target.SavePNG(%q); err != nil {
		log.Fatal(err)
	}
}
`, componentImport, outputPath)
}

func GenerateWasmMain(input WasmMainInput) string {
	componentImport := input.PackageImport
	if componentImport == "" {
		componentImport = "component"
	}
	canvasID := input.CanvasID
	if canvasID == "" {
		canvasID = "vugra-canvas"
	}
	width := input.Width
	if width <= 0 {
		width = 800
	}
	height := input.Height
	if height <= 0 {
		height = 600
	}
	layoutOption := ""
	if input.Layout == "css" {
		layoutOption = "\n\t\tLayout:      vugra.LayoutEngineCSS,"
	}
	assetBaseOption := ""
	if input.AssetBase != "" {
		assetBaseOption = fmt.Sprintf("\n\t\tAssetBase:   %q,", input.AssetBase)
	}
	state := strings.TrimSpace(input.State)
	if state == "" {
		state = "component.NewRuntimeState(nil)"
	}
	refreshHook := strings.TrimSpace(input.RefreshHook)
	return fmt.Sprintf(`//go:build js && wasm

package main

import (
	"syscall/js"

	component %q

	"github.com/vugra/vugra/pkg/vugra"
)

func main() {
	done := make(chan struct{})
	document := js.Global().Get("document")
	canvas := document.Call("getElementById", %q)
	canvasSize := func() (int, int) {
		width := 0
		if value := canvas.Get("clientWidth"); value.Truthy() {
			width = value.Int()
		}
		height := 0
		if value := canvas.Get("clientHeight"); value.Truthy() {
			height = value.Int()
		}
		if width <= 0 {
			width = %d
		}
		if height <= 0 {
			height = %d
		}
		return width, height
	}
	width, height := canvasSize()
	target := vugra.NewCanvasRenderer(canvas, width, height)
	measurer := vugra.NewCanvasMeasurer(canvas)
	mounted := vugra.Mount(component.Component(), %s, target, vugra.Options{
		Styles:      component.Styles(),
%s
		Constraints: vugra.Constraints{Width: float32(width), Height: float32(height)},
		Measurer:    measurer,
%s
	})
	mounted.Flush()
	syncAccessibility := func() {
		vugra.SyncAccessibility("vugra-a11y", mounted.LastFrame(), mounted.FocusedID())
	}
	syncAccessibility()
%s
	window := js.Global().Get("window")
	if window.Truthy() && window.Get("addEventListener").Truthy() {
		resize := js.FuncOf(func(this js.Value, args []js.Value) any {
			width, height := canvasSize()
			vugra.ResizeCanvasRenderer(target, width, height)
			mounted.Resize(float32(width), float32(height))
			mounted.Flush()
			syncAccessibility()
			return nil
		})
		window.Call("addEventListener", "resize", resize)
	}
	vugra.InstallAccessibilityEventHandlers("vugra-a11y", vugra.AccessibilityEvents{
		Click: func(x, y int) {
			if mounted.DispatchPointerEvent(float32(x), float32(y), vugra.Modifiers{}) {
				mounted.Flush()
				syncAccessibility()
			}
		},
		Focus: func(id string) {
			if mounted.FocusID(id) {
				mounted.Flush()
				syncAccessibility()
			}
		},
		Key: func(key string) {
			if mounted.DispatchKey(key) {
				mounted.Flush()
				syncAccessibility()
			}
		},
		Text: func(text string) {
			if mounted.DispatchTextInput(text) {
				mounted.Flush()
				syncAccessibility()
			}
		},
	})
	vugra.InstallPointerEventDetails(canvas, func(event string, x, y, deltaX, deltaY int, shift, ctrl, meta, alt bool) {
		modifiers := vugra.Modifiers{Shift: shift, Ctrl: ctrl, Meta: meta, Alt: alt}
		handled := false
		switch event {
		case "click":
			handled = mounted.DispatchPointerEvent(float32(x), float32(y), modifiers)
		case "dblclick":
			handled = mounted.DispatchDoubleClick(float32(x), float32(y), modifiers)
		case "contextmenu":
			handled = mounted.DispatchContextMenu(float32(x), float32(y), modifiers)
		case "hover":
			handled = mounted.DispatchHover(float32(x), float32(y), modifiers)
		case "drag":
			handled = mounted.DispatchDrag(float32(x), float32(y), float32(deltaX), float32(deltaY), modifiers)
		}
		if handled {
			mounted.Flush()
			syncAccessibility()
		}
	})
	vugra.InstallScrollEvents(canvas, func(x, y, deltaY int) {
		if mounted.DispatchScroll(float32(x), float32(y), float32(deltaY)) {
			mounted.Flush()
			syncAccessibility()
		}
	})
	vugra.InstallKeyboardEvents(func(key string) {
		if mounted.DispatchKey(key) {
			mounted.Flush()
			syncAccessibility()
		}
	})
	vugra.InstallTextEvents(func(text string) {
		if mounted.DispatchTextInput(text) {
			mounted.Flush()
			syncAccessibility()
		}
	})
	vugra.SetStatus("status", "Running")
	<-done
}
`, componentImport, canvasID, width, height, state, assetBaseOption, layoutOption, refreshHookSource(refreshHook))
}

func refreshHookSource(name string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf(`	refresh := js.FuncOf(func(this js.Value, args []js.Value) any {
		mounted.Flush()
		syncAccessibility()
		return nil
	})
	refreshTarget := js.Global().Get("window")
	if !refreshTarget.Truthy() {
		refreshTarget = js.Global()
	}
	refreshTarget.Set(%q, refresh)
`, name)
}

func withImports(script string, imports []string) string {
	var missing []string
	for _, importPath := range imports {
		if !strings.Contains(script, importPath) {
			missing = append(missing, importPath)
		}
	}
	if len(missing) == 0 {
		return script
	}
	fset := token.NewFileSet()
	wrapped := wrapScriptPackage(script)
	file, err := parser.ParseFile(fset, "script.go", wrapped, parser.ImportsOnly)
	if err != nil || len(file.Imports) == 0 {
		return fmt.Sprintf("import (\n%s\n)\n\n%s", importLines(missing), script)
	}
	offsetBase := len(wrapped) - len(script)
	decl := importDeclFor(file, file.Imports[0])
	if decl == nil || decl.Lparen.IsValid() {
		closeOffset := fset.Position(decl.Rparen).Offset - offsetBase
		if closeOffset < 0 || closeOffset > len(script) {
			return fmt.Sprintf("import (\n%s\n)\n\n%s", importLines(missing), script)
		}
		return script[:closeOffset] + "\n" + importLines(missing) + script[closeOffset:]
	}
	insertOffset := fset.Position(decl.Pos()).Offset - offsetBase
	if insertOffset < 0 || insertOffset > len(script) {
		return fmt.Sprintf("import (\n%s\n)\n\n%s", importLines(missing), script)
	}
	endOffset := fset.Position(decl.End()).Offset - offsetBase
	if endOffset < 0 || endOffset > len(script) {
		return fmt.Sprintf("import (\n%s\n)\n\n%s", importLines(missing), script)
	}
	existing := strings.TrimSpace(script[insertOffset:endOffset])
	existing = strings.TrimSpace(strings.TrimPrefix(existing, "import"))
	return script[:insertOffset] +
		"import (\n\t" + existing + "\n" + importLines(missing) + "\n)" +
		script[endOffset:]
}

func stripVugraImports(script string) string {
	fset := token.NewFileSet()
	wrapped := wrapScriptPackage(script)
	file, err := parser.ParseFile(fset, "script.go", wrapped, parser.ImportsOnly)
	if err != nil || len(file.Imports) == 0 {
		return script
	}
	offsetBase := len(wrapped) - len(script)
	type edit struct {
		start int
		end   int
		text  string
	}
	var edits []edit
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		var keep []string
		for _, rawSpec := range gen.Specs {
			spec, ok := rawSpec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path := strings.Trim(spec.Path.Value, `"`)
			if componentfile.IsComponentPath(path) {
				continue
			}
			keep = append(keep, strings.TrimSpace(wrapped[fset.Position(spec.Pos()).Offset:fset.Position(spec.End()).Offset]))
		}
		start := fset.Position(gen.Pos()).Offset - offsetBase
		end := fset.Position(gen.End()).Offset - offsetBase
		if start < 0 || end < start || end > len(script) {
			continue
		}
		text := ""
		if len(keep) == 1 {
			text = "import " + keep[0]
		} else if len(keep) > 1 {
			text = "import (\n\t" + strings.Join(keep, "\n\t") + "\n)"
		}
		edits = append(edits, edit{start: start, end: end, text: text})
	}
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		script = script[:e.start] + e.text + script[e.end:]
	}
	return strings.TrimSpace(script)
}

func stripImports(script string) string {
	fset := token.NewFileSet()
	wrapped := wrapScriptPackage(script)
	file, err := parser.ParseFile(fset, "script.go", wrapped, parser.ParseComments)
	if err != nil {
		return script
	}
	offsetBase := len(wrapped) - len(script)
	type edit struct {
		start int
		end   int
	}
	var edits []edit
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		start := fset.Position(gen.Pos()).Offset - offsetBase
		end := fset.Position(gen.End()).Offset - offsetBase
		if start >= 0 && end >= start && end <= len(script) {
			edits = append(edits, edit{start: start, end: end})
		}
	}
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		script = script[:e.start] + script[e.end:]
	}
	return strings.TrimSpace(script)
}

func nonVugraImports(script string) []string {
	fset := token.NewFileSet()
	wrapped := wrapScriptPackage(script)
	file, err := parser.ParseFile(fset, "script.go", wrapped, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	var out []string
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		if componentfile.IsComponentPath(path) {
			continue
		}
		out = append(out, strings.TrimSpace(wrapped[fset.Position(spec.Pos()).Offset:fset.Position(spec.End()).Offset]))
	}
	return out
}

func transformStateType(script, typeName string) string {
	if strings.TrimSpace(script) == "" {
		return ""
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "script.go", wrapScriptPackage(script), parser.ParseComments)
	if err != nil {
		return script
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			if n.Name == "State" {
				n.Name = typeName
			}
		case *ast.TypeSpec:
			if n.Name.Name == "State" {
				n.Name.Name = typeName
			}
		case *ast.FuncDecl:
			if n.Recv != nil {
				for _, field := range n.Recv.List {
					switch recv := field.Type.(type) {
					case *ast.StarExpr:
						if ident, ok := recv.X.(*ast.Ident); ok && ident.Name == "State" {
							ident.Name = typeName
						}
					case *ast.Ident:
						if recv.Name == "State" {
							recv.Name = typeName
						}
					}
				}
			}
		}
		return true
	})
	var b bytes.Buffer
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		if err := printer.Fprint(&b, fset, decl); err != nil {
			return script
		}
	}
	return b.String()
}

func wrapScriptPackage(script string) string {
	trimmed := strings.TrimSpace(script)
	if strings.HasPrefix(trimmed, "package ") {
		return script
	}
	return generatedPackagePrefix + script
}

func stripPackageClause(script string) string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "script.go", wrapScriptPackage(script), parser.ParseComments)
	if err != nil || file.Name == nil {
		return script
	}
	if !strings.HasPrefix(strings.TrimSpace(script), "package ") {
		return script
	}
	start := fset.Position(file.Package).Offset
	end := fset.Position(file.Name.End()).Offset
	if start < 0 || end < start || end > len(script) {
		return script
	}
	return strings.TrimSpace(script[:start] + script[end:])
}

func writeRuntimeStateFactory(b *strings.Builder, factoryName, typeName string, metadata goanalysis.Metadata) {
	fields := sortedFields(metadata)
	methods := sortedMethods(metadata)
	fmt.Fprintf(b, "func %s() vugra.State {\n", factoryName)
	if runtimeStateUsesState(fields, methods) {
		fmt.Fprintf(b, "\tstate := &%s{}\n", typeName)
	}
	fmt.Fprintln(b, "\treturn vugra.State{")
	fmt.Fprintln(b, "\t\tSignals: map[string]vugra.Signal{")
	for _, field := range fields {
		if !field.IsSignal {
			continue
		}
		fmt.Fprintf(b, "\t\t\t%q: &state.%s,\n", field.Alias, field.Name)
	}
	fmt.Fprintln(b, "\t\t},")
	fmt.Fprintln(b, "\t\tMethods: map[string]func(){")
	for _, method := range methods {
		if method.EventArg {
			continue
		}
		fmt.Fprintf(b, "\t\t\t%q: state.%s,\n", method.Name, method.Name)
	}
	fmt.Fprintln(b, "\t\t},")
	fmt.Fprintln(b, "\t\tEventMethods: map[string]func(vugra.Event){")
	for _, method := range methods {
		if !method.EventArg {
			continue
		}
		fmt.Fprintf(b, "\t\t\t%q: state.%s,\n", method.Name, method.Name)
	}
	fmt.Fprintln(b, "\t\t},")
	fmt.Fprintln(b, "\t}")
	fmt.Fprintln(b, "}")
}

func runtimeStateUsesState(fields []goanalysis.Field, methods []goanalysis.Method) bool {
	for _, field := range fields {
		if field.IsSignal {
			return true
		}
	}
	return len(methods) > 0
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func importLines(imports []string) string {
	var b strings.Builder
	for _, importPath := range imports {
		fmt.Fprintf(&b, "\t%s\n", importPath)
	}
	return strings.TrimRight(b.String(), "\n")
}

func importDeclFor(file *ast.File, spec *ast.ImportSpec) *ast.GenDecl {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, candidate := range gen.Specs {
			if candidate == spec {
				return gen
			}
		}
	}
	return nil
}

func writeComponent(b *strings.Builder, component *ir.Component, factories ...map[*ir.Component]string) {
	factoryMap := map[*ir.Component]string{}
	if len(factories) > 0 && factories[0] != nil {
		factoryMap = factories[0]
	}
	writeComponentWithFactories(b, component, map[*ir.Component]struct{}{}, factoryMap)
}

func writeComponentWithFactories(b *strings.Builder, component *ir.Component, seen map[*ir.Component]struct{}, factories map[*ir.Component]string) {
	if component == nil {
		b.WriteString("&vugra.Component{}")
		return
	}
	if _, ok := seen[component]; ok {
		fmt.Fprintf(b, "&vugra.Component{Name: %q}", component.Name)
		return
	}
	seen[component] = struct{}{}
	fmt.Fprintf(b, "&vugra.Component{Name: %q", component.Name)
	if factory := factories[component]; factory != "" {
		fmt.Fprintf(b, ", NewState: %s", factory)
	}
	if len(component.PropNames) > 0 {
		b.WriteString(", PropNames: []string{")
		for _, name := range component.PropNames {
			fmt.Fprintf(b, "%q, ", name)
		}
		b.WriteString("}")
	}
	if len(component.Props) > 0 {
		b.WriteString(", Props: []vugra.PropDef{")
		for _, prop := range component.Props {
			fmt.Fprintf(b, "{Name: %q", prop.Name)
			if prop.GoField != "" {
				fmt.Fprintf(b, ", GoField: %q", prop.GoField)
			}
			if prop.Type != "" {
				fmt.Fprintf(b, ", Type: %q", prop.Type)
			}
			if prop.Required {
				b.WriteString(", Required: true")
			}
			if prop.Default != "" {
				fmt.Fprintf(b, ", Default: %q", prop.Default)
			}
			b.WriteString("}, ")
		}
		b.WriteString("}")
	}
	if len(component.Provides) > 0 {
		b.WriteString(", Provides: []vugra.ProvideDef{")
		for _, provide := range component.Provides {
			fmt.Fprintf(b, "{Name: %q, Binding: %q}, ", provide.Name, provide.Binding)
		}
		b.WriteString("}")
	}
	if len(component.Injects) > 0 {
		b.WriteString(", Injects: []vugra.InjectDef{")
		for _, inject := range component.Injects {
			fmt.Fprintf(b, "{Name: %q", inject.Name)
			if inject.GoField != "" {
				fmt.Fprintf(b, ", GoField: %q", inject.GoField)
			}
			if inject.Type != "" {
				fmt.Fprintf(b, ", Type: %q", inject.Type)
			}
			if inject.Default != "" {
				fmt.Fprintf(b, ", Default: %q", inject.Default)
			}
			b.WriteString("}, ")
		}
		b.WriteString("}")
	}
	if len(component.Emits) > 0 {
		b.WriteString(", Emits: []vugra.EmitDef{")
		for _, emit := range component.Emits {
			fmt.Fprintf(b, "{Method: %q, Event: %q}, ", emit.Method, emit.Event)
		}
		b.WriteString("}")
	}
	if len(component.Lifecycle) > 0 {
		b.WriteString(", Lifecycle: []vugra.Lifecycle{")
		for _, lifecycle := range component.Lifecycle {
			fmt.Fprintf(b, "{Hook: %q, Method: %q}, ", lifecycle.Hook, lifecycle.Method)
		}
		b.WriteString("}")
	}
	if len(component.Nodes) > 0 {
		b.WriteString(", Nodes: []vugra.Node{")
		for _, node := range component.Nodes {
			writeNodeWithFactories(b, node, seen, factories)
			b.WriteString(", ")
		}
		b.WriteString("}")
	}
	b.WriteString("}")
}

func writeNode(b *strings.Builder, node ir.Node) {
	writeNodeWithFactories(b, node, map[*ir.Component]struct{}{}, nil)
}

func writeNodeWithFactories(b *strings.Builder, node ir.Node, seen map[*ir.Component]struct{}, factories map[*ir.Component]string) {
	switch n := node.(type) {
	case *ir.Element:
		b.WriteString("&vugra.Element{")
		fmt.Fprintf(b, "Tag: %q", n.Tag)
		if len(n.Props) > 0 {
			b.WriteString(", Props: []vugra.Prop{")
			for _, prop := range n.Props {
				writeProp(b, prop)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Events) > 0 {
			b.WriteString(", Events: []vugra.EventHandler{")
			for _, event := range n.Events {
				writeEvent(b, event)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Children) > 0 {
			b.WriteString(", Children: []vugra.Node{")
			for _, child := range n.Children {
				writeNodeWithFactories(b, child, seen, factories)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		b.WriteString("}")
	case *ir.Text:
		fmt.Fprintf(b, "&vugra.Text{Value: %q}", n.Value)
	case *ir.Interpolation:
		fmt.Fprintf(b, "&vugra.Interpolation{Binding: %q, GoField: %q}", n.Binding, n.GoField)
	case *ir.Conditional:
		fmt.Fprintf(b, "&vugra.Conditional{Expression: %q, Child: ", n.Expression)
		writeNodeWithFactories(b, n.Child, seen, factories)
		b.WriteString("}")
	case *ir.Repeater:
		fmt.Fprintf(b, "&vugra.Repeater{Expression: %q, Child: ", n.Expression)
		writeNodeWithFactories(b, n.Child, seen, factories)
		b.WriteString("}")
	case *ir.ComponentInstance:
		fmt.Fprintf(b, "&vugra.ComponentInstance{Alias: %q", n.Alias)
		if n.Component != nil {
			b.WriteString(", Component: ")
			writeComponentWithFactories(b, n.Component, seen, factories)
		}
		if len(n.Props) > 0 {
			b.WriteString(", Props: []vugra.Prop{")
			for _, prop := range n.Props {
				writeProp(b, prop)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Events) > 0 {
			b.WriteString(", Events: []vugra.EventHandler{")
			for _, event := range n.Events {
				writeEvent(b, event)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Slots) > 0 {
			b.WriteString(", Slots: []vugra.Slot{")
			for _, slot := range n.Slots {
				fmt.Fprintf(b, "{Name: %q", slot.Name)
				if slot.Scope != "" {
					fmt.Fprintf(b, ", Scope: %q", slot.Scope)
				}
				if len(slot.Nodes) > 0 {
					b.WriteString(", Nodes: []vugra.Node{")
					for _, child := range slot.Nodes {
						writeNodeWithFactories(b, child, seen, factories)
						b.WriteString(", ")
					}
					b.WriteString("}")
				}
				b.WriteString("}, ")
			}
			b.WriteString("}")
		}
		if len(n.Lifecycle) > 0 {
			b.WriteString(", Lifecycle: []vugra.Lifecycle{")
			for _, lifecycle := range n.Lifecycle {
				fmt.Fprintf(b, "{Hook: %q, Method: %q}, ", lifecycle.Hook, lifecycle.Method)
			}
			b.WriteString("}")
		}
		if len(n.Nodes) > 0 {
			b.WriteString(", Nodes: []vugra.Node{")
			for _, child := range n.Nodes {
				writeNodeWithFactories(b, child, seen, factories)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		b.WriteString("}")
	case *ir.DynamicComponent:
		fmt.Fprintf(b, "&vugra.DynamicComponent{Binding: %q", n.Binding)
		if len(n.Props) > 0 {
			b.WriteString(", Props: []vugra.Prop{")
			for _, prop := range n.Props {
				writeProp(b, prop)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Events) > 0 {
			b.WriteString(", Events: []vugra.EventHandler{")
			for _, event := range n.Events {
				writeEvent(b, event)
				b.WriteString(", ")
			}
			b.WriteString("}")
		}
		if len(n.Slots) > 0 {
			b.WriteString(", Slots: []vugra.Slot{")
			for _, slot := range n.Slots {
				fmt.Fprintf(b, "{Name: %q", slot.Name)
				if slot.Scope != "" {
					fmt.Fprintf(b, ", Scope: %q", slot.Scope)
				}
				if len(slot.Nodes) > 0 {
					b.WriteString(", Nodes: []vugra.Node{")
					for _, child := range slot.Nodes {
						writeNodeWithFactories(b, child, seen, factories)
						b.WriteString(", ")
					}
					b.WriteString("}")
				}
				b.WriteString("}, ")
			}
			b.WriteString("}")
		}
		if len(n.Cases) > 0 {
			b.WriteString(", Cases: []vugra.DynamicComponentCase{")
			for _, candidate := range n.Cases {
				fmt.Fprintf(b, "{Alias: %q", candidate.Alias)
				if candidate.Component != nil {
					b.WriteString(", Component: ")
					writeComponentWithFactories(b, candidate.Component, seen, factories)
				}
				if len(candidate.Nodes) > 0 {
					b.WriteString(", Nodes: []vugra.Node{")
					for _, child := range candidate.Nodes {
						writeNodeWithFactories(b, child, seen, factories)
						b.WriteString(", ")
					}
					b.WriteString("}")
				}
				b.WriteString("}, ")
			}
			b.WriteString("}")
		}
		b.WriteString("}")
	}
}

func writeProp(b *strings.Builder, prop ir.Prop) {
	fmt.Fprintf(b, "{Name: %q", prop.Name)
	if prop.Value != "" {
		fmt.Fprintf(b, ", Value: %q", prop.Value)
	}
	if prop.Binding != "" {
		fmt.Fprintf(b, ", Binding: %q", prop.Binding)
	}
	if prop.Bound {
		b.WriteString(", Bound: true")
	}
	b.WriteString("}")
}

func writeEvent(b *strings.Builder, event ir.EventHandler) {
	fmt.Fprintf(b, "{Event: %q, Method: %q}", event.Event, event.Method)
}

func writeStylesheet(b *strings.Builder, sheet *style.Stylesheet) {
	b.WriteString("&vugra.Stylesheet{")
	if len(sheet.Rules) > 0 {
		b.WriteString("Rules: []vugra.Rule{")
		for _, rule := range sheet.Rules {
			writeStyleRule(b, rule)
			b.WriteString(", ")
		}
		b.WriteString("}")
	}
	b.WriteString("}")
}

func writeStyleRule(b *strings.Builder, rule style.Rule) {
	fmt.Fprintf(b, "{Selector: %q, ClassName: %q", rule.Selector, rule.ClassName)
	if len(rule.Declarations) > 0 {
		b.WriteString(", Declarations: []vugra.Declaration{")
		for _, decl := range rule.Declarations {
			fmt.Fprintf(b, "{Name: %q, Value: %q}, ", decl.Name, decl.Value)
		}
		b.WriteString("}")
	}
	b.WriteString("}")
}

func sortedFields(metadata goanalysis.Metadata) []goanalysis.Field {
	if metadata.State == nil {
		return nil
	}
	fields := append([]goanalysis.Field(nil), metadata.State.Fields...)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Alias < fields[j].Alias
	})
	return fields
}

func sortedMethods(metadata goanalysis.Metadata) []goanalysis.Method {
	methods := append([]goanalysis.Method(nil), metadata.Methods...)
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})
	return methods
}
