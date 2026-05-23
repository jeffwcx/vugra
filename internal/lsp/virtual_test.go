package lsp_test

import (
	"testing"

	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/lsp"
)

func TestBuildVirtualFilesMapsStyleDiagnostics(t *testing.T) {
	result := compiler.Compile("Component.vue", []byte(`
<template><div class="card">{{ missing }}</div></template>
<script lang="go">
type State struct {}
</script>
<style>
.card {
  transform: scale(2);
}
</style>
`))
	files := lsp.BuildVirtualFiles("Component", result)
	if files.Template.FileName != "Component.template.html" || files.Script.Language != "go" || files.Style.Language != "css" {
		t.Fatalf("virtual files = %+v", files)
	}
	if files.Style.Content == "" || files.Style.Span.Start.Line == 0 {
		t.Fatalf("style virtual file missing source span: %+v", files.Style)
	}
	found := false
	for _, diag := range files.Diagnostics {
		if diag.Code == "style.unsupported_property" && diag.Span.Start.Line > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing mapped style diagnostic: %+v", files.Diagnostics)
	}
}
