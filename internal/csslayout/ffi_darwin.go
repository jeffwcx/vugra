//go:build darwin && cgo

package csslayout

/*
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef int (*vuego_css_layout_compute_fn)(const uint8_t*, size_t, uint8_t**, size_t*);
typedef void (*vuego_css_layout_free_fn)(uint8_t*, size_t);

static void* vuego_css_layout_dlopen(const char* path) {
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void* vuego_css_layout_dlsym(void* library, const char* name) {
	return dlsym(library, name);
}

static const char* vuego_css_layout_dlerror(void) {
	return dlerror();
}

static int vuego_css_layout_call_compute(
	vuego_css_layout_compute_fn fn,
	const uint8_t* input,
	size_t input_len,
	uint8_t** output,
	size_t* output_len
) {
	return fn(input, input_len, output, output_len);
}

static void vuego_css_layout_call_free(vuego_css_layout_free_fn fn, uint8_t* ptr, size_t len) {
	fn(ptr, len);
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

type ffiLibrary struct {
	compute C.vuego_css_layout_compute_fn
	free    C.vuego_css_layout_free_fn
}

var ffiCache sync.Map

func (e Engine) computeFFI(payload []byte) (Output, error) {
	path, err := e.libraryPath()
	if err != nil {
		return Output{}, err
	}
	lib, err := loadFFI(path)
	if err != nil {
		return Output{}, err
	}
	var input *C.uint8_t
	if len(payload) > 0 {
		input = (*C.uint8_t)(unsafe.Pointer(&payload[0]))
	}
	var output *C.uint8_t
	var outputLen C.size_t
	status := C.vuego_css_layout_call_compute(lib.compute, input, C.size_t(len(payload)), &output, &outputLen)
	if output == nil {
		return Output{}, fmt.Errorf("csslayout ffi: null output status=%d", int(status))
	}
	data := C.GoBytes(unsafe.Pointer(output), C.int(outputLen))
	C.vuego_css_layout_call_free(lib.free, output, outputLen)
	if status != 0 {
		return Output{}, fmt.Errorf("csslayout ffi: %s", string(data))
	}
	return decodeOutput(data)
}

func loadFFI(path string) (*ffiLibrary, error) {
	if cached, ok := ffiCache.Load(path); ok {
		return cached.(*ffiLibrary), nil
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	handle := C.vuego_css_layout_dlopen(cpath)
	if handle == nil {
		return nil, fmt.Errorf("csslayout ffi: dlopen %s: %s", path, dlerror())
	}
	computeName := C.CString("vuego_css_layout_compute")
	defer C.free(unsafe.Pointer(computeName))
	freeName := C.CString("vuego_css_layout_free")
	defer C.free(unsafe.Pointer(freeName))
	compute := C.vuego_css_layout_compute_fn(C.vuego_css_layout_dlsym(handle, computeName))
	if compute == nil {
		return nil, fmt.Errorf("csslayout ffi: missing vuego_css_layout_compute: %s", dlerror())
	}
	free := C.vuego_css_layout_free_fn(C.vuego_css_layout_dlsym(handle, freeName))
	if free == nil {
		return nil, fmt.Errorf("csslayout ffi: missing vuego_css_layout_free: %s", dlerror())
	}
	lib := &ffiLibrary{compute: compute, free: free}
	actual, _ := ffiCache.LoadOrStore(path, lib)
	return actual.(*ffiLibrary), nil
}

func dlerror() string {
	if err := C.vuego_css_layout_dlerror(); err != nil {
		return C.GoString(err)
	}
	return "unknown error"
}
