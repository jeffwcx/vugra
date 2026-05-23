package layout

import (
	"context"
	"fmt"
	"strings"

	"github.com/vugra/vugra/internal/csslayout"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/style"
)

func ComputeCSS(ctx context.Context, input Input) ([]Box, error) {
	width := input.Constraints.Width
	if width == 0 {
		width = 800
	}
	height := input.Constraints.Height
	root := csslayout.Node{
		ID:       "root",
		Tag:      "div",
		Class:    "__vuego_root",
		Children: make([]csslayout.Node, 0, len(input.Nodes)),
	}
	meta := map[string]nodeMeta{}
	for i, node := range input.Nodes {
		cssNode, ok := buildCSSNode(input, node, fmt.Sprintf("n%d", i+1), meta)
		if !ok {
			continue
		}
		root.Children = append(root.Children, cssNode)
	}
	if len(root.Children) == 0 {
		return nil, nil
	}
	out, err := csslayout.Compute(ctx, csslayout.Input{
		CSS:  viewportCSS(input.CSS, width, height),
		Root: root,
		Viewport: csslayout.Viewport{
			Width:  width,
			Height: floatPtr(height),
		},
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]csslayout.Box, len(out.Boxes))
	children := map[string][]string{}
	for _, box := range out.Boxes {
		byID[box.ID] = box
		if parent := meta[box.ID].parent; parent != "" {
			children[parent] = append(children[parent], box.ID)
		}
	}
	var boxes []Box
	for _, child := range root.Children {
		box, ok := convertCSSBox(input, child.ID, byID, children, meta)
		if ok {
			boxes = append(boxes, box)
		}
	}
	return boxes, nil
}

type nodeMeta struct {
	parent         string
	kind           string
	node           ir.Node
	props          map[string]string
	bindings       map[string]string
	events         []Event
	role           string
	text           string
	svg            string
	style          Style
	state          *ir.RuntimeState
	emitters       map[string]string
	parentBindings map[string]string
}

func buildCSSNode(input Input, node ir.Node, id string, meta map[string]nodeMeta) (csslayout.Node, bool) {
	switch n := node.(type) {
	case *ir.Element:
		props, bindings := evalProps(input, n.Props)
		computed := style.ComputeWithTokens(input.Styles, props["class"], input.SystemTokens)
		cssNode := csslayout.Node{ID: id, Tag: n.Tag, Class: props["class"]}
		meta[id] = nodeMeta{
			kind:           "element",
			node:           n,
			props:          props,
			bindings:       bindings,
			events:         events(n.Events),
			role:           roleForElement(n.Tag, props),
			style:          boxStyle(computed),
			state:          input.State,
			emitters:       input.Emitters,
			parentBindings: input.ParentBindings,
		}
		if n.Tag == "svg" {
			m := meta[id]
			m.kind = "svg"
			m.svg = props["__raw_svg"]
			if m.svg == "" {
				m.svg = svgMarkup(n, props)
			}
			meta[id] = m
		} else if n.Tag == "img" && strings.HasSuffix(strings.ToLower(props["src"]), ".svg") {
			svg := props["__raw_svg"]
			if svg == "" && input.ResolveAsset != nil {
				if resolved, ok := input.ResolveAsset(props["src"]); ok {
					svg = resolved
				}
			}
			if svg != "" {
				m := meta[id]
				m.kind = "svg"
				m.svg = svg
				meta[id] = m
			}
		}
		for i, child := range n.Children {
			childID := fmt.Sprintf("%s-%d", id, i+1)
			built, ok := buildCSSNode(input, child, childID, meta)
			if !ok {
				continue
			}
			meta[childID] = withParent(meta[childID], id)
			cssNode.Children = append(cssNode.Children, built)
		}
		if len(cssNode.Children) == 0 && isControl(n.Tag) && props["value"] != "" {
			cssNode.Text = props["value"]
		}
		return cssNode, true
	case *ir.Text:
		text := strings.TrimSpace(n.Value)
		if text == "" {
			return csslayout.Node{}, false
		}
		meta[id] = nodeMeta{kind: "text", node: n, text: text, role: "text", style: Style{Display: "inline"}}
		return csslayout.Node{ID: id, Tag: "span", Text: text}, true
	case *ir.Interpolation:
		text := ""
		if input.ResolveText != nil {
			text = input.ResolveText(n.Binding)
		}
		meta[id] = nodeMeta{kind: "text", node: n, text: text, role: "text", style: Style{Display: "inline"}}
		return csslayout.Node{ID: id, Tag: "span", Text: text}, true
	case *ir.Conditional:
		if input.ResolveTruthy != nil && !input.ResolveTruthy(n.Expression) {
			return csslayout.Node{}, false
		}
		return buildCSSNode(input, n.Child, id, meta)
	case *ir.Repeater:
		alias, source, ok := parseFor(n.Expression)
		if !ok || input.ResolveList == nil {
			return buildCSSNode(input, n.Child, id, meta)
		}
		group := csslayout.Node{ID: id, Tag: "div"}
		meta[id] = nodeMeta{kind: "element", role: "group"}
		for i, value := range input.ResolveList(source) {
			repeated := input
			previousResolveText := repeated.ResolveText
			repeated.ResolveText = func(binding string) string {
				if binding == alias {
					return value
				}
				if previousResolveText != nil {
					return previousResolveText(binding)
				}
				return ""
			}
			childID := fmt.Sprintf("%s-%d", id, i+1)
			built, ok := buildCSSNode(repeated, n.Child, childID, meta)
			if !ok {
				continue
			}
			meta[childID] = withParent(meta[childID], id)
			group.Children = append(group.Children, built)
		}
		return group, true
	case *ir.ComponentInstance:
		childInput := input
		if input.Component != nil {
			childInput = input.Component(n)
			if childInput.Styles == nil {
				childInput.Styles = input.Styles
			}
			if childInput.SystemTokens == nil {
				childInput.SystemTokens = input.SystemTokens
			}
		}
		group := csslayout.Node{ID: id, Tag: "div"}
		meta[id] = nodeMeta{kind: "element", role: "group"}
		for i, child := range n.Nodes {
			childID := fmt.Sprintf("%s-%d", id, i+1)
			built, ok := buildCSSNode(childInput, child, childID, meta)
			if !ok {
				continue
			}
			meta[childID] = withParent(meta[childID], id)
			group.Children = append(group.Children, built)
		}
		return group, len(group.Children) > 0
	case *ir.DynamicComponent:
		alias := ""
		if input.ResolveText != nil {
			alias = input.ResolveText(n.Binding)
		}
		for _, candidate := range n.Cases {
			if candidate.Alias != alias {
				continue
			}
			instance := &ir.ComponentInstance{
				Alias:     candidate.Alias,
				Component: candidate.Component,
				Props:     n.Props,
				Events:    n.Events,
				Slots:     n.Slots,
				Nodes:     candidate.Nodes,
				Span:      n.Span,
			}
			return buildCSSNode(input, instance, id, meta)
		}
		return csslayout.Node{}, false
	default:
		return csslayout.Node{}, false
	}
}

func withParent(meta nodeMeta, parent string) nodeMeta {
	meta.parent = parent
	return meta
}

func convertCSSBox(input Input, id string, byID map[string]csslayout.Box, children map[string][]string, meta map[string]nodeMeta) (Box, bool) {
	source, ok := byID[id]
	if !ok || id == "root" {
		return Box{}, false
	}
	m := meta[id]
	box := Box{
		ID:             id,
		Kind:           m.kind,
		Tag:            source.Tag,
		Text:           firstNonEmpty(m.text, source.Text),
		SVG:            m.svg,
		Role:           m.role,
		Rect:           Rect{X: source.X, Y: source.Y, Width: source.Width, Height: source.Height},
		Props:          m.props,
		Bindings:       m.bindings,
		Style:          mergeCSSStyle(m.style, source.Style),
		Events:         m.events,
		State:          m.state,
		Emitters:       m.emitters,
		ParentBindings: m.parentBindings,
	}
	if box.Kind == "" {
		box.Kind = "element"
	}
	if box.Kind == "text" {
		box.Tag = ""
		return box, true
	}
	for _, childID := range children[id] {
		child, ok := convertCSSBox(input, childID, byID, children, meta)
		if ok {
			box.Children = append(box.Children, child)
		}
	}
	return box, true
}

func evalProps(input Input, props []ir.Prop) (map[string]string, map[string]string) {
	out := map[string]string{}
	bindings := map[string]string{}
	for _, prop := range props {
		if prop.Bound {
			bindings[prop.Name] = prop.Binding
			if input.ResolveProp != nil {
				out[prop.Name] = input.ResolveProp(prop.Binding)
			} else {
				out[prop.Name] = prop.Binding
			}
			continue
		}
		out[prop.Name] = prop.Value
	}
	if len(bindings) == 0 {
		bindings = nil
	}
	return out, bindings
}

func viewportCSS(css string, width, height float32) string {
	if height <= 0 {
		return fmt.Sprintf(".__vuego_root { width: %.4gpx; }\n%s", width, css)
	}
	return fmt.Sprintf(".__vuego_root { width: %.4gpx; height: %.4gpx; }\n%s", width, height, css)
}

func mergeCSSStyle(base Style, css csslayout.Style) Style {
	if css.Display != "" {
		base.Display = css.Display
	}
	if css.FontSize > 0 {
		base.FontSize = css.FontSize
	}
	if css.LineHeight > 0 {
		base.LineHeight = css.LineHeight
	}
	if css.Color != "" {
		base.Color = css.Color
	}
	if css.BackgroundColor != "" {
		base.BackgroundColor = css.BackgroundColor
	}
	if css.Opacity > 0 {
		base.Opacity = css.Opacity
	}
	if css.BorderWidth > 0 {
		base.BorderWidth = css.BorderWidth
	}
	if css.BorderColor != "" {
		base.BorderColor = css.BorderColor
	}
	if css.BorderRadius > 0 {
		base.BorderRadius = css.BorderRadius
	}
	if css.Overflow != "" {
		base.Overflow = css.Overflow
	}
	return base
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func floatPtr(value float32) *float32 {
	if value <= 0 {
		return nil
	}
	return &value
}
