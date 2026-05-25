package rustanalysis

type sourceMap struct {
	source string
	base   BasePosition
	lines  []int
}

func newSourceMap(source string, base BasePosition) sourceMap {
	if base.Line == 0 {
		base.Line = 1
	}
	if base.Column == 0 {
		base.Column = 1
	}
	lines := []int{0}
	for i, ch := range source {
		if ch == '\n' {
			lines = append(lines, i+1)
		}
	}
	return sourceMap{source: source, base: base, lines: lines}
}

func (m sourceMap) span(start, end int) Span {
	return Span{Start: m.position(start), End: m.position(end)}
}

func (m sourceMap) position(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(m.source) {
		offset = len(m.source)
	}
	lineIndex := 0
	for i := len(m.lines) - 1; i >= 0; i-- {
		if m.lines[i] <= offset {
			lineIndex = i
			break
		}
	}
	line := m.base.Line + lineIndex
	column := offset - m.lines[lineIndex] + 1
	if lineIndex == 0 {
		column += m.base.Column - 1
	}
	return Position{
		Offset: m.base.Offset + offset,
		Line:   line,
		Column: column,
	}
}
