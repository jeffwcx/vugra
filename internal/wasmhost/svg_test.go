package wasmhost

import "testing"

func TestParseSVGShapesAndPath(t *testing.T) {
	doc, err := parseSVG(`<svg viewBox="0 0 32 32">
		<rect x="4" y="6" width="24" height="20" rx="5" fill="#2563eb"/>
		<circle cx="16" cy="16" r="12" fill="#f97316" />
		<path d="M10 16h12M16 10v12" stroke="#ffffff" stroke-width="3" fill="none"/>
	</svg>`)
	if err != nil {
		t.Fatalf("parseSVG returned error: %v", err)
	}
	if !doc.HasViewBox || doc.ViewBox.Width != 32 || doc.ViewBox.Height != 32 {
		t.Fatalf("unexpected viewBox: %+v has=%v", doc.ViewBox, doc.HasViewBox)
	}
	if len(doc.Shapes) != 3 {
		t.Fatalf("expected 3 shapes, got %d", len(doc.Shapes))
	}
	rect := doc.Shapes[0]
	if rect.Kind != "rect" || rect.X != 4 || rect.Y != 6 || rect.Width != 24 || rect.Height != 20 || rect.RX != 5 || rect.Fill != "#2563eb" {
		t.Fatalf("unexpected rect: %+v", rect)
	}
	circle := doc.Shapes[1]
	if circle.Kind != "circle" || circle.CX != 16 || circle.CY != 16 || circle.R != 12 || circle.Fill != "#f97316" {
		t.Fatalf("unexpected circle: %+v", circle)
	}
	path := doc.Shapes[2]
	if path.Kind != "path" || path.Fill != "none" || path.Stroke != "#ffffff" || path.StrokeWidth != 3 {
		t.Fatalf("unexpected path style: %+v", path)
	}
	if len(path.Path) != 4 {
		t.Fatalf("expected 4 path commands, got %d: %+v", len(path.Path), path.Path)
	}
	want := []svgPathCommand{
		{Op: 'M', X: 10, Y: 16},
		{Op: 'L', X: 22, Y: 16},
		{Op: 'M', X: 16, Y: 10},
		{Op: 'L', X: 16, Y: 22},
	}
	for index, command := range want {
		if path.Path[index] != command {
			t.Fatalf("path command %d = %+v, want %+v", index, path.Path[index], command)
		}
	}
}

func TestParseSVGRelativePath(t *testing.T) {
	path := parseSVGPath("M10 17l4 4 9-11z")
	want := []svgPathCommand{
		{Op: 'M', X: 10, Y: 17},
		{Op: 'L', X: 14, Y: 21},
		{Op: 'L', X: 23, Y: 10},
		{Op: 'Z', X: 10, Y: 17},
	}
	if len(path) != len(want) {
		t.Fatalf("got %d commands, want %d: %+v", len(path), len(want), path)
	}
	for index, command := range want {
		if path[index] != command {
			t.Fatalf("path command %d = %+v, want %+v", index, path[index], command)
		}
	}
}

func TestParseSVGStyleAttribute(t *testing.T) {
	doc, err := parseSVG(`<svg width="10" height="10"><path d="M0 0L10 10" style="fill:none; stroke:#123456; stroke-width:2"/></svg>`)
	if err != nil {
		t.Fatalf("parseSVG returned error: %v", err)
	}
	if len(doc.Shapes) != 1 {
		t.Fatalf("expected one shape, got %d", len(doc.Shapes))
	}
	shape := doc.Shapes[0]
	if shape.Fill != "none" || shape.Stroke != "#123456" || shape.StrokeWidth != 2 {
		t.Fatalf("unexpected style: %+v", shape)
	}
}
