package compiler_test

import (
	"testing"

	"github.com/vugra/vugra/internal/compiler"
)

func TestCompileCounterExample(t *testing.T) {
	result, err := compiler.CompileFile("../../examples/counter/Counter.vue")
	if err != nil {
		t.Fatalf("compile counter: %v", err)
	}
	if result.SFC == nil || result.SFC.Template == nil || result.SFC.Script == nil || result.SFC.Style == nil {
		t.Fatalf("incomplete SFC parse: %+v", result.SFC)
	}
	if result.Template == nil {
		t.Fatal("missing template document")
	}
	if result.Style == nil || len(result.Style.Rules) != 2 {
		t.Fatalf("missing style metadata: %+v", result.Style)
	}
	if result.Go.State == nil {
		t.Fatal("missing Go State metadata")
	}
	if result.IR == nil {
		t.Fatal("missing IR")
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
}
