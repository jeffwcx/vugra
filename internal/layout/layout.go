package layout

// Package layout produces backend-neutral layout boxes before painting.

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/style"
)

type Constraints struct {
	Width  float32
	Height float32
}

type Rect struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
}

type Box struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Tag            string            `json:"tag,omitempty"`
	Text           string            `json:"text,omitempty"`
	SVG            string            `json:"svg,omitempty"`
	Lines          []LineBox         `json:"lines,omitempty"`
	Glyphs         []GlyphRun        `json:"glyphs,omitempty"`
	Role           string            `json:"role,omitempty"`
	Rect           Rect              `json:"rect"`
	Props          map[string]string `json:"props,omitempty"`
	Bindings       map[string]string `json:"bindings,omitempty"`
	Style          Style             `json:"style,omitempty"`
	Events         []Event           `json:"events,omitempty"`
	State          *ir.RuntimeState  `json:"-"`
	Emitters       map[string]string `json:"-"`
	ParentBindings map[string]string `json:"-"`
	Children       []Box             `json:"children,omitempty"`
}

type Style struct {
	Display         string  `json:"display,omitempty"`
	FlexDirection   string  `json:"flexDirection,omitempty"`
	FlexWrap        string  `json:"flexWrap,omitempty"`
	AlignItems      string  `json:"alignItems,omitempty"`
	Justify         string  `json:"justify,omitempty"`
	Margin          float32 `json:"margin,omitempty"`
	PaddingLeft     float32 `json:"paddingLeft,omitempty"`
	FlexGrow        float32 `json:"flexGrow,omitempty"`
	FlexBasis       float32 `json:"flexBasis,omitempty"`
	FontSize        float32 `json:"fontSize,omitempty"`
	LineHeight      float32 `json:"lineHeight,omitempty"`
	TextAlign       string  `json:"textAlign,omitempty"`
	BackgroundColor string  `json:"backgroundColor,omitempty"`
	Opacity         float32 `json:"opacity,omitempty"`
	BorderWidth     float32 `json:"borderWidth,omitempty"`
	BorderWidthSet  bool    `json:"borderWidthSet,omitempty"`
	BorderColor     string  `json:"borderColor,omitempty"`
	BorderRadius    float32 `json:"borderRadius,omitempty"`
	Color           string  `json:"color,omitempty"`
	Overflow        string  `json:"overflow,omitempty"`
}

type Event struct {
	Event  string `json:"event"`
	Method string `json:"method"`
}

type LineBox struct {
	Text     string  `json:"text"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Width    float32 `json:"width"`
	Height   float32 `json:"height"`
	Baseline float32 `json:"baseline"`
}

type GlyphRun struct {
	Text     string  `json:"text"`
	Font     string  `json:"font,omitempty"`
	Size     float32 `json:"size"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Advance  float32 `json:"advance"`
	Baseline float32 `json:"baseline"`
}

type Measurer interface {
	MeasureText(string) (width float32, height float32)
}

type StyledMeasurer interface {
	MeasureStyledText(text string, fontSize, lineHeight float32) (width float32, height float32)
}

type FixedMeasurer struct {
	CharWidth  float32
	LineHeight float32
}

func (m FixedMeasurer) MeasureText(text string) (float32, float32) {
	charWidth := m.CharWidth
	if charWidth == 0 {
		charWidth = 8
	}
	lineHeight := m.LineHeight
	if lineHeight == 0 {
		lineHeight = 20
	}
	return float32(len([]rune(text))) * charWidth, lineHeight
}

func measureText(measurer Measurer, text string, textStyle textStyle) (float32, float32) {
	fontSize := textStyle.fontSize
	lineHeight := textStyle.lineHeight
	if styled, ok := measurer.(StyledMeasurer); ok {
		return styled.MeasureStyledText(text, fontSize, lineHeight)
	}
	width, height := measurer.MeasureText(text)
	if fontSize > 0 {
		width = float32(len([]rune(text))) * maxFloat(1, fontSize/2)
	}
	if lineHeight > 0 {
		height = lineHeight
	}
	return width, height
}

func measureLineHeight(measurer Measurer, textStyle textStyle) float32 {
	_, height := measureText(measurer, "M", textStyle)
	if height <= 0 {
		return 20
	}
	return height
}

type Input struct {
	Nodes          []ir.Node
	CSS            string
	Styles         *style.Stylesheet
	Constraints    Constraints
	Measurer       Measurer
	ResolveText    func(binding string) string
	ResolveProp    func(binding string) string
	ResolveTruthy  func(binding string) bool
	ResolveList    func(binding string) []string
	ResolveAsset   func(path string) (string, bool)
	Component      func(instance *ir.ComponentInstance) Input
	State          *ir.RuntimeState
	Emitters       map[string]string
	ParentBindings map[string]string
	SystemTokens   style.SystemTokens
}

func EvalNode(input Input, node ir.Node, x, y, width float32) (Box, bool) {
	l := newEngine(input)
	return l.layoutNode(node, x, y, width, input.Constraints.Height)
}

type textStyle struct {
	fontSize   float32
	lineHeight float32
	textAlign  string
	color      string
	opacity    float32
}

func Compute(input Input) []Box {
	l := newEngine(input)
	return l.layoutNodes(input.Nodes, 0, 0, l.constraints.Width, l.constraints.Height)
}

func newEngine(input Input) *engine {
	l := &engine{
		styles:         input.Styles,
		constraints:    input.Constraints,
		measurer:       input.Measurer,
		resolveText:    input.ResolveText,
		resolveProp:    input.ResolveProp,
		resolveTruthy:  input.ResolveTruthy,
		resolveList:    input.ResolveList,
		resolveAsset:   input.ResolveAsset,
		component:      input.Component,
		state:          input.State,
		emitters:       input.Emitters,
		parentBindings: input.ParentBindings,
		systemTokens:   input.SystemTokens,
		scope:          map[string]string{},
		textStyle:      textStyle{},
		nextID:         0,
	}
	if l.constraints.Width == 0 {
		l.constraints.Width = 800
	}
	if l.measurer == nil {
		l.measurer = FixedMeasurer{}
	}
	return l
}

type engine struct {
	styles         *style.Stylesheet
	constraints    Constraints
	measurer       Measurer
	resolveText    func(binding string) string
	resolveProp    func(binding string) string
	resolveTruthy  func(binding string) bool
	resolveList    func(binding string) []string
	resolveAsset   func(path string) (string, bool)
	component      func(instance *ir.ComponentInstance) Input
	state          *ir.RuntimeState
	emitters       map[string]string
	parentBindings map[string]string
	systemTokens   style.SystemTokens
	scope          map[string]string
	textStyle      textStyle
	nextID         int
}

func (e *engine) layoutNodes(nodes []ir.Node, x, y, width, height float32) []Box {
	var boxes []Box
	cursorY := y
	for _, node := range nodes {
		availableHeight := float32(0)
		if len(boxes) == 0 {
			availableHeight = height
		}
		box, ok := e.layoutNode(node, x, cursorY, width, availableHeight)
		if !ok {
			continue
		}
		boxes = append(boxes, box)
		cursorY += flowHeight(box)
	}
	return boxes
}

func (e *engine) layoutNode(node ir.Node, x, y, width, height float32) (Box, bool) {
	switch n := node.(type) {
	case *ir.Element:
		return e.layoutElement(n, x, y, width, height), true
	case *ir.Text:
		return e.layoutText(n.Value, x, y, width), strings.TrimSpace(n.Value) != ""
	case *ir.Interpolation:
		text := ""
		if scoped, ok := e.scope[n.Binding]; ok {
			text = scoped
		} else if e.resolveText != nil {
			text = e.resolveText(n.Binding)
		}
		return e.layoutText(text, x, y, width), true
	case *ir.Conditional:
		if e.resolveTruthy != nil && !e.resolveTruthy(n.Expression) {
			return Box{}, false
		}
		return e.layoutNode(n.Child, x, y, width, height)
	case *ir.Repeater:
		boxes := e.layoutRepeater(n, x, y, width)
		if len(boxes) == 0 {
			return Box{}, false
		}
		return groupBoxes(e.newID(), boxes), true
	case *ir.ComponentInstance:
		child := e.childEngine(n)
		boxes := child.layoutNodes(n.Nodes, x, y, width, height)
		e.nextID = child.nextID
		if len(boxes) == 0 {
			return Box{}, false
		}
		return groupBoxes(e.newID(), boxes), true
	case *ir.DynamicComponent:
		alias := ""
		if scoped, ok := e.scope[n.Binding]; ok {
			alias = scoped
		} else if e.resolveText != nil {
			alias = e.resolveText(n.Binding)
		}
		for _, candidate := range n.Cases {
			if candidate.Alias != alias {
				continue
			}
			if candidate.Component != nil {
				instance := &ir.ComponentInstance{
					Alias:     candidate.Alias,
					Component: candidate.Component,
					Props:     n.Props,
					Events:    n.Events,
					Slots:     n.Slots,
					Lifecycle: candidate.Component.Lifecycle,
					Nodes:     candidate.Nodes,
					Span:      n.Span,
				}
				child := e.childEngine(instance)
				boxes := child.layoutNodes(candidate.Nodes, x, y, width, height)
				e.nextID = child.nextID
				if len(boxes) == 0 {
					return Box{}, false
				}
				return groupBoxes(e.newID(), boxes), true
			}
			boxes := e.layoutNodes(candidate.Nodes, x, y, width, height)
			if len(boxes) == 0 {
				return Box{}, false
			}
			return groupBoxes(e.newID(), boxes), true
		}
		return Box{}, false
	default:
		return Box{}, false
	}
}

func (e *engine) childEngine(instance *ir.ComponentInstance) *engine {
	if e.component == nil {
		return e
	}
	input := e.component(instance)
	if input.Measurer == nil {
		input.Measurer = e.measurer
	}
	if input.Styles == nil {
		input.Styles = e.styles
	}
	if input.Constraints.Width == 0 {
		input.Constraints.Width = e.constraints.Width
	}
	if input.Constraints.Height == 0 {
		input.Constraints.Height = e.constraints.Height
	}
	child := newEngine(input)
	child.nextID = e.nextID
	return child
}

func (e *engine) layoutRepeater(repeater *ir.Repeater, x, y, width float32) []Box {
	alias, source, ok := parseFor(repeater.Expression)
	if !ok || e.resolveList == nil {
		box, ok := e.layoutNode(repeater.Child, x, y, width, 0)
		if !ok {
			return nil
		}
		return []Box{box}
	}
	values := e.resolveList(source)
	var boxes []Box
	cursorY := y
	previous, hadPrevious := e.scope[alias]
	for _, value := range values {
		e.scope[alias] = value
		box, ok := e.layoutNode(repeater.Child, x, cursorY, width, 0)
		if !ok {
			continue
		}
		boxes = append(boxes, box)
		cursorY += flowHeight(box)
	}
	if hadPrevious {
		e.scope[alias] = previous
	} else {
		delete(e.scope, alias)
	}
	return boxes
}

func groupBoxes(id string, boxes []Box) Box {
	if len(boxes) == 1 {
		return boxes[0]
	}
	rect := boxes[0].Rect
	maxX := rect.X + rect.Width
	maxY := rect.Y + rect.Height
	for _, box := range boxes[1:] {
		if box.Rect.X < rect.X {
			rect.X = box.Rect.X
		}
		if box.Rect.Y < rect.Y {
			rect.Y = box.Rect.Y
		}
		if x := box.Rect.X + box.Rect.Width; x > maxX {
			maxX = x
		}
		if y := box.Rect.Y + box.Rect.Height; y > maxY {
			maxY = y
		}
	}
	rect.Width = maxX - rect.X
	rect.Height = maxY - rect.Y
	return Box{ID: id, Kind: "element", Role: "group", Rect: rect, Children: boxes}
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

func (e *engine) layoutElement(elem *ir.Element, x, y, availableWidth, availableHeight float32) Box {
	props, bindings := e.propsMap(elem.Props)
	computed := style.ComputeWithTokens(e.styles, props["class"], e.systemTokens)
	previousTextStyle := e.textStyle
	defaultStyle := defaultTextStyle(elem.Tag)
	if defaultStyle.fontSize > 0 && e.textStyle.fontSize == 0 {
		e.textStyle.fontSize = defaultStyle.fontSize
	}
	if defaultStyle.lineHeight > 0 && e.textStyle.lineHeight == 0 {
		e.textStyle.lineHeight = defaultStyle.lineHeight
	}
	if computed.FontSize > 0 {
		e.textStyle.fontSize = computed.FontSize
	}
	if computed.LineHeight > 0 {
		e.textStyle.lineHeight = computed.LineHeight
	}
	if computed.TextAlign != "" {
		e.textStyle.textAlign = computed.TextAlign
	}
	if computed.Color != "" {
		e.textStyle.color = computed.Color
	}
	if computed.Opacity > 0 {
		e.textStyle.opacity = computed.Opacity
	}
	defer func() { e.textStyle = previousTextStyle }()
	margin := computed.Margin
	if margin < 0 {
		margin = 0
	}
	padding := computed.Padding
	paddingLeft := computed.PaddingLeft
	if paddingLeft == 0 {
		paddingLeft = padding
	}
	width := availableWidth - margin*2
	if width < 0 {
		width = 0
	}
	if computed.Width > 0 {
		width = computed.Width
	} else if computed.WidthPercent > 0 {
		width = availableWidth * computed.WidthPercent / 100
	}
	width = clampSize(width, computed.MinWidth, computed.MaxWidth)
	contentWidth := width - paddingLeft - padding
	if contentWidth < 0 {
		contentWidth = 0
	}

	box := Box{
		ID:             e.newID(),
		Kind:           "element",
		Tag:            elem.Tag,
		Role:           roleForElement(elem.Tag, props),
		Props:          props,
		Bindings:       bindings,
		Style:          boxStyle(computed),
		Events:         events(elem.Events),
		State:          e.state,
		Emitters:       e.emitters,
		ParentBindings: e.parentBindings,
		Rect: Rect{
			X:     x + margin,
			Y:     y + margin,
			Width: width,
		},
	}

	if elem.Tag == "svg" {
		box.Kind = "svg"
		box.SVG = props["__raw_svg"]
		if box.SVG == "" {
			box.SVG = svgMarkup(elem, props)
		}
	} else if elem.Tag == "img" && strings.HasSuffix(strings.ToLower(props["src"]), ".svg") {
		box.SVG = props["__raw_svg"]
		if box.SVG == "" && e.resolveAsset != nil {
			if svg, ok := e.resolveAsset(props["src"]); ok {
				box.SVG = svg
			}
		}
		if box.SVG != "" {
			box.Kind = "svg"
		}
	}

	height := float32(0)
	if elem.Tag != "svg" {
		switch computed.Display {
		case "flex":
			childHeight := resolvedContentHeight(computed, availableHeight, padding)
			box.Children, height = e.layoutFlexChildren(elem.Children, box.Rect.X+paddingLeft, box.Rect.Y+padding, contentWidth, childHeight, computed)
		case "grid":
			box.Children, height = e.layoutGridChildren(elem.Children, box.Rect.X+paddingLeft, box.Rect.Y+padding, contentWidth, computed)
		default:
			box.Children, height = e.layoutBlockChildren(elem.Children, box.Rect.X+paddingLeft, box.Rect.Y+padding, contentWidth)
		}
	}
	height += padding * 2
	if len(box.Children) == 0 && (isControl(elem.Tag) || elem.Tag == "img" || elem.Tag == "svg") {
		_, measuredHeight := e.measurer.MeasureText(props["value"])
		if elem.Tag == "img" || elem.Tag == "svg" {
			measuredHeight = width
			if measuredHeight <= 0 {
				measuredHeight = 24
			}
		}
		height = measuredHeight + padding*2
	}
	if computed.Height > 0 {
		height = computed.Height
	} else if computed.HeightPercent > 0 && availableHeight > 0 {
		height = availableHeight * computed.HeightPercent / 100
	} else if availableHeight > 0 && height < availableHeight {
		height = availableHeight
	}
	height = clampSize(height, computed.MinHeight, computed.MaxHeight)
	if height < 0 {
		height = 0
	}
	box.Rect.Height = height
	return box
}

func svgMarkup(elem *ir.Element, props map[string]string) string {
	var b strings.Builder
	writeSVGElement(&b, elem, props)
	return b.String()
}

func writeSVGElement(b *strings.Builder, elem *ir.Element, rootProps map[string]string) {
	b.WriteByte('<')
	b.WriteString(elem.Tag)
	props := rootProps
	if props == nil {
		props, _ = staticPropsMap(elem.Props)
	}
	if elem.Tag == "svg" {
		if _, ok := props["xmlns"]; !ok {
			b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
		}
	}
	for name, value := range props {
		if name == "" || strings.HasPrefix(name, "on:") {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(name)
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(value))
		b.WriteByte('"')
	}
	if len(elem.Children) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteByte('>')
	for _, child := range elem.Children {
		switch n := child.(type) {
		case *ir.Element:
			if isSVGElement(n.Tag) {
				writeSVGElement(b, n, nil)
			}
		case *ir.Text:
			b.WriteString(html.EscapeString(n.Value))
		}
	}
	b.WriteString("</")
	b.WriteString(elem.Tag)
	b.WriteByte('>')
}

func staticPropsMap(props []ir.Prop) (map[string]string, map[string]string) {
	out := map[string]string{}
	bindings := map[string]string{}
	for _, prop := range props {
		if prop.Bound {
			bindings[prop.Name] = prop.Binding
			continue
		}
		out[prop.Name] = prop.Value
	}
	if len(bindings) == 0 {
		bindings = nil
	}
	return out, bindings
}

func FileAssetResolver(basePath string) func(string) (string, bool) {
	baseDir := filepath.Dir(basePath)
	return func(path string) (string, bool) {
		if strings.TrimSpace(path) == "" {
			return "", false
		}
		clean := filepath.Clean(path)
		if !filepath.IsAbs(clean) {
			clean = filepath.Join(baseDir, clean)
		}
		data, err := os.ReadFile(clean)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}

func (e *engine) layoutBlockChildren(nodes []ir.Node, x, y, width float32) ([]Box, float32) {
	var boxes []Box
	cursorY := y
	for _, child := range nodes {
		childBox, ok := e.layoutNode(child, x, cursorY, width, 0)
		if !ok {
			continue
		}
		boxes = append(boxes, childBox)
		cursorY += flowHeight(childBox)
	}
	return boxes, cursorY - y
}

func resolvedContentHeight(computed style.Computed, availableHeight, padding float32) float32 {
	height := float32(0)
	if computed.Height > 0 {
		height = computed.Height
	} else if computed.HeightPercent > 0 && availableHeight > 0 {
		height = availableHeight * computed.HeightPercent / 100
	} else if availableHeight > 0 {
		height = availableHeight
	}
	height -= padding * 2
	if height < 0 {
		return 0
	}
	return height
}

func (e *engine) layoutFlexChildren(nodes []ir.Node, x, y, width, height float32, computed style.Computed) ([]Box, float32) {
	if computed.FlexDirection == "column" {
		return e.layoutColumnChildren(nodes, x, y, width, height, rowGap(computed), computed)
	}
	return e.layoutRowChildren(nodes, x, y, width, height, columnGap(computed), computed)
}

func (e *engine) layoutColumnChildren(nodes []ir.Node, x, y, width, height, gap float32, computed style.Computed) ([]Box, float32) {
	type entry struct {
		node         ir.Node
		box          Box
		ok           bool
		grow         float32
		baseWidth    float32
		explicitBase bool
	}
	var entries []entry
	fixedHeight := float32(0)
	totalGrow := float32(0)
	visible := 0
	for _, child := range nodes {
		grow := e.flexGrowForNode(child)
		if grow > 0 && height > 0 {
			entries = append(entries, entry{node: child, ok: true, grow: grow})
			totalGrow += grow
			visible++
			continue
		}
		childBox, ok := e.layoutNode(child, x, y, width, 0)
		if !ok {
			continue
		}
		entries = append(entries, entry{node: child, box: childBox, ok: true})
		fixedHeight += flowHeight(childBox)
		visible++
	}
	if visible == 0 {
		return nil, 0
	}
	remaining := height - fixedHeight - gap*float32(visible-1)
	if remaining < 0 {
		remaining = 0
	}
	usedGrow := float32(0)
	for i := range entries {
		if entries[i].grow <= 0 {
			continue
		}
		allocated := remaining * entries[i].grow / totalGrow
		usedGrow += allocated
		if usedGrow > remaining || i == len(entries)-1 {
			allocated -= usedGrow - remaining
		}
		childBox, ok := e.layoutNode(entries[i].node, x, y, width, allocated)
		if !ok {
			entries[i].ok = false
			continue
		}
		entries[i].box = childBox
	}
	var boxes []Box
	cursorY := y
	for _, entry := range entries {
		if !entry.ok {
			continue
		}
		targetOuterX := x + crossAxisOffset(width, flowWidth(entry.box), computed.AlignItems)
		targetOuterY := cursorY
		translateBox(&entry.box, targetOuterX-outerX(entry.box), targetOuterY-outerY(entry.box))
		boxes = append(boxes, entry.box)
		cursorY += flowHeight(entry.box) + gap
	}
	cursorY -= gap
	return boxes, cursorY - y
}

func (e *engine) layoutRowChildren(nodes []ir.Node, x, y, width, height, gap float32, computed style.Computed) ([]Box, float32) {
	type entry struct {
		node         ir.Node
		box          Box
		ok           bool
		grow         float32
		baseWidth    float32
		explicitBase bool
	}
	var entries []entry
	maxHeight := float32(0)
	fixedWidth := float32(0)
	totalGrow := float32(0)
	visible := 0
	for _, child := range nodes {
		grow, baseWidth, explicitBase := e.flexMetricsForNode(child)
		if grow > 0 && width > 0 {
			entries = append(entries, entry{node: child, ok: true, grow: grow, baseWidth: baseWidth, explicitBase: explicitBase})
			fixedWidth += baseWidth
			totalGrow += grow
			visible++
			continue
		}
		childBox, ok := e.layoutNode(child, 0, y, width, height)
		if !ok {
			continue
		}
		entries = append(entries, entry{box: childBox, ok: true})
		fixedWidth += flowWidth(childBox)
		visible++
		if h := flowHeight(childBox); h > maxHeight {
			maxHeight = h
		}
	}
	if visible == 0 {
		return nil, 0
	}
	remaining := width - fixedWidth - gap*float32(visible-1)
	if remaining < 0 {
		remaining = 0
	}
	usedGrow := float32(0)
	for i := range entries {
		if entries[i].grow <= 0 {
			continue
		}
		growWidth := remaining * entries[i].grow / totalGrow
		usedGrow += growWidth
		if usedGrow > remaining || i == len(entries)-1 {
			growWidth -= usedGrow - remaining
		}
		targetWidth := entries[i].baseWidth + growWidth
		childBox, ok := e.layoutNode(entries[i].node, 0, y, targetWidth, height)
		if !ok {
			entries[i].ok = false
			continue
		}
		if entries[i].explicitBase {
			resizeBoxWidth(&childBox, targetWidth)
		}
		entries[i].box = childBox
		if h := flowHeight(childBox); h > maxHeight {
			maxHeight = h
		}
	}
	var boxes []Box
	for _, entry := range entries {
		if entry.ok {
			boxes = append(boxes, entry.box)
		}
	}
	if computed.FlexWrap == "wrap" {
		return layoutWrappedRow(boxes, x, y, width, gap, rowGap(computed), computed)
	}
	totalWidth := float32(0)
	for _, box := range boxes {
		totalWidth += flowWidth(box)
	}
	if len(boxes) > 1 {
		totalWidth += gap * float32(len(boxes)-1)
	}
	cursorX := x + justifyOffset(width, totalWidth, computed.Justify)
	for i := range boxes {
		targetOuterX := cursorX
		targetOuterY := y + crossAxisOffset(maxHeight, flowHeight(boxes[i]), computed.AlignItems)
		translateBox(&boxes[i], targetOuterX-outerX(boxes[i]), targetOuterY-outerY(boxes[i]))
		cursorX += flowWidth(boxes[i]) + gap
	}
	return boxes, maxHeight
}

func (e *engine) flexGrowForNode(node ir.Node) float32 {
	grow, _, _ := e.flexMetricsForNode(node)
	return grow
}

func (e *engine) flexMetricsForNode(node ir.Node) (grow float32, baseWidth float32, explicitBase bool) {
	switch n := node.(type) {
	case *ir.Element:
		props, _ := e.propsMap(n.Props)
		computed := style.Compute(e.styles, props["class"])
		base := computed.FlexBasis
		if base <= 0 && computed.Width > 0 {
			base = computed.Width
		}
		return computed.FlexGrow, base, base > 0
	case *ir.Conditional:
		if e.resolveTruthy != nil && !e.resolveTruthy(n.Expression) {
			return 0, 0, false
		}
		return e.flexMetricsForNode(n.Child)
	default:
		return 0, 0, false
	}
}

func layoutWrappedRow(boxes []Box, x, y, width, columnGap, rowGap float32, computed style.Computed) ([]Box, float32) {
	if len(boxes) == 0 {
		return boxes, 0
	}
	type line struct {
		start  int
		end    int
		width  float32
		height float32
	}
	var lines []line
	current := line{start: 0}
	for i, box := range boxes {
		boxWidth := flowWidth(box)
		nextWidth := boxWidth
		if current.end > current.start {
			nextWidth = current.width + columnGap + boxWidth
		}
		if current.end > current.start && nextWidth > width {
			lines = append(lines, current)
			current = line{start: i}
			nextWidth = boxWidth
		}
		current.end = i + 1
		current.width = nextWidth
		if h := flowHeight(box); h > current.height {
			current.height = h
		}
	}
	lines = append(lines, current)

	cursorY := y
	for _, line := range lines {
		cursorX := x + justifyOffset(width, line.width, computed.Justify)
		for i := line.start; i < line.end; i++ {
			targetOuterX := cursorX
			targetOuterY := cursorY + crossAxisOffset(line.height, flowHeight(boxes[i]), computed.AlignItems)
			translateBox(&boxes[i], targetOuterX-outerX(boxes[i]), targetOuterY-outerY(boxes[i]))
			cursorX += flowWidth(boxes[i]) + columnGap
		}
		cursorY += line.height + rowGap
	}
	cursorY -= rowGap
	return boxes, cursorY - y
}

func (e *engine) layoutGridChildren(nodes []ir.Node, x, y, width float32, computed style.Computed) ([]Box, float32) {
	columns := resolveTracks(computed.GridTemplateColumns, width, columnGap(computed))
	if len(columns) == 0 {
		columns = []float32{width}
	}
	var boxes []Box
	rowHeights := map[int]float32{}
	visibleIndex := 0
	for _, child := range nodes {
		index := visibleIndex
		column := index % len(columns)
		row := index / len(columns)
		columnPlacement, rowPlacement := e.gridPlacement(child)
		if columnPlacement.Start > 0 {
			column = columnPlacement.Start - 1
			if column < 0 {
				column = 0
			}
		}
		if rowPlacement.Start > 0 {
			row = rowPlacement.Start - 1
			if row < 0 {
				row = 0
			}
		}
		if column >= len(columns) {
			column = len(columns) - 1
		}
		childX := x
		for i := 0; i < column; i++ {
			childX += columns[i] + columnGap(computed)
		}
		childY := y + gridRowsHeight(rowHeights, computed.GridTemplateRows, row, rowGap(computed))
		childWidth := gridSpanSize(columns, column, columnPlacement.Span, columnGap(computed))
		childBox, ok := e.layoutNode(child, childX, childY, childWidth, 0)
		if !ok {
			continue
		}
		visibleIndex++
		boxes = append(boxes, childBox)
		rowHeight := flowHeight(childBox)
		if rowPlacement.Span > 1 {
			rowHeight = maxFloat(rowHeight, gridRowsHeight(map[int]float32{0: rowHeight}, computed.GridTemplateRows, rowPlacement.Span, rowGap(computed)))
		}
		if row < len(computed.GridTemplateRows) && computed.GridTemplateRows[row].Unit == "px" && computed.GridTemplateRows[row].Value > rowHeight {
			rowHeight = computed.GridTemplateRows[row].Value
		}
		if computed.GridAutoRows > rowHeight {
			rowHeight = computed.GridAutoRows
		}
		if rowHeight > rowHeights[row] {
			rowHeights[row] = rowHeight
		}
	}
	totalHeight := float32(0)
	for row := 0; ; row++ {
		height, ok := rowHeights[row]
		if !ok {
			break
		}
		if row < len(computed.GridTemplateRows) && computed.GridTemplateRows[row].Unit == "px" && computed.GridTemplateRows[row].Value > height {
			height = computed.GridTemplateRows[row].Value
		}
		if row > 0 {
			totalHeight += rowGap(computed)
		}
		totalHeight += height
	}
	return boxes, totalHeight
}

func (e *engine) gridPlacement(node ir.Node) (style.GridPlacement, style.GridPlacement) {
	elem, ok := node.(*ir.Element)
	if !ok {
		return style.GridPlacement{}, style.GridPlacement{}
	}
	props, _ := e.propsMap(elem.Props)
	computed := style.Compute(e.styles, props["class"])
	return computed.GridColumn, computed.GridRow
}

func applyFlexGrow(boxes []Box, remaining float32) {
	if remaining <= 0 {
		return
	}
	totalGrow := float32(0)
	for _, box := range boxes {
		totalGrow += box.Style.FlexGrow
	}
	if totalGrow <= 0 {
		return
	}
	used := float32(0)
	for i := range boxes {
		grow := remaining * boxes[i].Style.FlexGrow / totalGrow
		if i == len(boxes)-1 {
			grow = remaining - used
		}
		used += grow
		resizeBoxWidth(&boxes[i], boxes[i].Rect.Width+grow)
	}
}

func resizeBoxWidth(box *Box, width float32) {
	if width < 0 {
		width = 0
	}
	box.Rect.Width = width
}

func gridSpanSize(columns []float32, start, span int, gap float32) float32 {
	if start < 0 || start >= len(columns) {
		return 0
	}
	if span <= 1 {
		return columns[start]
	}
	end := start + span
	if end > len(columns) {
		end = len(columns)
	}
	width := float32(0)
	for i := start; i < end; i++ {
		if i > start {
			width += gap
		}
		width += columns[i]
	}
	return width
}

func gridRowsHeight(rowHeights map[int]float32, tracks []style.Track, beforeRow int, gap float32) float32 {
	total := float32(0)
	for row := 0; row < beforeRow; row++ {
		height := rowHeights[row]
		if row < len(tracks) && tracks[row].Unit == "px" && tracks[row].Value > height {
			height = tracks[row].Value
		}
		if row > 0 {
			total += gap
		}
		total += height
	}
	if beforeRow > 0 {
		total += gap
	}
	return total
}

func rowGap(computed style.Computed) float32 {
	if computed.RowGap != 0 {
		return computed.RowGap
	}
	return computed.Gap
}

func columnGap(computed style.Computed) float32 {
	if computed.ColumnGap != 0 {
		return computed.ColumnGap
	}
	return computed.Gap
}

func resolveTracks(tracks []style.Track, width, gap float32) []float32 {
	if len(tracks) == 0 {
		return nil
	}
	totalGap := gap * float32(len(tracks)-1)
	if totalGap < 0 {
		totalGap = 0
	}
	fixed := float32(0)
	fr := float32(0)
	out := make([]float32, len(tracks))
	for i, track := range tracks {
		switch track.Unit {
		case "px":
			out[i] = track.Value
			fixed += track.Value
		case "fr":
			fr += track.Value
		}
	}
	remaining := width - fixed - totalGap
	if remaining < 0 {
		remaining = 0
	}
	for i, track := range tracks {
		if track.Unit == "fr" {
			if fr == 0 {
				out[i] = 0
			} else {
				out[i] = remaining * track.Value / fr
			}
		}
	}
	return out
}

func isControl(tag string) bool {
	return tag == "input" || tag == "button"
}

func isSVGElement(tag string) bool {
	switch tag {
	case "svg", "g", "path", "circle", "ellipse", "rect", "line", "polyline", "polygon":
		return true
	default:
		return false
	}
}

func events(handlers []ir.EventHandler) []Event {
	var out []Event
	for _, handler := range handlers {
		out = append(out, Event{Event: handler.Event, Method: handler.Method})
	}
	return out
}

func (e *engine) layoutText(text string, x, y, width float32) Box {
	w, h := measureText(e.measurer, text, e.textStyle)
	displayText := text
	if width > 0 && w > width {
		displayText, w, h = wrapText(text, width, e.textStyle, e.measurer)
	}
	if e.textStyle.textAlign != "" && width > w {
		x += justifyOffset(width, w, e.textStyle.textAlign)
	}
	return Box{
		ID:     e.newID(),
		Kind:   "text",
		Text:   displayText,
		Lines:  lineBoxes(displayText, x, y, w, e.textStyle, e.measurer),
		Glyphs: glyphRuns(displayText, x, y, e.textStyle, e.measurer),
		Role:   "text",
		Style: Style{
			FontSize:   e.textStyle.fontSize,
			LineHeight: e.textStyle.lineHeight,
			TextAlign:  e.textStyle.textAlign,
			Opacity:    e.textStyle.opacity,
			Color:      e.textStyle.color,
		},
		Rect: Rect{
			X:      x,
			Y:      y,
			Width:  w,
			Height: h,
		},
	}
}

func defaultTextStyle(tag string) textStyle {
	switch tag {
	case "h1":
		return textStyle{fontSize: 28, lineHeight: 38}
	case "h2":
		return textStyle{fontSize: 24, lineHeight: 34}
	case "h3":
		return textStyle{fontSize: 20, lineHeight: 28}
	case "h4", "h5", "h6":
		return textStyle{fontSize: 18, lineHeight: 26}
	default:
		return textStyle{}
	}
}

func boxStyle(computed style.Computed) Style {
	return Style{
		Display:         computed.Display,
		FlexDirection:   computed.FlexDirection,
		FlexWrap:        computed.FlexWrap,
		AlignItems:      computed.AlignItems,
		Justify:         computed.Justify,
		Margin:          computed.Margin,
		PaddingLeft:     computed.PaddingLeft,
		FlexGrow:        computed.FlexGrow,
		FlexBasis:       computed.FlexBasis,
		FontSize:        computed.FontSize,
		LineHeight:      computed.LineHeight,
		TextAlign:       computed.TextAlign,
		BackgroundColor: computed.BackgroundColor,
		Opacity:         computed.Opacity,
		BorderWidth:     computed.BorderWidth,
		BorderWidthSet:  computed.BorderWidthSet,
		BorderColor:     computed.BorderColor,
		BorderRadius:    computed.BorderRadius,
		Color:           computed.Color,
		Overflow:        computed.Overflow,
	}
}

func clampSize(value, minValue, maxValue float32) float32 {
	if minValue > 0 && value < minValue {
		value = minValue
	}
	if maxValue > 0 && value > maxValue {
		value = maxValue
	}
	return value
}

func wrapText(text string, width float32, textStyle textStyle, measurer Measurer) (string, float32, float32) {
	charWidth, _ := measureText(measurer, "M", textStyle)
	lineHeight := measureLineHeight(measurer, textStyle)
	if charWidth <= 0 {
		charWidth = 8
	}
	maxRunes := max(1, int(width/charWidth))
	runes := []rune(text)
	if len(runes) <= maxRunes {
		measuredWidth, _ := measureText(measurer, text, textStyle)
		return text, measuredWidth, lineHeight
	}
	var lines []string
	for len(runes) > 0 {
		n := maxRunes
		if n > len(runes) {
			n = len(runes)
		}
		lines = append(lines, string(runes[:n]))
		runes = runes[n:]
	}
	return strings.Join(lines, "\n"), width, float32(len(lines)) * lineHeight
}

func lineBoxes(text string, x, y, width float32, textStyle textStyle, measurer Measurer) []LineBox {
	lineHeight := measureLineHeight(measurer, textStyle)
	baseline := lineHeightBaseline(lineHeight, textStyle.fontSize)
	lines := strings.Split(text, "\n")
	out := make([]LineBox, 0, len(lines))
	for i, line := range lines {
		lineWidth, _ := measureText(measurer, line, textStyle)
		if width > lineWidth {
			lineWidth = width
		}
		out = append(out, LineBox{
			Text:     line,
			X:        x,
			Y:        y + float32(i)*lineHeight,
			Width:    lineWidth,
			Height:   lineHeight,
			Baseline: baseline,
		})
	}
	return out
}

func glyphRuns(text string, x, y float32, textStyle textStyle, measurer Measurer) []GlyphRun {
	fontSize := textStyle.fontSize
	if fontSize <= 0 {
		fontSize = 16
	}
	lineHeight := measureLineHeight(measurer, textStyle)
	baseline := lineHeightBaseline(lineHeight, fontSize)
	lines := strings.Split(text, "\n")
	out := make([]GlyphRun, 0, len(lines))
	for i, line := range lines {
		if line == "" {
			continue
		}
		out = append(out, GlyphRun{
			Text:     line,
			Size:     fontSize,
			X:        x,
			Y:        y + float32(i)*lineHeight,
			Advance:  lineWidth(measurer, line, textStyle),
			Baseline: baseline,
		})
	}
	return out
}

func lineWidth(measurer Measurer, text string, textStyle textStyle) float32 {
	width, _ := measureText(measurer, text, textStyle)
	return width
}

func lineHeightBaseline(lineHeight, fontSize float32) float32 {
	if fontSize <= 0 {
		fontSize = 16
	}
	return (lineHeight-fontSize)*0.5 + fontSize*0.82
}

func flowWidth(box Box) float32 {
	return box.Rect.Width + box.Style.Margin*2
}

func flowHeight(box Box) float32 {
	return box.Rect.Height + box.Style.Margin*2
}

func outerX(box Box) float32 {
	return box.Rect.X - box.Style.Margin
}

func outerY(box Box) float32 {
	return box.Rect.Y - box.Style.Margin
}

func translateBox(box *Box, dx, dy float32) {
	if dx == 0 && dy == 0 {
		return
	}
	box.Rect.X += dx
	box.Rect.Y += dy
	for i := range box.Lines {
		box.Lines[i].X += dx
		box.Lines[i].Y += dy
	}
	for i := range box.Glyphs {
		box.Glyphs[i].X += dx
		box.Glyphs[i].Y += dy
	}
	for i := range box.Children {
		translateBox(&box.Children[i], dx, dy)
	}
}

func justifyOffset(width, contentWidth float32, justify string) float32 {
	remaining := width - contentWidth
	if remaining <= 0 {
		return 0
	}
	switch justify {
	case "center":
		return remaining / 2
	case "flex-end", "end", "right":
		return remaining
	default:
		return 0
	}
}

func crossAxisOffset(available, child float32, align string) float32 {
	remaining := available - child
	if remaining <= 0 {
		return 0
	}
	switch align {
	case "center":
		return remaining / 2
	case "flex-end", "end":
		return remaining
	default:
		return 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func roleForTag(tag string) string {
	switch tag {
	case "button":
		return "button"
	case "input":
		return "textbox"
	case "img":
		return "image"
	case "svg":
		return "image"
	case "ul", "ol":
		return "list"
	case "li":
		return "listitem"
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return "heading"
	case "p", "span":
		return "text"
	default:
		return "group"
	}
}

func roleForElement(tag string, props map[string]string) string {
	if tag == "input" && props["type"] == "checkbox" {
		return "checkbox"
	}
	return roleForTag(tag)
}

func (e *engine) newID() string {
	e.nextID++
	return fmt.Sprintf("l%d", e.nextID)
}

func (e *engine) propsMap(props []ir.Prop) (map[string]string, map[string]string) {
	out := map[string]string{}
	bindings := map[string]string{}
	for _, prop := range props {
		if prop.Bound {
			bindings[prop.Name] = prop.Binding
			if e.resolveProp != nil {
				out[prop.Name] = e.resolveProp(prop.Binding)
			} else {
				out[prop.Name] = prop.Binding
			}
		} else {
			out[prop.Name] = prop.Value
		}
	}
	if len(bindings) == 0 {
		bindings = nil
	}
	return out, bindings
}
