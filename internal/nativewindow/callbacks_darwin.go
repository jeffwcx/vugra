//go:build darwin && cgo && vuego_native_window

package nativewindow

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"runtime/cgo"

	"github.com/vugra/vugra/internal/runtime"
)

func runtimeModifiers(shift C.int, ctrl C.int, meta C.int, alt C.int) runtime.Modifiers {
	return runtime.Modifiers{
		Shift: shift != 0,
		Ctrl:  ctrl != 0,
		Meta:  meta != 0,
		Alt:   alt != 0,
	}
}

//export vuegoDispatchMouse
func vuegoDispatchMouse(handle C.uintptr_t, x C.int, y C.int) {
	window := cgo.Handle(handle).Value().(*Window)
	window.dispatchMouse(int(x), int(y))
}

//export vuegoDispatchMouseEvent
func vuegoDispatchMouseEvent(handle C.uintptr_t, event *C.char, x C.int, y C.int, deltaX C.int, deltaY C.int, shift C.int, ctrl C.int, meta C.int, alt C.int) {
	window := cgo.Handle(handle).Value().(*Window)
	window.dispatchMouseEvent(C.GoString(event), int(x), int(y), int(deltaX), int(deltaY), runtimeModifiers(shift, ctrl, meta, alt))
}

//export vuegoDispatchScroll
func vuegoDispatchScroll(handle C.uintptr_t, x C.int, y C.int, deltaY C.int) {
	window := cgo.Handle(handle).Value().(*Window)
	window.dispatchScroll(int(x), int(y), int(deltaY))
}

//export vuegoDispatchKey
func vuegoDispatchKey(handle C.uintptr_t, key *C.char) {
	window := cgo.Handle(handle).Value().(*Window)
	window.dispatchKey(C.GoString(key))
}

//export vuegoDispatchText
func vuegoDispatchText(handle C.uintptr_t, text *C.char) {
	window := cgo.Handle(handle).Value().(*Window)
	window.dispatchText(C.GoString(text))
}

//export vuegoFlushRender
func vuegoFlushRender(handle C.uintptr_t) {
	window := cgo.Handle(handle).Value().(*Window)
	window.flushRender()
}

//export vuegoResizeWindow
func vuegoResizeWindow(handle C.uintptr_t, width C.int, height C.int) {
	window := cgo.Handle(handle).Value().(*Window)
	window.resize(int(width), int(height))
}
