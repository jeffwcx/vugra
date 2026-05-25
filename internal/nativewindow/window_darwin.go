//go:build darwin && cgo && vuego_native_window

package nativewindow

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

extern void vuegoDispatchMouse(uintptr_t handle, int x, int y);
extern void vuegoDispatchMouseEvent(uintptr_t handle, const char* event, int x, int y, int deltaX, int deltaY, int shift, int ctrl, int meta, int alt);
extern void vuegoDispatchScroll(uintptr_t handle, int x, int y, int deltaY);
extern void vuegoDispatchKey(uintptr_t handle, const char* key);
extern void vuegoDispatchText(uintptr_t handle, const char* text);
extern void vuegoFlushRender(uintptr_t handle);
extern void vuegoResizeWindow(uintptr_t handle, int width, int height);

@interface VugraView : NSView {
	uintptr_t handle;
	unsigned char* pixels;
	int pixelWidth;
	int pixelHeight;
}
- (id)initWithFrame:(NSRect)frame handle:(uintptr_t)appHandle;
- (void)updatePixels:(const unsigned char*)data width:(int)width height:(int)height;
@end

static NSWindow *vuegoWindow;

static double vuego_default_scale_factor(void) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		NSScreen *screen = [NSScreen mainScreen];
		if (screen != nil && [screen backingScaleFactor] > 0) {
			return [screen backingScaleFactor];
		}
		return 1.0;
	}
}

@implementation VugraView
- (id)initWithFrame:(NSRect)frame handle:(uintptr_t)appHandle {
	self = [super initWithFrame:frame];
	if (self) {
		handle = appHandle;
		[self setWantsLayer:YES];
	}
	return self;
}
- (BOOL)isFlipped {
	return YES;
}
- (BOOL)acceptsFirstResponder {
	return YES;
}
- (void)updatePixels:(const unsigned char*)data width:(int)width height:(int)height {
	size_t bytes = width * height * 4;
	unsigned char* nextPixels = (unsigned char*)malloc(bytes);
	memcpy(nextPixels, data, bytes);
	if (pixels != NULL) {
		free(pixels);
	}
	pixels = nextPixels;
	pixelWidth = width;
	pixelHeight = height;
	[self setNeedsDisplay:YES];
}
- (void)dealloc {
	if (pixels != NULL) {
		free(pixels);
	}
	[super dealloc];
}
- (void)drawRect:(NSRect)dirtyRect {
	[[NSColor colorWithCalibratedRed:0.98 green:0.98 blue:0.98 alpha:1.0] setFill];
	NSRectFill(dirtyRect);
	if (pixels == NULL || pixelWidth <= 0 || pixelHeight <= 0) {
		return;
	}
	CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
	CGDataProviderRef provider = CGDataProviderCreateWithData(NULL, pixels, pixelWidth * pixelHeight * 4, NULL);
	CGImageRef image = CGImageCreate(pixelWidth, pixelHeight, 8, 32, pixelWidth * 4, colorSpace, kCGImageAlphaLast | kCGBitmapByteOrder32Big, provider, NULL, false, kCGRenderingIntentDefault);
	NSGraphicsContext *context = [NSGraphicsContext currentContext];
	CGContextRef cg = [context CGContext];
	NSRect bounds = [self bounds];
	CGContextSaveGState(cg);
	CGContextTranslateCTM(cg, 0, bounds.size.height);
	CGContextScaleCTM(cg, 1, -1);
	CGContextDrawImage(cg, CGRectMake(0, 0, bounds.size.width, bounds.size.height), image);
	CGContextRestoreGState(cg);
	CGImageRelease(image);
	CGDataProviderRelease(provider);
	CGColorSpaceRelease(colorSpace);
}
- (void)mouseDown:(NSEvent*)event {
	[[self window] makeFirstResponder:self];
	NSPoint point = [self convertPoint:[event locationInWindow] fromView:nil];
	int x = (int)point.x;
	int y = (int)point.y;
	NSEventModifierFlags flags = [event modifierFlags];
	if ([event clickCount] >= 2) {
		vuegoDispatchMouseEvent(handle, "dblclick", x, y, 0, 0,
			(flags & NSEventModifierFlagShift) != 0,
			(flags & NSEventModifierFlagControl) != 0,
			(flags & NSEventModifierFlagCommand) != 0,
			(flags & NSEventModifierFlagOption) != 0);
		return;
	}
	vuegoDispatchMouseEvent(handle, "click", x, y, 0, 0,
		(flags & NSEventModifierFlagShift) != 0,
		(flags & NSEventModifierFlagControl) != 0,
		(flags & NSEventModifierFlagCommand) != 0,
		(flags & NSEventModifierFlagOption) != 0);
}
- (void)rightMouseDown:(NSEvent*)event {
	[[self window] makeFirstResponder:self];
	NSPoint point = [self convertPoint:[event locationInWindow] fromView:nil];
	NSEventModifierFlags flags = [event modifierFlags];
	vuegoDispatchMouseEvent(handle, "contextmenu", (int)point.x, (int)point.y, 0, 0,
		(flags & NSEventModifierFlagShift) != 0,
		(flags & NSEventModifierFlagControl) != 0,
		(flags & NSEventModifierFlagCommand) != 0,
		(flags & NSEventModifierFlagOption) != 0);
}
- (void)mouseMoved:(NSEvent*)event {
	NSPoint point = [self convertPoint:[event locationInWindow] fromView:nil];
	NSEventModifierFlags flags = [event modifierFlags];
	vuegoDispatchMouseEvent(handle, "hover", (int)point.x, (int)point.y, 0, 0,
		(flags & NSEventModifierFlagShift) != 0,
		(flags & NSEventModifierFlagControl) != 0,
		(flags & NSEventModifierFlagCommand) != 0,
		(flags & NSEventModifierFlagOption) != 0);
}
- (void)mouseDragged:(NSEvent*)event {
	NSPoint point = [self convertPoint:[event locationInWindow] fromView:nil];
	NSEventModifierFlags flags = [event modifierFlags];
	vuegoDispatchMouseEvent(handle, "drag", (int)point.x, (int)point.y, (int)[event deltaX], (int)[event deltaY],
		(flags & NSEventModifierFlagShift) != 0,
		(flags & NSEventModifierFlagControl) != 0,
		(flags & NSEventModifierFlagCommand) != 0,
		(flags & NSEventModifierFlagOption) != 0);
}
- (void)scrollWheel:(NSEvent*)event {
	NSPoint point = [self convertPoint:[event locationInWindow] fromView:nil];
	int x = (int)point.x;
	int y = (int)point.y;
	int deltaY = (int)round([event scrollingDeltaY] * -1.0);
	if (deltaY == 0 && [event scrollingDeltaY] != 0) {
		deltaY = [event scrollingDeltaY] < 0 ? 1 : -1;
	}
	vuegoDispatchScroll(handle, x, y, deltaY);
}
- (void)setFrameSize:(NSSize)newSize {
	[super setFrameSize:newSize];
	vuegoResizeWindow(handle, (int)newSize.width, (int)newSize.height);
}
- (void)keyDown:(NSEvent*)event {
	NSString *key = [event charactersIgnoringModifiers];
	if ([key length] == 1) {
		unichar ch = [key characterAtIndex:0];
		NSEventModifierFlags flags = [event modifierFlags];
		if ((flags & NSEventModifierFlagCommand) && (ch == 'a' || ch == 'A')) {
			vuegoDispatchKey(handle, "Mod+A");
			return;
		}
		if ((flags & NSEventModifierFlagControl) && (ch == 'a' || ch == 'A')) {
			vuegoDispatchKey(handle, "Mod+A");
			return;
		}
		if (ch == 9) {
			if ((flags & NSEventModifierFlagShift) != 0) {
				vuegoDispatchKey(handle, "Shift+Tab");
				return;
			}
			vuegoDispatchKey(handle, "Tab");
			return;
		}
		if (ch == 13 || ch == 3) {
			vuegoDispatchKey(handle, "Enter");
			return;
		}
		if (ch == 32) {
			vuegoDispatchKey(handle, " ");
			return;
		}
		if (ch == 127) {
			vuegoDispatchKey(handle, "Backspace");
			return;
		}
	}
	if ([event keyCode] == 126) {
		vuegoDispatchKey(handle, "ArrowUp");
		return;
	}
	if ([event keyCode] == 125) {
		vuegoDispatchKey(handle, "ArrowDown");
		return;
	}
	if ([event keyCode] == 123) {
		vuegoDispatchKey(handle, "ArrowLeft");
		return;
	}
	if ([event keyCode] == 124) {
		vuegoDispatchKey(handle, "ArrowRight");
		return;
	}
	if ([event keyCode] == 115) {
		vuegoDispatchKey(handle, "Home");
		return;
	}
	if ([event keyCode] == 119) {
		vuegoDispatchKey(handle, "End");
		return;
	}
	if ([event keyCode] == 117) {
		vuegoDispatchKey(handle, "Delete");
		return;
	}
	if ([event keyCode] == 53) {
		vuegoDispatchKey(handle, "Escape");
		return;
	}
	NSString *characters = [event characters];
	if ([characters length] > 0) {
		vuegoDispatchText(handle, [characters UTF8String]);
	}
}
@end

static void vuego_position_window_controls(NSWindow *window, int x, int y) {
	if (window == nil || x < 0 || y < 0) {
		return;
	}
	NSButton *closeButton = [window standardWindowButton:NSWindowCloseButton];
	NSButton *miniaturizeButton = [window standardWindowButton:NSWindowMiniaturizeButton];
	NSButton *zoomButton = [window standardWindowButton:NSWindowZoomButton];
	if (closeButton == nil || miniaturizeButton == nil || zoomButton == nil) {
		return;
	}
	NSView *container = [closeButton superview];
	if (container == nil) {
		return;
	}
	CGFloat convertedY = [container bounds].size.height - y - [closeButton frame].size.height;
	NSPoint closeOrigin = NSMakePoint(x, convertedY);
	CGFloat gap = [miniaturizeButton frame].origin.x - [closeButton frame].origin.x;
	if (gap <= 0) {
		gap = 20;
	}
	[closeButton setFrameOrigin:closeOrigin];
	[miniaturizeButton setFrameOrigin:NSMakePoint(closeOrigin.x + gap, closeOrigin.y)];
	[zoomButton setFrameOrigin:NSMakePoint(closeOrigin.x + gap * 2, closeOrigin.y)];
}

static void* vuego_create_window(const char* title, int width, int height, uintptr_t handle, int titlebar_hidden) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		NSRect rect = NSMakeRect(0, 0, width, height);
		NSWindowStyleMask styleMask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable;
		if (titlebar_hidden) {
			styleMask |= NSWindowStyleMaskFullSizeContentView;
		}
		NSWindow *window = [[NSWindow alloc]
			initWithContentRect:rect
			styleMask:styleMask
			backing:NSBackingStoreBuffered
			defer:NO];
		if (titlebar_hidden) {
			[window setTitleVisibility:NSWindowTitleHidden];
			[window setTitlebarAppearsTransparent:YES];
		}
		vuegoWindow = window;
		VugraView *view = [[VugraView alloc] initWithFrame:rect handle:handle];
		NSTrackingArea *tracking = [[NSTrackingArea alloc] initWithRect:rect
			options:(NSTrackingMouseMoved | NSTrackingActiveInKeyWindow | NSTrackingInVisibleRect)
			owner:view
			userInfo:nil];
		[view addTrackingArea:tracking];
		[window setTitle:[NSString stringWithUTF8String:title]];
		[window setContentView:view];
		[window makeFirstResponder:view];
		[window center];
		[window makeKeyAndOrderFront:nil];
		[window layoutIfNeeded];
		[NSApp activateIgnoringOtherApps:YES];
		return (void*)view;
	}
}

static void vuego_update_window_controls(void* view, int x, int y) {
	@autoreleasepool {
		NSWindow *window = [(VugraView*)view window];
		[window layoutIfNeeded];
		vuego_position_window_controls(window, x, y);
	}
}

static void vuego_update_window_controls_deferred(void* view, int x, int y) {
	dispatch_async(dispatch_get_main_queue(), ^{
		vuego_update_window_controls(view, x, y);
	});
}

static void vuego_update_window(void* view, const unsigned char* pixels, int width, int height) {
	@autoreleasepool {
		[(VugraView*)view updatePixels:pixels width:width height:height];
	}
}

static void vuego_run_window(void) {
	@autoreleasepool {
		[NSApp run];
	}
}

static void vuego_schedule_flush_render(uintptr_t handle) {
	dispatch_async(dispatch_get_main_queue(), ^{
		vuegoFlushRender(handle);
	});
}

static void vuego_test_mouse_down(void* view, int x, int y) {
	@autoreleasepool {
		CGFloat eventY = [(VugraView*)view bounds].size.height - y;
		NSEvent *event = [NSEvent mouseEventWithType:NSEventTypeLeftMouseDown
			location:NSMakePoint(x, eventY)
			modifierFlags:0
			timestamp:0
			windowNumber:[(VugraView*)view window].windowNumber
			context:nil
			eventNumber:1
			clickCount:1
			pressure:1.0];
		[(VugraView*)view mouseDown:event];
	}
}
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	goruntime "runtime"
	"runtime/cgo"
	"strconv"
	"sync"
	"unsafe"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/vello"
	"github.com/vugra/vugra/internal/vellonative"
	"github.com/vugra/vugra/pkg/system"
)

const hiddenTitlebarWindowControlsWidth = 72
const hiddenTitlebarWindowControlsHeight = 28
const defaultWindowControlsX = 0
const defaultWindowControlsY = 0

type Window struct {
	Title       string
	Width       int
	Height      int
	ScaleFactor float32
	Commands    []renderer.Command
	Clicks      int
	Keys        int
	Text        int

	app           *runtime.App
	velloRenderer *vello.NativeRenderer
	velloNative   *vellonative.Renderer
	software      *renderer.SoftwareRenderer
	rendererMode  string
	titlebarMode  string
	chrome        system.WindowChrome
	pixels        []byte
	status        string
	view          unsafe.Pointer
	handle        cgo.Handle
	deferRender   bool
	renderMu      sync.Mutex
	renderPending bool
}

func New(title string, width, height int) (*Window, error) {
	if title == "" {
		title = "Vugra"
	}
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	scaleFactor := scaleFactorFromEnv()
	pixelWidth, pixelHeight := scaledSize(width, height, scaleFactor)
	mode := rendererModeFromEnv()
	titlebarMode := titlebarModeFromEnv()
	chrome := windowChromeFromEnv(titlebarMode)
	return &Window{
		Title:         title,
		Width:         width,
		Height:        height,
		ScaleFactor:   scaleFactor,
		velloRenderer: vello.NewNativeRenderer(pixelWidth, pixelHeight),
		software:      renderer.NewSoftware(pixelWidth, pixelHeight),
		rendererMode:  mode,
		titlebarMode:  titlebarMode,
		chrome:        chrome,
	}, nil
}

func scaleFactorFromEnv() float32 {
	value := os.Getenv("VUGRA_NATIVE_SCALE_FACTOR")
	if value == "" {
		scale := float32(C.vuego_default_scale_factor())
		if scale <= 0 {
			return 1
		}
		return scale
	}
	scale, err := strconv.ParseFloat(value, 32)
	if err != nil || scale <= 0 {
		return 1
	}
	return float32(scale)
}

func scaledSize(width, height int, scale float32) (int, int) {
	if scale <= 0 {
		scale = 1
	}
	return maxInt(1, int(math.Ceil(float64(float32(width)*scale)))),
		maxInt(1, int(math.Ceil(float64(float32(height)*scale))))
}

func rendererModeFromEnv() string {
	mode := firstEnv("VUGRA_NATIVE_RENDERER", "VUEGO_NATIVE_RENDERER")
	if mode == "" {
		return "vello-native"
	}
	return mode
}

func titlebarModeFromEnv() string {
	mode := firstEnv("VUGRA_NATIVE_TITLEBAR", "VUEGO_NATIVE_TITLEBAR")
	switch mode {
	case "hidden", "default":
		return mode
	case "":
		return "default"
	default:
		return "default"
	}
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
	}
	return ""
}

func windowChromeFromEnv(titlebarMode string) system.WindowChrome {
	chrome := system.WindowChrome{
		Titlebar: system.WindowTitlebarMode(titlebarMode),
	}
	if titlebarMode != "hidden" {
		return chrome
	}
	chrome.Controls = system.WindowControls{
		Visible: true,
		Frame: system.Rect{
			X:      envInt("VUGRA_NATIVE_WINDOW_CONTROLS_X", defaultWindowControlsX),
			Y:      envInt("VUGRA_NATIVE_WINDOW_CONTROLS_Y", defaultWindowControlsY),
			Width:  hiddenTitlebarWindowControlsWidth,
			Height: hiddenTitlebarWindowControlsHeight,
		},
	}
	if os.Getenv("VUGRA_NATIVE_WINDOW_CONTROLS_X") != "" || os.Getenv("VUGRA_NATIVE_WINDOW_CONTROLS_Y") != "" {
		chrome.Controls.Positioned = true
	}
	return chrome
}

func envInt(name string, fallback int) float32 {
	value := os.Getenv(name)
	if value == "" {
		return float32(fallback)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return float32(fallback)
	}
	return float32(parsed)
}

func titlebarHiddenFlag(mode string) C.int {
	if mode == "hidden" {
		return 1
	}
	return 0
}

func nativeRenderLogEnabled() bool {
	return os.Getenv("VUGRA_NATIVE_RENDER_LOG") == "1"
}

func (w *Window) Render(commands []renderer.Command) {
	w.Commands = append([]renderer.Command(nil), commands...)
	if w.deferRender && w.view == nil {
		return
	}
	renderCommands := scaleCommands(commands, w.ScaleFactor)
	if w.rendererMode == "vello-native" {
		if err := w.ensureVelloNative(); err != nil {
			w.status = "vello-native unavailable: " + err.Error()
			w.renderSoftware(renderCommands)
		} else if err := w.velloNative.Render(renderCommands); err != nil {
			w.status = "vello-native fallback: " + err.Error()
			w.renderSoftware(renderCommands)
		} else {
			w.pixels = w.velloNative.Pixels()
			w.status = w.velloNative.Status()
		}
	} else if w.rendererMode == "vello" {
		w.velloRenderer.Render(renderCommands)
		w.pixels = w.velloRenderer.Pixels
		w.status = w.velloRenderer.Status
	} else {
		w.renderSoftware(renderCommands)
	}
	if w.view == nil || len(w.pixels) == 0 {
		return
	}
	if nativeRenderLogEnabled() && (w.rendererMode == "vello" || w.rendererMode == "vello-native") && w.status != "" {
		fmt.Fprintf(os.Stderr, "vugra native %s %s", w.rendererMode, w.status)
		if w.status[len(w.status)-1] != '\n' {
			fmt.Fprintln(os.Stderr)
		}
	}
	pixelWidth, pixelHeight := scaledSize(w.Width, w.Height, w.ScaleFactor)
	C.vuego_update_window(w.view, (*C.uchar)(unsafe.Pointer(&w.pixels[0])), C.int(pixelWidth), C.int(pixelHeight))
}

func (w *Window) Pixels() []byte {
	return append([]byte(nil), w.pixels...)
}

func (w *Window) ensureVelloNative() error {
	if w.velloNative != nil {
		return nil
	}
	pixelWidth, pixelHeight := scaledSize(w.Width, w.Height, w.ScaleFactor)
	nativeRenderer, err := vellonative.New(pixelWidth, pixelHeight)
	if err != nil {
		return err
	}
	w.velloNative = nativeRenderer
	return nil
}

func (w *Window) DeferRenderUntilOpen() {
	w.deferRender = true
}

func (w *Window) SystemTokens() style.SystemTokens {
	return style.SystemTokens(w.chrome.SystemTokens())
}

func (w *Window) WindowChrome() system.WindowChrome {
	return w.chrome
}

func (w *Window) SetWindowChrome(chrome system.WindowChrome) error {
	w.chrome = chrome
	w.titlebarMode = string(chrome.Titlebar)
	w.applyWindowChrome()
	if w.app != nil {
		w.app.SetSystemTokens(style.SystemTokens(w.chrome.SystemTokens()))
		w.scheduleRender()
	}
	return nil
}

func (w *Window) applyWindowChrome() {
	if w.view == nil {
		return
	}
	if w.chrome.Titlebar != system.WindowTitlebarHidden || !w.chrome.Controls.Positioned {
		return
	}
	C.vuego_update_window_controls(w.view, C.int(w.chrome.Controls.Frame.X), C.int(w.chrome.Controls.Frame.Y))
	C.vuego_update_window_controls_deferred(w.view, C.int(w.chrome.Controls.Frame.X), C.int(w.chrome.Controls.Frame.Y))
}

func (w *Window) renderSoftware(commands []renderer.Command) {
	w.software.Render(commands)
	if w.software.Image != nil {
		w.pixels = w.software.Image.Pix
	}
	if w.status == "" || w.rendererMode != "vello-native" {
		w.status = "software"
	}
}

func scaleCommands(commands []renderer.Command, scale float32) []renderer.Command {
	if scale <= 0 {
		scale = 1
	}
	if scale == 1 {
		out := make([]renderer.Command, len(commands))
		copy(out, commands)
		return out
	}
	out := make([]renderer.Command, len(commands))
	for i, command := range commands {
		out[i] = command
		out[i].Rect = scaleRect(command.Rect, scale)
		out[i].Lines = scaleLines(command.Lines, scale)
		out[i].Glyphs = scaleGlyphs(command.Glyphs, scale)
		out[i].Style.FontSize *= scale
		out[i].Style.LineHeight *= scale
		out[i].Style.BorderWidth *= scale
		out[i].Style.BorderRadius *= scale
	}
	return out
}

func scaleRect(rect renderer.Rect, scale float32) renderer.Rect {
	rect.X *= scale
	rect.Y *= scale
	rect.Width *= scale
	rect.Height *= scale
	return rect
}

func scaleLines(lines []renderer.LineBox, scale float32) []renderer.LineBox {
	out := make([]renderer.LineBox, len(lines))
	for i, line := range lines {
		out[i] = line
		out[i].X *= scale
		out[i].Y *= scale
		out[i].Width *= scale
		out[i].Height *= scale
		out[i].Baseline *= scale
	}
	return out
}

func scaleGlyphs(glyphs []renderer.GlyphRun, scale float32) []renderer.GlyphRun {
	out := make([]renderer.GlyphRun, len(glyphs))
	for i, glyph := range glyphs {
		out[i] = glyph
		out[i].Size *= scale
		out[i].X *= scale
		out[i].Y *= scale
		out[i].Advance *= scale
		out[i].Baseline *= scale
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (w *Window) Run(app *runtime.App) error {
	goruntime.LockOSThread()
	w.app = app
	w.handle = cgo.NewHandle(w)
	defer w.handle.Delete()
	title := C.CString(w.Title)
	defer C.free(unsafe.Pointer(title))
	w.view = C.vuego_create_window(title, C.int(w.Width), C.int(w.Height), C.uintptr_t(w.handle), titlebarHiddenFlag(w.titlebarMode))
	w.applyWindowChrome()
	if len(w.Commands) > 0 {
		w.Render(w.Commands)
	} else if w.app != nil {
		w.app.Flush()
	}
	C.vuego_run_window()
	return nil
}

func (w *Window) OpenForTest(app *runtime.App) error {
	goruntime.LockOSThread()
	w.app = app
	w.handle = cgo.NewHandle(w)
	title := C.CString(w.Title)
	defer C.free(unsafe.Pointer(title))
	w.view = C.vuego_create_window(title, C.int(w.Width), C.int(w.Height), C.uintptr_t(w.handle), titlebarHiddenFlag(w.titlebarMode))
	if w.view == nil {
		w.handle.Delete()
		return fmt.Errorf("create native test window")
	}
	w.applyWindowChrome()
	if len(w.Commands) > 0 {
		w.Render(w.Commands)
	} else if w.app != nil {
		w.app.Flush()
	}
	return nil
}

func (w *Window) CloseForTest() {
	if w.handle != 0 {
		w.handle.Delete()
		w.handle = 0
	}
	if w.velloNative != nil {
		w.velloNative.Close()
		w.velloNative = nil
	}
}

func (w *Window) MouseDownForTest(x, y int) {
	if w.view == nil {
		return
	}
	C.vuego_test_mouse_down(w.view, C.int(x), C.int(y))
}

func (w *Window) dispatchMouse(x, y int) {
	w.dispatchMouseEvent("click", x, y, 0, 0, runtime.Modifiers{})
}

func (w *Window) dispatchMouseEvent(event string, x, y, deltaX, deltaY int, modifiers runtime.Modifiers) {
	w.Clicks++
	if w.app == nil {
		if nativeRenderLogEnabled() {
			fmt.Fprintf(os.Stderr, "vugra native dispatchMouse no app event=%s x=%d y=%d\n", event, x, y)
		}
		return
	}
	id, hit := w.app.HitTest(float32(x), float32(y))
	if nativeRenderLogEnabled() {
		fmt.Fprintf(os.Stderr, "vugra native dispatchMouse event=%s x=%d y=%d hit=%t id=%s\n", event, x, y, hit, id)
	}
	handled := false
	switch event {
	case "click":
		handled = w.app.DispatchPointerEvent(float32(x), float32(y), modifiers)
	case "dblclick":
		handled = w.app.DispatchDoubleClick(float32(x), float32(y), modifiers)
	case "contextmenu":
		handled = w.app.DispatchContextMenu(float32(x), float32(y), modifiers)
	case "hover":
		handled = w.app.DispatchHover(float32(x), float32(y), modifiers)
	case "drag":
		handled = w.app.DispatchDrag(float32(x), float32(y), float32(deltaX), float32(deltaY), modifiers)
	}
	if handled {
		w.scheduleRender()
	}
}

func (w *Window) dispatchScroll(x, y, deltaY int) {
	if w.app != nil && w.app.DispatchScroll(float32(x), float32(y), float32(deltaY)) {
		w.scheduleRender()
	}
}

func (w *Window) resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if w.Width == width && w.Height == height {
		return
	}
	w.Width = width
	w.Height = height
	pixelWidth, pixelHeight := scaledSize(width, height, w.ScaleFactor)
	w.velloRenderer.Resize(pixelWidth, pixelHeight)
	w.software = renderer.NewSoftware(pixelWidth, pixelHeight)
	if w.velloNative != nil {
		w.velloNative.Resize(pixelWidth, pixelHeight)
	}
	if w.app != nil {
		w.app.Resize(float32(width), float32(height))
		w.scheduleRender()
	}
}

func (w *Window) dispatchKey(key string) {
	w.Keys++
	if w.app != nil && w.app.DispatchKey(key) {
		w.scheduleRender()
	}
}

func (w *Window) dispatchText(text string) {
	w.Text++
	if w.app != nil && w.app.DispatchTextInput(text) {
		w.scheduleRender()
	}
}

func (w *Window) scheduleRender() {
	w.renderMu.Lock()
	if w.renderPending {
		w.renderMu.Unlock()
		return
	}
	w.renderPending = true
	handle := w.handle
	w.renderMu.Unlock()
	if handle == 0 {
		return
	}
	C.vuego_schedule_flush_render(C.uintptr_t(handle))
}

func (w *Window) flushRender() {
	w.renderMu.Lock()
	w.renderPending = false
	w.renderMu.Unlock()
	if w.app != nil {
		w.app.Flush()
	}
}

func (w *Window) FlushForTest() {
	w.flushRender()
}
