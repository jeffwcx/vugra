package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"syscall"

	"github.com/vugra/vugra/internal/accessibility"
	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/codegen"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/project"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/sfc"
	"github.com/vugra/vugra/internal/vello"
	"github.com/vugra/vugra/internal/vellonative"
	"github.com/vugra/vugra/internal/wasmhost"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return usage()
	}

	switch args[0] {
	case "parse":
		return runParse(args[1:])
	case "check":
		return runCheck(args[1:])
	case "ir":
		return runIR(args[1:])
	case "frame":
		return runFrame(args[1:])
	case "vello-ops":
		return runVelloOps(args[1:])
	case "vello-png":
		return runVelloPNG(args[1:])
	case "png":
		return runPNG(args[1:])
	case "gen":
		return runGen(args[1:])
	case "gen-main":
		return runGenMain(args[1:])
	case "wasm-host":
		return runWasmHost(args[1:])
	case "gen-wasm-main":
		return runGenWasmMain(args[1:])
	case "wasm":
		return runWasm(args[1:])
	case "wasm-run":
		return runWasmRun(args[1:])
	case "wasm-serve":
		return runWasmServe(args[1:])
	case "native-png":
		return runNativePNG(args[1:])
	case "native-window":
		return runNativeWindow(args[1:])
	case "run":
		return runProject(args[1:])
	case "native-app":
		return runNativeApp(args[1:])
	case "a11y":
		return runA11y(args[1:])
	default:
		return usage()
	}
}

func usage() error {
	return fmt.Errorf("usage: vugra parse <file> | check <file> | ir <file> | frame <file> | vello-ops <file> | vello-png <file> <out.png> | a11y <file> | png <file> <out.png> | native-png <file> <out.png> | native-window <file> | run [config-or-dir] | native-app <binary> <out.app> [args...] | gen <file> | gen-main <component-import> <out.png> | gen-wasm-main <component-import> <canvas-id> | wasm-host <wasm-path> | wasm <file> <out-dir> | wasm-run <file-or-project> [addr] | wasm-serve <bundle-dir> [addr]")
}

func layoutEngineFromEnv() runtime.LayoutEngine {
	if firstEnv("VUGRA_LAYOUT_ENGINE", "VUEGO_LAYOUT_ENGINE") == "css" {
		return runtime.LayoutEngineCSS
	}
	return runtime.LayoutEngineGo
}

func layoutEngineFromConfig(value string) runtime.LayoutEngine {
	switch value {
	case "css":
		return runtime.LayoutEngineCSS
	default:
		return runtime.LayoutEngineGo
	}
}

func applyProjectRuntimeEnv(cfg project.Config) func() {
	previousRenderer, rendererWasSet := os.LookupEnv("VUGRA_NATIVE_RENDERER")
	previousLayout, layoutWasSet := os.LookupEnv("VUGRA_LAYOUT_ENGINE")
	if cfg.Runtime.Renderer != "" {
		os.Setenv("VUGRA_NATIVE_RENDERER", cfg.Runtime.Renderer)
	}
	if cfg.Runtime.Layout != "" && cfg.Runtime.Layout != "go" {
		os.Setenv("VUGRA_LAYOUT_ENGINE", cfg.Runtime.Layout)
	}
	return func() {
		restoreEnv("VUGRA_NATIVE_RENDERER", previousRenderer, rendererWasSet)
		restoreEnv("VUGRA_LAYOUT_ENGINE", previousLayout, layoutWasSet)
	}
}

func restoreEnv(name, value string, wasSet bool) {
	if wasSet {
		os.Setenv(name, value)
		return
	}
	os.Unsetenv(name)
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
	}
	return ""
}

func isFinderLiteExample(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == "examples/finder/FinderLite.vue" ||
		clean == "examples/finder/FinderLite.vugra" ||
		clean == "examples/finder/FinderLite.vuego" ||
		strings.HasSuffix(clean, "/examples/finder/FinderLite.vue") ||
		strings.HasSuffix(clean, "/examples/finder/FinderLite.vugra") ||
		strings.HasSuffix(clean, "/examples/finder/FinderLite.vuego")
}

func runParse(args []string) error {
	if len(args) != 1 {
		return usage()
	}

	path := args[0]
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	file := sfc.Parse(path, source)
	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode parse output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func runCheck(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	diagnostics := result.Diagnostics()
	if len(diagnostics) == 0 {
		fmt.Println("ok")
		return nil
	}
	encoded, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return fmt.Errorf("encode diagnostics: %w", err)
	}
	fmt.Println(string(encoded))
	return fmt.Errorf("check failed with %d diagnostic(s)", len(diagnostics))
}

func runIR(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	encoded, err := json.MarshalIndent(result.IR, "", "  ")
	if err != nil {
		return fmt.Errorf("encode IR: %w", err)
	}
	fmt.Println(string(encoded))
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		return fmt.Errorf("IR contains %d diagnostic(s)", len(diagnostics))
	}
	return nil
}

func runFrame(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("frame failed with %d diagnostic(s)", len(diagnostics))
	}

	scheduler := reactivity.NewScheduler()
	testRenderer := &renderer.TestRenderer{}
	app.Mount(result, demoStateFor(args[0], scheduler), testRenderer, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		Layout:      layoutEngineFromEnv(),
	})

	encoded, err := json.MarshalIndent(testRenderer.LastFrame(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func runA11y(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	frame, err := renderInitialFrame(args[0], &renderer.TestRenderer{})
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(accessibility.Build(frame), "", "  ")
	if err != nil {
		return fmt.Errorf("encode accessibility tree: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func runVelloOps(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	encoded, err := velloOpsJSON(args[0])
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func runVelloPNG(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	encoded, err := velloOpsJSON(args[0])
	if err != nil {
		return err
	}
	cmd := exec.Command("cargo", "run", "--manifest-path", "tools/vello-sidecar/Cargo.toml", "--", args[1])
	cmd.Stdin = bytes.NewReader(encoded)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("run Vello sidecar: %w", err)
	}
	fmt.Print(string(output))
	return nil
}

func velloOpsJSON(path string) ([]byte, error) {
	frame, err := renderInitialFrame(path, &renderer.TestRenderer{})
	if err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(vello.Translate(frame), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode vello ops: %w", err)
	}
	return encoded, nil
}

func runPNG(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("png failed with %d diagnostic(s)", len(diagnostics))
	}

	scheduler := reactivity.NewScheduler()
	software := renderer.NewSoftware(800, 600)
	app.Mount(result, demoStateFor(args[0], scheduler), software, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		Layout:      layoutEngineFromEnv(),
	})
	if err := software.SavePNG(args[1]); err != nil {
		return fmt.Errorf("write png %s: %w", args[1], err)
	}
	fmt.Println(args[1])
	return nil
}

func renderInitialFrame(path string, target renderer.Renderer) ([]renderer.Command, error) {
	result, err := compiler.CompileFile(path)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", path, err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return nil, fmt.Errorf("render failed with %d diagnostic(s)", len(diagnostics))
	}
	scheduler := reactivity.NewScheduler()
	testRenderer, isTest := target.(*renderer.TestRenderer)
	app.Mount(result, demoStateFor(path, scheduler), target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		Layout:      layoutEngineFromEnv(),
	})
	if isTest {
		return testRenderer.LastFrame(), nil
	}
	return nil, nil
}

func runGen(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("gen failed with %d diagnostic(s)", len(diagnostics))
	}
	generated := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "component",
		Script:      result.SFC.Script.Content,
		Metadata:    result.Go,
		Component:   result.IR,
		Styles:      result.Style,
		Imports:     codegenImports(result.Imports),
	})
	fmt.Print(generated)
	return nil
}

func codegenImports(imports []compiler.ImportResult) []codegen.ImportedComponentInput {
	out := make([]codegen.ImportedComponentInput, 0, len(imports))
	for _, imported := range imports {
		if imported.Result == nil || imported.Result.SFC == nil || imported.Result.SFC.Script == nil {
			continue
		}
		out = append(out, codegen.ImportedComponentInput{
			Alias:     imported.Alias,
			Path:      imported.Path,
			Script:    imported.Result.SFC.Script.Content,
			Metadata:  imported.Result.Go,
			Component: imported.Result.IR,
			Imports:   codegenImports(imported.Result.Imports),
		})
	}
	return out
}

func runGenMain(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	fmt.Print(codegen.GenerateSoftwareMain(codegen.SoftwareMainInput{
		PackageImport: args[0],
		OutputPath:    args[1],
	}))
	return nil
}

func runWasmHost(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	fmt.Print(wasmhost.GenerateHostPage(wasmhost.PageInput{
		WasmPath: args[0],
		Title:    "Vugra",
		CanvasID: "vugra-canvas",
		Width:    800,
		Height:   600,
	}))
	return nil
}

func runGenWasmMain(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	fmt.Print(codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: args[0],
		CanvasID:      args[1],
	}))
	return nil
}

func runWasm(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	input, err := resolveWasmInput(args[0])
	if err != nil {
		return err
	}
	outDir := args[1]
	result, err := compiler.CompileFile(input.EntryPath)
	if err != nil {
		return fmt.Errorf("compile %s: %w", input.EntryPath, err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("wasm failed with %d diagnostic(s)", len(diagnostics))
	}
	if result.SFC == nil || result.SFC.Script == nil {
		return fmt.Errorf("wasm requires a component with <script lang=\"go\">")
	}
	assetBase, err := filepath.Abs(input.EntryPath)
	if err != nil {
		return fmt.Errorf("resolve wasm asset base: %w", err)
	}
	inlineSVGAssets(result.IR, assetBase)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create wasm output dir %s: %w", outDir, err)
	}
	workDir, err := os.MkdirTemp("", "vugra-wasm-*")
	if err != nil {
		return fmt.Errorf("create wasm work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	componentDir := filepath.Join(workDir, "component")
	mainDir := filepath.Join(workDir, "cmd", "app")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		return fmt.Errorf("create component work dir: %w", err)
	}
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		return fmt.Errorf("create wasm main work dir: %w", err)
	}

	const modulePath = "vugra.local/wasmapp"
	componentSource := codegen.GenerateRuntimeState(codegen.RuntimeStateInput{
		PackageName: "component",
		Script:      result.SFC.Script.Content,
		Metadata:    result.Go,
		Component:   result.IR,
		Styles:      result.Style,
		Imports:     codegenImports(result.Imports),
	})
	if err := os.WriteFile(filepath.Join(componentDir, "component.go"), []byte(componentSource), 0o644); err != nil {
		return fmt.Errorf("write generated component: %w", err)
	}
	if err := writeWasmDemoState(componentDir, input.EntryPath); err != nil {
		return err
	}
	stateExpression := ""
	refreshHook := ""
	if strings.TrimSpace(wasmDemoStateSource(input.EntryPath)) != "" {
		stateExpression = "component.NewDemoState()"
		refreshHook = wasmRefreshHook
	}
	mainSource := codegen.GenerateWasmMain(codegen.WasmMainInput{
		PackageImport: modulePath + "/component",
		CanvasID:      "vugra-canvas",
		Width:         input.Width,
		Height:        input.Height,
		Layout:        input.Layout,
		AssetBase:     assetBase,
		State:         stateExpression,
		RefreshHook:   refreshHook,
	})
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(mainSource), 0o644); err != nil {
		return fmt.Errorf("write wasm main: %w", err)
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire github.com/vugra/vugra v0.0.0\n\nreplace github.com/vugra/vugra => %s\n", modulePath, moduleRoot())
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return fmt.Errorf("write wasm go.mod: %w", err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = workDir
	tidy.Stderr = os.Stderr
	if output, err := tidy.Output(); err != nil {
		return fmt.Errorf("go mod tidy for wasm bundle: %w%s", err, commandOutputSuffix(output))
	}
	wasmPath := filepath.Join(outDir, "app.wasm")
	build := exec.Command("go", "build", "-o", wasmPath, "./cmd/app")
	build.Dir = workDir
	build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	build.Stderr = os.Stderr
	if output, err := build.Output(); err != nil {
		return fmt.Errorf("build wasm bundle: %w%s", err, commandOutputSuffix(output))
	}
	if err := copyWasmAsset(wasmExecPath(), filepath.Join(outDir, "wasm_exec.js")); err != nil {
		return err
	}
	hostPage := wasmhost.GenerateHostPage(wasmhost.PageInput{
		WasmPath: "app.wasm",
		Title:    input.Title,
		CanvasID: "vugra-canvas",
		Width:    input.Width,
		Height:   input.Height,
	})
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte(hostPage), 0o644); err != nil {
		return fmt.Errorf("write wasm host page: %w", err)
	}
	fmt.Println(outDir)
	return nil
}

var wasmBuildBundle = runWasm
var wasmServeHTTP = func(server *http.Server) error {
	return server.ListenAndServe()
}

func runWasmRun(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return usage()
	}
	addr := "127.0.0.1:8000"
	if len(args) == 2 {
		addr = args[1]
	}
	bundleDir, err := os.MkdirTemp("", "vugra-wasm-run-*")
	if err != nil {
		return fmt.Errorf("create wasm-run bundle dir: %w", err)
	}
	defer os.RemoveAll(bundleDir)
	if err := wasmBuildBundle([]string{args[0], bundleDir}); err != nil {
		return err
	}
	return serveWasmBundleUntilSignal(bundleDir, addr)
}

func runWasmServe(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return usage()
	}
	addr := "127.0.0.1:8000"
	if len(args) == 2 {
		addr = args[1]
	}
	return serveWasmBundle(args[0], addr)
}

func serveWasmBundle(bundleDir, addr string) error {
	server, err := wasmBundleServer(bundleDir, addr)
	if err != nil {
		return err
	}
	fmt.Printf("serving %s at http://%s\n", bundleDir, addr)
	return wasmServeHTTP(server)
}

func serveWasmBundleUntilSignal(bundleDir, addr string) error {
	server, err := wasmBundleServer(bundleDir, addr)
	if err != nil {
		return err
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	errs := make(chan error, 1)
	go func() {
		fmt.Printf("serving %s at http://%s\n", bundleDir, addr)
		errs <- wasmServeHTTP(server)
	}()
	select {
	case err := <-errs:
		return err
	case sig := <-signals:
		if err := server.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("shutdown wasm-run server after %s: %w", sig, err)
		}
		if err := <-errs; err != nil && err != http.ErrServerClosed {
			return err
		}
		fmt.Printf("stopped wasm-run server after %s\n", sig)
		return nil
	}
}

func wasmBundleServer(bundleDir, addr string) (*http.Server, error) {
	handler, err := wasmBundleHandler(bundleDir)
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}, nil
}

func wasmBundleHandler(root string) (http.Handler, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve wasm bundle dir %s: %w", root, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat wasm bundle dir %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("wasm bundle path is not a directory: %s", root)
	}
	for _, name := range []string{"index.html", "app.wasm", "wasm_exec.js"} {
		if info, err := os.Stat(filepath.Join(abs, name)); err != nil {
			return nil, fmt.Errorf("missing wasm bundle asset %s in %s: %w", name, root, err)
		} else if info.IsDir() || info.Size() == 0 {
			return nil, fmt.Errorf("invalid wasm bundle asset %s in %s", name, root)
		}
	}
	files := http.FileServer(http.Dir(abs))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.html" {
			serveBundleAsset(w, r, filepath.Join(abs, "index.html"), "text/html; charset=utf-8")
			return
		}
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}
		files.ServeHTTP(w, r)
	}), nil
}

func serveBundleAsset(w http.ResponseWriter, r *http.Request, path, contentType string) {
	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func inlineSVGAssets(component *ir.Component, basePath string) {
	if component == nil {
		return
	}
	inlineSVGNodes(component.Nodes, basePath)
	for _, imported := range component.Imports {
		inlineSVGAssets(imported.Component, basePath)
	}
}

func inlineSVGNodes(nodes []ir.Node, basePath string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *ir.Element:
			inlineSVGElement(n, basePath)
			inlineSVGNodes(n.Children, basePath)
		case *ir.Conditional:
			inlineSVGNodes([]ir.Node{n.Child}, basePath)
		case *ir.Repeater:
			inlineSVGNodes([]ir.Node{n.Child}, basePath)
		case *ir.ComponentInstance:
			inlineSVGNodes(n.Nodes, basePath)
		case *ir.DynamicComponent:
			for _, candidate := range n.Cases {
				inlineSVGNodes(candidate.Nodes, basePath)
			}
		}
	}
}

func inlineSVGElement(elem *ir.Element, basePath string) {
	if elem.Tag != "img" || !strings.HasSuffix(strings.ToLower(staticProp(elem.Props, "src")), ".svg") || staticProp(elem.Props, "__raw_svg") != "" {
		return
	}
	src := staticProp(elem.Props, "src")
	if src == "" {
		return
	}
	path := filepath.Clean(src)
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(basePath), path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	elem.Props = append(elem.Props, ir.Prop{Name: "__raw_svg", Value: string(data), Span: elem.Span})
}

func staticProp(props []ir.Prop, name string) string {
	for _, prop := range props {
		if prop.Name == name && !prop.Bound {
			return prop.Value
		}
	}
	return ""
}

type wasmBundleInput struct {
	EntryPath string
	Title     string
	Width     int
	Height    int
	Layout    string
}

func resolveWasmInput(path string) (wasmBundleInput, error) {
	if isComponentInput(path) {
		return wasmBundleInput{
			EntryPath: path,
			Title:     "Vugra",
			Width:     800,
			Height:    600,
			Layout:    "go",
		}, nil
	}
	cfg, err := project.Load(path)
	if err != nil {
		return wasmBundleInput{}, fmt.Errorf("load wasm project input %s: %w", path, err)
	}
	return wasmBundleInput{
		EntryPath: cfg.EntryPath(),
		Title:     cfg.App.Title,
		Width:     cfg.App.Width,
		Height:    cfg.App.Height,
		Layout:    cfg.Runtime.Layout,
	}, nil
}

func isComponentInput(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".vue" || ext == ".vugra" || ext == ".vuego"
}

func moduleRoot() string {
	if root := firstEnv("VUGRA_MODULE_ROOT", "VUEGO_MODULE_ROOT"); root != "" {
		return root
	}
	root, err := filepath.Abs(".")
	if err != nil {
		return "."
	}
	return root
}

func wasmExecPath() string {
	if path := firstEnv("VUGRA_WASM_EXEC", "VUEGO_WASM_EXEC"); path != "" {
		return path
	}
	root := strings.TrimSpace(stdruntime.GOROOT())
	if root == "" {
		return "wasm_exec.js"
	}
	return filepath.Join(root, "lib", "wasm", "wasm_exec.js")
}

func copyWasmAsset(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s to %s: %w", source, dest, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dest, err)
	}
	return nil
}

func commandOutputSuffix(output []byte) string {
	if len(output) == 0 {
		return ""
	}
	return ": " + string(output)
}

func runNativePNG(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	frame, err := renderInitialFrame(args[0], &renderer.TestRenderer{})
	if err != nil {
		return err
	}
	mode := nativePNGRendererModeFromEnv()
	switch mode {
	case "vello-native":
		if err := writeVelloNativePNG(frame, 800, 600, args[1]); err != nil {
			return err
		}
	case "vello":
		velloRenderer := vello.NewNativeRenderer(800, 600)
		velloRenderer.Render(frame)
		if len(velloRenderer.Pixels) != 800*600*4 {
			return fmt.Errorf("render vello png: invalid pixel buffer len=%d status=%s", len(velloRenderer.Pixels), velloRenderer.Status)
		}
		if err := writeRGBAPNG(args[1], 800, 600, velloRenderer.Pixels); err != nil {
			return err
		}
	default:
		software := renderer.NewSoftware(800, 600)
		software.Render(frame)
		if err := software.SavePNG(args[1]); err != nil {
			return fmt.Errorf("write png %s: %w", args[1], err)
		}
	}
	fmt.Println(args[1])
	return nil
}

func nativePNGRendererModeFromEnv() string {
	mode := firstEnv("VUGRA_NATIVE_RENDERER", "VUEGO_NATIVE_RENDERER")
	if mode == "" {
		return "vello-native"
	}
	return mode
}

func writeVelloNativePNG(commands []renderer.Command, width, height int, path string) error {
	target, err := vellonative.New(width, height)
	if err != nil {
		return fmt.Errorf("create vello-native renderer: %w", err)
	}
	defer target.Close()
	if err := target.Render(commands); err != nil {
		return err
	}
	pixels := target.Pixels()
	if len(pixels) != width*height*4 {
		return fmt.Errorf("render vello-native png: invalid pixel buffer len=%d status=%s", len(pixels), target.Status())
	}
	return writeRGBAPNG(path, width, height, pixels)
}

func writeRGBAPNG(path string, width, height int, pixels []byte) error {
	if len(pixels) != width*height*4 {
		return fmt.Errorf("write png %s: invalid rgba buffer len=%d", path, len(pixels))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, pixels)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create png %s: %w", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("write png %s: %w", path, err)
	}
	return nil
}
