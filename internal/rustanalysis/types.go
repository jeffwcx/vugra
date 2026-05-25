package rustanalysis

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
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
}

type StateMetadata struct {
	TypeName string  `json:"typeName"`
	Fields   []Field `json:"fields"`
}

type Field struct {
	Name     string `json:"name"`
	Alias    string `json:"alias"`
	Type     string `json:"type"`
	IsSignal bool   `json:"isSignal"`
	Span     Span   `json:"span"`
	TypeSpan Span   `json:"typeSpan"`
	Exported bool   `json:"exported"`
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

type BasePosition struct {
	Offset int
	Line   int
	Column int
}
