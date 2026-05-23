package renderer_test

import (
	"image/color"
	"testing"

	"github.com/vugra/vugra/internal/renderer"
)

func TestSoftwareRendererPaintsElementsAndText(t *testing.T) {
	r := renderer.NewSoftware(120, 80)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "button",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 0, Y: 0, Width: 100, Height: 60},
		},
		{
			Kind: "text",
			ID:   "text",
			Text: "1",
			Rect: renderer.Rect{X: 10, Y: 10, Width: 8, Height: 20},
		},
	})
	if r.Image == nil {
		t.Fatal("missing rendered image")
	}
	border := r.Image.At(0, 0)
	if border == (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatal("border pixel was not painted")
	}
	textPixel := r.Image.At(12, 14)
	if textPixel == (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatal("text pixel was not painted")
	}
}

func TestSoftwareRendererPaintsCheckboxState(t *testing.T) {
	r := renderer.NewSoftware(40, 40)
	r.Render([]renderer.Command{
		{
			Kind:  "element",
			ID:    "check",
			Tag:   "input",
			Role:  "checkbox",
			Rect:  renderer.Rect{X: 4, Y: 4, Width: 20, Height: 20},
			Props: map[string]string{"checked": "true"},
		},
	})
	if r.Image == nil {
		t.Fatal("missing rendered image")
	}
	fill := r.Image.At(10, 10)
	if fill == (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatal("checkbox fill was not painted")
	}
}

func TestSoftwareRendererUsesCommandColors(t *testing.T) {
	r := renderer.NewSoftware(40, 40)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "card",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 24, Height: 18},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				BorderColor:     "#0f172a",
			},
		},
		{
			Kind:  "text",
			ID:    "text",
			Text:  "A",
			Rect:  renderer.Rect{X: 8, Y: 8, Width: 8, Height: 20},
			Style: renderer.Style{Color: "#ef4444"},
		},
	})
	if got := r.Image.At(4, 4); got != (color.RGBA{R: 15, G: 23, B: 42, A: 255}) {
		t.Fatalf("border color = %#v", got)
	}
	if got := r.Image.At(6, 6); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("fill color = %#v", got)
	}
}

func TestSoftwareRendererAppliesOpacity(t *testing.T) {
	r := renderer.NewSoftware(40, 40)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "card",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 20, Height: 20},
			Style: renderer.Style{
				BackgroundColor: "#2563eb",
				Opacity:         0.5,
			},
		},
	})
	got := color.RGBAModel.Convert(r.Image.At(8, 8)).(color.RGBA)
	if got.A != 127 {
		t.Fatalf("alpha = %d, want 127", got.A)
	}
}

func TestSoftwareRendererSuppressesExplicitZeroControlBorder(t *testing.T) {
	r := renderer.NewSoftware(40, 40)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "row",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 24, Height: 18},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				BorderWidthSet:  true,
				BorderWidth:     0,
			},
		},
	})
	if got := r.Image.At(4, 4); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("explicit zero button border should leave edge as fill, got %#v", got)
	}
	if got := r.Image.At(6, 6); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("fill color = %#v", got)
	}
}

func TestSoftwareRendererUsesBorderWidthAndRadius(t *testing.T) {
	r := renderer.NewSoftware(40, 40)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "card",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 24, Height: 24},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				BorderColor:     "#0f172a",
				BorderWidth:     3,
				BorderRadius:    8,
			},
		},
	})
	if got := r.Image.At(4, 4); got != (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatalf("rounded corner should remain background, got %#v", got)
	}
	if got := r.Image.At(12, 4); got != (color.RGBA{R: 15, G: 23, B: 42, A: 255}) {
		t.Fatalf("top border color = %#v", got)
	}
	if got := r.Image.At(12, 8); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("inner fill color = %#v", got)
	}
}

func TestSoftwareRendererDoesNotPaintPlainTextElements(t *testing.T) {
	r := renderer.NewSoftware(80, 50)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "title",
			Tag:  "h1",
			Role: "heading",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 60, Height: 24},
		},
		{
			Kind: "text",
			ID:   "title:text",
			Text: "Tasks",
			Rect: renderer.Rect{X: 8, Y: 8, Width: 40, Height: 20},
		},
		{
			Kind: "end",
			ID:   "title",
			Tag:  "h1",
		},
	})
	if got := r.Image.At(4, 4); got != (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatalf("plain h1 should not paint a default border, got %#v", got)
	}
	if got := r.Image.At(12, 12); got == (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatalf("text should still be painted")
	}
}

func TestSoftwareRendererClipsOverflowHiddenChildren(t *testing.T) {
	r := renderer.NewSoftware(60, 60)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "clip",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 20, Height: 20},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				Overflow:        "hidden",
			},
		},
		{
			Kind: "element",
			ID:   "outside",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 30, Y: 30, Width: 12, Height: 12},
			Style: renderer.Style{
				BackgroundColor: "#ef4444",
			},
		},
		{
			Kind: "end",
			ID:   "outside",
			Tag:  "div",
		},
		{
			Kind: "end",
			ID:   "clip",
			Tag:  "div",
		},
	})
	if got := r.Image.At(34, 34); got != (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatalf("outside clipped child should remain background, got %#v", got)
	}
	if got := r.Image.At(8, 8); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("clip container fill = %#v", got)
	}
}

func TestSoftwareRendererClipsOverflowScrollChildren(t *testing.T) {
	r := renderer.NewSoftware(60, 60)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			ID:   "scroll",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 4, Y: 4, Width: 20, Height: 20},
			Style: renderer.Style{
				BackgroundColor: "#ffffff",
				Overflow:        "scroll",
			},
		},
		{
			Kind: "element",
			ID:   "outside",
			Tag:  "div",
			Role: "group",
			Rect: renderer.Rect{X: 30, Y: 30, Width: 12, Height: 12},
			Style: renderer.Style{
				BackgroundColor: "#ef4444",
			},
		},
		{
			Kind: "end",
			ID:   "outside",
			Tag:  "div",
		},
		{
			Kind: "end",
			ID:   "scroll",
			Tag:  "div",
		},
	})
	if got := r.Image.At(34, 34); got != (color.RGBA{R: 250, G: 250, B: 250, A: 255}) {
		t.Fatalf("outside clipped child should remain background, got %#v", got)
	}
}
