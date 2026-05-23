package goanalysis

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/vugra/vugra/internal/componentfile"
)

type BasePosition struct {
	Offset int
	Line   int
	Column int
}

func Analyze(source string, base BasePosition) Metadata {
	if base.Line == 0 {
		base.Line = 1
	}
	if base.Column == 0 {
		base.Column = 1
	}

	wrapped, wrapperLen := wrapSource(source)
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "Component.script.go", wrapped, parser.AllErrors)
	a := analyzer{
		fset:       fset,
		source:     source,
		base:       base,
		wrapperLen: wrapperLen,
		mapper:     newSourceMap(source, base),
	}
	if err != nil {
		a.addParseDiagnostics(err)
	}
	if astFile != nil {
		a.inspect(astFile)
	}
	return a.metadata
}

type analyzer struct {
	fset       *token.FileSet
	source     string
	base       BasePosition
	wrapperLen int
	mapper     sourceMap
	metadata   Metadata
}

func (a *analyzer) inspect(file *ast.File) {
	a.inspectImports(file)
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			a.inspectGenDecl(d)
		case *ast.FuncDecl:
			a.inspectFuncDecl(d)
		}
	}
	sort.Slice(a.metadata.Methods, func(i, j int) bool {
		return a.metadata.Methods[i].Name < a.metadata.Methods[j].Name
	})
	sort.Slice(a.metadata.Imports, func(i, j int) bool {
		return a.metadata.Imports[i].Alias < a.metadata.Imports[j].Alias
	})
	if a.metadata.State == nil {
		a.error("goanalysis.missing_state", "missing State struct", Span{
			Start: a.mapper.position(0),
			End:   a.mapper.position(0),
		})
	}
}

func (a *analyzer) inspectImports(file *ast.File) {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !componentfile.IsComponentPath(path) {
			continue
		}
		alias := ""
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		if alias == "" || alias == "." || alias == "_" {
			alias = importAlias(path)
		}
		a.metadata.Imports = append(a.metadata.Imports, Import{
			Alias: alias,
			Path:  path,
			Span:  a.span(spec.Pos(), spec.End()),
		})
	}
}

func (a *analyzer) inspectGenDecl(decl *ast.GenDecl) {
	if decl.Tok != token.TYPE {
		return
	}
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || typeSpec.Name == nil || typeSpec.Name.Name != "State" {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			a.error("goanalysis.state_not_struct", "State must be a struct type", a.span(typeSpec.Pos(), typeSpec.End()))
			continue
		}
		state := &StateMetadata{TypeName: "State"}
		for _, field := range structType.Fields.List {
			state.Fields = append(state.Fields, a.fieldsFromAST(field)...)
		}
		a.metadata.State = state
	}
}

func (a *analyzer) fieldsFromAST(field *ast.Field) []Field {
	var fields []Field
	typ := a.exprString(field.Type)
	tag := vugraTag(field.Tag)
	typeSpan := a.span(field.Type.Pos(), field.Type.End())
	for _, name := range field.Names {
		fieldAlias := tag.Alias
		if fieldAlias == "" {
			fieldAlias = lowerFirst(name.Name)
		}
		fields = append(fields, Field{
			Name:        name.Name,
			Alias:       fieldAlias,
			Type:        typ,
			IsSignal:    isSignalType(typ),
			Span:        a.span(name.Pos(), field.End()),
			AliasSpan:   a.span(name.Pos(), name.End()),
			TypeSpan:    typeSpan,
			Exported:    name.IsExported(),
			HasVugraTag: tag.Present,
			Optional:    tag.Optional,
			Default:     tag.Default,
			Provide:     tag.Provide,
			Inject:      tag.Inject,
		})
	}
	return fields
}

func (a *analyzer) inspectFuncDecl(decl *ast.FuncDecl) {
	if decl.Recv == nil || decl.Name == nil || !decl.Name.IsExported() {
		return
	}
	receiver, onPointer := receiverString(decl.Recv)
	if receiver != "*State" {
		return
	}
	a.metadata.Methods = append(a.metadata.Methods, Method{
		Name:      decl.Name.Name,
		Receiver:  receiver,
		Span:      a.span(decl.Pos(), decl.End()),
		NameSpan:  a.span(decl.Name.Pos(), decl.Name.End()),
		Exported:  decl.Name.IsExported(),
		OnPointer: onPointer,
		EventArg:  a.hasEventArg(decl),
	})
	if hook := lifecycleHook(decl.Name.Name); hook != "" {
		a.metadata.Lifecycle = append(a.metadata.Lifecycle, Lifecycle{
			Hook:   hook,
			Method: decl.Name.Name,
			Span:   a.span(decl.Pos(), decl.End()),
		})
	}
	a.inspectEmits(decl)
}

func (a *analyzer) hasEventArg(decl *ast.FuncDecl) bool {
	if decl.Type == nil || decl.Type.Params == nil || len(decl.Type.Params.List) != 1 {
		return false
	}
	typ := a.exprString(decl.Type.Params.List[0].Type)
	return typ == "vugra.Event" || typ == "vuego.Event" || typ == "runtime.Event" || typ == "Event"
}

func lifecycleHook(method string) string {
	switch method {
	case "BeforeMount":
		return "beforeMount"
	case "Mounted":
		return "mounted"
	case "BeforeUpdate":
		return "beforeUpdate"
	case "Updated":
		return "updated"
	case "BeforeUnmount":
		return "beforeUnmount"
	case "Unmounted":
		return "unmounted"
	default:
		return ""
	}
}

func (a *analyzer) inspectEmits(decl *ast.FuncDecl) {
	if decl.Body == nil || decl.Name == nil {
		return
	}
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isEmitCall(call.Fun) || len(call.Args) == 0 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		event, err := strconv.Unquote(lit.Value)
		if err != nil || event == "" {
			return true
		}
		a.metadata.Emits = append(a.metadata.Emits, Emit{
			Method: decl.Name.Name,
			Event:  event,
			Span:   a.span(call.Pos(), call.End()),
		})
		return true
	})
}

func isEmitCall(expr ast.Expr) bool {
	switch n := expr.(type) {
	case *ast.SelectorExpr:
		return n.Sel != nil && n.Sel.Name == "Emit"
	case *ast.Ident:
		return n.Name == "Emit"
	default:
		return false
	}
}

func (a *analyzer) exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return a.exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + a.exprString(e.X)
	case *ast.ArrayType:
		return "[]" + a.exprString(e.Elt)
	case *ast.IndexExpr:
		return a.exprString(e.X) + "[" + a.exprString(e.Index) + "]"
	case *ast.IndexListExpr:
		var parts []string
		for _, idx := range e.Indices {
			parts = append(parts, a.exprString(idx))
		}
		return a.exprString(e.X) + "[" + strings.Join(parts, ", ") + "]"
	default:
		return strings.TrimSpace(a.sourceFor(e.Pos(), e.End()))
	}
}

func (a *analyzer) sourceFor(start, end token.Pos) string {
	startOff, ok1 := a.scriptOffset(start)
	endOff, ok2 := a.scriptOffset(end)
	if !ok1 || !ok2 || startOff < 0 || endOff > len(a.source) || startOff > endOff {
		return ""
	}
	return a.source[startOff:endOff]
}

func (a *analyzer) addParseDiagnostics(err error) {
	if list, ok := err.(scanner.ErrorList); ok {
		for _, item := range list {
			a.error("goanalysis.invalid_go", item.Msg, a.spanFromFileOffset(item.Pos.Offset, item.Pos.Offset))
		}
		return
	}
	a.error("goanalysis.invalid_go", err.Error(), Span{
		Start: a.mapper.position(0),
		End:   a.mapper.position(0),
	})
}

func (a *analyzer) error(code, message string, span Span) {
	a.metadata.Diagnostics = append(a.metadata.Diagnostics, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     span,
	})
}

func (a *analyzer) span(start, end token.Pos) Span {
	startOff, ok := a.scriptOffset(start)
	if !ok {
		startOff = 0
	}
	endOff, ok := a.scriptOffset(end)
	if !ok {
		endOff = startOff
	}
	if startOff < 0 {
		startOff = 0
	}
	if endOff < startOff {
		endOff = startOff
	}
	if endOff > len(a.source) {
		endOff = len(a.source)
	}
	return Span{
		Start: a.mapper.position(startOff),
		End:   a.mapper.position(endOff),
	}
}

func (a *analyzer) spanFromFileOffset(start, end int) Span {
	start -= a.wrapperLen
	end -= a.wrapperLen
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(a.source) {
		end = len(a.source)
	}
	return Span{
		Start: a.mapper.position(start),
		End:   a.mapper.position(end),
	}
}

func (a *analyzer) scriptOffset(pos token.Pos) (int, bool) {
	if !pos.IsValid() {
		return 0, false
	}
	position := a.fset.Position(pos)
	if position.Offset < a.wrapperLen {
		return 0, false
	}
	return position.Offset - a.wrapperLen, true
}

func wrapSource(source string) (string, int) {
	trimmed := strings.TrimLeftFunc(source, unicode.IsSpace)
	if strings.HasPrefix(trimmed, "package ") {
		return source, 0
	}
	prefix := "package vugra_component\n\n"
	return prefix + source, len(prefix)
}

type parsedVugraTag struct {
	Alias    string
	Present  bool
	Optional bool
	Default  string
	Provide  bool
	Inject   bool
}

func vugraTag(tag *ast.BasicLit) parsedVugraTag {
	if tag == nil {
		return parsedVugraTag{}
	}
	value, err := strconvUnquote(tag.Value)
	if err != nil {
		return parsedVugraTag{}
	}
	tagValue := reflect.StructTag(value).Get("vugra")
	if tagValue == "" {
		tagValue = reflect.StructTag(value).Get("vuego")
	}
	if tagValue == "" {
		return parsedVugraTag{}
	}
	out := parsedVugraTag{Present: true}
	parts := strings.Split(tagValue, ",")
	if len(parts) > 0 {
		out.Alias = strings.TrimSpace(parts[0])
	}
	for _, raw := range parts[1:] {
		part := strings.TrimSpace(raw)
		switch {
		case part == "optional":
			out.Optional = true
		case part == "provide":
			out.Provide = true
			out.Optional = true
		case part == "inject":
			out.Inject = true
			out.Optional = true
		case strings.HasPrefix(part, "default="):
			out.Default = strings.TrimPrefix(part, "default=")
			out.Optional = true
		}
	}
	return out
}

func strconvUnquote(value string) (string, error) {
	if len(value) >= 2 && value[0] == '`' && value[len(value)-1] == '`' {
		return value[1 : len(value)-1], nil
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return strings.ReplaceAll(value[1:len(value)-1], `\"`, `"`), nil
	}
	return value, nil
}

func receiverString(list *ast.FieldList) (string, bool) {
	if list == nil || len(list.List) != 1 {
		return "", false
	}
	switch typ := list.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := typ.X.(*ast.Ident); ok {
			return "*" + ident.Name, true
		}
	case *ast.Ident:
		return typ.Name, false
	}
	return "", false
}

func isSignalType(typ string) bool {
	return typ == "signal.Int" ||
		typ == "signal.String" ||
		typ == "signal.Bool" ||
		strings.HasPrefix(typ, "signal.Signal[") ||
		strings.HasPrefix(typ, "reactivity.Signal[")
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func importAlias(path string) string {
	base := path
	if slash := strings.LastIndexAny(base, `/\`); slash >= 0 {
		base = base[slash+1:]
	}
	base = componentfile.TrimComponentExt(base)
	if base == "" {
		return ""
	}
	var out []rune
	upperNext := true
	for _, r := range base {
		if r == '-' || r == '_' || r == ' ' || r == '.' {
			upperNext = true
			continue
		}
		if upperNext {
			out = append(out, unicode.ToUpper(r))
			upperNext = false
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

type sourceMap struct {
	base       BasePosition
	lineStarts []int
}

func newSourceMap(source string, base BasePosition) sourceMap {
	starts := []int{0}
	for i := 0; i < len(source); i++ {
		if source[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return sourceMap{base: base, lineStarts: starts}
}

func (m sourceMap) position(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	idx := sort.Search(len(m.lineStarts), func(i int) bool {
		return m.lineStarts[i] > offset
	}) - 1
	if idx < 0 {
		idx = 0
	}
	line := m.base.Line + idx
	column := offset - m.lineStarts[idx] + 1
	if idx == 0 {
		column = m.base.Column + offset
	}
	return Position{
		Offset: m.base.Offset + offset,
		Line:   line,
		Column: column,
	}
}
