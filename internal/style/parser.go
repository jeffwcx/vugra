package style

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/tdewolff/parse/v2"
	tcss "github.com/tdewolff/parse/v2/css"
)

type BasePosition struct {
	Offset int
	Line   int
	Column int
}

func Parse(source string, base BasePosition) *Stylesheet {
	if base.Line == 0 {
		base.Line = 1
	}
	if base.Column == 0 {
		base.Column = 1
	}
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

func (p *parser) parse() *Stylesheet {
	var rules []Rule
	input := parse.NewInputString(p.source)
	defer input.Restore()
	cssParser := tcss.NewParser(input, false)
	var current *Rule
	for {
		grammar, tokenType, data := cssParser.Next()
		switch grammar {
		case tcss.BeginRulesetGrammar:
			rule := p.beginRule(cssParser)
			rules = append(rules, rule)
			current = &rules[len(rules)-1]
		case tcss.DeclarationGrammar, tcss.CustomPropertyGrammar:
			if current != nil {
				current.Declarations = append(current.Declarations, p.declaration(cssParser, data))
			}
		case tcss.EndRulesetGrammar:
			if current != nil {
				end := cssParser.Offset()
				if tokenType != tcss.RightBraceToken {
					end = len(p.source)
					p.error("style.missing_closing_brace", "missing closing brace in style rule", p.span(max(0, end-1), end))
				}
				current.Span.End = p.mapper.position(end)
				current = nil
			}
		case tcss.AtRuleGrammar, tcss.BeginAtRuleGrammar:
			start := max(0, cssParser.Offset()-len(data))
			p.error("style.unsupported_at_rule", fmt.Sprintf("unsupported at-rule %q", string(data)), p.span(start, cssParser.Offset()))
		case tcss.ErrorGrammar:
			err := cssParser.Err()
			if err == io.EOF {
				if current != nil {
					p.error("style.missing_closing_brace", "missing closing brace in style rule", p.span(max(0, cssParser.Offset()-1), len(p.source)))
				}
				return &Stylesheet{Rules: rules, Diagnostics: p.diags}
			}
			p.parseError(err, cssParser.Offset())
			return &Stylesheet{Rules: rules, Diagnostics: p.diags}
		}
	}
	return &Stylesheet{Rules: rules, Diagnostics: p.diags}
}

func (p *parser) beginRule(cssParser *tcss.Parser) Rule {
	selector := tokensString(cssParser.Values())
	open := cssParser.Offset()
	selectorStart := findSelectorStart(p.source, selector, open)
	rule := Rule{
		Selector:  selector,
		ClassName: classSelector(selector),
		Span:      p.span(selectorStart, open),
	}
	if rule.ClassName == "" {
		p.error("style.unsupported_selector", fmt.Sprintf("unsupported selector %q; only class selectors are supported", selector), p.span(selectorStart, open))
	}
	return rule
}

func (p *parser) declaration(cssParser *tcss.Parser, nameBytes []byte) Declaration {
	name := strings.ToLower(string(nameBytes))
	value := tokensString(cssParser.Values())
	end := cssParser.Offset()
	nameStart := findDeclarationNameStart(p.source, name, end)
	valueStart := nameStart + len(name)
	if colon := strings.IndexByte(p.source[valueStart:min(end, len(p.source))], ':'); colon >= 0 {
		valueStart += colon + 1
	}
	valueStart = skipSpace(p.source, valueStart, min(end, len(p.source)))
	valueEnd := valueStart + len(value)
	if valueEnd > len(p.source) || p.source[valueStart:valueEnd] != value {
		if idx := strings.Index(p.source[valueStart:min(end, len(p.source))], value); idx >= 0 {
			valueStart += idx
			valueEnd = valueStart + len(value)
		} else {
			valueEnd = end
		}
	}
	decl := Declaration{
		Name:      name,
		Value:     strings.TrimSpace(value),
		NameSpan:  p.span(nameStart, nameStart+len(name)),
		ValueSpan: p.span(valueStart, valueEnd),
	}
	if !supportedProperty(name) {
		p.warning("style.unsupported_property", fmt.Sprintf("unsupported CSS Profile v1 property %q; declaration is ignored", name), decl.NameSpan)
	}
	return decl
}

func Compute(sheet *Stylesheet, classList string) Computed {
	return ComputeWithTokens(sheet, classList, nil)
}

func ComputeWithTokens(sheet *Stylesheet, classList string, tokens SystemTokens) Computed {
	out := Computed{Display: "block", FlexDirection: "row"}
	if sheet == nil {
		return out
	}
	classes := map[string]struct{}{}
	for _, className := range strings.Fields(classList) {
		classes[className] = struct{}{}
	}
	for _, rule := range sheet.Rules {
		if _, ok := classes[rule.ClassName]; !ok {
			continue
		}
		for _, decl := range rule.Declarations {
			applyDeclaration(&out, decl, tokens)
		}
	}
	return out
}

func applyDeclaration(out *Computed, decl Declaration, tokens SystemTokens) {
	switch decl.Name {
	case "display":
		out.Display = decl.Value
	case "box-sizing":
		out.BoxSizing = decl.Value
	case "flex-direction":
		out.FlexDirection = decl.Value
	case "flex-wrap":
		out.FlexWrap = decl.Value
	case "gap":
		out.Gap = parsePixels(decl.Value)
		out.RowGap = out.Gap
		out.ColumnGap = out.Gap
	case "row-gap":
		out.RowGap = parsePixels(decl.Value)
	case "column-gap":
		out.ColumnGap = parsePixels(decl.Value)
	case "padding":
		out.Padding = parsePixelsWithTokens(decl.Value, tokens)
	case "padding-left":
		out.PaddingLeft = parsePixelsWithTokens(decl.Value, tokens)
	case "margin":
		out.Margin = parsePixelsWithTokens(decl.Value, tokens)
	case "width":
		out.Width, out.WidthPercent = parseLength(decl.Value)
	case "height":
		out.Height, out.HeightPercent = parseLength(decl.Value)
	case "min-width":
		out.MinWidth = parsePixelsWithTokens(decl.Value, tokens)
	case "max-width":
		out.MaxWidth = parsePixelsWithTokens(decl.Value, tokens)
	case "min-height":
		out.MinHeight = parsePixelsWithTokens(decl.Value, tokens)
	case "max-height":
		out.MaxHeight = parsePixelsWithTokens(decl.Value, tokens)
	case "align-items":
		out.AlignItems = decl.Value
	case "align-self":
		out.AlignSelf = decl.Value
	case "justify-content":
		out.Justify = decl.Value
	case "flex":
		out.FlexGrow, out.FlexShrink, out.FlexBasis = parseFlex(decl.Value)
	case "flex-grow":
		out.FlexGrow = parsePositiveNumber(decl.Value)
	case "flex-shrink":
		out.FlexShrink = parsePositiveNumber(decl.Value)
	case "flex-basis":
		out.FlexBasis = parsePixelsWithTokens(decl.Value, tokens)
	case "grid-template-columns":
		out.GridTemplateColumns = parseTracks(decl.Value)
	case "grid-template-rows":
		out.GridTemplateRows = parseTracks(decl.Value)
	case "grid-auto-rows":
		out.GridAutoRows = parsePixelsWithTokens(decl.Value, tokens)
	case "grid-column":
		out.GridColumn = parseGridPlacement(decl.Value)
	case "grid-row":
		out.GridRow = parseGridPlacement(decl.Value)
	case "font-size":
		out.FontSize = parsePixelsWithTokens(decl.Value, tokens)
	case "font-family":
		out.FontFamily = decl.Value
	case "font-weight":
		out.FontWeight = decl.Value
	case "line-height":
		out.LineHeight = parsePixelsWithTokens(decl.Value, tokens)
	case "text-align":
		out.TextAlign = decl.Value
	case "white-space":
		out.WhiteSpace = decl.Value
	case "background":
		out.Background = decl.Value
		if color := backgroundColorValue(decl.Value); color != "" {
			out.BackgroundColor = color
		}
	case "background-color":
		out.BackgroundColor = decl.Value
	case "opacity":
		out.Opacity = parsePositiveNumber(decl.Value)
	case "border":
		out.BorderWidth, out.BorderStyle, out.BorderColor = parseBorder(decl.Value)
		out.BorderWidthSet = true
	case "border-width":
		out.BorderWidth = parsePixelsWithTokens(decl.Value, tokens)
		out.BorderWidthSet = true
	case "border-color":
		out.BorderColor = decl.Value
	case "border-style":
		out.BorderStyle = decl.Value
	case "border-radius":
		out.BorderRadius = parsePixelsWithTokens(decl.Value, tokens)
	case "color":
		out.Color = decl.Value
	case "overflow":
		out.Overflow = decl.Value
		out.OverflowX = decl.Value
		out.OverflowY = decl.Value
	case "overflow-x":
		out.OverflowX = decl.Value
	case "overflow-y":
		out.OverflowY = decl.Value
	}
}

func parseBorder(value string) (float32, string, string) {
	width := float32(0)
	style := ""
	color := ""
	for _, field := range strings.Fields(value) {
		if strings.HasSuffix(field, "px") {
			width = parsePixels(field)
			continue
		}
		switch field {
		case "none", "solid", "dashed", "dotted":
			style = field
			continue
		}
		if strings.HasPrefix(field, "#") {
			color = field
		}
	}
	return width, style, color
}

func backgroundColorValue(value string) string {
	for _, field := range strings.Fields(value) {
		if strings.HasPrefix(field, "#") && (len(field) == 4 || len(field) == 7) {
			return field
		}
	}
	return ""
}

func parsePixels(value string) float32 {
	return parsePixelsWithTokens(value, nil)
}

func parsePixelsWithTokens(value string, tokens SystemTokens) float32 {
	value = strings.TrimSpace(strings.TrimSuffix(value, "px"))
	if token, fallback, ok := parseEnvToken(value); ok {
		if resolved, exists := tokens[token]; exists {
			return resolved
		}
		return parsePixelsWithTokens(fallback, tokens)
	}
	n, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0
	}
	return float32(n)
}

func parseEnvToken(value string) (name string, fallback string, ok bool) {
	if !strings.HasPrefix(value, "env(") || !strings.HasSuffix(value, ")") {
		return "", "", false
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "env("), ")"))
	if body == "" {
		return "", "", false
	}
	name = body
	if comma := strings.IndexByte(body, ','); comma >= 0 {
		name = strings.TrimSpace(body[:comma])
		fallback = strings.TrimSpace(body[comma+1:])
	}
	if name == "" {
		return "", "", false
	}
	return name, fallback, true
}

func parseLength(value string) (px float32, percent float32) {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "%") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 32)
		if err != nil || n <= 0 {
			return 0, 0
		}
		return 0, float32(n)
	}
	return parsePixels(value), 0
}

func parsePositiveInt(value string) int {
	n := parsePositiveNumber(value)
	if n <= 0 {
		return 0
	}
	return int(math.Round(float64(n)))
}

func parsePositiveNumber(value string) float32 {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil || n < 0 {
		return 0
	}
	return float32(n)
}

func parseFlex(value string) (float32, float32, float32) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, 0, 0
	}
	grow := parsePositiveNumber(fields[0])
	shrink := float32(0)
	basis := float32(0)
	for _, field := range fields[1:] {
		switch {
		case strings.HasSuffix(field, "px"):
			basis = parsePixels(field)
		default:
			shrink = parsePositiveNumber(field)
		}
	}
	return grow, shrink, basis
}

func parseGridPlacement(value string) GridPlacement {
	value = strings.TrimSpace(value)
	if value == "" {
		return GridPlacement{}
	}
	parts := strings.Split(value, "/")
	start := parsePositiveInt(parts[0])
	placement := GridPlacement{Start: start}
	if len(parts) > 1 {
		end := strings.TrimSpace(parts[1])
		if strings.HasPrefix(end, "span ") {
			placement.Span = parsePositiveInt(strings.TrimPrefix(end, "span "))
		} else if stop := parsePositiveInt(end); stop > start && start > 0 {
			placement.Span = stop - start
		}
	}
	if placement.Span <= 0 && placement.Start > 0 {
		placement.Span = 1
	}
	return placement
}

func parseTracks(value string) []Track {
	fields := strings.Fields(value)
	tracks := make([]Track, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		switch {
		case strings.HasSuffix(field, "px"):
			tracks = append(tracks, Track{Unit: "px", Value: parsePixels(field)})
		case strings.HasSuffix(field, "fr"):
			n, err := strconv.ParseFloat(strings.TrimSuffix(field, "fr"), 32)
			if err != nil || n <= 0 {
				n = 1
			}
			tracks = append(tracks, Track{Unit: "fr", Value: float32(n)})
		default:
			if px := parsePixels(field); px > 0 {
				tracks = append(tracks, Track{Unit: "px", Value: px})
			}
		}
	}
	return tracks
}

func classSelector(selector string) string {
	selector = strings.TrimSpace(selector)
	if !strings.HasPrefix(selector, ".") {
		return ""
	}
	name := strings.TrimPrefix(selector, ".")
	if name == "" || strings.ContainsAny(name, " \t\r\n>+~#:[") {
		return ""
	}
	return name
}

func supportedProperty(name string) bool {
	switch name {
	case "display",
		"box-sizing",
		"width",
		"height",
		"min-width",
		"min-height",
		"max-width",
		"max-height",
		"margin",
		"padding",
		"padding-left",
		"flex-direction",
		"flex-wrap",
		"flex",
		"flex-grow",
		"flex-shrink",
		"flex-basis",
		"align-items",
		"align-self",
		"justify-content",
		"gap",
		"row-gap",
		"column-gap",
		"background",
		"background-color",
		"color",
		"opacity",
		"border",
		"border-width",
		"border-color",
		"border-style",
		"border-radius",
		"font-family",
		"font-size",
		"font-weight",
		"line-height",
		"text-align",
		"white-space",
		"overflow",
		"overflow-x",
		"overflow-y",
		"grid-template-columns",
		"grid-template-rows",
		"grid-auto-rows",
		"grid-column",
		"grid-row":
		return true
	default:
		return false
	}
}

func tokensString(tokens []tcss.Token) string {
	var b bytes.Buffer
	for _, token := range tokens {
		b.Write(token.Data)
	}
	return strings.TrimSpace(b.String())
}

func findSelectorStart(source, selector string, end int) int {
	if selector == "" {
		return max(0, end)
	}
	limit := min(end, len(source))
	if idx := strings.LastIndex(source[:limit], selector); idx >= 0 {
		return idx
	}
	return max(0, end-len(selector)-1)
}

func findDeclarationNameStart(source, name string, end int) int {
	limit := min(end, len(source))
	if idx := strings.LastIndex(source[:limit], name); idx >= 0 {
		return idx
	}
	return max(0, end-len(name))
}

func (p *parser) parseError(err error, offset int) {
	code := "style.parse_error"
	message := err.Error()
	if strings.Contains(message, "expected colon") {
		code = "style.missing_colon"
		message = "style declaration is missing ':'"
	} else if strings.Contains(message, "unexpected ending") {
		code = "style.missing_closing_brace"
		message = "missing closing brace in style rule"
	} else if strings.Contains(message, "comment") {
		code = "style.unclosed_comment"
		message = "unclosed style comment"
	}
	p.error(code, message, p.span(max(0, offset-1), min(len(p.source), offset)))
}

func (p *parser) error(code, message string, span Span) {
	p.diags = append(p.diags, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     span,
	})
}

func (p *parser) warning(code, message string, span Span) {
	p.diags = append(p.diags, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "warning",
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
