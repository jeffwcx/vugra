package sfc_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goldentest"
	"github.com/vugra/vugra/internal/sfc"
)

func TestParseValidComponentGolden(t *testing.T) {
	source := `<template>
  <button @click="Inc">{{ count }}</button>
</template>

<script lang="go">
type State struct {
    Count signal.Int ` + "`vugra:\"count\"`" + `
}
</script>

<style>
button { padding: 8px; }
</style>
`
	assertGolden(t, "valid_component.json", source)
}

func TestParseCommentsAndWhitespaceGolden(t *testing.T) {
	source := `
<!-- lead comment -->

<template>
  <div>{{ count }}</div>
</template>

<!-- between blocks -->
<script
  lang="go"
>
type State struct{}
</script>
`
	assertGolden(t, "comments_whitespace.json", source)
}

func TestParseDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		codes []string
	}{
		{
			name:  "missing template",
			src:   `<script lang="go">type State struct{}</script>`,
			codes: []string{"sfc.missing_template"},
		},
		{
			name:  "missing script",
			src:   `<template><div></div></template>`,
			codes: []string{"sfc.missing_script"},
		},
		{
			name: "duplicate blocks",
			src: `<template></template>
<template></template>
<script lang="go"></script>`,
			codes: []string{"sfc.duplicate_block"},
		},
		{
			name: "unsupported script language",
			src: `<template></template>
<script lang="js"></script>`,
			codes: []string{"sfc.unsupported_script_lang"},
		},
		{
			name: "missing closing tag",
			src: `<template>
  <div></div>
<script lang="go"></script>`,
			codes: []string{"sfc.missing_closing_tag", "sfc.missing_template"},
		},
		{
			name:  "malformed start tag",
			src:   `<template lang="html></template><script lang="go"></script>`,
			codes: []string{"sfc.malformed_tag", "sfc.missing_template"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := sfc.ParseString("Component.vue", tt.src)
			var got []string
			for _, diag := range file.Diagnostics {
				got = append(got, diag.Code)
			}
			for _, want := range tt.codes {
				if !contains(got, want) {
					t.Fatalf("expected diagnostic %s, got %v", want, got)
				}
			}
		})
	}
}

func TestBlockContentAndOffsets(t *testing.T) {
	source := "<template>\n  hi\n</template>\n<script lang=\"go\">\npackage demo\n</script>"
	file := sfc.ParseString("Component.vue", source)

	if file.Template == nil {
		t.Fatal("missing template block")
	}
	if got := file.Template.Content; got != "\n  hi\n" {
		t.Fatalf("template content = %q", got)
	}
	if file.Template.ContentSpan.Start.Offset != len("<template>") {
		t.Fatalf("template content start offset = %d", file.Template.ContentSpan.Start.Offset)
	}
	if file.Template.ContentSpan.Start.Line != 1 || file.Template.ContentSpan.Start.Column != 11 {
		t.Fatalf("template content start position = %+v", file.Template.ContentSpan.Start)
	}

	if file.Script == nil {
		t.Fatal("missing script block")
	}
	if file.Script.Lang != "go" {
		t.Fatalf("script lang = %q", file.Script.Lang)
	}
}

func TestParseTemplateBlockWithNestedTemplateSlot(t *testing.T) {
	source := `<template>
  <Badge>
    <template #meta>
      <span>named slot</span>
    </template>
  </Badge>
</template>
<script lang="go">
type State struct{}
</script>`

	file := sfc.ParseString("Component.vue", source)
	if diagnostics := file.Diagnostics; len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	if len(file.Blocks) != 2 {
		t.Fatalf("blocks = %d", len(file.Blocks))
	}
	if file.Template == nil {
		t.Fatal("missing template block")
	}
	if !strings.Contains(file.Template.Content, `<template #meta>`) {
		t.Fatalf("template content = %q", file.Template.Content)
	}
	if strings.Contains(file.Script.Content, "named slot") {
		t.Fatalf("script content included nested template content: %q", file.Script.Content)
	}
}

func assertGolden(t *testing.T, name, source string) {
	t.Helper()
	file := sfc.ParseString("Component.vue", source)
	goldentest.Assert(t, name, renderSnapshot(file))
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func renderSnapshot(file *sfc.File) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "path=%s\n", file.Path)
	fmt.Fprintf(&b, "diagnostics=%d\n", len(file.Diagnostics))
	for i, diag := range file.Diagnostics {
		fmt.Fprintf(&b, "diagnostic[%d]=%s %s %s\n", i, diag.Code, diag.Severity, formatSpan(diag.Span))
	}
	fmt.Fprintf(&b, "blocks=%d\n", len(file.Blocks))
	for i, block := range file.Blocks {
		fmt.Fprintf(&b, "block[%d].type=%s lang=%s\n", i, block.Type, block.Lang)
		fmt.Fprintf(&b, "block[%d].span=%s\n", i, formatSpan(block.Span))
		fmt.Fprintf(&b, "block[%d].startTag=%s\n", i, formatSpan(block.StartTag))
		fmt.Fprintf(&b, "block[%d].contentSpan=%s\n", i, formatSpan(block.ContentSpan))
		fmt.Fprintf(&b, "block[%d].endTag=%s\n", i, formatSpan(block.EndTag))
		for j, attr := range block.Attrs {
			fmt.Fprintf(
				&b,
				"block[%d].attr[%d]=%s value=%s hasValue=%t name=%s valueSpan=%s\n",
				i,
				j,
				attr.Name,
				strconv.Quote(attr.Value),
				attr.HasValue,
				formatSpan(attr.NameSpan),
				formatSpan(attr.ValueSpan),
			)
		}
		fmt.Fprintf(&b, "block[%d].content=%s\n", i, strconv.Quote(block.Content))
	}
	return []byte(b.String())
}

func formatSpan(span sfc.Span) string {
	return formatPosition(span.Start) + "-" + formatPosition(span.End)
}

func formatPosition(pos sfc.Position) string {
	return fmt.Sprintf("%d:%d@%d", pos.Line, pos.Column, pos.Offset)
}
