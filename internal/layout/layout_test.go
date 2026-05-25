package layout_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/goldentest"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
)

func TestCounterLayoutGolden(t *testing.T) {
	component := buildComponent(t)
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 200px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		ResolveText: func(binding string) string {
			if binding == "count" {
				return "7"
			}
			return ""
		},
	})
	goldentest.Assert(t, "counter_layout.txt", renderBoxes(boxes))
}

func TestBlockLayoutUsesAvailableWidth(t *testing.T) {
	component := buildComponent(t)
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		ResolveText: func(binding string) string { return "1" },
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 240 {
		t.Fatalf("root width = %g", root.Rect.Width)
	}
}

func TestPercentWidthUsesAvailableWidth(t *testing.T) {
	templateDoc := template.Parse(`<div class="fill"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "PercentWidth", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.fill {
  width: 100%;
  padding: 10px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 240 {
		t.Fatalf("percent width rect = %+v", root.Rect)
	}
	if len(root.Children) != 1 || root.Children[0].Rect.Width != 220 {
		t.Fatalf("percent width child = %+v", root.Children)
	}
}

func TestBoxSizingBorderBoxKeepsExplicitOuterSize(t *testing.T) {
	templateDoc := template.Parse(`<div class="panel"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "BorderBox", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.panel {
  box-sizing: border-box;
  width: 200px;
  height: 80px;
  padding: 20px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 200 || root.Rect.Height != 80 {
		t.Fatalf("border-box rect = %+v", root.Rect)
	}
	if len(root.Children) != 1 || root.Children[0].Rect.X != 20 || root.Children[0].Rect.Width != 160 {
		t.Fatalf("border-box child = %+v", root.Children)
	}
}

func TestBoxSizingContentBoxAddsPaddingToExplicitSize(t *testing.T) {
	templateDoc := template.Parse(`<div class="panel"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "ContentBox", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.panel {
  box-sizing: content-box;
  width: 200px;
  height: 80px;
  padding: 20px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 240 || root.Rect.Height != 120 {
		t.Fatalf("content-box rect = %+v", root.Rect)
	}
	if len(root.Children) != 1 || root.Children[0].Rect.X != 20 || root.Children[0].Rect.Width != 200 {
		t.Fatalf("content-box child = %+v", root.Children)
	}
}

func TestSystemTokenPaddingLeftAffectsChildLayout(t *testing.T) {
	templateDoc := template.Parse(`<div class="toolbar"><button>Back</button></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Toolbar", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.toolbar {
  display: flex;
  padding: 10px;
  padding-left: env(vugra-window-controls-left, 10px);
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		SystemTokens: style.SystemTokens{
			"vugra-window-controls-left": 72,
		},
	})
	root := firstElement(t, boxes)
	if root.Style.PaddingLeft != 72 {
		t.Fatalf("root style = %+v", root.Style)
	}
	if len(root.Children) != 1 || root.Children[0].Rect.X != 72 {
		t.Fatalf("toolbar child should start after window controls token: %+v", root.Children)
	}
}

func TestOpacityPropagatesToElementAndTextStyle(t *testing.T) {
	templateDoc := template.Parse(`<div class="panel"><p>Dim</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Opacity", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.panel {
  opacity: 0.5;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Style.Opacity != 0.5 {
		t.Fatalf("root opacity = %g", root.Style.Opacity)
	}
	text := firstText(t, root)
	if text.Style.Opacity != 0.5 {
		t.Fatalf("text opacity = %g", text.Style.Opacity)
	}
}

func TestInlineSVGProducesSVGBox(t *testing.T) {
	templateDoc := template.Parse(`<svg class="icon" viewBox="0 0 24 24"><path d="M2 12h20" stroke="#111"/></svg>`, 0)
	component := ir.Build(ir.BuildInput{Name: "InlineSVG", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.icon { width: 24px; height: 24px; }`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 100},
	})
	root := firstBoxKind(t, boxes, "svg")
	if root.Kind != "svg" || root.Role != "image" {
		t.Fatalf("svg box kind/role = %s/%s", root.Kind, root.Role)
	}
	if !strings.Contains(root.SVG, `<path`) || !strings.Contains(root.SVG, `viewBox="0 0 24 24"`) {
		t.Fatalf("svg markup = %q", root.SVG)
	}
	if len(root.Children) != 0 {
		t.Fatalf("inline svg children should not be laid out separately: %+v", root.Children)
	}
}

func TestImageSVGResolvesAsset(t *testing.T) {
	templateDoc := template.Parse(`<img class="icon" src="./icon.svg" />`, 0)
	component := ir.Build(ir.BuildInput{Name: "ImageSVG", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.icon { width: 16px; height: 16px; }`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 100},
		ResolveAsset: func(path string) (string, bool) {
			if path != "./icon.svg" {
				t.Fatalf("asset path = %q", path)
			}
			return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"></svg>`, true
		},
	})
	root := firstBoxKind(t, boxes, "svg")
	if root.Kind != "svg" || !strings.Contains(root.SVG, `<svg`) {
		t.Fatalf("svg image box = %+v", root)
	}
}

func TestImageSVGUsesInlineRawSVG(t *testing.T) {
	templateDoc := template.Parse(`<img class="icon" src="./icon.svg" __raw_svg="<svg viewBox='0 0 16 16'></svg>" />`, 0)
	component := ir.Build(ir.BuildInput{Name: "ImageSVG", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.icon { width: 16px; height: 16px; }`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 100},
		ResolveAsset: func(path string) (string, bool) {
			t.Fatalf("inline raw svg should not resolve asset path %q", path)
			return "", false
		},
	})
	root := firstBoxKind(t, boxes, "svg")
	if root.Kind != "svg" || !strings.Contains(root.SVG, `viewBox='0 0 16 16'`) {
		t.Fatalf("svg image box = %+v", root)
	}
}

func TestRootHeightUsesConstraints(t *testing.T) {
	templateDoc := template.Parse(`<div class="app"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "RootHeight", Template: templateDoc, Go: goanalysis.Metadata{}})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Constraints: layout.Constraints{Width: 240, Height: 180},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Height != 180 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
}

func TestRootPercentHeightUsesConstraints(t *testing.T) {
	templateDoc := template.Parse(`<div class="app"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "RootPercentHeight", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.app { height: 100%; }`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240, Height: 180},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Height != 180 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
}

func TestLaterBlockFlexChildReceivesRemainingHeight(t *testing.T) {
	templateDoc := template.Parse(`
<div class="app">
  <div class="toolbar"></div>
  <div class="main"></div>
  <div class="status"></div>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "BlockRemainingHeight", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.app {
  height: 100%;
}
.toolbar {
  height: 52px;
}
.main {
  flex: 1;
}
.status {
  height: 28px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240, Height: 180},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if len(root.Children) != 3 {
		t.Fatalf("children = %d", len(root.Children))
	}
	if root.Children[1].Rect.Height != 100 {
		t.Fatalf("main rect = %+v", root.Children[1].Rect)
	}
	if root.Children[2].Rect.Y != 152 {
		t.Fatalf("status rect = %+v", root.Children[2].Rect)
	}
}

func TestDynamicComponentChoosesCaseFromBinding(t *testing.T) {
	nodes := []ir.Node{&ir.DynamicComponent{
		Binding: "current",
		Cases: []ir.DynamicComponentCase{
			{Alias: "Card", Nodes: []ir.Node{&ir.Element{Tag: "p", Children: []ir.Node{&ir.Text{Value: "card"}}}}},
			{Alias: "Badge", Nodes: []ir.Node{&ir.Element{Tag: "span", Children: []ir.Node{&ir.Text{Value: "badge"}}}}},
		},
	}}
	boxes := layout.Compute(layout.Input{
		Nodes:       nodes,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		ResolveText: func(binding string) string {
			if binding == "current" {
				return "Badge"
			}
			return ""
		},
	})
	root := firstElement(t, boxes)
	if root.Tag != "span" || len(root.Children) != 1 || root.Children[0].Text != "badge" {
		t.Fatalf("dynamic component box = %+v", root)
	}
}

func TestFlexRowLayout(t *testing.T) {
	component := buildTwoButtonComponent(t)
	sheet := style.Parse(`
.toolbar {
  display: flex;
  flex-direction: row;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 8px;
  width: 240px;
  height: 80px;
}
.tool {
  width: 48px;
  height: 32px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 240 || root.Rect.Height != 80 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d", len(root.Children))
	}
	if root.Children[0].Rect.X != 66 || root.Children[1].Rect.X != 126 {
		t.Fatalf("child x positions = %g, %g", root.Children[0].Rect.X, root.Children[1].Rect.X)
	}
	if root.Children[0].Rect.Y != 8 {
		t.Fatalf("child y = %g", root.Children[0].Rect.Y)
	}
}

func TestFlexRowUsesAvailableCrossAxisHeight(t *testing.T) {
	templateDoc := template.Parse(`
<div class="main">
  <div class="sidebar"></div>
  <div class="content"></div>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "FlexRowCrossAxis", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.main {
  display: flex;
  flex-direction: row;
  height: 100%;
}
.sidebar {
  width: 80px;
  height: 100%;
}
.content {
  flex: 1;
  height: 100%;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300, Height: 180},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Height != 180 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
	if root.Children[0].Rect.Height != 180 || root.Children[1].Rect.Height != 180 {
		t.Fatalf("child rects = %+v", root.Children)
	}
}

func TestFlexRowJustifySpaceBetweenDistributesRemainingWidth(t *testing.T) {
	templateDoc := template.Parse(`
<div class="statusbar">
  <p>12 items</p>
  <p>0 selected</p>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "StatusBar", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.statusbar {
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
  width: 800px;
  height: 28px;
  padding: 6px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 800, Height: 28},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if len(root.Children) != 2 {
		t.Fatalf("children = %d", len(root.Children))
	}
	firstText := root.Children[0].Children[0]
	secondText := root.Children[1].Children[0]
	if firstText.Rect.X != 6 {
		t.Fatalf("first text x = %g", firstText.Rect.X)
	}
	if secondText.Rect.X != 714 {
		t.Fatalf("second text x = %g", secondText.Rect.X)
	}
}

func TestFlexRowMovesNestedChildrenWithParent(t *testing.T) {
	templateDoc := template.Parse(`
<div class="toolbar">
  <button class="tool"><span>A</span></button>
  <button class="tool"><span>B</span></button>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "ToolbarNested", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.toolbar {
  display: flex;
  flex-direction: row;
  justify-content: center;
  gap: 12px;
  width: 240px;
}
.tool {
  width: 48px;
  height: 32px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	first := root.Children[0]
	if first.Rect.X != 66 {
		t.Fatalf("button x = %g", first.Rect.X)
	}
	if len(first.Children) != 1 || len(first.Children[0].Children) != 1 {
		t.Fatalf("unexpected nested children: %+v", first.Children)
	}
	text := first.Children[0].Children[0]
	if text.Rect.X != 66 {
		t.Fatalf("nested text x = %g, want moved with button", text.Rect.X)
	}
}

func TestFlexWrapLayout(t *testing.T) {
	templateDoc := template.Parse(`
<div class="wrap">
  <button class="tool">A</button>
  <button class="tool">B</button>
  <button class="tool">C</button>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "Wrap", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.wrap {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 8px;
  width: 120px;
}
.tool {
  width: 56px;
  height: 20px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 200},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Height != 48 {
		t.Fatalf("root height = %g", root.Rect.Height)
	}
	if root.Children[0].Rect.X != 0 || root.Children[1].Rect.X != 64 || root.Children[2].Rect.Y != 28 {
		t.Fatalf("wrapped children = %+v", root.Children)
	}
}

func TestFlexGrowDistributesRemainingWidth(t *testing.T) {
	templateDoc := template.Parse(`
<div class="bar">
  <button class="fixed">A</button>
  <button class="grow">B</button>
  <button class="grow2">C</button>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "FlexGrow", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.bar {
  display: flex;
  flex-direction: row;
  gap: 10px;
  width: 300px;
}
.fixed {
  width: 50px;
  height: 20px;
}
.grow {
  width: 50px;
  height: 20px;
  flex-grow: 1;
}
.grow2 {
  width: 50px;
  height: 20px;
  flex-grow: 2;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 320},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Children[0].Rect.Width != 50 || !near(root.Children[1].Rect.Width, 50+130.0/3.0) || !near(root.Children[2].Rect.Width, 50+260.0/3.0) {
		t.Fatalf("grown widths = %g, %g, %g", root.Children[0].Rect.Width, root.Children[1].Rect.Width, root.Children[2].Rect.Width)
	}
	if root.Children[1].Rect.X != 60 || !near(root.Children[2].Rect.X, 60+50+130.0/3.0+10) {
		t.Fatalf("grown x positions = %g, %g", root.Children[1].Rect.X, root.Children[2].Rect.X)
	}
}

func TestTranslatedFlexTextLinesMoveWithTextRect(t *testing.T) {
	templateDoc := template.Parse(`
<div class="row">
  <button class="left">Left</button>
  <button class="right">Right</button>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "TextTranslate", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.row {
  display: flex;
  gap: 10px;
  width: 220px;
}
.left {
  width: 60px;
  height: 30px;
}
.right {
  width: 80px;
  height: 30px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	rightText := root.Children[1].Children[0]
	if rightText.Rect.X < root.Children[1].Rect.X {
		t.Fatalf("right text rect was not translated: text=%+v button=%+v", rightText.Rect, root.Children[1].Rect)
	}
	if len(rightText.Lines) == 0 || rightText.Lines[0].X < root.Children[1].Rect.X {
		t.Fatalf("right text line was not translated with rect: lines=%+v button=%+v", rightText.Lines, root.Children[1].Rect)
	}
}

func TestGridLayout(t *testing.T) {
	component := buildGridComponent(t)
	sheet := style.Parse(`
.grid {
  display: grid;
  grid-template-columns: 80px 1fr;
  grid-template-rows: 40px 50px;
  gap: 10px;
  padding: 5px;
  width: 220px;
}
.cell {
  height: 30px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 220 || root.Rect.Height != 110 {
		t.Fatalf("root rect = %+v", root.Rect)
	}
	if len(root.Children) != 4 {
		t.Fatalf("children = %d", len(root.Children))
	}
	if root.Children[0].Rect.X != 5 || root.Children[0].Rect.Y != 5 {
		t.Fatalf("cell 0 rect = %+v", root.Children[0].Rect)
	}
	if root.Children[1].Rect.X != 95 || root.Children[1].Rect.Width != 120 {
		t.Fatalf("cell 1 rect = %+v", root.Children[1].Rect)
	}
	if root.Children[2].Rect.Y != 55 {
		t.Fatalf("cell 2 rect = %+v", root.Children[2].Rect)
	}
}

func TestGridFrRowPassesAvailableHeightToChild(t *testing.T) {
	templateDoc := template.Parse(`
<div class="grid">
  <div class="fill"></div>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "GridFrHeight", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.grid {
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
  height: 100%;
}
.fill {
  flex: 1;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 240, Height: 180},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Height != 180 || root.Children[0].Rect.Height != 180 {
		t.Fatalf("grid rects root=%+v child=%+v", root.Rect, root.Children[0].Rect)
	}
}

func TestGridPlacementSpansTracks(t *testing.T) {
	templateDoc := template.Parse(`
<div class="grid">
  <div class="hero">A</div>
  <div class="side">B</div>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "GridPlacement", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.grid {
  display: grid;
  grid-template-columns: 80px 1fr 40px;
  grid-template-rows: 30px 50px;
  gap: 10px;
  width: 240px;
}
.hero {
  grid-column: 1 / span 2;
  grid-row: 2;
  height: 20px;
}
.side {
  grid-column: 3;
  grid-row: 1 / span 2;
  height: 60px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 260},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Children[0].Rect.X != 0 || root.Children[0].Rect.Y != 40 || root.Children[0].Rect.Width != 190 {
		t.Fatalf("hero rect = %+v", root.Children[0].Rect)
	}
	if root.Children[1].Rect.X != 200 || root.Children[1].Rect.Y != 0 || root.Children[1].Rect.Width != 40 {
		t.Fatalf("side rect = %+v", root.Children[1].Rect)
	}
}

func TestMarginParticipatesInBlockFlow(t *testing.T) {
	templateDoc := template.Parse(`
<div>
  <p class="card">A</p>
  <p class="card">B</p>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "Margin", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.card {
  margin: 4px;
  width: 40px;
  height: 20px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 100},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Children[0].Rect.X != 4 || root.Children[0].Rect.Y != 4 {
		t.Fatalf("first child rect = %+v", root.Children[0].Rect)
	}
	if root.Children[1].Rect.Y != 32 {
		t.Fatalf("second child y = %g", root.Children[1].Rect.Y)
	}
	if root.Rect.Height != 56 {
		t.Fatalf("root height = %g", root.Rect.Height)
	}
}

func TestMinMaxSizeClampsLayout(t *testing.T) {
	templateDoc := template.Parse(`<div class="panel"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Clamp", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	sheet := style.Parse(`
.panel {
  width: 400px;
  max-width: 180px;
  min-height: 80px;
  max-height: 90px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 500},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if root.Rect.Width != 180 || root.Rect.Height != 80 {
		t.Fatalf("clamped rect = %+v", root.Rect)
	}
}

func TestTypographyAffectsTextMeasurement(t *testing.T) {
	templateDoc := template.Parse(`<p class="title">Hello</p>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Typography", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.title {
  font-size: 20px;
  line-height: 32px;
  text-align: center;
  width: 200px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	if len(root.Children) != 1 {
		t.Fatalf("children = %d", len(root.Children))
	}
	text := root.Children[0]
	if text.Rect.Width != 50 || text.Rect.Height != 32 {
		t.Fatalf("text rect = %+v", text.Rect)
	}
	if text.Rect.X != 75 {
		t.Fatalf("text x = %g", text.Rect.X)
	}
}

func TestTypographyUsesStyledMeasurerWhenAvailable(t *testing.T) {
	templateDoc := template.Parse(`<p class="title">Hello</p>`, 0)
	component := ir.Build(ir.BuildInput{Name: "StyledTypography", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.title {
  font-size: 20px;
  line-height: 32px;
}
`, style.BasePosition{})
	measurer := &recordingStyledMeasurer{width: 73, height: 32}
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 300},
		Measurer:    measurer,
	})
	text := firstElement(t, boxes).Children[0]
	if text.Rect.Width != 73 || text.Rect.Height != 32 {
		t.Fatalf("text rect = %+v", text.Rect)
	}
	if text.Glyphs[0].Advance != 73 {
		t.Fatalf("glyph advance = %g", text.Glyphs[0].Advance)
	}
	if measurer.lastText != "Hello" || measurer.lastFontSize != 20 || measurer.lastLineHeight != 32 {
		t.Fatalf("measurer input = text %q size %g line-height %g", measurer.lastText, measurer.lastFontSize, measurer.lastLineHeight)
	}
}

func TestTextWrapsToAvailableWidth(t *testing.T) {
	templateDoc := template.Parse(`<p class="copy">HelloWorld</p>`, 0)
	component := ir.Build(ir.BuildInput{Name: "WrapText", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`
.copy {
  width: 32px;
  line-height: 10px;
}
`, style.BasePosition{})
	boxes := layout.Compute(layout.Input{
		Nodes:       component.Nodes,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 100},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	root := firstElement(t, boxes)
	text := root.Children[0]
	if text.Text != "Hell\noWor\nld" {
		t.Fatalf("wrapped text = %q", text.Text)
	}
	if text.Rect.Width != 32 || text.Rect.Height != 30 {
		t.Fatalf("wrapped text rect = %+v", text.Rect)
	}
}

func buildComponent(t *testing.T) *ir.Component {
	t.Helper()
	templateDoc := template.Parse(`
<div class="counter">
  <p>{{ count }}</p>
  <button @click="Inc">+</button>
</div>
`, 0)
	goMeta := goanalysis.Analyze(`
type State struct {
    Count signal.Int `+"`vugra:\"count\"`"+`
}
func (s *State) Inc() {}
`, goanalysis.BasePosition{})
	component := ir.Build(ir.BuildInput{Name: "Counter", Template: templateDoc, Go: goMeta})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	return component
}

func buildTwoButtonComponent(t *testing.T) *ir.Component {
	t.Helper()
	templateDoc := template.Parse(`
<div class="toolbar">
  <button class="tool">A</button>
  <button class="tool">B</button>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "Toolbar", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	return component
}

func buildGridComponent(t *testing.T) *ir.Component {
	t.Helper()
	templateDoc := template.Parse(`
<div class="grid">
  <div class="cell">A</div>
  <div class="cell">B</div>
  <div class="cell">C</div>
  <div class="cell">D</div>
</div>
`, 0)
	component := ir.Build(ir.BuildInput{Name: "Grid", Template: templateDoc, Go: goanalysis.Metadata{}})
	if len(component.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", component.Diagnostics)
	}
	return component
}

func renderBoxes(boxes []layout.Box) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "boxes=%d\n", len(boxes))
	for i, box := range boxes {
		renderBox(&b, fmt.Sprintf("box[%d]", i), box)
	}
	return []byte(b.String())
}

func renderBox(b *strings.Builder, prefix string, box layout.Box) {
	fmt.Fprintf(b, "%s=%s tag=%s role=%s text=%s rect=%g,%g,%g,%g children=%d\n", prefix, box.Kind, box.Tag, box.Role, strconv.Quote(box.Text), box.Rect.X, box.Rect.Y, box.Rect.Width, box.Rect.Height, len(box.Children))
	for i, child := range box.Children {
		renderBox(b, fmt.Sprintf("%s.child[%d]", prefix, i), child)
	}
}

func firstElement(t *testing.T, boxes []layout.Box) layout.Box {
	t.Helper()
	for _, box := range boxes {
		if box.Kind == "element" {
			return box
		}
	}
	t.Fatal("no element box")
	return layout.Box{}
}

func firstBoxKind(t *testing.T, boxes []layout.Box, kind string) layout.Box {
	t.Helper()
	for _, box := range boxes {
		if found, ok := findBoxKind(box, kind); ok {
			return found
		}
	}
	t.Fatalf("no %s box", kind)
	return layout.Box{}
}

func firstText(t *testing.T, box layout.Box) layout.Box {
	t.Helper()
	if box.Kind == "text" {
		return box
	}
	for _, child := range box.Children {
		if found, ok := findBoxKind(child, "text"); ok {
			return found
		}
	}
	t.Fatal("no text box")
	return layout.Box{}
}

func findBoxKind(box layout.Box, kind string) (layout.Box, bool) {
	if box.Kind == kind {
		return box, true
	}
	for _, child := range box.Children {
		if found, ok := findBoxKind(child, kind); ok {
			return found, true
		}
	}
	return layout.Box{}, false
}

type recordingStyledMeasurer struct {
	width          float32
	height         float32
	lastText       string
	lastFontSize   float32
	lastLineHeight float32
}

func (m *recordingStyledMeasurer) MeasureText(text string) (float32, float32) {
	return float32(len([]rune(text))) * 8, 20
}

func (m *recordingStyledMeasurer) MeasureStyledText(text string, fontSize, lineHeight float32) (float32, float32) {
	m.lastText = text
	m.lastFontSize = fontSize
	m.lastLineHeight = lineHeight
	return m.width, m.height
}

func near(got, want float32) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.0001
}
