package style

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

type Stylesheet struct {
	Rules       []Rule       `json:"rules"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type Rule struct {
	Selector     string        `json:"selector"`
	ClassName    string        `json:"className"`
	Declarations []Declaration `json:"declarations"`
	Span         Span          `json:"span"`
}

type Declaration struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	NameSpan  Span   `json:"nameSpan"`
	ValueSpan Span   `json:"valueSpan"`
}

type Computed struct {
	Display             string
	BoxSizing           string
	FlexDirection       string
	FlexWrap            string
	Gap                 float32
	RowGap              float32
	ColumnGap           float32
	Padding             float32
	PaddingLeft         float32
	Margin              float32
	Width               float32
	WidthPercent        float32
	Height              float32
	HeightPercent       float32
	MinWidth            float32
	MaxWidth            float32
	MinHeight           float32
	MaxHeight           float32
	AlignItems          string
	AlignSelf           string
	Justify             string
	FlexGrow            float32
	FlexShrink          float32
	FlexBasis           float32
	GridTemplateColumns []Track
	GridTemplateRows    []Track
	GridAutoRows        float32
	GridColumn          GridPlacement
	GridRow             GridPlacement
	FontFamily          string
	FontSize            float32
	FontWeight          string
	LineHeight          float32
	TextAlign           string
	WhiteSpace          string
	Background          string
	BackgroundColor     string
	Opacity             float32
	BorderWidth         float32
	BorderWidthSet      bool
	BorderColor         string
	BorderStyle         string
	BorderRadius        float32
	Color               string
	Overflow            string
	OverflowX           string
	OverflowY           string
}

type SystemTokens map[string]float32

type Track struct {
	Unit  string
	Value float32
}

type GridPlacement struct {
	Start int
	Span  int
}
