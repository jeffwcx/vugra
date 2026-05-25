package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/project"
)

func TestIsFinderLiteExample(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"examples/finder/FinderLite.vue", true},
		{"/repo/examples/finder/FinderLite.vue", true},
		{"examples/finder/FinderLite.vugra", true},
		{"/repo/examples/finder/FinderLite.vugra", true},
		{"examples/finder/FinderToolbar.vue", false},
		{"FinderLite.vue", false},
	}
	for _, tc := range cases {
		if got := isFinderLiteExample(tc.path); got != tc.want {
			t.Fatalf("isFinderLiteExample(%q) = %t, want %t", tc.path, got, tc.want)
		}
	}
}

func TestUsageIncludesProjectRun(t *testing.T) {
	if err := usage(); err == nil || !strings.Contains(err.Error(), "run [config-or-dir]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "go-finder-lite [smoke|native|run|native-window-smoke]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "finder-parity-smoke") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "rust-sfc-smoke [file]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "rust-finder-sfc [native|native-software|native-vello|native-wgpu|native-window-smoke|native-software-window-smoke|native-vello-window-smoke|native-wgpu-window-smoke]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "gui-runtime-smoke [window]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "wasm <file> <out-dir>") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "wasm-run <file-or-project> [addr]") {
		t.Fatalf("usage = %v", err)
	}
	if err := usage(); err == nil || !strings.Contains(err.Error(), "wasm-serve <bundle-dir> [addr]") {
		t.Fatalf("usage = %v", err)
	}
}

func TestRunGoFinderLiteSmoke(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runGoFinderLite([]string{"smoke"}); err != nil {
			t.Fatalf("run go finder lite smoke: %v", err)
		}
	})
	for _, want := range []string{
		"go software commands=",
		"go runtime path=Documents row=Roadmap.md",
		"go-finder-lite smoke ok",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunGoFinderLiteRejectsUnknownVariant(t *testing.T) {
	if err := runGoFinderLite([]string{"bogus"}); err == nil || !strings.Contains(err.Error(), "go-finder-lite variant must be smoke, native, run, or native-window-smoke") {
		t.Fatalf("expected go finder variant error, got %v", err)
	}
}

func TestRunFinderParitySmoke(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runFinderParitySmoke(nil); err != nil {
			t.Fatalf("run finder parity smoke: %v", err)
		}
	})
	for _, want := range []string{
		"finder-parity go commands=",
		"finder-parity rust commands=",
		"finder-parity-smoke ok",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunGoFinderLiteNativeUsesFinderComponent(t *testing.T) {
	oldNative := goFinderNativeWindow
	t.Cleanup(func() { goFinderNativeWindow = oldNative })
	var got []string
	goFinderNativeWindow = func(args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	if err := runGoFinderLite([]string{"native"}); err != nil {
		t.Fatalf("run go finder native: %v", err)
	}
	if len(got) != 1 || !strings.HasSuffix(filepath.ToSlash(got[0]), "examples/finder/FinderLite.vue") {
		t.Fatalf("native args = %#v", got)
	}
}

func TestRunGoFinderLiteRunUsesFinderProject(t *testing.T) {
	oldRun := goFinderProjectRun
	t.Cleanup(func() { goFinderProjectRun = oldRun })
	var got []string
	goFinderProjectRun = func(args []string) error {
		got = append([]string(nil), args...)
		return nil
	}
	if err := runGoFinderLite([]string{"run"}); err != nil {
		t.Fatalf("run go finder project: %v", err)
	}
	if len(got) != 1 || !strings.HasSuffix(filepath.ToSlash(got[0]), "examples/finder") {
		t.Fatalf("run args = %#v", got)
	}
}

func TestRunGoFinderLiteNativeWindowSmokeUsesHook(t *testing.T) {
	oldSmoke := goFinderNativeWindowSmoke
	t.Cleanup(func() { goFinderNativeWindowSmoke = oldSmoke })
	called := false
	goFinderNativeWindowSmoke = func() error {
		called = true
		return nil
	}
	if err := runGoFinderLite([]string{"native-window-smoke"}); err != nil {
		t.Fatalf("run go finder native window smoke: %v", err)
	}
	if !called {
		t.Fatal("native window smoke hook was not called")
	}
}

func TestRunRustSFCSmoke(t *testing.T) {
	out := captureStdout(t, func() {
		if err := runRustSFCSmoke(nil); err != nil {
			t.Fatalf("run rust SFC smoke: %v", err)
		}
	})
	for _, want := range []string{
		"rust-sfc path=",
		"fields=1",
		"methods=1",
		"adapter_bytes=",
		"rust-sfc-codegen signals=1 methods=1",
		"rust-sfc-smoke ok",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunRustFinderSFCRejectsUnknownVariant(t *testing.T) {
	if err := runRustFinderSFC([]string{"go"}); err == nil || !strings.Contains(err.Error(), "rust-finder-sfc variant must be native, native-software, native-vello, native-wgpu, native-window-smoke, native-software-window-smoke, native-vello-window-smoke, or native-wgpu-window-smoke") {
		t.Fatalf("expected rust finder SFC variant error, got %v", err)
	}
}

func TestRunGUIRuntimeSmoke(t *testing.T) {
	old := rustFinderCommand
	t.Cleanup(func() { rustFinderCommand = old })
	rustFinderCommand = func(variant string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=TestRustFinderLiteHelperProcess", "--", variant)
	}
	t.Setenv("VUGRA_RUST_FINDER_HELPER", "1")

	out := captureStdout(t, func() {
		if err := runGUIRuntimeSmoke(nil); err != nil {
			t.Fatalf("run gui runtime smoke: %v", err)
		}
	})
	for _, want := range []string{
		"go-finder-lite smoke ok",
		"rust-sfc-smoke ok",
		"helper variant=native-smoke",
		"helper variant=generated-adapter-smoke",
		"helper variant=wgpu-device-smoke",
		"gui-runtime-smoke ok",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunGUIRuntimeSmokeWindow(t *testing.T) {
	old := rustFinderCommand
	t.Cleanup(func() { rustFinderCommand = old })
	oldGoWindowSmoke := goFinderNativeWindowSmoke
	t.Cleanup(func() { goFinderNativeWindowSmoke = oldGoWindowSmoke })
	rustFinderCommand = func(variant string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=TestRustFinderLiteHelperProcess", "--", variant)
	}
	goFinderNativeWindowSmoke = func() error {
		fmt.Println("go-finder-lite native-window-smoke ok")
		return nil
	}
	t.Setenv("VUGRA_RUST_FINDER_HELPER", "1")

	out := captureStdout(t, func() {
		if err := runGUIRuntimeSmoke([]string{"window"}); err != nil {
			t.Fatalf("run gui runtime smoke window: %v", err)
		}
	})
	for _, want := range []string{
		"go-finder-lite smoke ok",
		"go-finder-lite native-window-smoke ok",
		"rust-sfc-smoke ok",
		"helper variant=native-smoke",
		"helper variant=generated-adapter-smoke",
		"helper variant=wgpu-device-smoke",
		"rust-finder-sfc native-window-smoke ok",
		"rust-finder-sfc native-software-window-smoke ok",
		"rust-finder-sfc native-wgpu-window-smoke ok",
		"helper variant=native-window-smoke",
		"helper variant=native-software-window-smoke",
		"helper variant=native-wgpu-window-smoke",
		"helper variant=abi-window-smoke",
		"helper variant=abi-software-window-smoke",
		"helper variant=abi-wgpu-window-smoke",
		"gui-runtime-smoke ok",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestNativeAutoRelaunchAvailability(t *testing.T) {
	original := os.Getenv(nativeAutoRelaunchEnv)
	t.Cleanup(func() {
		if original == "" {
			os.Unsetenv(nativeAutoRelaunchEnv)
		} else {
			os.Setenv(nativeAutoRelaunchEnv, original)
		}
	})
	os.Unsetenv(nativeAutoRelaunchEnv)
	if !canAutoRelaunchNative() {
		t.Fatal("expected native auto relaunch on darwin+cgo test host")
	}
	os.Setenv(nativeAutoRelaunchEnv, "1")
	if canAutoRelaunchNative() {
		t.Fatal("auto relaunch should be disabled after recursive relaunch")
	}
}

func TestApplyProjectRuntimeEnv(t *testing.T) {
	t.Setenv("VUGRA_NATIVE_RENDERER", "software")
	os.Unsetenv("VUGRA_LAYOUT_ENGINE")
	restore := applyProjectRuntimeEnv(project.Config{
		Runtime: project.RuntimeConfig{
			Renderer: "vello-native",
			Layout:   "css",
		},
	})
	if got := os.Getenv("VUGRA_NATIVE_RENDERER"); got != "vello-native" {
		t.Fatalf("renderer env = %q", got)
	}
	if got := os.Getenv("VUGRA_LAYOUT_ENGINE"); got != "css" {
		t.Fatalf("layout env = %q", got)
	}
	restore()
	if got := os.Getenv("VUGRA_NATIVE_RENDERER"); got != "software" {
		t.Fatalf("restored renderer env = %q", got)
	}
	if _, ok := os.LookupEnv("VUGRA_LAYOUT_ENGINE"); ok {
		t.Fatal("layout env should be unset after restore")
	}
}

func TestProjectConfigLoadsFinderExample(t *testing.T) {
	cfg, err := project.Load(filepath.Join("..", "..", "examples", "finder"))
	if err != nil {
		t.Fatalf("load finder project config: %v", err)
	}
	if cfg.Name != "finder-lite" {
		t.Fatalf("project name = %q", cfg.Name)
	}
	if filepath.Base(cfg.EntryPath()) != "FinderLite.vue" {
		t.Fatalf("entry path = %q", cfg.EntryPath())
	}
	if len(cfg.LaunchActions()) != 1 {
		t.Fatalf("launch actions = %+v", cfg.LaunchActions())
	}
}

func TestRunWasmBuildsBrowserBundle(t *testing.T) {
	outDir := t.TempDir()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	t.Setenv("VUGRA_MODULE_ROOT", repoRoot)
	if err := runWasm([]string{filepath.Join("..", "..", "examples", "counter", "Counter.vue"), outDir}); err != nil {
		t.Fatalf("run wasm: %v", err)
	}
	for _, name := range []string{"index.html", "app.wasm", "wasm_exec.js"} {
		info, err := os.Stat(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	if !strings.Contains(string(index), `fetch("app.wasm")`) || !strings.Contains(string(index), `wasm_exec.js`) {
		t.Fatalf("host page does not reference wasm assets:\n%s", index)
	}
}

func TestRunWasmBuildsProjectBrowserBundle(t *testing.T) {
	outDir := t.TempDir()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	t.Setenv("VUGRA_MODULE_ROOT", repoRoot)
	if err := runWasm([]string{filepath.Join("..", "..", "examples", "finder"), outDir}); err != nil {
		t.Fatalf("run wasm project: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	page := string(index)
	for _, want := range []string{
		"<title>Finder Lite</title>",
		`<canvas id="vugra-canvas" tabindex="0"></canvas>`,
		`canvas { display: block; width: 100vw; height: 100vh;`,
		`fetch("app.wasm")`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("host page missing %q:\n%s", want, page)
		}
	}
	if info, err := os.Stat(filepath.Join(outDir, "app.wasm")); err != nil || info.Size() == 0 {
		t.Fatalf("app.wasm missing or empty: info=%v err=%v", info, err)
	}
}

func TestWasmBundleHandlerServesWasmMIME(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"index.html":   "<!doctype html>",
		"app.wasm":     "wasm",
		"wasm_exec.js": "js",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler, err := wasmBundleHandler(dir)
	if err != nil {
		t.Fatalf("wasm bundle handler: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app.wasm", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/wasm" {
		t.Fatalf("content type = %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status = %d body=%q", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("index content type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("index body = %q", rec.Body.String())
	}
}

func TestWasmBundleHandlerRejectsMissingAssets(t *testing.T) {
	dir := t.TempDir()
	if _, err := wasmBundleHandler(dir); err == nil || !strings.Contains(err.Error(), "missing wasm bundle asset index.html") {
		t.Fatalf("expected missing asset error, got %v", err)
	}
}

func TestRunWasmRunBuildsTempBundleAndServes(t *testing.T) {
	oldBuildBundle := wasmBuildBundle
	oldServeHTTP := wasmServeHTTP
	t.Cleanup(func() {
		wasmBuildBundle = oldBuildBundle
		wasmServeHTTP = oldServeHTTP
		signal.Reset(syscall.SIGTERM)
	})

	var builtInput string
	var builtDir string
	wasmBuildBundle = func(args []string) error {
		if len(args) != 2 {
			t.Fatalf("build args = %v", args)
		}
		builtInput = args[0]
		builtDir = args[1]
		for name, content := range map[string]string{
			"index.html":   "<!doctype html>",
			"app.wasm":     "wasm",
			"wasm_exec.js": "js",
		} {
			if err := os.WriteFile(filepath.Join(builtDir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}

	var servedAddr string
	var servedHandler http.Handler
	wasmServeHTTP = func(server *http.Server) error {
		servedAddr = server.Addr
		servedHandler = server.Handler
		req := httptest.NewRequest(http.MethodGet, "/app.wasm", nil)
		rec := httptest.NewRecorder()
		server.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%q", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != "application/wasm" {
			t.Fatalf("content type = %q", got)
		}
		return nil
	}

	if err := runWasmRun([]string{"examples/counter/Counter.vue", "127.0.0.1:9001"}); err != nil {
		t.Fatalf("run wasm-run: %v", err)
	}
	if builtInput != "examples/counter/Counter.vue" {
		t.Fatalf("built input = %q", builtInput)
	}
	if builtDir == "" {
		t.Fatal("build output dir was empty")
	}
	if _, err := os.Stat(builtDir); !os.IsNotExist(err) {
		t.Fatalf("temp bundle dir should be removed after server exits: %v", err)
	}
	if servedAddr != "127.0.0.1:9001" {
		t.Fatalf("served addr = %q", servedAddr)
	}
	if servedHandler == nil {
		t.Fatal("server was not started")
	}
}

func TestRunWasmRunUsesDefaultAddr(t *testing.T) {
	oldBuildBundle := wasmBuildBundle
	oldServeHTTP := wasmServeHTTP
	t.Cleanup(func() {
		wasmBuildBundle = oldBuildBundle
		wasmServeHTTP = oldServeHTTP
	})

	wasmBuildBundle = func(args []string) error {
		for name, content := range map[string]string{
			"index.html":   "<!doctype html>",
			"app.wasm":     "wasm",
			"wasm_exec.js": "js",
		} {
			if err := os.WriteFile(filepath.Join(args[1], name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}
	var servedAddr string
	wasmServeHTTP = func(server *http.Server) error {
		servedAddr = server.Addr
		return nil
	}

	if err := runWasmRun([]string{"examples/finder"}); err != nil {
		t.Fatalf("run wasm-run: %v", err)
	}
	if servedAddr != "127.0.0.1:8000" {
		t.Fatalf("served addr = %q", servedAddr)
	}
}

func TestRunWasmRunRemovesTempBundleOnInterrupt(t *testing.T) {
	oldBuildBundle := wasmBuildBundle
	oldServeHTTP := wasmServeHTTP
	t.Cleanup(func() {
		wasmBuildBundle = oldBuildBundle
		wasmServeHTTP = oldServeHTTP
	})

	var builtDir string
	wasmBuildBundle = func(args []string) error {
		builtDir = args[1]
		for name, content := range map[string]string{
			"index.html":   "<!doctype html>",
			"app.wasm":     "wasm",
			"wasm_exec.js": "js",
		} {
			if err := os.WriteFile(filepath.Join(builtDir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return nil
	}

	serverStarted := make(chan struct{})
	wasmServeHTTP = func(server *http.Server) error {
		shutdown := make(chan struct{})
		server.RegisterOnShutdown(func() {
			close(shutdown)
		})
		close(serverStarted)
		<-shutdown
		return http.ErrServerClosed
	}

	done := make(chan error, 1)
	go func() {
		done <- runWasmRun([]string{"examples/counter/Counter.vue"})
	}()
	select {
	case <-serverStarted:
	case err := <-done:
		t.Fatalf("runWasmRun returned before serving: %v", err)
	case <-time.After(time.Second):
		t.Fatal("server did not start")
	}
	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := self.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run wasm-run after interrupt: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runWasmRun did not stop after interrupt")
	}
	if _, err := os.Stat(builtDir); !os.IsNotExist(err) {
		t.Fatalf("temp bundle dir should be removed after interrupt: %v", err)
	}
}

func TestInlineSVGAssetsEmbedsReferencedSVG(t *testing.T) {
	dir := t.TempDir()
	componentPath := filepath.Join(dir, "Icon.vue")
	if err := os.WriteFile(filepath.Join(dir, "icon.svg"), []byte(`<svg viewBox="0 0 16 16"><rect width="16" height="16" fill="#2563eb"/></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}
	component := &ir.Component{Nodes: []ir.Node{
		&ir.Element{Tag: "img", Props: []ir.Prop{{Name: "src", Value: "./icon.svg"}}},
	}}
	inlineSVGAssets(component, componentPath)
	elem := component.Nodes[0].(*ir.Element)
	if !strings.Contains(staticProp(elem.Props, "__raw_svg"), `fill="#2563eb"`) {
		t.Fatalf("missing inlined svg prop: %+v", elem.Props)
	}
}

func TestResolveWasmInputReadsProjectLayout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "App.vue"), []byte(`<template><div></div></template>
<script lang="go">
type State struct{}
</script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vugra.config.json"), []byte(`{
  "name": "css-app",
  "entry": "App.vue",
  "app": { "title": "CSS App", "width": 640, "height": 360 },
  "runtime": { "layout": "css" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	input, err := resolveWasmInput(dir)
	if err != nil {
		t.Fatalf("resolve wasm input: %v", err)
	}
	if input.EntryPath != filepath.Join(dir, "App.vue") {
		t.Fatalf("entry path = %q", input.EntryPath)
	}
	if input.Title != "CSS App" || input.Width != 640 || input.Height != 360 || input.Layout != "css" {
		t.Fatalf("wasm input = %+v", input)
	}
}
