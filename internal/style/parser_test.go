package style_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goldentest"
	"github.com/vugra/vugra/internal/style"
)

func TestParseClassRulesGolden(t *testing.T) {
	source := `
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 320px;
}

.button {
  height: 44px;
  align-items: center;
}
`
	sheet := style.Parse(source, style.BasePosition{Offset: 200, Line: 20, Column: 8})
	goldentest.Assert(t, "class_rules.txt", renderStylesheet(sheet))
}

func TestComputeClassStyle(t *testing.T) {
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
}
.wide {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-width: 200px;
  max-width: 480px;
  min-height: 60px;
  max-height: 120px;
  display: grid;
  flex: 2 1 40px;
  grid-template-columns: 120px 1fr 2fr;
  grid-column: 1 / span 2;
  grid-row: 2 / 4;
  row-gap: 6px;
  column-gap: 10px;
  margin: 4px;
  font-family: system-ui;
  font-size: 18px;
  font-weight: 600;
  line-height: 24px;
  text-align: center;
  white-space: normal;
  background: #f8fafc;
  background-color: #ffffff;
  opacity: 0.85;
  border: 2px solid #cbd5e1;
  border-radius: 8px;
  color: #0f172a;
  overflow: hidden;
  overflow-y: scroll;
}
`, style.BasePosition{})
	computed := style.Compute(sheet, "counter wide")
	if computed.Display != "grid" ||
		computed.BoxSizing != "border-box" ||
		computed.FlexDirection != "column" ||
		computed.Gap != 8 ||
		computed.Padding != 16 ||
		computed.WidthPercent != 100 ||
		computed.HeightPercent != 100 ||
		computed.MinWidth != 200 ||
		computed.MaxWidth != 480 ||
		computed.MinHeight != 60 ||
		computed.MaxHeight != 120 ||
		computed.FlexGrow != 2 ||
		computed.FlexShrink != 1 ||
		computed.FlexBasis != 40 ||
		len(computed.GridTemplateColumns) != 3 ||
		computed.GridColumn.Start != 1 ||
		computed.GridColumn.Span != 2 ||
		computed.GridRow.Start != 2 ||
		computed.GridRow.Span != 2 ||
		computed.RowGap != 6 ||
		computed.ColumnGap != 10 ||
		computed.Margin != 4 ||
		computed.FontFamily != "system-ui" ||
		computed.FontSize != 18 ||
		computed.FontWeight != "600" ||
		computed.LineHeight != 24 ||
		computed.TextAlign != "center" ||
		computed.WhiteSpace != "normal" ||
		computed.BackgroundColor != "#ffffff" ||
		computed.Opacity != 0.85 ||
		computed.BorderWidth != 2 ||
		computed.BorderStyle != "solid" ||
		computed.BorderColor != "#cbd5e1" ||
		computed.BorderRadius != 8 ||
		computed.Color != "#0f172a" ||
		computed.Overflow != "hidden" ||
		computed.OverflowX != "hidden" ||
		computed.OverflowY != "scroll" {
		t.Fatalf("unexpected computed style: %+v", computed)
	}
}

func TestBackgroundShorthandColorFeedsBackgroundColor(t *testing.T) {
	sheet := style.Parse(`
.panel {
  background: #123456;
}
`, style.BasePosition{})
	computed := style.Compute(sheet, "panel")
	if computed.Background != "#123456" || computed.BackgroundColor != "#123456" {
		t.Fatalf("computed background = %+v", computed)
	}
}

func TestComputeSystemTokenEnvValues(t *testing.T) {
	sheet := style.Parse(`
.toolbar {
  padding: 10px;
  padding-left: env(vugra-window-controls-left, 10px);
}
`, style.BasePosition{})
	computed := style.ComputeWithTokens(sheet, "toolbar", style.SystemTokens{
		"vugra-window-controls-left": 72,
	})
	if computed.Padding != 10 || computed.PaddingLeft != 72 {
		t.Fatalf("token computed style = %+v", computed)
	}
	computed = style.ComputeWithTokens(sheet, "toolbar", nil)
	if computed.Padding != 10 || computed.PaddingLeft != 10 {
		t.Fatalf("fallback computed style = %+v", computed)
	}
}

func TestParseCSSWithTdewolffParserValues(t *testing.T) {
	sheet := style.Parse(`
/* keep parser on CSS grammar */
.grid {
  grid-template-columns: minmax(80px, 1fr) 2fr;
  width: calc(100px + 20px);
  transform: scale(2);
}
`, style.BasePosition{})
	if !containsDiag(sheet.Diagnostics, "style.unsupported_property") {
		t.Fatalf("expected unsupported property diagnostics, got %+v", sheet.Diagnostics)
	}
	if len(sheet.Rules) != 1 || len(sheet.Rules[0].Declarations) != 3 {
		t.Fatalf("rules = %+v", sheet.Rules)
	}
	if got := sheet.Rules[0].Declarations[0].Value; got != "minmax(80px,1fr) 2fr" {
		t.Fatalf("grid-template-columns value = %q", got)
	}
	if got := sheet.Rules[0].Declarations[1].Value; got != "calc(100px + 20px)" {
		t.Fatalf("width value = %q", got)
	}
}

func TestStyleDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
	}{
		{name: "unsupported selector", src: `div { display: flex; }`, code: "style.unsupported_selector"},
		{name: "missing brace", src: `.counter { display: flex;`, code: "style.missing_closing_brace"},
		{name: "missing colon", src: `.counter { display flex; }`, code: "style.missing_colon"},
		{name: "unsupported property", src: `.counter { transform: scale(2); }`, code: "style.unsupported_property"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet := style.Parse(tt.src, style.BasePosition{})
			if !containsDiag(sheet.Diagnostics, tt.code) {
				t.Fatalf("expected %s, got %+v", tt.code, sheet.Diagnostics)
			}
		})
	}
}

func renderStylesheet(sheet *style.Stylesheet) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "diagnostics=%d\n", len(sheet.Diagnostics))
	for i, diag := range sheet.Diagnostics {
		fmt.Fprintf(&b, "diagnostic[%d]=%s %s %s %s\n", i, diag.Code, diag.Severity, formatSpan(diag.Span), strconv.Quote(diag.Message))
	}
	fmt.Fprintf(&b, "rules=%d\n", len(sheet.Rules))
	for i, rule := range sheet.Rules {
		fmt.Fprintf(&b, "rule[%d]=%s class=%s span=%s declarations=%d\n", i, rule.Selector, rule.ClassName, formatSpan(rule.Span), len(rule.Declarations))
		for j, decl := range rule.Declarations {
			fmt.Fprintf(&b, "rule[%d].decl[%d]=%s:%s name=%s value=%s\n", i, j, decl.Name, decl.Value, formatSpan(decl.NameSpan), formatSpan(decl.ValueSpan))
		}
	}
	return []byte(b.String())
}

func formatSpan(span style.Span) string {
	return formatPosition(span.Start) + "-" + formatPosition(span.End)
}

func formatPosition(pos style.Position) string {
	return fmt.Sprintf("%d:%d@%d", pos.Line, pos.Column, pos.Offset)
}

func containsDiag(diags []style.Diagnostic, code string) bool {
	for _, diag := range diags {
		if diag.Code == code {
			return true
		}
	}
	return false
}
