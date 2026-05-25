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

func TestCompileRustScriptBuildsIR(t *testing.T) {
	result := compiler.Compile("Counter.vue", []byte(`
<template>
  <button @click="Inc">{{ count }}</button>
</template>
<script lang="rust">
pub struct State {
    pub count: Signal<i32>,
}

impl State {
    pub fn Inc(&mut self) {}
}
</script>
`))
	if result.SFC == nil || result.SFC.Script == nil || result.SFC.Script.Lang != "rust" {
		t.Fatalf("missing rust script: %+v", result.SFC)
	}
	if result.Rust.State == nil || len(result.Rust.State.Fields) != 1 {
		t.Fatalf("missing rust metadata: %+v", result.Rust)
	}
	if result.Go.State != nil {
		t.Fatalf("rust script should not be analyzed as Go: %+v", result.Go)
	}
	if result.IR == nil || len(result.IR.Nodes) == 0 {
		t.Fatalf("missing IR: %+v", result.IR)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}
}

func TestCompileRustScriptDiagnosticsMapToSource(t *testing.T) {
	result := compiler.Compile("Broken.vue", []byte(`
<template><button @click="Missing">{{ count }}</button></template>
<script lang="rust">
pub struct State {
    pub count: Signal<i32>,
}

impl State {
    pub fn Inc(&mut self) {}
}
</script>
`))
	diagnostics := result.Diagnostics()
	if !hasDiagnostic(diagnostics, "ir.unknown_event_handler") {
		t.Fatalf("missing unknown event diagnostic: %+v", diagnostics)
	}
	if hasDiagnostic(diagnostics, "goanalysis.invalid_go") {
		t.Fatalf("rust script should not produce Go diagnostics: %+v", diagnostics)
	}
}

func hasDiagnostic(diagnostics []compiler.Diagnostic, code string) bool {
	for _, diag := range diagnostics {
		if diag.Code == code {
			return true
		}
	}
	return false
}
