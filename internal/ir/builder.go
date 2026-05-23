package ir

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/template"
)

type BuildInput struct {
	Name     string
	Template *template.Document
	Go       goanalysis.Metadata
	Imports  []Import
}

func Build(input BuildInput) *Component {
	b := builder{
		component:  &Component{Name: input.Name, Imports: input.Imports},
		state:      map[string]goanalysis.Field{},
		methods:    map[string]goanalysis.Method{},
		aliases:    map[string]struct{}{},
		imports:    map[string]*Component{},
		imported:   map[string]struct{}{},
		slotScopes: map[string]struct{}{},
	}
	for _, imported := range input.Imports {
		b.imported[imported.Alias] = struct{}{}
		if imported.Component != nil {
			b.imports[imported.Alias] = imported.Component
		}
	}
	if input.Go.State != nil {
		for _, field := range input.Go.State.Fields {
			b.state[field.Alias] = field
		}
	}
	b.component.PropNames = sortedPropNames(b.state)
	b.component.Props = propDefs(input.Go.State)
	b.component.Provides = provideDefs(input.Go.State)
	b.component.Injects = injectDefs(input.Go.State)
	for _, method := range input.Go.Methods {
		b.methods[method.Name] = method
	}
	for _, emit := range input.Go.Emits {
		b.component.Emits = append(b.component.Emits, Emit{Method: emit.Method, Event: emit.Event, Span: fromGoSpan(emit.Span)})
	}
	for _, lifecycle := range input.Go.Lifecycle {
		b.component.Lifecycle = append(b.component.Lifecycle, Lifecycle{Hook: lifecycle.Hook, Method: lifecycle.Method, Span: fromGoSpan(lifecycle.Span)})
	}
	for _, diag := range input.Go.Diagnostics {
		b.component.Diagnostics = append(b.component.Diagnostics, fromGoDiagnostic(diag))
	}
	if input.Template == nil {
		b.error("ir.missing_template_ast", "missing template AST", Span{})
		return b.component
	}
	for _, diag := range input.Template.Diagnostics {
		b.component.Diagnostics = append(b.component.Diagnostics, fromTemplateDiagnostic(diag))
	}
	for _, node := range input.Template.Nodes {
		built := b.buildNode(node)
		if built != nil {
			b.component.Nodes = append(b.component.Nodes, built)
		}
	}
	return b.component
}

type builder struct {
	component  *Component
	state      map[string]goanalysis.Field
	methods    map[string]goanalysis.Method
	aliases    map[string]struct{}
	imports    map[string]*Component
	imported   map[string]struct{}
	slotScopes map[string]struct{}
}

func (b *builder) buildNode(node template.Node) Node {
	switch n := node.(type) {
	case *template.Element:
		return b.buildElement(n)
	case *template.Text:
		return &Text{Value: n.Value, Span: fromTemplateSpan(n.Span)}
	case *template.Interpolation:
		return b.buildInterpolation(n)
	case *template.Comment:
		return nil
	default:
		b.error("ir.unsupported_template_node", fmt.Sprintf("unsupported template node %T", node), fromTemplateSpan(node.NodeSpan()))
		return nil
	}
}

func (b *builder) buildElement(elem *template.Element) Node {
	if elem.Tag == "component" {
		if b.rejectAsyncComponent(elem) {
			return nil
		}
		return b.buildDynamicComponent(elem)
	}
	if alias, component, ok := b.resolveComponent(elem); ok {
		if b.rejectAsyncComponent(elem) {
			return nil
		}
		props, events, slots := b.componentAttrs(elem)
		props = b.applyInjectedProps(props, component)
		props = canonicalComponentProps(props, component)
		b.validateComponentProps(alias, props, component)
		props = applyDefaultProps(props, component)
		componentProps, fallthroughProps := splitComponentProps(props, component.PropNames)
		nodes := applyComponentContent(cloneNodes(component.Nodes), componentProps, fallthroughProps, slots, events, component.Emits, false)
		b.validateExpandedComponentState(alias, nodes, component, componentProps)
		return &ComponentInstance{
			Alias:     alias,
			Component: component,
			Props:     props,
			Events:    events,
			Slots:     slots,
			Lifecycle: component.Lifecycle,
			Nodes:     nodes,
			Span:      fromTemplateSpan(elem.Span),
		}
	}
	if alias, ok := b.resolveImportedAlias(elem); ok {
		b.error("ir.unresolved_component_import", fmt.Sprintf("component %q was imported but could not be resolved", alias), fromTemplateSpan(elem.Span))
		return nil
	}
	if isComponentTag(elem.RawTag) || isKebabComponentTag(elem.RawTag) {
		b.error("ir.unknown_component", fmt.Sprintf("unknown component %q; import a .vue component with this alias", elem.RawTag), fromTemplateSpan(elem.Span))
		return nil
	}
	out := &Element{
		Tag:    elem.Tag,
		RawTag: elem.RawTag,
		Span:   fromTemplateSpan(elem.Span),
	}
	var conditional string
	var repeater string
	for _, attr := range elem.Attrs {
		switch attr.Kind {
		case template.AttrStatic:
			out.Props = append(out.Props, Prop{
				Name:  attr.Name,
				Value: attr.Value,
				Span:  fromTemplateSpan(attr.NameSpan),
			})
		case template.AttrBoundProp:
			out.Props = append(out.Props, Prop{
				Name:    attr.Arg,
				Binding: attr.Value,
				Bound:   true,
				Span:    fromTemplateSpan(attr.NameSpan),
			})
			b.validateStateBinding(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrModel:
			name := attr.Arg
			if name == "" {
				name = "modelValue"
			}
			out.Props = append(out.Props, Prop{
				Name:    name,
				Binding: attr.Value,
				Bound:   true,
				Span:    fromTemplateSpan(attr.NameSpan),
			})
			b.validateStateBinding(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrEvent:
			out.Events = append(out.Events, EventHandler{
				Event:  attr.Arg,
				Method: attr.Value,
				Span:   fromTemplateSpan(attr.NameSpan),
			})
			b.validateEventHandler(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrIf:
			conditional = attr.Value
			b.validateStateBinding(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrFor:
			repeater = attr.Value
		case template.AttrSlot:
		}
	}
	alias, _, hasAlias := parseFor(repeater)
	if hasAlias {
		b.aliases[alias] = struct{}{}
	}
	for _, child := range elem.Children {
		built := b.buildNode(child)
		if built != nil {
			out.Children = append(out.Children, built)
		}
	}
	if hasAlias {
		delete(b.aliases, alias)
	}
	var node Node = out
	if repeater != "" {
		node = &Repeater{
			Expression: repeater,
			Child:      node,
			Span:       out.Span,
		}
	}
	if conditional != "" {
		node = &Conditional{
			Expression: conditional,
			Child:      node,
			Span:       out.Span,
		}
	}
	return node
}

func (b *builder) rejectAsyncComponent(elem *template.Element) bool {
	for _, attr := range elem.Attrs {
		if attr.Name == "async" || attr.Name == "v-async" || (attr.Kind == template.AttrBoundProp && attr.Arg == "async") {
			b.error("ir.unsupported_async_component", "async components are not supported yet", fromTemplateSpan(attr.NameSpan))
			return true
		}
	}
	return false
}

func (b *builder) resolveComponent(elem *template.Element) (string, *Component, bool) {
	if component, ok := b.imports[elem.RawTag]; ok {
		return elem.RawTag, component, true
	}
	if alias := pascalFromKebab(elem.RawTag); alias != "" {
		if component, ok := b.imports[alias]; ok {
			return alias, component, true
		}
	}
	return "", nil, false
}

func (b *builder) resolveImportedAlias(elem *template.Element) (string, bool) {
	if _, ok := b.imported[elem.RawTag]; ok {
		return elem.RawTag, true
	}
	if alias := pascalFromKebab(elem.RawTag); alias != "" {
		if _, ok := b.imported[alias]; ok {
			return alias, true
		}
	}
	return "", false
}

func (b *builder) buildDynamicComponent(elem *template.Element) Node {
	binding := dynamicComponentBinding(elem)
	if binding == "" {
		b.error("ir.missing_dynamic_component_binding", `dynamic <component> requires :is`, fromTemplateSpan(elem.Span))
		return nil
	}
	b.validateStateBinding(binding, fromTemplateSpan(elem.Span))
	props, events, slots := b.componentAttrs(elem)
	out := &DynamicComponent{
		Binding: binding,
		Props:   props,
		Events:  events,
		Slots:   slots,
		Span:    fromTemplateSpan(elem.Span),
	}
	for alias, component := range b.imports {
		caseProps := b.applyInjectedProps(props, component)
		caseProps = canonicalComponentProps(caseProps, component)
		b.validateComponentProps(alias, caseProps, component)
		caseProps = applyDefaultProps(caseProps, component)
		componentProps, fallthroughProps := splitComponentProps(caseProps, component.PropNames)
		nodes := applyComponentContent(cloneNodes(component.Nodes), componentProps, fallthroughProps, slots, events, component.Emits, false)
		b.validateExpandedComponentState(alias, nodes, component, componentProps)
		out.Cases = append(out.Cases, DynamicComponentCase{
			Alias:     alias,
			Component: component,
			Nodes:     nodes,
		})
	}
	sort.Slice(out.Cases, func(i, j int) bool {
		return out.Cases[i].Alias < out.Cases[j].Alias
	})
	if len(out.Cases) == 0 {
		b.error("ir.empty_dynamic_component_cases", "dynamic <component> has no resolved imported components", fromTemplateSpan(elem.Span))
	}
	return out
}

func (b *builder) validateExpandedComponentState(alias string, nodes []Node, component *Component, props []Prop) {
	stateNames := map[string]struct{}{}
	for _, prop := range component.Props {
		stateNames[prop.Name] = struct{}{}
	}
	for _, inject := range component.Injects {
		stateNames[inject.Name] = struct{}{}
	}
	if len(stateNames) == 0 {
		return
	}
	for _, prop := range props {
		delete(stateNames, prop.Name)
	}
	if len(stateNames) == 0 {
		return
	}
	b.validateExpandedComponentStateNodes(alias, nodes, stateNames)
}

func (b *builder) validateExpandedComponentStateNodes(alias string, nodes []Node, stateNames map[string]struct{}) {
	for _, node := range nodes {
		b.validateExpandedComponentStateNode(alias, node, stateNames)
	}
}

func (b *builder) validateExpandedComponentStateNode(alias string, node Node, stateNames map[string]struct{}) {
	switch n := node.(type) {
	case *Element:
		for _, prop := range n.Props {
			if prop.Bound {
				b.validateExpandedComponentBinding(alias, prop.Binding, prop.Span, stateNames)
			}
		}
		b.validateExpandedComponentStateNodes(alias, n.Children, stateNames)
	case *Interpolation:
		b.validateExpandedComponentBinding(alias, n.Binding, n.Span, stateNames)
	case *Conditional:
		b.validateExpandedComponentBinding(alias, n.Expression, n.Span, stateNames)
		b.validateExpandedComponentStateNode(alias, n.Child, stateNames)
	case *Repeater:
		b.validateExpandedComponentStateNode(alias, n.Child, stateNames)
	case *ComponentInstance:
		b.validateExpandedComponentStateNodes(alias, n.Nodes, stateNames)
	case *DynamicComponent:
		for _, candidate := range n.Cases {
			b.validateExpandedComponentStateNodes(alias, candidate.Nodes, stateNames)
		}
	}
}

func (b *builder) validateExpandedComponentBinding(alias, binding string, span Span, stateNames map[string]struct{}) {
	if _, ok := stateNames[binding]; !ok {
		return
	}
	b.error("ir.unbound_component_state", fmt.Sprintf("component %s state %q requires a prop, default, or provide/inject binding", alias, binding), span)
}

func (b *builder) applyInjectedProps(props []Prop, component *Component) []Prop {
	if len(component.Injects) == 0 || len(b.component.Provides) == 0 {
		return props
	}
	out := append([]Prop(nil), props...)
	providedProps := map[string]struct{}{}
	for _, prop := range out {
		providedProps[prop.Name] = struct{}{}
	}
	provides := map[string]ProvideDef{}
	for _, provide := range b.component.Provides {
		provides[provide.Name] = provide
	}
	for _, inject := range component.Injects {
		if _, ok := providedProps[inject.Name]; ok {
			continue
		}
		provide, ok := provides[inject.Name]
		if !ok {
			continue
		}
		out = append(out, Prop{Name: inject.Name, Binding: provide.Binding, Bound: true, Span: provide.Span})
	}
	return out
}

func dynamicComponentBinding(elem *template.Element) string {
	for _, attr := range elem.Attrs {
		if attr.Kind == template.AttrBoundProp && attr.Arg == "is" {
			return attr.Value
		}
	}
	return ""
}

func (b *builder) componentAttrs(elem *template.Element) ([]Prop, []EventHandler, []Slot) {
	var props []Prop
	var events []EventHandler
	for _, attr := range elem.Attrs {
		switch attr.Kind {
		case template.AttrStatic:
			props = append(props, Prop{Name: attr.Name, Value: attr.Value, Span: fromTemplateSpan(attr.NameSpan)})
		case template.AttrBoundProp:
			props = append(props, Prop{Name: attr.Arg, Binding: attr.Value, Bound: true, Span: fromTemplateSpan(attr.NameSpan)})
			b.validateStateBinding(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrModel:
			name := attr.Arg
			if name == "" {
				name = "modelValue"
			}
			props = append(props, Prop{Name: name, Binding: attr.Value, Bound: true, Span: fromTemplateSpan(attr.NameSpan)})
			events = append(events, EventHandler{Event: "update:" + name, Method: attr.Value, Span: fromTemplateSpan(attr.NameSpan)})
			b.validateStateBinding(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrEvent:
			events = append(events, EventHandler{Event: attr.Arg, Method: attr.Value, Span: fromTemplateSpan(attr.NameSpan)})
			b.validateEventHandler(attr.Value, fromTemplateSpan(attr.ValueOrNameSpan()))
		case template.AttrSlot:
		}
	}
	defaultSlot := Slot{Name: "default", Scope: componentDefaultSlotScope(elem)}
	named := map[string][]Node{}
	namedScopes := map[string]string{}
	b.pushSlotScope(defaultSlot.Scope)
	for _, child := range elem.Children {
		if childElem, ok := child.(*template.Element); ok && childElem.Tag == "template" {
			slotName, slotScope := slotInfoForTemplate(childElem)
			if slotName != "" {
				namedScopes[slotName] = slotScope
				b.pushSlotScope(slotScope)
				for _, grandchild := range childElem.Children {
					if built := b.buildNode(grandchild); built != nil {
						named[slotName] = append(named[slotName], applyScopedSlotAliases(built, slotScope)...)
					}
				}
				b.popSlotScope(slotScope)
				continue
			}
		}
		if childElem, ok := child.(*template.Element); ok {
			slotName, slotScope := slotInfoForTemplate(childElem)
			if slotName != "" {
				namedScopes[slotName] = slotScope
				clean := *childElem
				clean.Attrs = attrsWithoutSlot(childElem.Attrs)
				b.pushSlotScope(slotScope)
				if built := b.buildNode(&clean); built != nil {
					named[slotName] = append(named[slotName], applyScopedSlotAliases(built, slotScope)...)
				}
				b.popSlotScope(slotScope)
				continue
			}
		}
		if built := b.buildNode(child); built != nil {
			defaultSlot.Nodes = append(defaultSlot.Nodes, applyScopedSlotAliases(built, defaultSlot.Scope)...)
		}
	}
	b.popSlotScope(defaultSlot.Scope)
	var slots []Slot
	if len(defaultSlot.Nodes) > 0 {
		slots = append(slots, defaultSlot)
	}
	for name, nodes := range named {
		slots = append(slots, Slot{Name: name, Scope: namedScopes[name], Nodes: nodes})
	}
	return props, events, slots
}

func componentDefaultSlotScope(elem *template.Element) string {
	for _, attr := range elem.Attrs {
		if attr.Kind == template.AttrSlot && attr.Arg == "default" {
			return attr.Value
		}
	}
	return ""
}

func (b *builder) pushSlotScope(scope string) {
	if scope == "" {
		return
	}
	b.slotScopes[scope] = struct{}{}
}

func (b *builder) popSlotScope(scope string) {
	if scope == "" {
		return
	}
	delete(b.slotScopes, scope)
}

func applyScopedSlotAliases(node Node, scope string) []Node {
	if scope == "" {
		return []Node{node}
	}
	return applyScopedSlotAliasNode(node, scope)
}

func applyScopedSlotAliasNode(node Node, scope string) []Node {
	switch n := node.(type) {
	case *Element:
		clone := *n
		clone.Props = replaceScopedProps(n.Props, scope)
		clone.Children = nil
		for _, child := range n.Children {
			clone.Children = append(clone.Children, applyScopedSlotAliasNode(child, scope)...)
		}
		return []Node{&clone}
	case *Interpolation:
		if strings.HasPrefix(n.Binding, scope+".") {
			clone := *n
			clone.Binding = strings.TrimPrefix(n.Binding, scope+".")
			return []Node{&clone}
		}
		clone := *n
		return []Node{&clone}
	case *Conditional:
		clone := *n
		clone.Expression = strings.TrimPrefix(clone.Expression, scope+".")
		children := applyScopedSlotAliasNode(n.Child, scope)
		if len(children) == 1 {
			clone.Child = children[0]
		}
		return []Node{&clone}
	case *Repeater:
		clone := *n
		clone.Child = cloneNode(n.Child)
		return []Node{&clone}
	default:
		return []Node{node}
	}
}

func replaceScopedProps(props []Prop, scope string) []Prop {
	out := append([]Prop(nil), props...)
	prefix := scope + "."
	for i, prop := range out {
		if prop.Bound && strings.HasPrefix(prop.Binding, prefix) {
			out[i].Binding = strings.TrimPrefix(prop.Binding, prefix)
		}
	}
	return out
}

func attrsWithoutSlot(attrs []template.Attribute) []template.Attribute {
	out := make([]template.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		if attr.Kind == template.AttrSlot {
			continue
		}
		out = append(out, attr)
	}
	return out
}

func slotNameForTemplate(elem *template.Element) string {
	name, _ := slotInfoForTemplate(elem)
	return name
}

func slotInfoForTemplate(elem *template.Element) (string, string) {
	for _, attr := range elem.Attrs {
		if attr.Kind == template.AttrSlot {
			if attr.Arg != "" {
				return attr.Arg, attr.Value
			}
			return "default", attr.Value
		}
	}
	return "", ""
}

func (b *builder) buildInterpolation(interp *template.Interpolation) Node {
	binding := strings.TrimSpace(interp.Expression)
	out := &Interpolation{
		Binding: binding,
		Span:    fromTemplateSpan(interp.Span),
	}
	if field, ok := b.state[binding]; ok {
		out.GoField = field.Name
		return out
	}
	if _, ok := b.aliases[binding]; ok {
		return out
	}
	if b.isSlotScopeBinding(binding) {
		return out
	}
	b.error("ir.unknown_state_binding", fmt.Sprintf("unknown state binding %q", binding), fromTemplateSpan(interp.ExprSpan))
	return out
}

func (b *builder) validateStateBinding(binding string, span Span) {
	if binding == "" {
		b.error("ir.empty_binding", "binding expression cannot be empty", span)
		return
	}
	if _, ok := b.state[binding]; !ok {
		if _, aliasOK := b.aliases[binding]; aliasOK {
			return
		}
		if b.isSlotScopeBinding(binding) {
			return
		}
		b.error("ir.unknown_state_binding", fmt.Sprintf("unknown state binding %q", binding), span)
	}
}

func (b *builder) isSlotScopeBinding(binding string) bool {
	scope, _, ok := strings.Cut(binding, ".")
	if !ok {
		return false
	}
	_, exists := b.slotScopes[scope]
	return exists
}

func parseFor(expression string) (string, string, bool) {
	parts := strings.Split(expression, " in ")
	if len(parts) != 2 {
		return "", "", false
	}
	alias := strings.TrimSpace(parts[0])
	source := strings.TrimSpace(parts[1])
	return alias, source, alias != "" && source != ""
}

func (b *builder) validateEventHandler(method string, span Span) {
	if method == "" {
		b.error("ir.empty_event_handler", "event handler expression cannot be empty", span)
		return
	}
	if _, ok := b.methods[method]; !ok {
		b.error("ir.unknown_event_handler", fmt.Sprintf("unknown event handler %q", method), span)
	}
}

func (b *builder) error(code, message string, span Span) {
	b.component.Diagnostics = append(b.component.Diagnostics, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     span,
	})
}

func fromTemplateDiagnostic(diag template.Diagnostic) Diagnostic {
	return Diagnostic{
		Code:     diag.Code,
		Message:  diag.Message,
		Severity: diag.Severity,
		Span:     fromTemplateSpan(diag.Span),
	}
}

func fromGoDiagnostic(diag goanalysis.Diagnostic) Diagnostic {
	return Diagnostic{
		Code:     diag.Code,
		Message:  diag.Message,
		Severity: diag.Severity,
		Span:     fromGoSpan(diag.Span),
	}
}

func fromTemplateSpan(span template.Span) Span {
	return Span{
		Start: Position(span.Start),
		End:   Position(span.End),
	}
}

func fromGoSpan(span goanalysis.Span) Span {
	return Span{
		Start: Position(span.Start),
		End:   Position(span.End),
	}
}

func cloneNodes(nodes []Node) []Node {
	out := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, cloneNode(node))
	}
	return out
}

func cloneNode(node Node) Node {
	switch n := node.(type) {
	case *Element:
		clone := *n
		clone.Props = append([]Prop(nil), n.Props...)
		clone.Events = append([]EventHandler(nil), n.Events...)
		clone.Children = cloneNodes(n.Children)
		return &clone
	case *Text:
		clone := *n
		return &clone
	case *Interpolation:
		clone := *n
		return &clone
	case *Conditional:
		clone := *n
		clone.Child = cloneNode(n.Child)
		return &clone
	case *Repeater:
		clone := *n
		clone.Child = cloneNode(n.Child)
		return &clone
	case *ComponentInstance:
		clone := *n
		clone.Component = n.Component
		clone.Props = append([]Prop(nil), n.Props...)
		clone.Events = append([]EventHandler(nil), n.Events...)
		clone.Slots = append([]Slot(nil), n.Slots...)
		clone.Lifecycle = append([]Lifecycle(nil), n.Lifecycle...)
		clone.Nodes = cloneNodes(n.Nodes)
		return &clone
	case *DynamicComponent:
		clone := *n
		clone.Props = append([]Prop(nil), n.Props...)
		clone.Events = append([]EventHandler(nil), n.Events...)
		clone.Slots = append([]Slot(nil), n.Slots...)
		clone.Cases = make([]DynamicComponentCase, 0, len(n.Cases))
		for _, candidate := range n.Cases {
			clone.Cases = append(clone.Cases, DynamicComponentCase{
				Alias:     candidate.Alias,
				Component: candidate.Component,
				Nodes:     cloneNodes(candidate.Nodes),
			})
		}
		return &clone
	default:
		return node
	}
}

func applyComponentContent(nodes []Node, props []Prop, fallthroughProps []Prop, slots []Slot, events []EventHandler, emits []Emit, mapEmits bool) []Node {
	propMap := map[string]Prop{}
	for _, prop := range props {
		propMap[prop.Name] = prop
	}
	slotMap := map[string][]Node{}
	for _, slot := range slots {
		name := slot.Name
		if name == "" {
			name = "default"
		}
		slotMap[name] = cloneNodes(slot.Nodes)
	}
	var out []Node
	appliedRootAttrs := false
	var emitEvents map[string]EventHandler
	if mapEmits {
		emitEvents = eventsByEmit(emits, events)
	}
	for _, node := range nodes {
		rootProps := []Prop(nil)
		rootEvents := []EventHandler(nil)
		if !appliedRootAttrs && acceptsComponentRootAttrs(node) {
			rootProps = fallthroughProps
			rootEvents = fallthroughRootEvents(events, emits)
			appliedRootAttrs = true
		}
		out = append(out, applyComponentNode(node, propMap, slotMap, rootProps, rootEvents, emitEvents)...)
	}
	return out
}

func fallthroughRootEvents(events []EventHandler, emits []Emit) []EventHandler {
	if len(events) == 0 {
		return nil
	}
	emitted := map[string]struct{}{}
	for _, emit := range emits {
		emitted[emit.Event] = struct{}{}
	}
	out := make([]EventHandler, 0, len(events))
	for _, event := range events {
		if _, ok := emitted[event.Event]; ok {
			continue
		}
		out = append(out, event)
	}
	return out
}

func eventsByEmit(emits []Emit, events []EventHandler) map[string]EventHandler {
	if len(emits) == 0 || len(events) == 0 {
		return nil
	}
	listeners := map[string]EventHandler{}
	for _, event := range events {
		listeners[event.Event] = event
	}
	out := map[string]EventHandler{}
	for _, emit := range emits {
		if listener, ok := listeners[emit.Event]; ok {
			out[emit.Method] = listener
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func replaceEmitEvents(events []EventHandler, emitEvents map[string]EventHandler) []EventHandler {
	out := append([]EventHandler(nil), events...)
	for i, event := range out {
		if replacement, ok := emitEvents[event.Method]; ok {
			replacement.Event = event.Event
			replacement.Span = event.Span
			out[i] = replacement
		}
	}
	return out
}

func acceptsComponentRootAttrs(node Node) bool {
	switch n := node.(type) {
	case *Element:
		return true
	case *Conditional:
		return acceptsComponentRootAttrs(n.Child)
	case *Repeater:
		return acceptsComponentRootAttrs(n.Child)
	default:
		return false
	}
}

func applyComponentNode(node Node, props map[string]Prop, slots map[string][]Node, rootProps []Prop, rootEvents []EventHandler, emitEvents map[string]EventHandler) []Node {
	switch n := node.(type) {
	case *Element:
		if n.Tag == "slot" {
			name := slotElementName(n)
			if name == "" {
				name = "default"
			}
			if replacement, ok := slots[name]; ok {
				return applySlotProps(cloneNodes(replacement), replaceProps(n.Props, props))
			}
			return cloneNodes(n.Children)
		}
		clone := *n
		clone.Props = replaceProps(n.Props, props)
		if len(rootProps) > 0 {
			clone.Props = mergeRootProps(clone.Props, rootProps)
		}
		clone.Events = replaceEmitEvents(n.Events, emitEvents)
		if len(rootEvents) > 0 {
			clone.Events = append(clone.Events, rootEvents...)
		}
		clone.Children = nil
		for _, child := range n.Children {
			clone.Children = append(clone.Children, applyComponentNode(child, props, slots, nil, nil, emitEvents)...)
		}
		return []Node{&clone}
	case *Interpolation:
		if prop, ok := props[n.Binding]; ok {
			if prop.Bound {
				clone := *n
				clone.Binding = prop.Binding
				clone.GoField = ""
				return []Node{&clone}
			}
			return []Node{&Text{Value: prop.Value, Span: n.Span}}
		}
		clone := *n
		return []Node{&clone}
	case *Conditional:
		clone := *n
		if prop, ok := props[n.Expression]; ok {
			clone.Expression = propValueExpression(prop)
		}
		children := applyComponentNode(n.Child, props, slots, rootProps, rootEvents, emitEvents)
		if len(children) == 1 {
			clone.Child = children[0]
		} else {
			clone.Child = &ComponentInstance{Alias: "fragment", Nodes: children}
		}
		return []Node{&clone}
	case *Repeater:
		clone := *n
		clone.Child = cloneNode(n.Child)
		return []Node{&clone}
	case *ComponentInstance:
		clone := *n
		clone.Component = n.Component
		clone.Props = append([]Prop(nil), n.Props...)
		clone.Events = append([]EventHandler(nil), n.Events...)
		clone.Slots = append([]Slot(nil), n.Slots...)
		clone.Lifecycle = append([]Lifecycle(nil), n.Lifecycle...)
		clone.Nodes = applyComponentContent(n.Nodes, n.Props, nil, n.Slots, n.Events, nil, false)
		return []Node{&clone}
	case *DynamicComponent:
		clone := *n
		clone.Cases = make([]DynamicComponentCase, 0, len(n.Cases))
		for _, candidate := range n.Cases {
			clone.Cases = append(clone.Cases, DynamicComponentCase{
				Alias:     candidate.Alias,
				Component: candidate.Component,
				Nodes:     cloneNodes(candidate.Nodes),
			})
		}
		return []Node{&clone}
	case *Text:
		clone := *n
		return []Node{&clone}
	default:
		return []Node{node}
	}
}

func sortedPropNames(state map[string]goanalysis.Field) []string {
	out := make([]string, 0, len(state))
	for name := range state {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func propDefs(state *goanalysis.StateMetadata) []PropDef {
	if state == nil {
		return nil
	}
	out := make([]PropDef, 0, len(state.Fields))
	for _, field := range state.Fields {
		out = append(out, PropDef{
			Name:     field.Alias,
			GoField:  field.Name,
			Type:     field.Type,
			Required: field.HasVugraTag && !field.Optional,
			Default:  field.Default,
			Span:     fromGoSpan(field.Span),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func provideDefs(state *goanalysis.StateMetadata) []ProvideDef {
	if state == nil {
		return nil
	}
	var out []ProvideDef
	for _, field := range state.Fields {
		if !field.Provide {
			continue
		}
		out = append(out, ProvideDef{
			Name:    field.Alias,
			Binding: field.Alias,
			Span:    fromGoSpan(field.Span),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func injectDefs(state *goanalysis.StateMetadata) []InjectDef {
	if state == nil {
		return nil
	}
	var out []InjectDef
	for _, field := range state.Fields {
		if !field.Inject {
			continue
		}
		out = append(out, InjectDef{
			Name:    field.Alias,
			GoField: field.Name,
			Type:    field.Type,
			Default: field.Default,
			Span:    fromGoSpan(field.Span),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (b *builder) validateComponentProps(alias string, props []Prop, component *Component) {
	if len(component.PropNames) == 0 {
		return
	}
	declared := map[string]struct{}{}
	for _, name := range component.PropNames {
		declared[name] = struct{}{}
	}
	for _, prop := range props {
		if _, ok := declared[prop.Name]; ok {
			continue
		}
		if isFallthroughProp(prop.Name) {
			continue
		}
		b.error("ir.unknown_component_prop", fmt.Sprintf("unknown prop %q on component %s", prop.Name, alias), prop.Span)
	}
	provided := map[string]struct{}{}
	for _, prop := range props {
		provided[prop.Name] = struct{}{}
	}
	for _, prop := range component.Props {
		if !prop.Required {
			continue
		}
		if _, ok := provided[prop.Name]; !ok {
			b.error("ir.missing_required_component_prop", fmt.Sprintf("missing required prop %q on component %s", prop.Name, alias), Span{})
		}
	}
}

func canonicalComponentProps(props []Prop, component *Component) []Prop {
	if len(props) == 0 || len(component.PropNames) == 0 {
		return props
	}
	canonical := map[string]string{}
	for _, name := range component.PropNames {
		canonical[strings.ToLower(name)] = name
	}
	out := append([]Prop(nil), props...)
	for i, prop := range out {
		if name, ok := canonical[strings.ToLower(prop.Name)]; ok {
			out[i].Name = name
		}
	}
	return out
}

func applyDefaultProps(props []Prop, component *Component) []Prop {
	if len(component.Props) == 0 {
		return props
	}
	out := append([]Prop(nil), props...)
	provided := map[string]struct{}{}
	for _, prop := range out {
		provided[prop.Name] = struct{}{}
	}
	for _, prop := range component.Props {
		if prop.Default == "" {
			continue
		}
		if _, ok := provided[prop.Name]; ok {
			continue
		}
		out = append(out, Prop{Name: prop.Name, Value: prop.Default, Span: prop.Span})
	}
	return out
}

func isFallthroughProp(name string) bool {
	switch name {
	case "class", "id", "style", "title", "role", "tabindex", "aria-label", "focus-scope":
		return true
	default:
		return strings.HasPrefix(name, "data-") || strings.HasPrefix(name, "aria-")
	}
}

func splitComponentProps(props []Prop, propNames []string) ([]Prop, []Prop) {
	if len(propNames) == 0 {
		return nil, append([]Prop(nil), props...)
	}
	declared := map[string]struct{}{}
	for _, name := range propNames {
		declared[name] = struct{}{}
	}
	var componentProps []Prop
	var fallthroughProps []Prop
	for _, prop := range props {
		if _, ok := declared[prop.Name]; ok {
			componentProps = append(componentProps, prop)
		} else {
			fallthroughProps = append(fallthroughProps, prop)
		}
	}
	return componentProps, fallthroughProps
}

func mergeRootProps(root []Prop, fallthroughProps []Prop) []Prop {
	out := append([]Prop(nil), root...)
	for _, prop := range fallthroughProps {
		if prop.Name == "class" {
			merged := false
			for i := range out {
				if out[i].Name == "class" && !out[i].Bound && !prop.Bound {
					out[i].Value = strings.TrimSpace(out[i].Value + " " + prop.Value)
					merged = true
					break
				}
			}
			if merged {
				continue
			}
		}
		out = append(out, prop)
	}
	return out
}

func applySlotProps(nodes []Node, slotProps []Prop) []Node {
	if len(slotProps) == 0 {
		return nodes
	}
	propMap := map[string]Prop{}
	for _, prop := range slotProps {
		propMap[prop.Name] = prop
	}
	var out []Node
	for _, node := range nodes {
		out = append(out, applySlotPropNode(node, propMap)...)
	}
	return out
}

func applySlotPropNode(node Node, props map[string]Prop) []Node {
	switch n := node.(type) {
	case *Element:
		clone := *n
		clone.Props = replaceProps(n.Props, props)
		clone.Children = nil
		for _, child := range n.Children {
			clone.Children = append(clone.Children, applySlotPropNode(child, props)...)
		}
		return []Node{&clone}
	case *Interpolation:
		if prop, ok := props[n.Binding]; ok {
			if prop.Bound {
				clone := *n
				clone.Binding = prop.Binding
				clone.GoField = ""
				return []Node{&clone}
			}
			return []Node{&Text{Value: prop.Value, Span: n.Span}}
		}
		clone := *n
		return []Node{&clone}
	case *Conditional:
		clone := *n
		if prop, ok := props[n.Expression]; ok {
			clone.Expression = propValueExpression(prop)
		}
		children := applySlotPropNode(n.Child, props)
		if len(children) == 1 {
			clone.Child = children[0]
		}
		return []Node{&clone}
	case *Repeater:
		clone := *n
		clone.Child = cloneNode(n.Child)
		return []Node{&clone}
	default:
		return []Node{node}
	}
}

func isComponentTag(rawTag string) bool {
	for _, r := range rawTag {
		return unicode.IsUpper(r)
	}
	return false
}

func isKebabComponentTag(rawTag string) bool {
	return strings.Contains(rawTag, "-")
}

func pascalFromKebab(rawTag string) string {
	if !strings.Contains(rawTag, "-") {
		return ""
	}
	parts := strings.Split(rawTag, "-")
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			return ""
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		out.WriteString(string(runes))
	}
	return out.String()
}

func replaceProps(in []Prop, props map[string]Prop) []Prop {
	out := append([]Prop(nil), in...)
	for i, prop := range out {
		if replacement, ok := props[prop.Binding]; prop.Bound && ok {
			replacement.Name = prop.Name
			replacement.Span = prop.Span
			out[i] = replacement
		}
	}
	return out
}

func propValueExpression(prop Prop) string {
	if prop.Bound {
		return prop.Binding
	}
	return prop.Value
}

func slotElementName(elem *Element) string {
	for _, prop := range elem.Props {
		if prop.Name == "name" {
			return prop.Value
		}
	}
	return "default"
}
