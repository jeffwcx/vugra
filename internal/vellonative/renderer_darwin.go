//go:build darwin && cgo

package vellonative

/*
#cgo CFLAGS: -std=c11
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef void* (*vuego_create_fn)(uint32_t, uint32_t);
typedef void (*vuego_destroy_fn)(void*);
typedef void (*vuego_resize_fn)(void*, uint32_t, uint32_t);
typedef int (*vuego_render_fn)(void*, const uint8_t*, size_t);
typedef const uint8_t* (*vuego_pixels_fn)(const void*);
typedef size_t (*vuego_pixels_len_fn)(const void*);
typedef const char* (*vuego_status_fn)(const void*);
typedef const char* (*vuego_version_fn)(void);
typedef int (*vuego_measure_text_fn)(const char*, float, float, void*);

typedef struct {
	void* library;
	vuego_create_fn create;
	vuego_destroy_fn destroy;
	vuego_resize_fn resize;
	vuego_render_fn render;
	vuego_pixels_fn pixels;
	vuego_pixels_len_fn pixels_len;
	vuego_status_fn status;
	vuego_version_fn version;
	vuego_measure_text_fn measure_text;
	char error[512];
} VugraVelloNativeSymbols;

static int vuego_load_symbol(void* library, void** out, const char* name, char* error, size_t error_len) {
	*out = dlsym(library, name);
	if (*out == NULL) {
		snprintf(error, error_len, "missing symbol %s: %s", name, dlerror());
		return -1;
	}
	return 0;
}

static int vuego_load_vello_native(const char* path, VugraVelloNativeSymbols* symbols) {
	memset(symbols, 0, sizeof(*symbols));
	void* library = dlopen(path, RTLD_NOW | RTLD_LOCAL);
	if (library == NULL) {
		snprintf(symbols->error, sizeof(symbols->error), "%s", dlerror());
		return -1;
	}
	symbols->library = library;
	if (vuego_load_symbol(library, (void**)&symbols->create, "vuego_native_renderer_create", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->destroy, "vuego_native_renderer_destroy", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->resize, "vuego_native_renderer_resize", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->render, "vuego_native_renderer_render", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->pixels, "vuego_native_renderer_pixels", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->pixels_len, "vuego_native_renderer_pixels_len", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->status, "vuego_native_renderer_status", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->version, "vuego_native_renderer_version", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	if (vuego_load_symbol(library, (void**)&symbols->measure_text, "vuego_native_measure_text", symbols->error, sizeof(symbols->error)) != 0) goto fail;
	return 0;
fail:
	dlclose(library);
	symbols->library = NULL;
	return -1;
}

static void vuego_close_vello_native(VugraVelloNativeSymbols* symbols) {
	if (symbols->library != NULL) {
		dlclose(symbols->library);
	}
	memset(symbols, 0, sizeof(*symbols));
}

static void* vuego_bridge_create(VugraVelloNativeSymbols* symbols, uint32_t width, uint32_t height) {
	return symbols->create(width, height);
}

static void vuego_bridge_destroy(VugraVelloNativeSymbols* symbols, void* renderer) {
	symbols->destroy(renderer);
}

static void vuego_bridge_resize(VugraVelloNativeSymbols* symbols, void* renderer, uint32_t width, uint32_t height) {
	symbols->resize(renderer, width, height);
}

static int vuego_bridge_render(VugraVelloNativeSymbols* symbols, void* renderer, const uint8_t* json, size_t len) {
	return symbols->render(renderer, json, len);
}

static const uint8_t* vuego_bridge_pixels(VugraVelloNativeSymbols* symbols, void* renderer) {
	return symbols->pixels(renderer);
}

static size_t vuego_bridge_pixels_len(VugraVelloNativeSymbols* symbols, void* renderer) {
	return symbols->pixels_len(renderer);
}

static const char* vuego_bridge_status(VugraVelloNativeSymbols* symbols, void* renderer) {
	return symbols->status(renderer);
}

static const char* vuego_bridge_version(VugraVelloNativeSymbols* symbols) {
	return symbols->version();
}

typedef struct {
	float width;
	float height;
	float baseline;
} VugraTextMetrics;

static int vuego_bridge_measure_text(VugraVelloNativeSymbols* symbols, const char* text, float font_size, float line_height, VugraTextMetrics* out) {
	return symbols->measure_text(text, font_size, line_height, out);
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/vello"
)

type Renderer struct {
	mu      sync.Mutex
	symbols C.VugraVelloNativeSymbols
	handle  unsafe.Pointer
	width   int
	height  int
	pixels  []byte
	status  string
	closed  bool
}

func New(width, height int) (*Renderer, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	path, err := LibraryPath()
	if err != nil {
		return nil, err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	r := &Renderer{width: width, height: height}
	if C.vuego_load_vello_native(cpath, &r.symbols) != 0 {
		return nil, fmt.Errorf("load vello-native %s: %s", path, C.GoString(&r.symbols.error[0]))
	}
	r.handle = C.vuego_bridge_create(&r.symbols, C.uint32_t(width), C.uint32_t(height))
	if r.handle == nil {
		C.vuego_close_vello_native(&r.symbols)
		return nil, fmt.Errorf("create vello-native renderer: nil handle")
	}
	r.status = C.GoString(C.vuego_bridge_version(&r.symbols))
	return r, nil
}

func (r *Renderer) Render(commands []renderer.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.handle == nil {
		return fmt.Errorf("vello-native renderer is closed")
	}
	encoded, err := vello.EncodeOps(commands)
	if err != nil {
		return fmt.Errorf("encode Vello ops: %w", err)
	}
	var data *C.uint8_t
	if len(encoded) > 0 {
		data = (*C.uint8_t)(unsafe.Pointer(&encoded[0]))
	}
	if C.vuego_bridge_render(&r.symbols, r.handle, data, C.size_t(len(encoded))) != 0 {
		r.status = C.GoString(C.vuego_bridge_status(&r.symbols, r.handle))
		return fmt.Errorf("render vello-native: %s", r.status)
	}
	pixelsPtr := C.vuego_bridge_pixels(&r.symbols, r.handle)
	pixelsLen := int(C.vuego_bridge_pixels_len(&r.symbols, r.handle))
	if pixelsPtr == nil || pixelsLen != r.width*r.height*4 {
		return fmt.Errorf("render vello-native: invalid pixel buffer len=%d", pixelsLen)
	}
	r.pixels = C.GoBytes(unsafe.Pointer(pixelsPtr), C.int(pixelsLen))
	r.status = C.GoString(C.vuego_bridge_status(&r.symbols, r.handle))
	return nil
}

func (r *Renderer) Resize(width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	r.width = width
	r.height = height
	if !r.closed && r.handle != nil {
		C.vuego_bridge_resize(&r.symbols, r.handle, C.uint32_t(width), C.uint32_t(height))
	}
}

func (r *Renderer) Pixels() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.pixels...)
}

func (r *Renderer) Status() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Renderer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	if r.handle != nil {
		C.vuego_bridge_destroy(&r.symbols, r.handle)
		r.handle = nil
	}
	C.vuego_close_vello_native(&r.symbols)
	r.closed = true
}

func LibraryPath() (string, error) {
	if path := os.Getenv("VUGRA_VELLO_NATIVE_LIB"); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("stat VUGRA_VELLO_NATIVE_LIB: %w", err)
		}
		return path, nil
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	for _, candidate := range []string{
		filepath.Join(root, "tools", "vello-native", "target", "debug", "libvello_native.dylib"),
		filepath.Join(root, "tools", "vello-native", "target", "release", "libvello_native.dylib"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("locate tools/vello-native/target/{debug,release}/libvello_native.dylib; run cargo build --manifest-path tools/vello-native/Cargo.toml")
}

type Measurer struct {
	mu      sync.Mutex
	symbols C.VugraVelloNativeSymbols
	loaded  bool
	loadErr error
	cache   map[textMeasureKey][2]float32
}

type textMeasureKey struct {
	text       string
	fontSize   float32
	lineHeight float32
}

func NewMeasurer() (*Measurer, error) {
	m := newMeasurer()
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func NewLazyMeasurer() *Measurer {
	return newMeasurer()
}

func newMeasurer() *Measurer {
	return &Measurer{cache: map[textMeasureKey][2]float32{}}
}

func (m *Measurer) load() error {
	if m.loaded {
		return nil
	}
	if m.loadErr != nil {
		return m.loadErr
	}
	path, err := LibraryPath()
	if err != nil {
		m.loadErr = err
		return err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	if C.vuego_load_vello_native(cpath, &m.symbols) != 0 {
		m.loadErr = fmt.Errorf("load vello-native measurer %s: %s", path, C.GoString(&m.symbols.error[0]))
		return m.loadErr
	}
	m.loaded = true
	return nil
}

func (m *Measurer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loaded {
		C.vuego_close_vello_native(&m.symbols)
		m.loaded = false
	}
}

func (m *Measurer) MeasureText(text string) (float32, float32) {
	return m.MeasureStyledText(text, 16, 20)
}

func (m *Measurer) MeasureStyledText(text string, fontSize, lineHeight float32) (float32, float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.loaded {
		if err := m.load(); err != nil {
			return 0, 0
		}
	}
	if !m.loaded {
		return 0, 0
	}
	key := textMeasureKey{text: text, fontSize: fontSize, lineHeight: lineHeight}
	if cached, ok := m.cache[key]; ok {
		return cached[0], cached[1]
	}
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))
	var metrics C.VugraTextMetrics
	if C.vuego_bridge_measure_text(&m.symbols, ctext, C.float(fontSize), C.float(lineHeight), &metrics) != 0 {
		return 0, 0
	}
	width, height := float32(metrics.width), float32(metrics.height)
	m.cache[key] = [2]float32{width, height}
	return width, height
}

func (m *Measurer) Loaded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loaded
}

func (m *Measurer) LoadError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadErr
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "tools", "vello-native", "Cargo.toml")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("locate tools/vello-native/Cargo.toml")
}
