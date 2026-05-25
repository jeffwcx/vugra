//go:build darwin && cgo && vuego_native_window

package nativewindow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
)

func TestNativeWindowDispatchMouseRepaints(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	window, count := openCounterWindow(t)
	defer window.CloseForTest()
	window.dispatchMouse(20, 50)
	if window.Clicks != 1 {
		t.Fatalf("clicks = %d", window.Clicks)
	}
	if count.Get() != 1 {
		t.Fatalf("count = %d", count.Get())
	}
	window.FlushForTest()
	window.dispatchMouse(20, 180)
	if count.Get() != 1 {
		t.Fatalf("bottom-left click should not hit top button, count = %d", count.Get())
	}
	window.FlushForTest()
	if !frameContainsText(window.Commands, "1") {
		t.Fatalf("commands did not repaint count: %+v", window.Commands)
	}
	if got, want := len(window.pixels), scaledPixelLen(window); got != want {
		t.Fatalf("native pixel buffer = %d", len(window.pixels))
	}
	if window.status != "software" {
		t.Fatalf("status = %q, want software", window.status)
	}
}

func TestNativeWindowDispatchMouseSchedulesAsyncRepaint(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	window, count := openCounterWindow(t)
	defer window.CloseForTest()

	window.dispatchMouse(20, 50)
	if count.Get() != 1 {
		t.Fatalf("event should update state immediately, count = %d", count.Get())
	}
	if !window.renderPending {
		t.Fatal("expected pending async render")
	}
	if frameContainsText(window.Commands, "1") {
		t.Fatalf("commands repainted synchronously: %+v", window.Commands)
	}
	window.FlushForTest()
	if !frameContainsText(window.Commands, "1") {
		t.Fatalf("commands did not repaint after drain: %+v", window.Commands)
	}
}

func TestNativeWindowCoalescesPendingRepaints(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	window, count := openCounterWindow(t)
	defer window.CloseForTest()

	window.dispatchMouse(20, 50)
	window.dispatchMouse(20, 50)
	if count.Get() != 2 {
		t.Fatalf("count = %d", count.Get())
	}
	window.renderMu.Lock()
	pending := window.renderPending
	window.renderMu.Unlock()
	if !pending {
		t.Fatal("missing coalesced pending render")
	}
	if frameContainsText(window.Commands, "2") {
		t.Fatalf("commands repainted synchronously: %+v", window.Commands)
	}
	window.FlushForTest()
	if !frameContainsText(window.Commands, "2") {
		t.Fatalf("commands did not render latest state: %+v", window.Commands)
	}
}

func TestNativeWindowDefaultRendererModeIsVelloNative(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "")
	if got := rendererModeFromEnv(); got != "vello-native" {
		t.Fatalf("default renderer mode = %q, want vello-native", got)
	}
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	if got := rendererModeFromEnv(); got != "software" {
		t.Fatalf("explicit renderer mode = %q, want software", got)
	}
}

func TestNativeWindowTitlebarModeFromEnv(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	t.Setenv("VUGRA_NATIVE_TITLEBAR", "")
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new default titlebar window: %v", err)
	}
	if window.titlebarMode != "default" {
		t.Fatalf("default titlebar mode = %q", window.titlebarMode)
	}

	t.Setenv("VUGRA_NATIVE_TITLEBAR", "hidden")
	window, err = New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new hidden titlebar window: %v", err)
	}
	if window.titlebarMode != "hidden" {
		t.Fatalf("hidden titlebar mode = %q", window.titlebarMode)
	}
	if titlebarHiddenFlag(window.titlebarMode) != 1 {
		t.Fatal("hidden titlebar flag not enabled")
	}
	if got := window.SystemTokens()["vugra-window-controls-left"]; got != 72 {
		t.Fatalf("window controls token = %g", got)
	}
	if got := window.SystemTokens()["vugra-window-controls-width"]; got != 72 {
		t.Fatalf("window controls width token = %g", got)
	}
	if got := window.SystemTokens()["vugra-window-controls-height"]; got != 28 {
		t.Fatalf("window controls height token = %g", got)
	}

	t.Setenv("VUGRA_NATIVE_WINDOW_CONTROLS_X", "18")
	t.Setenv("VUGRA_NATIVE_WINDOW_CONTROLS_Y", "11")
	window, err = New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new positioned hidden titlebar window: %v", err)
	}
	if chrome := window.WindowChrome(); !chrome.Controls.Positioned || chrome.Controls.Frame.X != 18 || chrome.Controls.Frame.Y != 11 {
		t.Fatalf("positioned window chrome = %+v", chrome)
	}
	if got := window.SystemTokens()["vugra-window-controls-left"]; got != 90 {
		t.Fatalf("positioned window controls left token = %g", got)
	}

	t.Setenv("VUGRA_NATIVE_TITLEBAR", "unknown")
	window, err = New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new fallback titlebar window: %v", err)
	}
	if window.titlebarMode != "default" {
		t.Fatalf("invalid titlebar mode fallback = %q", window.titlebarMode)
	}
	if tokens := window.SystemTokens(); len(tokens) != 0 {
		t.Fatalf("default titlebar should not expose window control tokens: %+v", tokens)
	}
}

func TestNativeWindowScaleFactorScalesRenderSurfaceOnly(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	t.Setenv("VUGRA_NATIVE_SCALE_FACTOR", "2")
	window, err := New("Vugra", 160, 100)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	if window.ScaleFactor != 2 {
		t.Fatalf("scale factor = %g", window.ScaleFactor)
	}
	window.Render([]renderer.Command{
		{
			Kind: "text",
			Text: "Hi",
			Lines: []renderer.LineBox{{
				Text:     "Hi",
				X:        4,
				Y:        6,
				Width:    20,
				Height:   10,
				Baseline: 8,
			}},
			Glyphs: []renderer.GlyphRun{{
				Text:     "Hi",
				Size:     10,
				X:        4,
				Y:        6,
				Advance:  20,
				Baseline: 8,
			}},
			Rect: renderer.Rect{X: 4, Y: 6, Width: 20, Height: 10},
		},
	})
	if len(window.pixels) != 320*200*4 {
		t.Fatalf("scaled pixel buffer = %d", len(window.pixels))
	}
	if window.Commands[0].Rect.X != 4 || window.Commands[0].Lines[0].Baseline != 8 {
		t.Fatalf("logical commands were mutated: %+v", window.Commands[0])
	}
}

func TestNativeWindowDefaultScaleFactorUsesBackingScale(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	t.Setenv("VUGRA_NATIVE_SCALE_FACTOR", "")
	window, err := New("Vugra", 160, 100)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	if window.ScaleFactor < 1 {
		t.Fatalf("default scale factor = %g", window.ScaleFactor)
	}
}

func TestNativeWindowResizeUpdatesLayoutAndSurface(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	templateDoc := template.Parse(`<div class="fill"><p>A</p></div>`, 0)
	component := ir.Build(ir.BuildInput{Name: "Resize", Template: templateDoc, Go: goanalysis.Metadata{}})
	sheet := style.Parse(`.fill { width: 100%; }`, style.BasePosition{})
	scheduler := reactivity.NewScheduler()
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{},
		Methods: map[string]func(){},
	}, window, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 320, Height: 200},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
	})
	window.app = app
	window.resize(640, 360)
	if window.Width != 640 || window.Height != 360 {
		t.Fatalf("window size = %dx%d", window.Width, window.Height)
	}
	if !window.renderPending {
		t.Fatal("expected resize to schedule render")
	}
	window.FlushForTest()
	root := firstCommandByClass(window.Commands, "fill")
	if root.Rect.Width != 640 {
		t.Fatalf("resized root rect = %+v", root.Rect)
	}
	if got, want := len(window.pixels), scaledPixelLen(window); got != want {
		t.Fatalf("resized pixel buffer = %d", len(window.pixels))
	}
}

func TestNativeWindowDefaultRendererUsesVelloNativeWhenAvailable(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "")
	if !velloNativeDylibAvailable() {
		t.Skip("build vello-native first: cargo build --manifest-path tools/vello-native/Cargo.toml")
	}
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	defer window.CloseForTest()
	window.Render([]renderer.Command{
		{
			Kind: "element",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 16, Y: 16, Width: 120, Height: 40},
		},
		{
			Kind: "text",
			Text: "Default Vello",
			Rect: renderer.Rect{X: 24, Y: 24, Width: 120, Height: 20},
		},
	})
	if window.rendererMode != "vello-native" {
		t.Fatalf("renderer mode = %q, want vello-native", window.rendererMode)
	}
	if window.velloNative == nil {
		t.Fatal("expected vello-native renderer")
	}
	if !strings.Contains(window.status, `"backend":"vello-native"`) {
		t.Fatalf("expected vello-native status, got %q", window.status)
	}
}

func TestNativeWindowVelloNativeRendererInitializesLazily(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "vello-native")
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	defer window.CloseForTest()
	if window.velloNative != nil {
		t.Fatal("vello-native renderer should not load during window construction")
	}
	window.Render([]renderer.Command{
		{
			Kind: "element",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 16, Y: 16, Width: 120, Height: 40},
		},
	})
	if velloNativeDylibAvailable() {
		if window.velloNative == nil {
			t.Fatal("expected vello-native renderer after first render")
		}
	} else if window.velloNative != nil {
		t.Fatal("vello-native renderer loaded without library")
	}
}

func TestNativeWindowCanUseVelloRenderer(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "vello")
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	window.Render([]renderer.Command{
		{
			Kind: "element",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 16, Y: 16, Width: 120, Height: 40},
		},
		{
			Kind: "text",
			Text: "Vello",
			Rect: renderer.Rect{X: 24, Y: 24, Width: 40, Height: 20},
		},
	})
	if got, want := len(window.pixels), scaledPixelLen(window); got != want {
		t.Fatalf("native pixel buffer = %d", len(window.pixels))
	}
	if window.velloRenderer.Status == "" {
		t.Fatal("expected Vello renderer status")
	}
	if strings.Contains(window.velloRenderer.Status, "fallback:") {
		if !strings.Contains(window.velloRenderer.Status, "run Vello sidecar") {
			t.Fatalf("unexpected Vello fallback status: %q", window.velloRenderer.Status)
		}
		return
	}
	if !strings.Contains(window.velloRenderer.Status, `"render":"texture"`) {
		t.Fatalf("expected Vello texture render status or sidecar fallback, got %q", window.velloRenderer.Status)
	}
}

func TestNativeWindowCanUseVelloNativeRenderer(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "vello-native")
	if !velloNativeDylibAvailable() {
		t.Skip("build vello-native first: cargo build --manifest-path tools/vello-native/Cargo.toml")
	}
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	defer window.CloseForTest()
	window.Render([]renderer.Command{
		{
			Kind: "element",
			Tag:  "button",
			Role: "button",
			Rect: renderer.Rect{X: 16, Y: 16, Width: 120, Height: 40},
		},
		{
			Kind: "text",
			Text: "中文 Vello",
			Rect: renderer.Rect{X: 24, Y: 24, Width: 96, Height: 24},
		},
	})
	if got, want := len(window.pixels), scaledPixelLen(window); got != want {
		t.Fatalf("native pixel buffer = %d", len(window.pixels))
	}
	if window.velloNative == nil {
		t.Fatal("expected vello-native renderer")
	}
	if !strings.Contains(window.status, `"backend":"vello-native"`) {
		t.Fatalf("expected vello-native status, got %q", window.status)
	}
	if !strings.Contains(window.status, `"glyphOps":`) {
		t.Fatalf("expected shaped glyph status, got %q", window.status)
	}
}

func scaledPixelLen(window *Window) int {
	return int(float32(window.Width)*window.ScaleFactor) * int(float32(window.Height)*window.ScaleFactor) * 4
}

func BenchmarkNativeWindowVelloDispatchMouseRepaint(b *testing.B) {
	benchmarkNativeWindowDispatchMouseRepaint(b, "vello")
}

func BenchmarkNativeWindowVelloNativeDispatchMouseRepaint(b *testing.B) {
	if !velloNativeDylibAvailable() {
		b.Skip("build vello-native first: cargo build --manifest-path tools/vello-native/Cargo.toml")
	}
	benchmarkNativeWindowDispatchMouseRepaint(b, "vello-native")
}

func velloNativeDylibAvailable() bool {
	for _, candidate := range []string{
		filepath.Join("..", "..", "tools", "vello-native", "target", "debug", "libvello_native.dylib"),
		filepath.Join("..", "..", "tools", "vello-native", "target", "debug", "deps", "libvello_native.dylib"),
		filepath.Join("..", "..", "tools", "vello-native", "target", "release", "libvello_native.dylib"),
		filepath.Join("..", "..", "tools", "vello-native", "target", "release", "deps", "libvello_native.dylib"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

func BenchmarkNativeWindowSoftwareDispatchMouseRepaint(b *testing.B) {
	benchmarkNativeWindowDispatchMouseRepaint(b, "software")
}

func benchmarkNativeWindowDispatchMouseRepaint(b *testing.B, rendererMode string) {
	b.Helper()
	b.Setenv("VUGRA_NATIVE_RENDERER", rendererMode)
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
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
}
`, style.BasePosition{})

	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	window, err := New("Vugra", 320, 200)
	if err != nil {
		b.Fatalf("new window: %v", err)
	}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Update(func(value int) int { return value + 1 }) },
		},
	}, window, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 320, Height: 200},
		Measurer:    layout.FixedMeasurer{},
	})
	window.app = app

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		window.dispatchMouse(20, 50)
		window.FlushForTest()
	}
}

func openCounterWindow(t *testing.T) (*Window, *runtime.SignalValue[int]) {
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
	sheet := style.Parse(`
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
}
`, style.BasePosition{})

	scheduler := reactivity.NewScheduler()
	count := runtime.NewSignal(0, scheduler)
	window, err := New("Vugra", 320, 200)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	app := runtime.MountWithOptions(component, runtime.State{
		Signals: map[string]runtime.Signal{"count": count},
		Methods: map[string]func(){
			"Inc": func() { count.Update(func(value int) int { return value + 1 }) },
		},
	}, window, runtime.Options{
		Scheduler:   scheduler,
		Styles:      sheet,
		Constraints: layout.Constraints{Width: 320, Height: 200},
		Measurer:    layout.FixedMeasurer{},
	})
	window.app = app
	return window, count
}

func frameContainsText(frame []renderer.Command, text string) bool {
	for _, command := range frame {
		if command.Kind == "text" && command.Text == text {
			return true
		}
	}
	return false
}

func firstCommandByClass(frame []renderer.Command, className string) renderer.Command {
	for _, command := range frame {
		if command.Kind == "element" && command.Props["class"] == className {
			return command
		}
	}
	return renderer.Command{}
}
