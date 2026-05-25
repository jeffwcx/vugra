package rustanalysis

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func Analyze(source string, base BasePosition) Metadata {
	if base.Line == 0 {
		base.Line = 1
	}
	if base.Column == 0 {
		base.Column = 1
	}
	a := analyzer{
		source: source,
		mapper: newSourceMap(source, base),
	}
	a.inspect()
	return a.metadata
}

type analyzer struct {
	source   string
	mapper   sourceMap
	metadata Metadata
}

func (a *analyzer) inspect() {
	a.inspectState()
	a.inspectMethods()
	if a.metadata.State == nil {
		a.error("rustanalysis.missing_state", "missing State struct", 0, 0)
	}
	sort.Slice(a.metadata.Methods, func(i, j int) bool {
		return a.metadata.Methods[i].Name < a.metadata.Methods[j].Name
	})
}

func (a *analyzer) inspectState() {
	for i := 0; i < len(a.source); {
		idx := strings.Index(a.source[i:], "struct")
		if idx < 0 {
			return
		}
		start := i + idx
		if !tokenBoundary(a.source, start-1) || !tokenBoundary(a.source, start+len("struct")) {
			i = start + len("struct")
			continue
		}
		nameStart := skipSpaceAndVisibility(a.source, start+len("struct"))
		name, nameEnd := readIdent(a.source, nameStart)
		if name != "State" {
			i = max(start+len("struct"), nameEnd)
			continue
		}
		bodyStart := strings.IndexByte(a.source[nameEnd:], '{')
		if bodyStart < 0 {
			a.error("rustanalysis.invalid_rust", "State struct is missing a body", start, nameEnd)
			return
		}
		bodyStart += nameEnd
		bodyEnd, ok := findMatchingBrace(a.source, bodyStart)
		if !ok {
			a.error("rustanalysis.invalid_rust", "State struct body is not closed", bodyStart, len(a.source))
			return
		}
		state := &StateMetadata{TypeName: "State"}
		for _, field := range parseFields(a.source, bodyStart+1, bodyEnd, a.mapper) {
			state.Fields = append(state.Fields, field)
		}
		a.metadata.State = state
		return
	}
}

func (a *analyzer) inspectMethods() {
	for i := 0; i < len(a.source); {
		idx := strings.Index(a.source[i:], "impl")
		if idx < 0 {
			return
		}
		start := i + idx
		if !tokenBoundary(a.source, start-1) || !tokenBoundary(a.source, start+len("impl")) {
			i = start + len("impl")
			continue
		}
		nameStart := skipSpace(a.source, start+len("impl"))
		name, nameEnd := readIdent(a.source, nameStart)
		if name != "State" {
			i = max(start+len("impl"), nameEnd)
			continue
		}
		bodyStart := strings.IndexByte(a.source[nameEnd:], '{')
		if bodyStart < 0 {
			a.error("rustanalysis.invalid_rust", "impl State is missing a body", start, nameEnd)
			return
		}
		bodyStart += nameEnd
		bodyEnd, ok := findMatchingBrace(a.source, bodyStart)
		if !ok {
			a.error("rustanalysis.invalid_rust", "impl State body is not closed", bodyStart, len(a.source))
			return
		}
		a.metadata.Methods = append(a.metadata.Methods, parseMethods(a.source, bodyStart+1, bodyEnd, a.mapper)...)
		i = bodyEnd + 1
	}
}

func parseFields(source string, start, end int, mapper sourceMap) []Field {
	var fields []Field
	i := start
	for i < end {
		i = skipSpace(source, i)
		if i >= end {
			break
		}
		for i < end && source[i] == '#' {
			next := strings.IndexByte(source[i:end], '\n')
			if next < 0 {
				return fields
			}
			i += next + 1
			i = skipSpace(source, i)
		}
		fieldStart := i
		exported := false
		if strings.HasPrefix(source[i:end], "pub") && tokenBoundary(source, i+3) {
			exported = true
			i = skipSpace(source, i+3)
			if i < end && source[i] == '(' {
				close := strings.IndexByte(source[i:end], ')')
				if close < 0 {
					return fields
				}
				i = skipSpace(source, i+close+1)
			}
		}
		name, nameEnd := readIdent(source, i)
		if name == "" {
			i = skipField(source, i, end)
			continue
		}
		i = skipSpace(source, nameEnd)
		if i >= end || source[i] != ':' {
			i = skipField(source, i, end)
			continue
		}
		i++
		typeStart := skipSpace(source, i)
		typeEnd := scanTypeEnd(source, typeStart, end)
		typ := strings.TrimSpace(source[typeStart:typeEnd])
		if typ != "" {
			fields = append(fields, Field{
				Name:     name,
				Alias:    lowerFirst(name),
				Type:     typ,
				IsSignal: isSignalType(typ),
				Span:     mapper.span(fieldStart, typeEnd),
				TypeSpan: mapper.span(typeStart, typeEnd),
				Exported: exported || isUpperFirst(name),
			})
		}
		i = skipField(source, typeEnd, end)
	}
	return fields
}

func parseMethods(source string, start, end int, mapper sourceMap) []Method {
	var methods []Method
	for i := start; i < end; {
		idx := strings.Index(source[i:end], "fn")
		if idx < 0 {
			return methods
		}
		fnStart := i + idx
		if !tokenBoundary(source, fnStart-1) || !tokenBoundary(source, fnStart+2) {
			i = fnStart + 2
			continue
		}
		declStart := fnStart
		pubStart := previousNonSpaceTokenStart(source, start, fnStart)
		exported := false
		if pubStart >= start && strings.HasPrefix(source[pubStart:fnStart], "pub") {
			declStart = pubStart
			exported = true
		}
		nameStart := skipSpace(source, fnStart+2)
		name, nameEnd := readIdent(source, nameStart)
		if name == "" {
			i = fnStart + 2
			continue
		}
		paramsStart := skipSpace(source, nameEnd)
		if paramsStart >= end || source[paramsStart] != '(' {
			i = nameEnd
			continue
		}
		paramsEnd, ok := findMatchingParen(source, paramsStart)
		if !ok || paramsEnd > end {
			i = nameEnd
			continue
		}
		bodyStart := strings.IndexByte(source[paramsEnd:end], '{')
		methodEnd := paramsEnd
		if bodyStart >= 0 {
			bodyStart += paramsEnd
			if close, ok := findMatchingBrace(source, bodyStart); ok {
				methodEnd = close + 1
			}
		}
		params := strings.TrimSpace(source[paramsStart+1 : paramsEnd])
		methods = append(methods, Method{
			Name:      name,
			Receiver:  "State",
			Span:      mapper.span(declStart, methodEnd),
			NameSpan:  mapper.span(nameStart, nameEnd),
			Exported:  exported,
			OnPointer: strings.HasPrefix(params, "&mut self") || strings.HasPrefix(params, "&self") || params == "self",
			EventArg:  hasEventArg(params),
		})
		i = max(methodEnd, nameEnd)
	}
	return methods
}

func skipSpaceAndVisibility(source string, i int) int {
	i = skipSpace(source, i)
	if strings.HasPrefix(source[i:], "pub") && tokenBoundary(source, i+3) {
		i = skipSpace(source, i+3)
	}
	return i
}

func previousNonSpaceTokenStart(source string, start, end int) int {
	i := end - 1
	for i >= start && unicode.IsSpace(rune(source[i])) {
		i--
	}
	tokenEnd := i + 1
	for i >= start && isIdentChar(rune(source[i])) {
		i--
	}
	if strings.TrimSpace(source[i+1:tokenEnd]) == "pub" {
		return i + 1
	}
	return end
}

func skipField(source string, i, end int) int {
	depth := 0
	for i < end {
		switch source[i] {
		case '<', '[', '(', '{':
			depth++
		case '>', ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				return i + 1
			}
		case '\n':
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return i
}

func scanTypeEnd(source string, i, end int) int {
	depth := 0
	for i < end {
		switch source[i] {
		case '<', '[', '(', '{':
			depth++
		case '>', ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case ',', '\n':
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return i
}

func findMatchingBrace(source string, open int) (int, bool) {
	return findMatching(source, open, '{', '}')
}

func findMatchingParen(source string, open int) (int, bool) {
	return findMatching(source, open, '(', ')')
}

func findMatching(source string, open int, left, right byte) (int, bool) {
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case left:
			depth++
		case right:
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func readIdent(source string, i int) (string, int) {
	if i >= len(source) || !isIdentStart(rune(source[i])) {
		return "", i
	}
	start := i
	i++
	for i < len(source) && isIdentChar(rune(source[i])) {
		i++
	}
	return source[start:i], i
}

func skipSpace(source string, i int) int {
	for i < len(source) && unicode.IsSpace(rune(source[i])) {
		i++
	}
	return i
}

func tokenBoundary(source string, i int) bool {
	if i < 0 || i >= len(source) {
		return true
	}
	return !isIdentChar(rune(source[i]))
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func isUpperFirst(value string) bool {
	if value == "" {
		return false
	}
	r, _ := utf8DecodeRuneInString(value)
	return unicode.IsUpper(r)
}

func isSignalType(typ string) bool {
	return strings.Contains(typ, "Signal<") || strings.Contains(typ, "signal::") || strings.Contains(typ, "SignalCell")
}

func hasEventArg(params string) bool {
	if params == "" {
		return false
	}
	for _, part := range strings.Split(params, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "Event") && !strings.Contains(part, "self") {
			return true
		}
	}
	return false
}

func (a *analyzer) error(code, message string, start, end int) {
	a.metadata.Diagnostics = append(a.metadata.Diagnostics, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     a.mapper.span(start, end),
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func utf8DecodeRuneInString(value string) (rune, int) {
	for _, r := range value {
		return r, len(string(r))
	}
	return 0, 0
}

func Format(metadata Metadata) string {
	return fmt.Sprintf("state=%v methods=%d diagnostics=%d", metadata.State != nil, len(metadata.Methods), len(metadata.Diagnostics))
}
