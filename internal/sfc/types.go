package sfc

type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Span     Span   `json:"span"`
}

type Attribute struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	HasValue  bool   `json:"hasValue"`
	NameSpan  Span   `json:"nameSpan"`
	ValueSpan Span   `json:"valueSpan,omitempty"`
}

type Block struct {
	Type        string      `json:"type"`
	Lang        string      `json:"lang,omitempty"`
	Attrs       []Attribute `json:"attrs,omitempty"`
	Content     string      `json:"content"`
	Span        Span        `json:"span"`
	StartTag    Span        `json:"startTag"`
	ContentSpan Span        `json:"contentSpan"`
	EndTag      Span        `json:"endTag"`
}

type File struct {
	Path        string       `json:"path,omitempty"`
	Blocks      []Block      `json:"blocks"`
	Template    *Block       `json:"template,omitempty"`
	Script      *Block       `json:"script,omitempty"`
	Style       *Block       `json:"style,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
