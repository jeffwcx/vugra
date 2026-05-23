package goanalysis

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

type Metadata struct {
	State       *StateMetadata `json:"state,omitempty"`
	Methods     []Method       `json:"methods,omitempty"`
	Imports     []Import       `json:"imports,omitempty"`
	Emits       []Emit         `json:"emits,omitempty"`
	Lifecycle   []Lifecycle    `json:"lifecycle,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
}

type Import struct {
	Alias string `json:"alias"`
	Path  string `json:"path"`
	Span  Span   `json:"span"`
}

type StateMetadata struct {
	TypeName string  `json:"typeName"`
	Fields   []Field `json:"fields"`
}

type Field struct {
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	Type        string `json:"type"`
	IsSignal    bool   `json:"isSignal"`
	Span        Span   `json:"span"`
	AliasSpan   Span   `json:"aliasSpan"`
	TypeSpan    Span   `json:"typeSpan"`
	Exported    bool   `json:"exported"`
	HasVugraTag bool   `json:"hasVugraTag"`
	Optional    bool   `json:"optional,omitempty"`
	Default     string `json:"default,omitempty"`
	Provide     bool   `json:"provide,omitempty"`
	Inject      bool   `json:"inject,omitempty"`
}

type Method struct {
	Name      string `json:"name"`
	Receiver  string `json:"receiver"`
	Span      Span   `json:"span"`
	NameSpan  Span   `json:"nameSpan"`
	Exported  bool   `json:"exported"`
	OnPointer bool   `json:"onPointer"`
	EventArg  bool   `json:"eventArg,omitempty"`
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
