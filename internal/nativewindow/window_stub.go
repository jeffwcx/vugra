//go:build !darwin || !cgo || !vuego_native_window

package nativewindow

import (
	"fmt"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
)

type Window struct{}

func New(title string, width, height int) (*Window, error) {
	return nil, fmt.Errorf("native window backend is not enabled; build on darwin with cgo and -tags vuego_native_window")
}

func (w *Window) Render(commands []renderer.Command) {}

func (w *Window) Pixels() []byte { return nil }

func (w *Window) Run(app *runtime.App) error {
	return fmt.Errorf("native window backend is not enabled")
}
