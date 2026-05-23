//go:build darwin && cgo

package vellonative_test

import (
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/vellonative"
)

func TestRendererUsesRustVelloNativeBridge(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	r, err := vellonative.New(180, 96)
	if err != nil {
		t.Fatalf("new vello-native renderer: %v", err)
	}
	defer r.Close()

	err = r.Render([]renderer.Command{
		{
			Kind: "element",
			Role: "button",
			Tag:  "button",
			Rect: renderer.Rect{X: 12, Y: 12, Width: 144, Height: 40},
		},
		{
			Kind: "text",
			Text: "中文 Todo fi",
			Role: "text",
			Tag:  "p",
			Rect: renderer.Rect{X: 18, Y: 18, Width: 132, Height: 32},
			Style: renderer.Style{
				FontSize:   18,
				LineHeight: 26,
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(r.Pixels()) != 180*96*4 {
		t.Fatalf("pixels = %d", len(r.Pixels()))
	}
	status := r.Status()
	if !strings.Contains(status, `"backend":"vello-native"`) {
		t.Fatalf("status = %q", status)
	}
	if !strings.Contains(status, `"glyphOps":`) {
		t.Fatalf("expected shaped glyph status, got %q", status)
	}
}

func TestMeasurerUsesNativeTextShaping(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	measurer, err := vellonative.NewMeasurer()
	if err != nil {
		t.Fatalf("new vello-native measurer: %v", err)
	}
	defer measurer.Close()

	narrow, narrowHeight := measurer.MeasureStyledText("iiii", 18, 26)
	wide, wideHeight := measurer.MeasureStyledText("WWWW", 18, 26)
	cjk, cjkHeight := measurer.MeasureStyledText("文件", 18, 26)
	if narrow <= 0 || wide <= 0 || cjk <= 0 {
		t.Fatalf("invalid native metrics: narrow=%g wide=%g cjk=%g", narrow, wide, cjk)
	}
	if narrowHeight != 26 || wideHeight != 26 || cjkHeight != 26 {
		t.Fatalf("unexpected heights: narrow=%g wide=%g cjk=%g", narrowHeight, wideHeight, cjkHeight)
	}
	if wide <= narrow {
		t.Fatalf("expected shaped metrics to distinguish wide text from narrow text: narrow=%g wide=%g", narrow, wide)
	}
}

func TestRendererDrawsSVGCommands(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	r, err := vellonative.New(80, 48)
	if err != nil {
		t.Fatalf("new vello-native renderer: %v", err)
	}
	defer r.Close()

	err = r.Render([]renderer.Command{
		{
			Kind: "svg",
			Role: "image",
			Tag:  "svg",
			Rect: renderer.Rect{X: 8, Y: 8, Width: 32, Height: 32},
			SVG:  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="12" fill="#2563eb"/><path d="M10 16h12" stroke="#ffffff" stroke-width="3" fill="none"/></svg>`,
		},
	})
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}
	if len(r.Pixels()) != 80*48*4 {
		t.Fatalf("pixels = %d", len(r.Pixels()))
	}
	if status := r.Status(); !strings.Contains(status, `"svgOps":1`) {
		t.Fatalf("expected svg status, got %q", status)
	}
	pixels := r.Pixels()
	if changed := countNonBackgroundPixels(pixels, 80, 48, 8, 8, 40, 40); changed < 20 {
		t.Fatalf("expected visible svg pixels, got %d changed pixels status=%s sample=%v", changed, r.Status(), samplePixel(pixels, 80, 16, 16))
	}
	if blue := countApproxColorPixels(pixels, 80, 48, 8, 8, 40, 40, 0x25, 0x63, 0xeb, 48); blue < 20 {
		t.Fatalf("expected blue svg pixels, got %d", blue)
	}
}

func TestRendererDrawsFillRectCommands(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	r, err := vellonative.New(80, 48)
	if err != nil {
		t.Fatalf("new vello-native renderer: %v", err)
	}
	defer r.Close()

	err = r.Render([]renderer.Command{
		{
			Kind: "element",
			Role: "group",
			Tag:  "div",
			Rect: renderer.Rect{X: 8, Y: 8, Width: 32, Height: 32},
			Style: renderer.Style{
				BackgroundColor: "#ff0000",
			},
		},
	})
	if err != nil {
		t.Fatalf("render fill rect: %v", err)
	}
	pixels := r.Pixels()
	if changed := countNonBackgroundPixels(pixels, 80, 48, 8, 8, 40, 40); changed < 20 {
		t.Fatalf("expected visible fill rect pixels, got %d changed pixels sample=%v", changed, samplePixel(pixels, 80, 16, 16))
	}
}

func TestRendererSuppressesExplicitZeroControlBorder(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	r, err := vellonative.New(80, 48)
	if err != nil {
		t.Fatalf("new vello-native renderer: %v", err)
	}
	defer r.Close()

	err = r.Render([]renderer.Command{
		{
			Kind: "element",
			Role: "button",
			Tag:  "button",
			Rect: renderer.Rect{X: 8, Y: 8, Width: 32, Height: 20},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				BorderWidthSet:  true,
				BorderWidth:     0,
			},
		},
	})
	if err != nil {
		t.Fatalf("render explicit zero control border: %v", err)
	}
	pixels := r.Pixels()
	if blue := countApproxColorPixels(pixels, 80, 48, 8, 8, 40, 28, 0x25, 0x63, 0xeb, 8); blue != 0 {
		t.Fatalf("explicit zero button border should not draw default blue stroke, got %d pixels", blue)
	}
	if white := countApproxColorPixels(pixels, 80, 48, 8, 8, 40, 28, 0xff, 0xff, 0xff, 4); white < 200 {
		t.Fatalf("expected white button fill, got %d pixels", white)
	}
}

func TestRendererDrawsFinderSVGIconPaths(t *testing.T) {
	if _, err := vellonative.LibraryPath(); err != nil {
		t.Skip(err)
	}
	r, err := vellonative.New(80, 48)
	if err != nil {
		t.Fatalf("new vello-native renderer: %v", err)
	}
	defer r.Close()

	err = r.Render([]renderer.Command{
		{
			Kind: "svg",
			Role: "image",
			Tag:  "svg",
			Rect: renderer.Rect{X: 8, Y: 8, Width: 24, Height: 24},
			SVG:  `<svg class="finder-icon" viewBox="0 0 24 24"><path d="M3 7h7l2 2h9v8.5c0 1.4-.9 2.5-2.2 2.5H5.2C3.9 20 3 18.9 3 17.5V7z" fill="#78b7ff" /><path d="M3 8.5c0-1.4.9-2.5 2.2-2.5h4.5l2 2H21v2H3V8.5z" fill="#4f9cf9" /><path d="M3 10h18v7.5c0 1.4-.9 2.5-2.2 2.5H5.2C3.9 20 3 18.9 3 17.5V10z" fill="#6fb2ff" /></svg>`,
		},
	})
	if err != nil {
		t.Fatalf("render finder svg: %v", err)
	}
	pixels := r.Pixels()
	if changed := countNonBackgroundPixels(pixels, 80, 48, 8, 8, 32, 32); changed < 20 {
		t.Fatalf("expected visible finder svg pixels, got %d changed pixels status=%s sample=%v", changed, r.Status(), samplePixel(pixels, 80, 16, 16))
	}
	if blue := countApproxColorPixels(pixels, 80, 48, 8, 8, 32, 32, 0x6f, 0xb2, 0xff, 56); blue < 20 {
		t.Fatalf("expected finder blue svg pixels, got %d", blue)
	}
}

func countNonBackgroundPixels(pixels []byte, width, height, x0, y0, x1, y1 int) int {
	return countMatchingPixels(pixels, width, height, x0, y0, x1, y1, func(r, g, b, a byte) bool {
		return r != 250 || g != 250 || b != 250 || a != 255
	})
}

func countApproxColorPixels(pixels []byte, width, height, x0, y0, x1, y1 int, wantR, wantG, wantB byte, tolerance int) int {
	return countMatchingPixels(pixels, width, height, x0, y0, x1, y1, func(r, g, b, a byte) bool {
		return a > 0 &&
			absInt(int(r)-int(wantR)) <= tolerance &&
			absInt(int(g)-int(wantG)) <= tolerance &&
			absInt(int(b)-int(wantB)) <= tolerance
	})
}

func countMatchingPixels(pixels []byte, width, height, x0, y0, x1, y1 int, match func(r, g, b, a byte) bool) int {
	changed := 0
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > width {
		x1 = width
	}
	if y1 > height {
		y1 = height
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := (y*width + x) * 4
			if i+3 >= len(pixels) {
				continue
			}
			if match(pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]) {
				changed++
			}
		}
	}
	return changed
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func samplePixel(pixels []byte, width, x, y int) []byte {
	i := (y*width + x) * 4
	if i+3 >= len(pixels) {
		return nil
	}
	return []byte{pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]}
}
