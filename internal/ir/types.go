package ir

// Package ir defines Vugra's renderer-neutral component intermediate
// representation and validation from template plus Go metadata.

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

type Component struct {
	Name        string              `json:"name"`
	Nodes       []Node              `json:"nodes"`
	Imports     []Import            `json:"imports,omitempty"`
	PropNames   []string            `json:"propNames,omitempty"`
	Props       []PropDef           `json:"props,omitempty"`
	Provides    []ProvideDef        `json:"provides,omitempty"`
	Injects     []InjectDef         `json:"injects,omitempty"`
	Emits       []Emit              `json:"emits,omitempty"`
	Lifecycle   []Lifecycle         `json:"lifecycle,omitempty"`
	Diagnostics []Diagnostic        `json:"diagnostics,omitempty"`
	NewState    func() RuntimeState `json:"-"`
}

type Signal interface {
	GetAny() any
	SetAny(any)
	Subscribe(func()) func()
}

type RuntimeState struct {
	Signals      map[string]Signal
	Methods      map[string]func()
	EventMethods map[string]func(Event)
}

type Modifiers struct {
	Shift bool
	Ctrl  bool
	Meta  bool
	Alt   bool
}

func (m Modifiers) Command() bool {
	return m.Ctrl || m.Meta
}

type Event struct {
	Type      string
	Key       string
	X         float32
	Y         float32
	DeltaX    float32
	DeltaY    float32
	Modifiers Modifiers
}

type Import struct {
	Alias     string     `json:"alias"`
	Path      string     `json:"path"`
	Component *Component `json:"component,omitempty"`
}

type Node interface {
	node()
	NodeKind() string
}

type Element struct {
	Tag      string         `json:"tag"`
	RawTag   string         `json:"rawTag,omitempty"`
	Props    []Prop         `json:"props,omitempty"`
	Events   []EventHandler `json:"events,omitempty"`
	Children []Node         `json:"children,omitempty"`
	Span     Span           `json:"span"`
}

func (*Element) node()            {}
func (*Element) NodeKind() string { return "element" }

type Text struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*Text) node()            {}
func (*Text) NodeKind() string { return "text" }

type Interpolation struct {
	Binding string `json:"binding"`
	GoField string `json:"goField,omitempty"`
	Span    Span   `json:"span"`
}

func (*Interpolation) node()            {}
func (*Interpolation) NodeKind() string { return "interpolation" }

type Conditional struct {
	Expression string `json:"expression"`
	Child      Node   `json:"child"`
	Span       Span   `json:"span"`
}

func (*Conditional) node()            {}
func (*Conditional) NodeKind() string { return "conditional" }

type Repeater struct {
	Expression string `json:"expression"`
	Child      Node   `json:"child"`
	Span       Span   `json:"span"`
}

func (*Repeater) node()            {}
func (*Repeater) NodeKind() string { return "repeater" }

type ComponentInstance struct {
	Alias     string         `json:"alias"`
	Component *Component     `json:"component,omitempty"`
	Props     []Prop         `json:"props,omitempty"`
	Events    []EventHandler `json:"events,omitempty"`
	Slots     []Slot         `json:"slots,omitempty"`
	Lifecycle []Lifecycle    `json:"lifecycle,omitempty"`
	Nodes     []Node         `json:"nodes"`
	Span      Span           `json:"span"`
}

func (*ComponentInstance) node()            {}
func (*ComponentInstance) NodeKind() string { return "component" }

type DynamicComponent struct {
	Binding string                 `json:"binding"`
	Props   []Prop                 `json:"props,omitempty"`
	Events  []EventHandler         `json:"events,omitempty"`
	Slots   []Slot                 `json:"slots,omitempty"`
	Cases   []DynamicComponentCase `json:"cases"`
	Span    Span                   `json:"span"`
}

func (*DynamicComponent) node()            {}
func (*DynamicComponent) NodeKind() string { return "dynamicComponent" }

type DynamicComponentCase struct {
	Alias     string     `json:"alias"`
	Component *Component `json:"component,omitempty"`
	Nodes     []Node     `json:"nodes"`
}

type Prop struct {
	Name    string `json:"name"`
	Value   string `json:"value,omitempty"`
	Binding string `json:"binding,omitempty"`
	Bound   bool   `json:"bound"`
	Span    Span   `json:"span"`
}

type PropDef struct {
	Name     string `json:"name"`
	GoField  string `json:"goField,omitempty"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
	Default  string `json:"default,omitempty"`
	Span     Span   `json:"span"`
}

type ProvideDef struct {
	Name    string `json:"name"`
	Binding string `json:"binding"`
	Span    Span   `json:"span"`
}

type InjectDef struct {
	Name    string `json:"name"`
	GoField string `json:"goField,omitempty"`
	Type    string `json:"type,omitempty"`
	Default string `json:"default,omitempty"`
	Span    Span   `json:"span"`
}

type Slot struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
	Nodes []Node `json:"nodes"`
}

type Emit struct {
	Method string `json:"method"`
	Event  string `json:"event"`
	Span   Span   `json:"span"`
}

type Lifecycle struct {
	Hook   string `json:"hook"`
	Method string `json:"method"`
	Span   Span   `json:"span"`
}

type EventHandler struct {
	Event  string `json:"event"`
	Method string `json:"method"`
	Span   Span   `json:"span"`
}
