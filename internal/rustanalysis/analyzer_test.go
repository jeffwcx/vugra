package rustanalysis_test

import (
	"testing"

	"github.com/vugra/vugra/internal/rustanalysis"
)

func TestAnalyzeRustStateAndMethods(t *testing.T) {
	metadata := rustanalysis.Analyze(`
pub struct State {
    pub count: Signal<i32>,
    title: String,
}

impl State {
    pub fn inc(&mut self) {}
    fn hidden(&mut self) {}
    pub fn select(&mut self, event: Event) {}
}
`, rustanalysis.BasePosition{Offset: 100, Line: 5, Column: 3})

	if metadata.State == nil {
		t.Fatal("missing state")
	}
	if len(metadata.State.Fields) != 2 {
		t.Fatalf("fields = %+v", metadata.State.Fields)
	}
	if field := metadata.State.Fields[0]; field.Name != "count" || field.Alias != "count" || field.Type != "Signal<i32>" || !field.IsSignal || !field.Exported {
		t.Fatalf("count field = %+v", field)
	}
	if field := metadata.State.Fields[1]; field.Name != "title" || field.Alias != "title" || field.Exported {
		t.Fatalf("title field = %+v", field)
	}
	if len(metadata.Methods) != 3 {
		t.Fatalf("methods = %+v", metadata.Methods)
	}
	if metadata.Methods[0].Name != "hidden" || metadata.Methods[0].Exported {
		t.Fatalf("hidden method = %+v", metadata.Methods[0])
	}
	if metadata.Methods[2].Name != "select" || !metadata.Methods[2].EventArg {
		t.Fatalf("select method = %+v", metadata.Methods[2])
	}
	if metadata.Methods[2].NameSpan.Start.Offset < 100 {
		t.Fatalf("method span not mapped into source: %+v", metadata.Methods[2].NameSpan)
	}
	if len(metadata.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", metadata.Diagnostics)
	}
}

func TestAnalyzeRustMissingStateDiagnostic(t *testing.T) {
	metadata := rustanalysis.Analyze(`impl State { pub fn inc(&mut self) {} }`, rustanalysis.BasePosition{})
	if len(metadata.Diagnostics) == 0 || metadata.Diagnostics[0].Code != "rustanalysis.missing_state" {
		t.Fatalf("diagnostics = %+v", metadata.Diagnostics)
	}
}
