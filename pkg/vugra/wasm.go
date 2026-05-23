//go:build js && wasm

package vugra

import (
	"syscall/js"

	"github.com/vugra/vugra/internal/wasmhost"
)

func NewCanvasRenderer(canvas js.Value, width, height int) Renderer {
	return wasmhost.NewCanvasRenderer(canvas, width, height)
}

func ResizeCanvasRenderer(target Renderer, width, height int) {
	if canvas, ok := target.(*wasmhost.CanvasRenderer); ok {
		canvas.Resize(width, height)
	}
}

func NewCanvasMeasurer(canvas js.Value) Measurer {
	return wasmhost.NewCanvasMeasurer(canvas)
}

func InstallPointerEvents(canvas js.Value, dispatch func(x, y int)) js.Func {
	return wasmhost.InstallPointerEvents(canvas, dispatch)
}

func InstallPointerEventDetails(canvas js.Value, dispatch func(event string, x, y, deltaX, deltaY int, shift, ctrl, meta, alt bool)) js.Func {
	return wasmhost.InstallPointerEventDetails(canvas, dispatch)
}

func InstallScrollEvents(canvas js.Value, dispatch func(x, y, deltaY int)) js.Func {
	return wasmhost.InstallScrollEvents(canvas, dispatch)
}

func InstallKeyboardEvents(dispatch func(key string)) js.Func {
	return wasmhost.InstallKeyboardEvents(dispatch)
}

func InstallTextEvents(dispatch func(text string)) js.Func {
	return wasmhost.InstallTextEvents(dispatch)
}

func SetStatus(id string, text string) {
	wasmhost.SetStatus(id, text)
}

func SyncAccessibility(containerID string, commands []Command, focusedID string) {
	wasmhost.SyncAccessibility(containerID, commands, focusedID)
}

func InstallAccessibilityEvents(containerID string, dispatch func(x, y int)) js.Func {
	return wasmhost.InstallAccessibilityEvents(containerID, dispatch)
}

type AccessibilityEvents = wasmhost.AccessibilityEvents

func InstallAccessibilityEventHandlers(containerID string, handlers AccessibilityEvents) js.Func {
	return wasmhost.InstallAccessibilityEventHandlers(containerID, handlers)
}
