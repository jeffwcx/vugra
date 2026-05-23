package template_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goldentest"
	"github.com/vugra/vugra/internal/template"
)

func TestParseNestedDirectivesGolden(t *testing.T) {
	source := `
<div class="counter" v-if="visible">
  <p>{{ count }}</p>
  <button @click="Inc" :disabled="isDisabled">+</button>
  <ul>
    <li v-for="item in items">{{ item }}</li>
  </ul>
</div>
`
	assertGolden(t, "nested_directives.txt", source, 100)
}

func TestParseCommentsAndVoidElementsGolden(t *testing.T) {
	source := `
<!-- keep me -->
<label for="name">Name</label>
<input :value="name">
<img src="avatar.png" />
`
	assertGolden(t, "comments_void.txt", source, 20)
}

func TestTemplateDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		codes []string
	}{
		{
			name:  "unsupported tag",
			src:   `<section></section>`,
			codes: []string{"template.unsupported_tag"},
		},
		{
			name:  "unclosed interpolation",
			src:   `<div>{{ count</div>`,
			codes: []string{"template.unclosed_interpolation", "template.missing_closing_tag"},
		},
		{
			name:  "missing closing tag",
			src:   `<div><span>text</div>`,
			codes: []string{"template.mismatched_closing_tag", "template.missing_closing_tag", "template.missing_closing_tag"},
		},
		{
			name:  "unexpected closing tag",
			src:   `</div>`,
			codes: []string{"template.unexpected_closing_tag"},
		},
		{
			name:  "malformed attr",
			src:   `<div class="broken></div>`,
			codes: []string{"template.malformed_tag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := template.Parse(tt.src, 0)
			var got []string
			for _, diag := range doc.Diagnostics {
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

func TestInterpolationExpressionSpan(t *testing.T) {
	source := `<p>{{  count  }}</p>`
	doc := template.Parse(source, 50)
	elem := firstElement(t, doc.Nodes)
	if len(elem.Children) != 1 {
		t.Fatalf("expected one child, got %d", len(elem.Children))
	}
	interp, ok := elem.Children[0].(*template.Interpolation)
	if !ok {
		t.Fatalf("expected interpolation, got %T", elem.Children[0])
	}
	if interp.Expression != "count" {
		t.Fatalf("expression = %q", interp.Expression)
	}
	if interp.ExprSpan.Start.Offset != 57 || interp.ExprSpan.End.Offset != 62 {
		t.Fatalf("expr span = %+v", interp.ExprSpan)
	}
}

func TestParseHTMLWithTdewolffLexerAttributes(t *testing.T) {
	doc := template.Parse(`<input disabled :value=name>{{ count }}`, 10)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", doc.Diagnostics)
	}
	if len(doc.Nodes) != 2 {
		t.Fatalf("nodes = %d", len(doc.Nodes))
	}
	elem := firstElement(t, doc.Nodes)
	if len(elem.Attrs) != 2 {
		t.Fatalf("attrs = %+v", elem.Attrs)
	}
	if elem.Attrs[0].Name != "disabled" || elem.Attrs[0].HasValue {
		t.Fatalf("boolean attr = %+v", elem.Attrs[0])
	}
	if elem.Attrs[1].Kind != template.AttrBoundProp || elem.Attrs[1].Value != "name" {
		t.Fatalf("bound attr = %+v", elem.Attrs[1])
	}
	interp, ok := doc.Nodes[1].(*template.Interpolation)
	if !ok || interp.Expression != "count" {
		t.Fatalf("interpolation = %#v", doc.Nodes[1])
	}
}

func TestParseModelDirectives(t *testing.T) {
	doc := template.Parse(`<Editor v-model="title" v-model:subtitle="subtitle"></Editor>`, 0)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", doc.Diagnostics)
	}
	elem := firstElement(t, doc.Nodes)
	if len(elem.Attrs) != 2 {
		t.Fatalf("attrs = %+v", elem.Attrs)
	}
	if elem.Attrs[0].Kind != template.AttrModel || elem.Attrs[0].Arg != "modelValue" || elem.Attrs[0].Value != "title" {
		t.Fatalf("default model attr = %+v", elem.Attrs[0])
	}
	if elem.Attrs[1].Kind != template.AttrModel || elem.Attrs[1].Arg != "subtitle" || elem.Attrs[1].Value != "subtitle" {
		t.Fatalf("named model attr = %+v", elem.Attrs[1])
	}
}

func TestParseScopedSlotDirective(t *testing.T) {
	doc := template.Parse(`<template #item="scope">{{ scope.label }}</template>`, 0)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", doc.Diagnostics)
	}
	elem := firstElement(t, doc.Nodes)
	if len(elem.Attrs) != 1 {
		t.Fatalf("attrs = %+v", elem.Attrs)
	}
	if elem.Attrs[0].Kind != template.AttrSlot || elem.Attrs[0].Arg != "item" || elem.Attrs[0].Value != "scope" {
		t.Fatalf("scoped slot attr = %+v", elem.Attrs[0])
	}
}

func TestParseComponentDefaultScopedSlotDirective(t *testing.T) {
	doc := template.Parse(`<List v-slot="item">{{ item.label }}</List>`, 0)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", doc.Diagnostics)
	}
	elem := firstElement(t, doc.Nodes)
	if len(elem.Attrs) != 1 {
		t.Fatalf("attrs = %+v", elem.Attrs)
	}
	if elem.Attrs[0].Kind != template.AttrSlot || elem.Attrs[0].Arg != "default" || elem.Attrs[0].Value != "item" {
		t.Fatalf("default scoped slot attr = %+v", elem.Attrs[0])
	}
}

func TestParseKebabCaseComponentTag(t *testing.T) {
	doc := template.Parse(`<plain-card></plain-card>`, 0)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", doc.Diagnostics)
	}
	elem := firstElement(t, doc.Nodes)
	if elem.Tag != "plain-card" || elem.RawTag != "plain-card" {
		t.Fatalf("element = %+v", elem)
	}
}

func TestParseWithBaseMapsOriginalLineAndColumn(t *testing.T) {
	doc := template.ParseWithBase("<p>{{ count }}</p>", template.BasePosition{
		Offset: 134,
		Line:   8,
		Column: 19,
	})
	elem := firstElement(t, doc.Nodes)
	if elem.StartTag.Start.Offset != 134 || elem.StartTag.Start.Line != 8 || elem.StartTag.Start.Column != 19 {
		t.Fatalf("element start = %+v", elem.StartTag.Start)
	}
	interp, ok := elem.Children[0].(*template.Interpolation)
	if !ok {
		t.Fatalf("expected interpolation, got %T", elem.Children[0])
	}
	if interp.ExprSpan.Start.Offset != 140 || interp.ExprSpan.Start.Line != 8 || interp.ExprSpan.Start.Column != 25 {
		t.Fatalf("interpolation expr start = %+v", interp.ExprSpan.Start)
	}
}

func assertGolden(t *testing.T, name, source string, baseOffset int) {
	t.Helper()
	doc := template.Parse(source, baseOffset)
	goldentest.Assert(t, name, renderDocument(doc))
}

func renderDocument(doc *template.Document) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "diagnostics=%d\n", len(doc.Diagnostics))
	for i, diag := range doc.Diagnostics {
		fmt.Fprintf(&b, "diagnostic[%d]=%s %s %s\n", i, diag.Code, diag.Severity, formatSpan(diag.Span))
	}
	fmt.Fprintf(&b, "nodes=%d\n", len(doc.Nodes))
	for i, node := range doc.Nodes {
		renderNode(&b, fmt.Sprintf("node[%d]", i), node)
	}
	return []byte(b.String())
}

func renderNode(b *strings.Builder, prefix string, node template.Node) {
	switch n := node.(type) {
	case *template.Element:
		fmt.Fprintf(b, "%s.kind=element tag=%s span=%s start=%s", prefix, n.Tag, formatSpan(n.Span), formatSpan(n.StartTag))
		if n.EndTag != nil {
			fmt.Fprintf(b, " end=%s", formatSpan(*n.EndTag))
		}
		fmt.Fprintln(b)
		for i, attr := range n.Attrs {
			valueSpan := "<nil>"
			if attr.ValueSpan != nil {
				valueSpan = formatSpan(*attr.ValueSpan)
			}
			fmt.Fprintf(
				b,
				"%s.attr[%d]=%s kind=%s arg=%s value=%s hasValue=%t name=%s valueSpan=%s\n",
				prefix,
				i,
				attr.Name,
				attr.Kind,
				attr.Arg,
				strconv.Quote(attr.Value),
				attr.HasValue,
				formatSpan(attr.NameSpan),
				valueSpan,
			)
			if attr.Directive != nil {
				fmt.Fprintf(
					b,
					"%s.attr[%d].directive=%s arg=%s expr=%s\n",
					prefix,
					i,
					attr.Directive.Name,
					attr.Directive.Argument,
					strconv.Quote(attr.Directive.Expression),
				)
			}
		}
		for i, child := range n.Children {
			renderNode(b, fmt.Sprintf("%s.child[%d]", prefix, i), child)
		}
	case *template.Text:
		fmt.Fprintf(b, "%s.kind=text span=%s value=%s\n", prefix, formatSpan(n.Span), strconv.Quote(n.Value))
	case *template.Interpolation:
		fmt.Fprintf(b, "%s.kind=interpolation span=%s exprSpan=%s expr=%s\n", prefix, formatSpan(n.Span), formatSpan(n.ExprSpan), strconv.Quote(n.Expression))
	case *template.Comment:
		fmt.Fprintf(b, "%s.kind=comment span=%s value=%s\n", prefix, formatSpan(n.Span), strconv.Quote(n.Value))
	default:
		fmt.Fprintf(b, "%s.kind=unknown %T\n", prefix, node)
	}
}

func firstElement(t *testing.T, nodes []template.Node) *template.Element {
	t.Helper()
	for _, node := range nodes {
		elem, ok := node.(*template.Element)
		if ok {
			return elem
		}
	}
	t.Fatal("no element found")
	return nil
}

func formatSpan(span template.Span) string {
	return formatPosition(span.Start) + "-" + formatPosition(span.End)
}

func formatPosition(pos template.Position) string {
	return fmt.Sprintf("%d:%d@%d", pos.Line, pos.Column, pos.Offset)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
