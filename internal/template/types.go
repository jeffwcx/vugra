package template

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

type Document struct {
	Nodes       []Node       `json:"nodes"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type Node interface {
	node()
	NodeKind() string
	NodeSpan() Span
}

type Element struct {
	Tag      string      `json:"tag"`
	RawTag   string      `json:"rawTag,omitempty"`
	Attrs    []Attribute `json:"attrs,omitempty"`
	Children []Node      `json:"children,omitempty"`
	Span     Span        `json:"span"`
	StartTag Span        `json:"startTag"`
	EndTag   *Span       `json:"endTag,omitempty"`
}

func (*Element) node()            {}
func (*Element) NodeKind() string { return "element" }
func (e *Element) NodeSpan() Span { return e.Span }

type Text struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*Text) node()            {}
func (*Text) NodeKind() string { return "text" }
func (t *Text) NodeSpan() Span { return t.Span }

type Interpolation struct {
	Expression string `json:"expression"`
	Span       Span   `json:"span"`
	ExprSpan   Span   `json:"exprSpan"`
}

func (*Interpolation) node()            {}
func (*Interpolation) NodeKind() string { return "interpolation" }
func (i *Interpolation) NodeSpan() Span { return i.Span }

type Comment struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*Comment) node()            {}
func (*Comment) NodeKind() string { return "comment" }
func (c *Comment) NodeSpan() Span { return c.Span }

type Attribute struct {
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
	Kind      AttrKind   `json:"kind"`
	Arg       string     `json:"arg,omitempty"`
	HasValue  bool       `json:"hasValue"`
	NameSpan  Span       `json:"nameSpan"`
	ValueSpan *Span      `json:"valueSpan,omitempty"`
	Directive *Directive `json:"directive,omitempty"`
}

func (a Attribute) ValueOrNameSpan() Span {
	if a.ValueSpan != nil {
		return *a.ValueSpan
	}
	return a.NameSpan
}

type AttrKind string

const (
	AttrStatic    AttrKind = "static"
	AttrIf        AttrKind = "if"
	AttrFor       AttrKind = "for"
	AttrBoundProp AttrKind = "boundProp"
	AttrEvent     AttrKind = "event"
	AttrSlot      AttrKind = "slot"
	AttrModel     AttrKind = "model"
)

type Directive struct {
	Name       string `json:"name"`
	Argument   string `json:"argument,omitempty"`
	Expression string `json:"expression,omitempty"`
}
