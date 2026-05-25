# Vugra

Vugra is an early-stage cross-platform UI framework built around `.vue` single-file components, standard HTML template syntax, real Go component logic, Signal-based reactivity, and renderer-neutral compiler IR.

## Current milestone

This repository currently implements a non-LSP GUI framework slice from `IMPLEMENTATION_PLAN.md`: SFC parsing, template parsing, style parsing, Go metadata extraction, component IR validation, public Signal APIs, generated component/runtime glue, Signal-based updates, `v-if`/`v-for`, backend-neutral layout boxes, pointer/keyboard/text event routing, focus handling, text input and checkbox control semantics, semantic roles/accessibility trees, a public `pkg/vugra` runtime facade, a public `pkg/system` boundary with native OS and wasm Web Storage file-system backends, a test renderer, a software renderer, a native software-render host adapter, generated native/wasm entry points, a wasm canvas host, and a Cocoa native window backend that the CLI can auto-enable on macOS with cgo.

The CSS/layout MVP supports class selectors, block layout, flex row/column, flex wrapping, flex grow/basis, fixed/fractional grid columns, grid rows, simple grid column/row placement, gaps, padding, margin, min/max width/height, basic text wrapping, and typography metadata such as font size, line height, and text align. A `internal/vello` adapter defines the Vugra render-command to Vello-scene operation contract. The `tools/vello-sidecar` Rust crate consumes those operations and builds a real Linebender Vello `Scene` for CLI PNG/raw output. The `tools/vello-native` Rust crate exposes a cgo-loaded C ABI that keeps wgpu, Vello, cosmic-text, and font caches alive across frames; it shapes text through cosmic-text before drawing glyphs with Vello. The Cocoa native window defaults to the persistent cgo/Rust Vello renderer, maps common focus and text-editing keys into the same runtime `DispatchKey` path used by wasm, can use the old per-frame sidecar with `VUGRA_NATIVE_RENDERER=vello`, and keeps the software pixel renderer available only as an explicit experimental path with `VUGRA_NATIVE_RENDERER=software`.

Direct native surface presentation, WebGPU wasm rendering, richer widgets, full CSS cascade, and full platform accessibility bridges are still future backend work. LSP support is intentionally excluded from this milestone.

The Rust GUI runtime slice is also executable. The workspace includes a Rust kernel crate set, direct and ABI-shaped Finder Lite state paths, Rust SFC parsing/codegen smoke tests, and a generated Rust Finder Lite SFC path that opens a native window through `vugra-host-native`. This is a narrow vertical slice rather than a finished Rust product surface, but it proves that both Go and Rust GUI runtime paths can mount state, render frames, dispatch events, and present native windows.

## Commands

```sh
go test ./...
sh docs/site/verify_docs_site.sh
go run ./cmd/vugra wasm-run docs/site
go run ./cmd/vugra wasm docs/site /tmp/vugra-docs-site
go run ./cmd/vugra wasm-serve /tmp/vugra-docs-site
go run ./cmd/vugra parse examples/counter/Counter.vue
go run ./cmd/vugra check examples/counter/Counter.vue
go run ./cmd/vugra ir examples/counter/Counter.vue
go run ./cmd/vugra frame examples/counter/Counter.vue
go run ./cmd/vugra vello-ops examples/counter/Counter.vue
go run ./cmd/vugra vello-ops examples/counter/Counter.vue | cargo run --manifest-path tools/vello-sidecar/Cargo.toml
go run ./cmd/vugra vello-png examples/counter/Counter.vue /tmp/vugra-vello-counter.png
go run ./cmd/vugra a11y examples/counter/Counter.vue
go run ./cmd/vugra png examples/counter/Counter.vue /tmp/vugra-counter.png
go run ./cmd/vugra native-png examples/counter/Counter.vue /tmp/vugra-native-counter.png
go run ./cmd/vugra native-window examples/counter/Counter.vue
go run ./cmd/vugra run examples/finder
go run ./cmd/vugra go-finder-lite smoke
go run ./cmd/vugra go-finder-lite native-window-smoke
go run ./cmd/vugra go-finder-lite native
go run ./cmd/vugra go-finder-lite run
go run ./cmd/vugra rust-sfc-smoke
go run ./cmd/vugra rust-finder-sfc native-window-smoke
go run ./cmd/vugra rust-finder-sfc native-software-window-smoke
go run ./cmd/vugra rust-finder-sfc native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-sfc native
go run ./cmd/vugra rust-finder-sfc native-software
go run ./cmd/vugra rust-finder-sfc native-vello
go run ./cmd/vugra rust-finder-sfc native-wgpu
go run ./cmd/vugra rust-finder-lite native-smoke
go run ./cmd/vugra rust-finder-lite native-window-smoke
go run ./cmd/vugra rust-finder-lite native-software-window-smoke
go run ./cmd/vugra rust-finder-lite native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite abi-window-smoke
go run ./cmd/vugra rust-finder-lite abi-software-window-smoke
go run ./cmd/vugra rust-finder-lite abi-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite native
go run ./cmd/vugra rust-finder-lite native-software
go run ./cmd/vugra gui-runtime-smoke
go run ./cmd/vugra gui-runtime-smoke window
go build -tags vuego_native_window -o /tmp/vugra ./cmd/vugra
go run ./cmd/vugra native-app /tmp/vugra /tmp/VugraCounter.app native-window examples/counter/Counter.vue
open /tmp/VugraCounter.app
cargo build --manifest-path tools/vello-native/Cargo.toml
go run ./cmd/vugra native-window examples/counter/Counter.vue
VUGRA_NATIVE_RENDERER=software go run ./cmd/vugra native-window examples/counter/Counter.vue
go run ./cmd/vugra gen examples/counter/Counter.vue
go run ./cmd/vugra gen examples/system/SystemFiles.vue
go run ./cmd/vugra gen-main github.com/example/counter/component counter.png
go run ./cmd/vugra gen-wasm-main github.com/example/counter/component vugra-canvas
go run ./cmd/vugra wasm-host counter.wasm > index.html
go run ./cmd/vugra wasm examples/counter/Counter.vue /tmp/vugra-counter-wasm
go run ./cmd/vugra wasm examples/finder /tmp/vugra-finder-wasm
go run ./cmd/vugra wasm-run examples/counter/Counter.vue
go run ./cmd/vugra wasm-serve /tmp/vugra-counter-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --screenshot /tmp/vugra-counter-browser.png
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --click 30,60 --expect-text 1 --expect-a11y button,+
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --a11y-click button,+ --expect-text 1 --expect-a11y button,+
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --a11y-focus button,+ --expect-text 0 --expect-a11y-focused button,+
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --a11y-focus button,+ --key Enter --expect-text 1 --expect-a11y-focused button,+
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --key Tab --key Enter --expect-text 1 --expect-a11y-focused button,+
node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --key Tab --key ' ' --expect-text 1 --expect-a11y-focused button,+
go run ./cmd/vugra wasm examples/focus/FocusCycle.vue /tmp/vugra-focus-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-focus-wasm --key Tab --key Tab --expect-a11y-focused button,Second
node tools/wasm-browser-check/run.mjs /tmp/vugra-focus-wasm --key Tab --key Tab --key Shift+Tab --expect-a11y-focused button,First
go run ./cmd/vugra wasm examples/finder /tmp/vugra-finder-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-finder-wasm --expect-title 'Finder Lite' --expect-canvas 800,600 --expect-a11y text,Favorites --expect-a11y text,Workspace
go run ./cmd/vugra wasm examples/textinput/TextInput.vue /tmp/vugra-textinput-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --expect-text abc --expect-a11y-focused textbox,abc
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key ArrowLeft --text X --expect-text abXc --expect-a11y textbox,abXc
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key Home --text X --key End --text Y --expect-text XabcY --expect-a11y textbox,XabcY
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text draft --key Mod+A --text final --expect-text final --expect-a11y textbox,final
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --a11y-text textbox,,xy --a11y-key textbox,xy,Backspace --expect-text x --expect-a11y textbox,x
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --compose 你好 --expect-text 你好 --expect-a11y textbox,你好
node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --paste pasted --expect-text pasted --expect-a11y textbox,pasted
go run ./cmd/vugra wasm examples/checkbox/CheckboxBox.vue /tmp/vugra-checkbox-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-checkbox-wasm --click 20,20 --expect-text true --expect-a11y-checked checkbox,,true --expect-a11y-focused checkbox,
go run ./cmd/vugra wasm examples/disabled/DisabledButton.vue /tmp/vugra-disabled-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-disabled-wasm --click 20,50 --expect-text 0 --expect-a11y-disabled button,Disabled,true
node tools/wasm-browser-check/run.mjs /tmp/vugra-disabled-wasm --a11y-click button,Disabled --expect-text 0 --expect-a11y-disabled button,Disabled,true
node tools/wasm-browser-check/run.mjs /tmp/vugra-disabled-wasm --a11y-focus button,Disabled --expect-text 0 --expect-a11y-disabled button,Disabled,true --expect-a11y-not-focused button,Disabled
go run ./cmd/vugra wasm examples/scroll/ScrollBox.vue /tmp/vugra-scroll-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-scroll-wasm --wheel 10,10,30 --expect-a11y-y-lt text,Delta,60 --expect-a11y text,Delta
go run ./cmd/vugra wasm examples/styled/StyledBox.vue /tmp/vugra-styled-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-styled-wasm --expect-text Styled --expect-pixel 10,10,#123456
go run ./cmd/vugra wasm examples/background/BackgroundBox.vue /tmp/vugra-background-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-background-wasm --expect-text Background --expect-pixel 10,10,#3b0764
go run ./cmd/vugra wasm examples/opacity/OpacityBox.vue /tmp/vugra-opacity-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-opacity-wasm --expect-text Opacity --expect-pixel 10,10,#90aff3
go run ./cmd/vugra wasm examples/svg/SVGDemo.vue /tmp/vugra-svg-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-svg-wasm --expect-pixel 24,36,#2563eb --expect-pixel 76,28,#f97316
go run ./cmd/vugra wasm examples/pointer/PointerBox.vue /tmp/vugra-pointer-wasm
node tools/wasm-browser-check/run.mjs /tmp/vugra-pointer-wasm --hover 20,20 --expect-text hover
node tools/wasm-browser-check/run.mjs /tmp/vugra-pointer-wasm --drag 20,20,45,20 --expect-text 'drag 25'
node tools/wasm-browser-check/run.mjs /tmp/vugra-pointer-wasm --dblclick 20,20 --expect-text double
node tools/wasm-browser-check/run.mjs /tmp/vugra-pointer-wasm --contextmenu 20,20 --expect-text menu
node tools/wasm-smoke/run.mjs /tmp/vugra-counter-wasm
node tools/wasm-smoke/run.mjs /tmp/vugra-counter-wasm --click 30,60 --expect-text 1 --expect-a11y button,+
node tools/wasm-smoke/run.mjs /tmp/vugra-counter-wasm --a11y-click button,+ --expect-text 1
node tools/wasm-smoke/run.mjs /tmp/vugra-focus-wasm --key Tab --key Tab --key Shift+Tab --expect-a11y-focused button,First
node tools/wasm-smoke/run.mjs /tmp/vugra-disabled-wasm --a11y-focus button,Disabled --expect-text 0 --expect-a11y-disabled button,Disabled,true --expect-a11y-not-focused button,Disabled
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --key Tab --expect-a11y-focused textbox,
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --expect-text abc --expect-a11y-focused textbox,abc
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key Backspace --expect-text ab
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key ArrowLeft --text X --expect-text abXc
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key Home --text X --key End --text Y --expect-text XabcY
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text draft --key Mod+A --text final --expect-text final
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --a11y-text textbox,,xy --a11y-key textbox,xy,Backspace --expect-text x
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --compose 你好 --expect-text 你好
node tools/wasm-smoke/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --paste pasted --expect-text pasted
node tools/wasm-smoke/run.mjs /tmp/vugra-checkbox-wasm --click 20,20 --expect-text true --expect-checkmark --expect-a11y-checked checkbox,,true
node tools/wasm-smoke/run.mjs /tmp/vugra-styled-wasm --expect-text Styled --expect-fill '#123456' --expect-stroke '#abcdef' --expect-fill '#fedcba' --expect-line-width 3 --expect-rounded
node tools/wasm-smoke/run.mjs /tmp/vugra-background-wasm --expect-text Background --expect-fill '#3b0764' --expect-stroke '#f59e0b'
node tools/wasm-smoke/run.mjs /tmp/vugra-opacity-wasm --expect-text Opacity --expect-fill '#2563eb' --expect-stroke '#0f172a' --expect-alpha 0.5
node tools/wasm-smoke/run.mjs /tmp/vugra-scroll-wasm --wheel 10,10,30 --expect-latest-text-y-lt Delta,70 --expect-clip
node tools/wasm-smoke/run.mjs /tmp/vugra-svg-wasm --expect-fill '#2563eb' --expect-fill '#f97316' --expect-stroke '#ffffff' --expect-line-width 3 --expect-arc --expect-svg-path
python3 -m http.server 8000 --directory /tmp/vugra-counter-wasm
```

`parse` prints structured SFC blocks and diagnostics, including source ranges from the original `.vue` file. `check` runs SFC parsing, template parsing, style parsing, Go metadata extraction, and IR validation. `ir` prints the renderer-neutral component IR. `frame` runs the compiler, layout, runtime, and test renderer and prints the first resolved render frame with layout rectangles. `vello-ops` translates that frame into the Vello scene-operation contract; piping it into `tools/vello-sidecar` verifies that the ops build and render through a real Vello `Scene`. `vello-png` runs that sidecar and writes a Vello-rendered PNG when a wgpu adapter is available. `a11y` prints the backend-neutral accessibility tree for the first frame. `png` renders the first frame through the software renderer. `native-png` runs the native host adapter and writes a PNG frame. `native-window` opens the Cocoa-backed native window on macOS with cgo; when invoked through `go run ./cmd/vugra`, the CLI automatically relaunches itself with the native build tag. `run` reads `vugra.config.json` from a project directory or config path, compiles the configured entry component, applies project runtime settings and declarative system GUI actions, then starts the native host. The config schema and project convention are documented in `docs/project-config.md` and `docs/vugra.config.schema.json`. `go-finder-lite smoke`, `rust-finder-lite native-smoke`, and `gui-runtime-smoke` are the repeatable non-window checks for the Go and Rust GUI runtime slices. `go-finder-lite native-window-smoke` opens a short-lived Cocoa native window for the Go Finder Lite runtime, verifies command/pixel output and a row click, then exits. `go-finder-lite native` opens the Go Finder Lite component through the native window path, while `go-finder-lite run` opens the configured Finder Lite project. Rust native GUI commands now default to the Vello-lowered backend; explicit `native-software*` and `abi-software*` variants keep the software fallback available for comparison and tests. `rust-finder-lite *window-smoke` commands open short-lived native windows across Vello-lowered, software fallback, and wgpu-lowered backends for both direct Rust and ABI-shaped state. `rust-sfc-smoke` compiles `<script lang="rust">` examples, generates Rust state adapters, and runs temporary Cargo tests. `rust-finder-sfc *window-smoke` opens short-lived native windows from the generated Rust Finder Lite SFC path across Vello-lowered, software fallback, and wgpu-lowered backends; `rust-finder-sfc native`, `native-software`, `native-vello`, and `native-wgpu` open the same generated native app until it is closed. `native-app` wraps a built binary in a macOS `.app` bundle and can bake in launch arguments such as `native-window examples/counter/Counter.vue`, which is the most reliable way to launch the GUI from Finder or `open`. Set `VUGRA_NATIVE_RENDERER=software` only to compare the experimental software path, or `VUGRA_NATIVE_RENDERER=vello` to compare the old per-frame sidecar path. `gen` emits a generated component package with the original Go `State`, static Vugra IR/style constructors, and runtime state glue. `gen-main` emits a small software-rendered Go entry point for that generated package. `gen-wasm-main` emits a wasm/canvas Go entry point for that generated package. `wasm-host` emits a browser host page for a wasm build. `wasm` compiles a `.vue` file, project directory, or `vugra.config.json` into a browser bundle containing `index.html`, `app.wasm`, and Go's `wasm_exec.js`; project inputs use config title, viewport size, and layout engine. `wasm-run` builds the same bundle in a temporary directory and serves it locally. `wasm-serve` serves an existing static bundle locally with the correct `application/wasm` MIME type; any static HTTP server can be used for deployment.
