package sfc

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func Parse(path string, source []byte) *File {
	return ParseString(path, string(source))
}

func ParseString(path string, source string) *File {
	p := parser{
		path:   path,
		source: source,
		mapper: newSourceMap(source),
	}
	return p.parse()
}

type parser struct {
	path   string
	source string
	mapper sourceMap
	diags  []Diagnostic
}

func (p *parser) parse() *File {
	file := &File{Path: p.path}

	for i := 0; i < len(p.source); {
		next := strings.IndexByte(p.source[i:], '<')
		if next < 0 {
			break
		}
		tagStart := i + next

		if strings.HasPrefix(p.source[tagStart:], "<!--") {
			end := strings.Index(p.source[tagStart+4:], "-->")
			if end < 0 {
				p.error("sfc.unclosed_comment", "unclosed comment", tagStart, len(p.source))
				break
			}
			i = tagStart + 4 + end + len("-->")
			continue
		}

		if tagStart+1 < len(p.source) && p.source[tagStart+1] == '/' {
			tagEnd := p.findTagEnd(tagStart)
			if tagEnd < 0 {
				p.error("sfc.malformed_tag", "malformed closing tag", tagStart, len(p.source))
				break
			}
			name, ok := readTagName(p.source, tagStart+2)
			if ok {
				p.error("sfc.unexpected_closing_tag", fmt.Sprintf("unexpected closing <%s> block", name), tagStart, tagEnd)
			}
			i = tagEnd
			continue
		}

		start, ok := p.parseStartTag(tagStart)
		if !ok {
			break
		}

		if !isBlockName(start.name) {
			i = start.end
			continue
		}

		if start.selfClosing {
			p.error("sfc.self_closing_block", fmt.Sprintf("<%s> block cannot be self-closing", start.name), start.offset, start.end)
			i = start.end
			continue
		}

		closeStart, closeEnd, found := p.findClosingBlock(start.name, start.end)
		if !found {
			p.error("sfc.missing_closing_tag", fmt.Sprintf("missing </%s> closing tag", start.name), start.offset, start.end)
			i = start.end
			continue
		}

		block := Block{
			Type:        start.name,
			Attrs:       start.attrs,
			Content:     p.source[start.end:closeStart],
			Span:        p.span(start.offset, closeEnd),
			StartTag:    p.span(start.offset, start.end),
			ContentSpan: p.span(start.end, closeStart),
			EndTag:      p.span(closeStart, closeEnd),
		}
		if block.Type == "script" {
			block.Lang = attrValue(block.Attrs, "lang")
			if block.Lang != "go" && block.Lang != "rust" {
				p.error("sfc.unsupported_script_lang", `<script> blocks must use lang="go" or lang="rust"`, start.offset, start.end)
			}
		}

		file.Blocks = append(file.Blocks, block)
		p.assignBlock(file, &file.Blocks[len(file.Blocks)-1])
		i = closeEnd
	}

	if file.Template == nil {
		p.error("sfc.missing_template", "missing <template> block", 0, 0)
	}
	if file.Script == nil {
		p.error("sfc.missing_script", `missing <script lang="go"> or <script lang="rust"> block`, 0, 0)
	}

	file.Diagnostics = append(file.Diagnostics, p.diags...)
	return file
}

func (p *parser) assignBlock(file *File, block *Block) {
	var current **Block
	switch block.Type {
	case "template":
		current = &file.Template
	case "script":
		current = &file.Script
	case "style":
		current = &file.Style
	default:
		return
	}

	if *current != nil {
		p.error("sfc.duplicate_block", fmt.Sprintf("duplicate <%s> block", block.Type), block.StartTag.Start.Offset, block.StartTag.End.Offset)
		return
	}
	*current = block
}

type startTag struct {
	name        string
	attrs       []Attribute
	offset      int
	end         int
	selfClosing bool
}

func (p *parser) parseStartTag(offset int) (startTag, bool) {
	end := p.findTagEnd(offset)
	if end < 0 {
		p.error("sfc.malformed_tag", "unterminated start tag", offset, len(p.source))
		return startTag{}, false
	}

	name, nameEnd, ok := readTagNameEnd(p.source, offset+1)
	if !ok {
		p.error("sfc.malformed_tag", "malformed start tag", offset, end)
		return startTag{}, false
	}

	attrs, selfClosing, attrOK := p.parseAttrs(nameEnd, end-1)
	if !attrOK {
		return startTag{}, false
	}
	return startTag{
		name:        strings.ToLower(name),
		attrs:       attrs,
		offset:      offset,
		end:         end,
		selfClosing: selfClosing,
	}, true
}

func (p *parser) parseAttrs(start, end int) ([]Attribute, bool, bool) {
	var attrs []Attribute
	selfClosing := false
	i := start
	for i < end {
		i = skipSpace(p.source, i, end)
		if i >= end {
			break
		}
		if p.source[i] == '/' {
			selfClosing = true
			i++
			i = skipSpace(p.source, i, end)
			if i != end {
				p.error("sfc.malformed_tag", "unexpected content after self-closing marker", i, end)
				return nil, false, false
			}
			break
		}

		nameStart := i
		for i < end && isAttrNameChar(rune(p.source[i])) {
			i++
		}
		if nameStart == i {
			p.error("sfc.malformed_tag", "malformed attribute", i, end)
			return nil, false, false
		}
		nameEnd := i
		attr := Attribute{
			Name:     strings.ToLower(p.source[nameStart:nameEnd]),
			NameSpan: p.span(nameStart, nameEnd),
		}

		i = skipSpace(p.source, i, end)
		if i < end && p.source[i] == '=' {
			attr.HasValue = true
			i++
			i = skipSpace(p.source, i, end)
			if i >= end {
				p.error("sfc.malformed_tag", "missing attribute value", nameStart, end)
				return nil, false, false
			}

			valueStart := i
			switch p.source[i] {
			case '"', '\'':
				quote := p.source[i]
				i++
				valueStart = i
				for i < end && p.source[i] != quote {
					i++
				}
				if i >= end {
					p.error("sfc.malformed_tag", "unterminated quoted attribute value", valueStart-1, end)
					return nil, false, false
				}
				attr.Value = p.source[valueStart:i]
				attr.ValueSpan = p.span(valueStart, i)
				i++
			default:
				for i < end && !unicode.IsSpace(rune(p.source[i])) && p.source[i] != '/' {
					i++
				}
				attr.Value = p.source[valueStart:i]
				attr.ValueSpan = p.span(valueStart, i)
			}
		}

		attrs = append(attrs, attr)
	}

	return attrs, selfClosing, true
}

func (p *parser) findTagEnd(offset int) int {
	quote := byte(0)
	for i := offset; i < len(p.source); i++ {
		ch := p.source[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '>' {
			return i + 1
		}
	}
	return -1
}

func (p *parser) findClosingBlock(name string, offset int) (int, int, bool) {
	blockName := strings.ToLower(name)
	depth := 1
	for i := offset; i < len(p.source); {
		next := strings.IndexByte(p.source[i:], '<')
		if next < 0 {
			return 0, 0, false
		}
		tagStart := i + next
		if strings.HasPrefix(p.source[tagStart:], "<!--") {
			end := strings.Index(p.source[tagStart+4:], "-->")
			if end < 0 {
				return 0, 0, false
			}
			i = tagStart + 4 + end + len("-->")
			continue
		}

		if tagStart+1 < len(p.source) && p.source[tagStart+1] == '/' {
			tagName, tagNameEnd, ok := readTagNameEnd(p.source, tagStart+2)
			if !ok || strings.ToLower(tagName) != blockName {
				i = tagStart + 2
				continue
			}
			tagEnd, ok := p.closingTagEnd(tagNameEnd)
			if !ok {
				i = tagNameEnd
				continue
			}
			depth--
			if depth == 0 {
				return tagStart, tagEnd, true
			}
			i = tagEnd
			continue
		}

		tagName, tagNameEnd, ok := readTagNameEnd(p.source, tagStart+1)
		if !ok || strings.ToLower(tagName) != blockName {
			i = tagStart + 1
			continue
		}
		tagEnd := p.findTagEnd(tagStart)
		if tagEnd < 0 {
			return 0, 0, false
		}
		if !isSelfClosingTag(p.source[tagNameEnd:tagEnd]) {
			depth++
		}
		i = tagEnd
	}
	return 0, 0, false
}

func isSelfClosingTag(afterName string) bool {
	trimmed := strings.TrimSpace(afterName)
	return strings.HasSuffix(trimmed, "/>")
}

func (p *parser) closingTagEnd(offset int) (int, bool) {
	i := skipSpace(p.source, offset, len(p.source))
	if i >= len(p.source) || p.source[i] != '>' {
		return 0, false
	}
	return i + 1, true
}

func (p *parser) error(code, message string, start, end int) {
	p.diags = append(p.diags, Diagnostic{
		Code:     code,
		Message:  message,
		Severity: "error",
		Span:     p.span(start, end),
	})
}

func (p *parser) span(start, end int) Span {
	return Span{
		Start: p.mapper.position(start),
		End:   p.mapper.position(end),
	}
}

func isBlockName(name string) bool {
	switch strings.ToLower(name) {
	case "template", "script", "style":
		return true
	default:
		return false
	}
}

func attrValue(attrs []Attribute, name string) string {
	name = strings.ToLower(name)
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value
		}
	}
	return ""
}

func readTagName(source string, offset int) (string, bool) {
	name, _, ok := readTagNameEnd(source, offset)
	return name, ok
}

func readTagNameEnd(source string, offset int) (string, int, bool) {
	if offset >= len(source) || !isTagNameStart(rune(source[offset])) {
		return "", offset, false
	}
	i := offset + 1
	for i < len(source) && isTagNameChar(rune(source[i])) {
		i++
	}
	return source[offset:i], i, true
}

func skipSpace(source string, offset, end int) int {
	i := offset
	for i < end && unicode.IsSpace(rune(source[i])) {
		i++
	}
	return i
}

func isTagNameStart(ch rune) bool {
	return unicode.IsLetter(ch)
}

func isTagNameChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_'
}

func isAttrNameChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_' || ch == ':' || ch == '@'
}

type sourceMap struct {
	lineStarts []int
}

func newSourceMap(source string) sourceMap {
	starts := []int{0}
	for i := 0; i < len(source); i++ {
		if source[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return sourceMap{lineStarts: starts}
}

func (m sourceMap) position(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	last := m.lineStarts[len(m.lineStarts)-1]
	if offset > last && len(m.lineStarts) == 0 {
		offset = 0
	}

	idx := sort.Search(len(m.lineStarts), func(i int) bool {
		return m.lineStarts[i] > offset
	}) - 1
	if idx < 0 {
		idx = 0
	}
	return Position{
		Offset: offset,
		Line:   idx + 1,
		Column: offset - m.lineStarts[idx] + 1,
	}
}
