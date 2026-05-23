package template

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"github.com/tdewolff/parse/v2"
	thtml "github.com/tdewolff/parse/v2/html"
)

var intrinsicTags = map[string]struct{}{
	"div":       {},
	"span":      {},
	"p":         {},
	"button":    {},
	"input":     {},
	"img":       {},
	"svg":       {},
	"g":         {},
	"path":      {},
	"circle":    {},
	"ellipse":   {},
	"rect":      {},
	"line":      {},
	"polyline":  {},
	"polygon":   {},
	"label":     {},
	"h1":        {},
	"h2":        {},
	"h3":        {},
	"h4":        {},
	"h5":        {},
	"h6":        {},
	"ul":        {},
	"ol":        {},
	"li":        {},
	"slot":      {},
	"template":  {},
	"component": {},
}

type BasePosition struct {
	Offset int
	Line   int
	Column int
}

func Parse(source string, baseOffset int) *Document {
	return ParseWithBase(source, BasePosition{Offset: baseOffset, Line: 1, Column: 1})
}

func ParseWithBase(source string, base BasePosition) *Document {
	p := parser{
		source: source,
		mapper: newSourceMap(source, base),
	}
	return p.parse()
}

type parser struct {
	source string
	mapper sourceMap
	diags  []Diagnostic
}

func (p *parser) parse() *Document {
	var roots []Node
	var stack []*Element

	addNode := func(node Node) {
		if len(stack) == 0 {
			roots = append(roots, node)
			return
		}
		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, node)
	}

	input := parse.NewInputString(p.source)
	defer input.Restore()
	lexer := thtml.NewTemplateLexer(input, thtml.GoTemplate)
	var pending *Element
	for {
		tokenType, data := lexer.Next()
		start := input.Offset() - len(data)
		end := input.Offset()
		switch tokenType {
		case thtml.ErrorToken:
			if err := lexer.Err(); err != nil && err != io.EOF {
				p.error("template.malformed_tag", err.Error(), p.span(max(0, start), min(len(p.source), end)))
			}
			if pending != nil {
				p.error("template.malformed_tag", "unterminated start tag", pending.StartTag)
				pending = nil
			}
			goto done
		case thtml.CommentToken:
			addNode(&Comment{Value: commentValue(string(data)), Span: p.span(start, end)})
		case thtml.TextToken:
			nodes := p.textNodes(string(data), start)
			for _, node := range nodes {
				addNode(node)
			}
		case thtml.TemplateToken:
			if interp, ok := p.interpolationFromToken(string(data), start, end); ok {
				addNode(interp)
			}
		case thtml.SVGToken:
			addNode(p.svgElementFromToken(string(data), start, end))
		case thtml.StartTagToken:
			pending = p.startElement(lexer, start, end)
		case thtml.AttributeToken:
			if pending == nil {
				p.error("template.malformed_tag", "attribute outside start tag", p.span(start, end))
				continue
			}
			pending.Attrs = append(pending.Attrs, p.attributeFromToken(lexer, start, end))
		case thtml.StartTagCloseToken, thtml.StartTagVoidToken:
			if pending == nil {
				continue
			}
			pending.StartTag = p.span(pending.StartTag.Start.Offset-p.mapper.base.Offset, end)
			pending.Span = pending.StartTag
			addNode(pending)
			selfClosing := tokenType == thtml.StartTagVoidToken
			if !selfClosing && !isVoidElement(pending.Tag) {
				stack = append(stack, pending)
			}
			pending = nil
		case thtml.EndTagToken:
			p.closeElement(strings.ToLower(string(lexer.Text())), p.span(start, end), &stack)
		}
	}

done:
	for i := len(stack) - 1; i >= 0; i-- {
		elem := stack[i]
		p.error("template.missing_closing_tag", fmt.Sprintf("missing </%s> closing tag", elem.Tag), elem.StartTag)
	}

	return &Document{
		Nodes:       roots,
		Diagnostics: p.diags,
	}
}

func (p *parser) svgElementFromToken(data string, start, end int) *Element {
	elem := &Element{
		Tag:      "svg",
		RawTag:   "svg",
		Span:     p.span(start, end),
		StartTag: p.span(start, svgStartTagEnd(data, start)),
		EndTag:   ptrSpan(p.span(max(start, end-len("</svg>")), end)),
		Attrs: []Attribute{{
			Name:     "__raw_svg",
			Value:    data,
			Kind:     AttrStatic,
			HasValue: true,
			NameSpan: p.span(start, start),
		}},
	}
	decoder := xml.NewDecoder(strings.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return elem
		}
		startElement, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		for _, attr := range startElement.Attr {
			if attr.Name.Local == "" {
				continue
			}
			elem.Attrs = append(elem.Attrs, Attribute{
				Name:     attr.Name.Local,
				Value:    attr.Value,
				Kind:     AttrStatic,
				HasValue: true,
				NameSpan: p.span(start, start),
			})
		}
		return elem
	}
}

func svgStartTagEnd(data string, offset int) int {
	end := strings.Index(data, ">")
	if end < 0 {
		return offset + len(data)
	}
	return offset + end + 1
}

func (p *parser) startElement(lexer *thtml.Lexer, start, end int) *Element {
	rawTag := p.rawTagName(start, end)
	if rawTag == "" {
		rawTag = string(lexer.Text())
	}
	tag := strings.ToLower(rawTag)
	if _, ok := intrinsicTags[tag]; !ok && rawTag == tag && !strings.Contains(tag, "-") {
		nameStart := start + 1
		p.error("template.unsupported_tag", fmt.Sprintf("unsupported intrinsic tag <%s>", tag), p.span(nameStart, nameStart+len(tag)))
	}
	span := p.span(start, end)
	return &Element{
		Tag:      tag,
		RawTag:   rawTag,
		Span:     span,
		StartTag: span,
	}
}

func (p *parser) rawTagName(start, end int) string {
	if start < 0 || start >= len(p.source) {
		return ""
	}
	if end > len(p.source) {
		end = len(p.source)
	}
	i := start
	if p.source[i] == '<' {
		i++
	}
	for i < end && unicode.IsSpace(rune(p.source[i])) {
		i++
	}
	nameStart := i
	for i < end {
		r := rune(p.source[i])
		if unicode.IsSpace(r) || r == '/' || r == '>' {
			break
		}
		i++
	}
	if i <= nameStart {
		return ""
	}
	return p.source[nameStart:i]
}

func (p *parser) closeElement(name string, span Span, stack *[]*Element) {
	if len(*stack) == 0 {
		p.error("template.unexpected_closing_tag", fmt.Sprintf("unexpected closing </%s> tag", name), span)
		return
	}
	top := (*stack)[len(*stack)-1]
	if top.Tag != name {
		p.error("template.mismatched_closing_tag", fmt.Sprintf("expected closing </%s> tag, got </%s>", top.Tag, name), span)
		return
	}
	top.EndTag = &span
	top.Span.End = span.End
	*stack = (*stack)[:len(*stack)-1]
}

func (p *parser) attributeFromToken(lexer *thtml.Lexer, start, end int) Attribute {
	name := string(lexer.AttrKey())
	rawValue := string(lexer.AttrVal())
	nameStart := findAttributeNameStart(p.source, name, start, end)
	attr := Attribute{
		Name:     name,
		Kind:     AttrStatic,
		NameSpan: p.span(nameStart, nameStart+len(name)),
	}
	if rawValue != "" {
		attr.HasValue = true
		value, valueStart, valueEnd, ok := normalizeAttrValue(rawValue, p.source, start, end)
		attr.Value = value
		attr.ValueSpan = ptrSpan(p.span(valueStart, valueEnd))
		if !ok {
			p.error("template.malformed_tag", "unterminated quoted attribute value", p.span(valueStart, valueEnd))
		}
	}
	attr.applyDirective()
	return attr
}

func (p *parser) textNodes(text string, offset int) []Node {
	var nodes []Node
	for len(text) > 0 {
		rel := strings.Index(text, "{{")
		if rel < 0 {
			nodes = append(nodes, &Text{Value: text, Span: p.span(offset, offset+len(text))})
			break
		}
		if rel > 0 {
			nodes = append(nodes, &Text{Value: text[:rel], Span: p.span(offset, offset+rel)})
		}
		closeRel := strings.Index(text[rel+2:], "}}")
		if closeRel < 0 {
			span := p.span(offset+rel, offset+len(text))
			p.error("template.unclosed_interpolation", "unclosed interpolation", span)
			break
		}
		endRel := rel + 2 + closeRel + 2
		interp, ok := p.interpolationFromToken(text[rel:endRel], offset+rel, offset+endRel)
		if ok {
			nodes = append(nodes, interp)
		}
		offset += endRel
		text = text[endRel:]
	}
	return nodes
}

func (p *parser) interpolationFromToken(token string, start, end int) (*Interpolation, bool) {
	if !strings.HasPrefix(token, "{{") || !strings.HasSuffix(token, "}}") {
		p.error("template.unclosed_interpolation", "unclosed interpolation", p.span(start, end))
		return nil, false
	}
	exprStart := start + 2
	exprEnd := end - 2
	trimStart, trimEnd := trimSpaceRange(p.source, exprStart, exprEnd)
	return &Interpolation{
		Expression: strings.TrimSpace(p.source[exprStart:exprEnd]),
		Span:       p.span(start, end),
		ExprSpan:   p.span(trimStart, trimEnd),
	}, true
}

func (a *Attribute) applyDirective() {
	switch {
	case a.Name == "v-if":
		a.Kind = AttrIf
		a.Directive = &Directive{Name: "if", Expression: a.Value}
	case a.Name == "v-for":
		a.Kind = AttrFor
		a.Directive = &Directive{Name: "for", Expression: a.Value}
	case a.Name == "v-model":
		a.Kind = AttrModel
		a.Arg = "modelValue"
		a.Directive = &Directive{Name: "model", Argument: a.Arg, Expression: a.Value}
	case strings.HasPrefix(a.Name, "v-model:") && len(a.Name) > len("v-model:"):
		a.Kind = AttrModel
		a.Arg = a.Name[len("v-model:"):]
		a.Directive = &Directive{Name: "model", Argument: a.Arg, Expression: a.Value}
	case a.Name == "slot":
		a.Kind = AttrSlot
		a.Arg = a.Value
		a.Directive = &Directive{Name: "slot", Argument: a.Arg, Expression: a.Value}
	case a.Name == "v-slot":
		a.Kind = AttrSlot
		a.Arg = "default"
		a.Directive = &Directive{Name: "slot", Argument: a.Arg, Expression: a.Value}
	case strings.HasPrefix(a.Name, ":") && len(a.Name) > 1:
		a.Kind = AttrBoundProp
		a.Arg = a.Name[1:]
		a.Directive = &Directive{Name: "bind", Argument: a.Arg, Expression: a.Value}
	case strings.HasPrefix(a.Name, "#") && len(a.Name) > 1:
		a.Kind = AttrSlot
		a.Arg = a.Name[1:]
		a.Directive = &Directive{Name: "slot", Argument: a.Arg, Expression: a.Value}
	case strings.HasPrefix(a.Name, "v-slot:") && len(a.Name) > len("v-slot:"):
		a.Kind = AttrSlot
		a.Arg = a.Name[len("v-slot:"):]
		a.Directive = &Directive{Name: "slot", Argument: a.Arg, Expression: a.Value}
	case strings.HasPrefix(a.Name, "@") && len(a.Name) > 1:
		a.Kind = AttrEvent
		a.Arg = a.Name[1:]
		a.Directive = &Directive{Name: "on", Argument: a.Arg, Expression: a.Value}
	}
}

func (p *parser) error(code, message string, span Span) {
	p.diags = append(p.diags, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     span,
	})
}

func (p *parser) span(start, end int) Span {
	return Span{
		Start: p.mapper.position(start),
		End:   p.mapper.position(end),
	}
}

func skipSpace(source string, offset, end int) int {
	i := offset
	for i < end && unicode.IsSpace(rune(source[i])) {
		i++
	}
	return i
}

func trimSpaceRange(source string, start, end int) (int, int) {
	for start < end && unicode.IsSpace(rune(source[start])) {
		start++
	}
	for end > start && unicode.IsSpace(rune(source[end-1])) {
		end--
	}
	return start, end
}

func commentValue(raw string) string {
	if strings.HasPrefix(raw, "<!--") && strings.HasSuffix(raw, "-->") {
		return raw[len("<!--") : len(raw)-len("-->")]
	}
	return raw
}

func findAttributeNameStart(source, name string, start, end int) int {
	if name == "" {
		return start
	}
	if idx := strings.Index(source[start:end], name); idx >= 0 {
		return start + idx
	}
	return start
}

func normalizeAttrValue(rawValue, source string, start, end int) (string, int, int, bool) {
	if rawValue == "" {
		return "", end, end, true
	}
	valueStart := strings.Index(source[start:end], rawValue)
	if valueStart < 0 {
		return rawValue, start, end, true
	}
	valueStart += start
	valueEnd := valueStart + len(rawValue)
	if len(rawValue) >= 2 {
		quote := rawValue[0]
		if (quote == '"' || quote == '\'') && rawValue[len(rawValue)-1] == quote {
			return rawValue[1 : len(rawValue)-1], valueStart + 1, valueEnd - 1, true
		}
		if quote == '"' || quote == '\'' {
			return strings.TrimPrefix(rawValue, string(quote)), valueStart + 1, valueEnd, false
		}
	}
	return rawValue, valueStart, valueEnd, true
}

func ptrSpan(span Span) *Span {
	return &span
}

func isVoidElement(tag string) bool {
	return tag == "input" || tag == "img"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type sourceMap struct {
	base       BasePosition
	lineStarts []int
}

func newSourceMap(source string, base BasePosition) sourceMap {
	if base.Line == 0 {
		base.Line = 1
	}
	if base.Column == 0 {
		base.Column = 1
	}
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
