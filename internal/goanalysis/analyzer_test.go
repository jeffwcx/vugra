package goanalysis_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/goldentest"
)

func TestAnalyzeStateAndMethodsGolden(t *testing.T) {
	source := `
type State struct {
    Count signal.Int ` + "`vugra:\"count\"`" + `
    Name signal.Signal[string] ` + "`vugra:\"displayName\"`" + `
    hidden int
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}

func (s State) IgnoredValueReceiver() {}

func (s *State) hiddenMethod() {}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{Offset: 200, Line: 8, Column: 19})
	goldentest.Assert(t, "state_methods.txt", renderMetadata(metadata))
}

func TestAnalyzeInvalidGoDiagnostics(t *testing.T) {
	source := `
type State struct {
    Count signal.Int
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{Offset: 50, Line: 3, Column: 1})
	if len(metadata.Diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	if !containsDiag(metadata.Diagnostics, "goanalysis.invalid_go") {
		t.Fatalf("expected invalid_go diagnostic, got %+v", metadata.Diagnostics)
	}
	diag := metadata.Diagnostics[0]
	if diag.Span.Start.Offset < 50 {
		t.Fatalf("diagnostic did not map into script source: %+v", diag.Span)
	}
}

func TestAnalyzeMissingState(t *testing.T) {
	metadata := goanalysis.Analyze(`func Nope() {}`, goanalysis.BasePosition{Offset: 10, Line: 2, Column: 3})
	if !containsDiag(metadata.Diagnostics, "goanalysis.missing_state") {
		t.Fatalf("expected missing_state diagnostic, got %+v", metadata.Diagnostics)
	}
}

func TestAnalyzeVugraComponentImports(t *testing.T) {
	source := `
import (
    Badge "./Badge.vue"
    "./plain-card.vue"
    Legacy "./Legacy.vugra"
    "fmt"
)

type State struct {}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if len(metadata.Imports) != 3 {
		t.Fatalf("imports = %+v", metadata.Imports)
	}
	if metadata.Imports[0].Alias != "Badge" || metadata.Imports[0].Path != "./Badge.vue" {
		t.Fatalf("first import = %+v", metadata.Imports[0])
	}
	if metadata.Imports[1].Alias != "Legacy" || metadata.Imports[1].Path != "./Legacy.vugra" {
		t.Fatalf("second import = %+v", metadata.Imports[1])
	}
	if metadata.Imports[2].Alias != "PlainCard" || metadata.Imports[2].Path != "./plain-card.vue" {
		t.Fatalf("third import = %+v", metadata.Imports[2])
	}
}

func TestAnalyzeVugraEmits(t *testing.T) {
	source := `
type State struct {}

func (s *State) Pressed() {
    vugra.Emit("select")
}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if len(metadata.Emits) != 1 {
		t.Fatalf("emits = %+v", metadata.Emits)
	}
	if metadata.Emits[0].Method != "Pressed" || metadata.Emits[0].Event != "select" {
		t.Fatalf("emit = %+v", metadata.Emits[0])
	}
}

func TestAnalyzeVugraPropTagOptions(t *testing.T) {
	source := `
type State struct {
    Title signal.String ` + "`vugra:\"title,optional,default=Untitled\"`" + `
    Theme signal.String ` + "`vugra:\"theme,provide\"`" + `
    Locale signal.String ` + "`vugra:\"locale,inject,default=en\"`" + `
}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if metadata.State == nil || len(metadata.State.Fields) != 3 {
		t.Fatalf("state = %+v", metadata.State)
	}
	field := metadata.State.Fields[0]
	if field.Alias != "title" || !field.Optional || field.Default != "Untitled" {
		t.Fatalf("field = %+v", field)
	}
	if !metadata.State.Fields[1].Provide || !metadata.State.Fields[1].Optional {
		t.Fatalf("provide field = %+v", metadata.State.Fields[1])
	}
	if !metadata.State.Fields[2].Inject || !metadata.State.Fields[2].Optional || metadata.State.Fields[2].Default != "en" {
		t.Fatalf("inject field = %+v", metadata.State.Fields[2])
	}
}

func TestAnalyzeLegacyVuegoPropTag(t *testing.T) {
	source := `
type State struct {
    Title signal.String ` + "`vuego:\"title\"`" + `
}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if metadata.State == nil || len(metadata.State.Fields) != 1 {
		t.Fatalf("state = %+v", metadata.State)
	}
	if field := metadata.State.Fields[0]; field.Alias != "title" || !field.HasVugraTag {
		t.Fatalf("field = %+v", field)
	}
}

func TestAnalyzeLifecycleHooks(t *testing.T) {
	source := `
type State struct {}

func (s *State) Mounted() {}
func (s *State) Updated() {}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if len(metadata.Lifecycle) != 2 {
		t.Fatalf("lifecycle = %+v", metadata.Lifecycle)
	}
	if metadata.Lifecycle[0].Hook != "mounted" || metadata.Lifecycle[1].Hook != "updated" {
		t.Fatalf("lifecycle = %+v", metadata.Lifecycle)
	}
}

func TestAnalyzeEventMethodArgument(t *testing.T) {
	source := `
type State struct {}

func (s *State) Select(event vugra.Event) {}
`
	metadata := goanalysis.Analyze(source, goanalysis.BasePosition{})
	if len(metadata.Methods) != 1 || !metadata.Methods[0].EventArg {
		t.Fatalf("methods = %+v", metadata.Methods)
	}
}

func renderMetadata(metadata goanalysis.Metadata) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "diagnostics=%d\n", len(metadata.Diagnostics))
	for i, diag := range metadata.Diagnostics {
		fmt.Fprintf(&b, "diagnostic[%d]=%s %s %s %s\n", i, diag.Code, diag.Severity, formatSpan(diag.Span), strconv.Quote(diag.Message))
	}
	if metadata.State == nil {
		fmt.Fprintln(&b, "state=<nil>")
	} else {
		fmt.Fprintf(&b, "state=%s fields=%d\n", metadata.State.TypeName, len(metadata.State.Fields))
		for i, field := range metadata.State.Fields {
			fmt.Fprintf(
				&b,
				"field[%d]=%s alias=%s type=%s signal=%t exported=%t tagged=%t span=%s typeSpan=%s\n",
				i,
				field.Name,
				field.Alias,
				field.Type,
				field.IsSignal,
				field.Exported,
				field.HasVugraTag,
				formatSpan(field.Span),
				formatSpan(field.TypeSpan),
			)
		}
	}
	fmt.Fprintf(&b, "methods=%d\n", len(metadata.Methods))
	for i, method := range metadata.Methods {
		fmt.Fprintf(
			&b,
			"method[%d]=%s receiver=%s pointer=%t exported=%t span=%s name=%s\n",
			i,
			method.Name,
			method.Receiver,
			method.OnPointer,
			method.Exported,
			formatSpan(method.Span),
			formatSpan(method.NameSpan),
		)
	}
	return []byte(b.String())
}

func formatSpan(span goanalysis.Span) string {
	return formatPosition(span.Start) + "-" + formatPosition(span.End)
}

func formatPosition(pos goanalysis.Position) string {
	return fmt.Sprintf("%d:%d@%d", pos.Line, pos.Column, pos.Offset)
}

func containsDiag(diags []goanalysis.Diagnostic, code string) bool {
	for _, diag := range diags {
		if diag.Code == code {
			return true
		}
	}
	return false
}
