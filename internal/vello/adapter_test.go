package vello_test

import (
	"testing"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/vello"
)

func TestTranslateRenderCommandsToVelloOps(t *testing.T) {
	ops := vello.Translate([]renderer.Command{
		{
			Kind: "element",
			Role: "button",
			Rect: renderer.Rect{X: 1.25, Y: 2.5, Width: 30.75, Height: 40.125},
			Style: renderer.Style{
				Opacity: 0.5,
			},
		},
		{
			Kind: "text",
			Text: "Go",
			Lines: []renderer.LineBox{{
				Text:     "Go",
				X:        4.5,
				Y:        5.25,
				Width:    16.75,
				Height:   20.5,
				Baseline: 14.25,
			}},
			Glyphs: []renderer.GlyphRun{{
				Text:     "Go",
				Size:     16,
				X:        4.5,
				Y:        5.25,
				Advance:  16.75,
				Baseline: 14.25,
			}},
			Role: "text",
			Rect: renderer.Rect{X: 4.5, Y: 5.25, Width: 16.75, Height: 20.5},
		},
	})
	if len(ops) != 3 {
		t.Fatalf("ops = %+v", ops)
	}
	if ops[0].Kind != "fill-rect" || ops[1].Kind != "stroke-rect" {
		t.Fatalf("element ops = %+v", ops[:2])
	}
	if ops[2].Kind != "text" || ops[2].Text != "Go" {
		t.Fatalf("text op = %+v", ops[2])
	}
	if ops[0].Rect.X != 1.25 || ops[0].Rect.Width != 30.75 {
		t.Fatalf("fractional element rect was truncated: %+v", ops[0].Rect)
	}
	if ops[0].Style.Opacity != 0.5 || ops[1].Style.Opacity != 0.5 {
		t.Fatalf("opacity not preserved in element ops: %+v %+v", ops[0].Style, ops[1].Style)
	}
	if len(ops[2].Lines) != 1 || ops[2].Lines[0].Baseline != 14.25 || len(ops[2].Glyphs) != 1 || ops[2].Glyphs[0].Advance != 16.75 {
		t.Fatalf("text layout data not preserved: %+v", ops[2])
	}
}

func TestTranslateOverflowHiddenToClipOps(t *testing.T) {
	ops := vello.Translate([]renderer.Command{
		{
			Kind:  "element",
			ID:    "clip",
			Tag:   "div",
			Rect:  renderer.Rect{X: 1, Y: 2, Width: 30, Height: 40},
			Style: renderer.Style{Overflow: "hidden"},
		},
		{
			Kind: "text",
			Text: "Go",
			Role: "text",
			Rect: renderer.Rect{X: 4, Y: 5, Width: 16, Height: 20},
		},
		{
			Kind: "end",
			ID:   "clip",
			Tag:  "div",
			Rect: renderer.Rect{X: 1, Y: 2, Width: 30, Height: 40},
		},
	})
	kinds := make([]string, len(ops))
	for i, op := range ops {
		kinds[i] = op.Kind
	}
	want := []string{"fill-rect", "stroke-rect", "begin-clip", "text", "end-clip"}
	if len(kinds) != len(want) {
		t.Fatalf("ops = %+v", ops)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("op kinds = %v", kinds)
		}
	}
	if ops[2].Rect != (renderer.Rect{X: 1, Y: 2, Width: 30, Height: 40}) {
		t.Fatalf("clip rect = %+v", ops[2].Rect)
	}
}

func TestTranslateOverflowScrollToClipOps(t *testing.T) {
	ops := vello.Translate([]renderer.Command{
		{
			Kind:  "element",
			ID:    "scroll",
			Tag:   "div",
			Rect:  renderer.Rect{X: 1, Y: 2, Width: 30, Height: 40},
			Style: renderer.Style{Overflow: "scroll"},
		},
		{
			Kind: "end",
			ID:   "scroll",
			Tag:  "div",
		},
	})
	if len(ops) != 4 {
		t.Fatalf("ops = %+v", ops)
	}
	if ops[2].Kind != "begin-clip" || ops[3].Kind != "end-clip" {
		t.Fatalf("clip ops = %+v", ops)
	}
}

func TestBackendStatusDocumentsMissingLinkedVello(t *testing.T) {
	r := vello.New()
	if err := r.BackendStatus(); err == nil {
		t.Fatal("expected missing backend status")
	}
}

func TestNativeRendererFallsBackWhenSidecarCannotRun(t *testing.T) {
	t.Setenv("PATH", "")
	r := vello.NewNativeRenderer(64, 64)
	r.Render([]renderer.Command{
		{
			Kind: "element",
			Role: "button",
			Rect: renderer.Rect{X: 1, Y: 2, Width: 30, Height: 20},
		},
		{
			Kind: "text",
			Text: "1",
			Role: "text",
			Rect: renderer.Rect{X: 4, Y: 5, Width: 8, Height: 20},
		},
	})
	if len(r.Pixels) != 64*64*4 {
		t.Fatalf("pixels = %d", len(r.Pixels))
	}
	if r.Status == "" {
		t.Fatal("expected fallback status")
	}
}
