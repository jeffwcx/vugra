package csslayout

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestComputeUsesRustCSSLayoutEngine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := Compute(ctx, sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Boxes) != 3 {
		t.Fatalf("expected 3 boxes, got %d", len(out.Boxes))
	}
	add := boxByID(t, out, "add")
	done := boxByID(t, out, "done")
	if add.Width <= 0 || add.Height != 32 {
		t.Fatalf("unexpected add box size: %+v", add)
	}
	if done.X <= add.X {
		t.Fatalf("expected second flex child to be right of first: add=%+v done=%+v", add, done)
	}
	if add.Style.BorderWidth != 1 || add.Style.BorderRadius != 6 || add.Style.Overflow != "scroll" {
		t.Fatalf("expected CSS paint metadata on add box, got %+v", add.Style)
	}
}

func TestComputeResolvesPercentWidthAgainstParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := Compute(ctx, Input{
		CSS: `.viewport { width: 320px; }
.app { width: 100%; padding: 24px; }
.panel { width: 50%; height: 20px; }`,
		Viewport: Viewport{Width: 320},
		Root: Node{
			ID:    "root",
			Tag:   "div",
			Class: "viewport",
			Children: []Node{
				{
					ID:    "app",
					Tag:   "div",
					Class: "app",
					Children: []Node{
						{ID: "panel", Tag: "div", Class: "panel"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	app := boxByID(t, out, "app")
	panel := boxByID(t, out, "panel")
	if app.Width != 320 {
		t.Fatalf("expected app to fill parent width, got %+v", app)
	}
	if panel.Width != 136 {
		t.Fatalf("expected nested percent width to resolve against parent, got %+v", panel)
	}
}

func TestComputeResolvesPercentMinHeightAgainstParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := Compute(ctx, Input{
		CSS: `.viewport { width: 320px; height: 180px; }
.app { min-height: 100%; }`,
		Viewport: Viewport{Width: 320, Height: float32Ptr(180)},
		Root: Node{
			ID:    "root",
			Tag:   "div",
			Class: "viewport",
			Children: []Node{
				{ID: "app", Tag: "div", Class: "app", Text: "A"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	app := boxByID(t, out, "app")
	if app.Height != 180 {
		t.Fatalf("expected app to use parent height, got %+v", app)
	}
}

func TestCommandSupportsExplicitProcessFallback(t *testing.T) {
	manifest, err := defaultManifestPath()
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := (Engine{ManifestPath: manifest}).command(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cmd.Path); err == nil && cmd.Args[0] == cmd.Path {
		return
	}
	if filepath.Base(cmd.Path) != "cargo" {
		t.Fatalf("expected built binary or cargo fallback, got path=%q args=%v", cmd.Path, cmd.Args)
	}
}

func float32Ptr(value float32) *float32 {
	return &value
}

func TestDefaultComputeIgnoresProcessBinaryWhenFFIAvailable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("default FFI test is currently wired for darwin")
	}
	manifest, err := defaultManifestPath()
	if err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(filepath.Dir(manifest), "target", "debug", "libvuego_css_layout.dylib")
	if _, err := os.Stat(lib); err != nil {
		t.Skipf("build css layout library first: %v", err)
	}
	t.Setenv("VUGRA_CSS_LAYOUT_BIN", "/definitely/missing")
	out, err := Compute(context.Background(), sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Boxes) != 3 {
		t.Fatalf("boxes = %d", len(out.Boxes))
	}
}

func TestComputeWithExplicitFFILibrary(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("explicit FFI library test is currently wired for darwin")
	}
	manifest, err := defaultManifestPath()
	if err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(filepath.Dir(manifest), "target", "debug", "libvuego_css_layout.dylib")
	if _, err := os.Stat(lib); err != nil {
		t.Skipf("build css layout library first: %v", err)
	}
	out, err := (Engine{LibraryPath: lib}).Compute(context.Background(), sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Boxes) != 3 {
		t.Fatalf("boxes = %d", len(out.Boxes))
	}
}

func sampleInput() Input {
	return Input{
		CSS: `.row { display: flex; gap: 12px; width: 220px; }
.item { height: 32px; border: 1px solid #ddd; border-radius: 6px; overflow: scroll; }`,
		Viewport: Viewport{Width: 320},
		Root: Node{
			ID:    "root",
			Tag:   "div",
			Class: "row",
			Children: []Node{
				{ID: "add", Tag: "button", Class: "item", Text: "Add"},
				{ID: "done", Tag: "button", Class: "item", Text: "Done"},
			},
		},
	}
}

func boxByID(t *testing.T, out Output, id string) Box {
	t.Helper()
	for _, box := range out.Boxes {
		if box.ID == id {
			return box
		}
	}
	t.Fatalf("missing box %q in %+v", id, out.Boxes)
	return Box{}
}
