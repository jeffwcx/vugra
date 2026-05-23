package nativehost

import (
	"github.com/vugra/vugra/internal/accessibility"
	"github.com/vugra/vugra/internal/host"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
)

type Host struct {
	Renderer *renderer.SoftwareRenderer
	Clicks   []host.PointerEvent
	Keys     []host.KeyEvent
	Text     []host.TextEvent
	Scroll   []host.ScrollEvent
	OutPNG   string
	Frames   int
	A11y     []accessibility.Node
}

func New(width, height int, outPNG string) *Host {
	return &Host{
		Renderer: renderer.NewSoftware(width, height),
		OutPNG:   outPNG,
	}
}

func (h *Host) Render(commands []renderer.Command) {
	h.Present(commands)
}

func (h *Host) Present(commands []renderer.Command) {
	h.Frames++
	h.Renderer.Render(commands)
	h.A11y = accessibility.Build(commands)
}

func (h *Host) Run(app *runtime.App) error {
	for _, click := range h.Clicks {
		if app.DispatchPointer(float32(click.X), float32(click.Y)) {
			app.Flush()
		}
	}
	for _, scroll := range h.Scroll {
		if app.DispatchScroll(float32(scroll.X), float32(scroll.Y), float32(scroll.DeltaY)) {
			app.Flush()
		}
	}
	for _, key := range h.Keys {
		if app.DispatchKey(key.Key) {
			app.Flush()
		}
	}
	for _, text := range h.Text {
		if app.DispatchTextInput(text.Text) {
			app.Flush()
		}
	}
	if h.OutPNG != "" {
		return h.Renderer.SavePNG(h.OutPNG)
	}
	return nil
}
