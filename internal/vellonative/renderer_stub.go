//go:build !darwin || !cgo

package vellonative

import (
	"fmt"

	"github.com/vugra/vugra/internal/renderer"
)

type Renderer struct{}

func New(width, height int) (*Renderer, error) {
	return nil, fmt.Errorf("vello-native requires darwin+cgo")
}

func (r *Renderer) Render(commands []renderer.Command) error {
	return fmt.Errorf("vello-native requires darwin+cgo")
}

func (r *Renderer) Resize(width, height int) {}

func (r *Renderer) Pixels() []byte {
	return nil
}

func (r *Renderer) Status() string {
	return "vello-native unavailable"
}

func (r *Renderer) Close() {}

func LibraryPath() (string, error) {
	return "", fmt.Errorf("vello-native requires darwin+cgo")
}

type Measurer struct{}

func NewMeasurer() (*Measurer, error) {
	return nil, fmt.Errorf("vello-native requires darwin+cgo")
}

func NewLazyMeasurer() *Measurer {
	return &Measurer{}
}

func (m *Measurer) Close() {}

func (m *Measurer) MeasureText(text string) (float32, float32) {
	return 0, 0
}

func (m *Measurer) MeasureStyledText(text string, fontSize, lineHeight float32) (float32, float32) {
	return 0, 0
}

func (m *Measurer) Loaded() bool {
	return false
}

func (m *Measurer) LoadError() error {
	return fmt.Errorf("vello-native requires darwin+cgo")
}
