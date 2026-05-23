package host

import (
	"github.com/vugra/vugra/internal/accessibility"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
)

type PointerEvent struct {
	X int
	Y int
}

type KeyEvent struct {
	Key string
}

type TextEvent struct {
	Text string
}

type ScrollEvent struct {
	X      int
	Y      int
	DeltaY int
}

type FrameSink interface {
	Present([]renderer.Command)
}

type Host interface {
	FrameSink
	Run(*runtime.App) error
}

type MemoryHost struct {
	Frames [][]renderer.Command
	A11y   [][]accessibility.Node
	Clicks []PointerEvent
	Keys   []KeyEvent
	Text   []TextEvent
	Scroll []ScrollEvent
}

func (h *MemoryHost) Present(commands []renderer.Command) {
	frame := make([]renderer.Command, len(commands))
	copy(frame, commands)
	h.Frames = append(h.Frames, frame)
	h.A11y = append(h.A11y, accessibility.Build(frame))
}

func (h *MemoryHost) Run(app *runtime.App) error {
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
	return nil
}

func (h *MemoryHost) Render(commands []renderer.Command) {
	h.Present(commands)
}

func (h *MemoryHost) LastFrame() []renderer.Command {
	if len(h.Frames) == 0 {
		return nil
	}
	return h.Frames[len(h.Frames)-1]
}

func (h *MemoryHost) LastAccessibilityTree() []accessibility.Node {
	if len(h.A11y) == 0 {
		return nil
	}
	return h.A11y[len(h.A11y)-1]
}
