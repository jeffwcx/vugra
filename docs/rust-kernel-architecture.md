# Rust Kernel Architecture

This note sketches a possible Vugra architecture where Rust becomes the
foundation of the framework, while Go and Rust user-facing APIs are derived from
the same runtime core.

The main principle is:

```text
Vugra = language-neutral component IR + Rust kernel + language bindings.
```

This is intentionally different from treating Rust as only an acceleration layer
under a Go framework. In this model, Rust owns the core platform, and Go is one
developer-facing language frontend and binding.

## Target Shape

```text
                         +---------------------------+
                         |        .vue / .vugra       |
                         |  template + style + script |
                         +-------------+-------------+
                                       |
                    +------------------+------------------+
                    |                                     |
        +-----------v-----------+             +-----------v-----------+
        | Go Language Frontend   |             | Rust Language Frontend |
        | <script lang="go">     |             | <script lang="rust">   |
        | gopls metadata         |             | rust-analyzer metadata |
        | Go codegen             |             | Rust codegen           |
        +-----------+-----------+             +-----------+-----------+
                    |                                     |
                    +------------------+------------------+
                                       |
                                       v
                         +---------------------------+
                         |      Vugra Component IR   |
                         | language-neutral schema   |
                         | nodes / bindings / events |
                         | props / styles / spans    |
                         +-------------+-------------+
                                       |
                                       v
              +------------------------------------------------+
              |                 Rust Vugra Kernel               |
              |                                                |
              |  +------------------------------------------+  |
              |  | Runtime Core                             |  |
              |  | component tree / state slots / effects   |  |
              |  | scheduler / lifecycle / invalidation     |  |
              |  +------------------------------------------+  |
              |                                                |
              |  +------------------------------------------+  |
              |  | Reactive Memory Model                    |  |
              |  | stable signal layout / typed values      |  |
              |  | subscriptions / dirty flags / batching   |  |
              |  +------------------------------------------+  |
              |                                                |
              |  +------------------------------------------+  |
              |  | UI Semantics                             |  |
              |  | events / focus / input / accessibility   |  |
              |  | hit-test / text selection / scroll       |  |
              |  +------------------------------------------+  |
              |                                                |
              |  +------------------------------------------+  |
              |  | Layout + Text                            |  |
              |  | Taffy / Parley / font fallback           |  |
              |  | boxes / glyph runs / baselines           |  |
              |  +------------------------------------------+  |
              |                                                |
              |  +------------------------------------------+  |
              |  | Scene / Display List                     |  |
              |  | retained scene / clips / scroll nodes    |  |
              |  | render commands / dirty regions          |  |
              |  +------------------------------------------+  |
              +-------------------+----------------------------+
                                  |
           +----------------------+----------------------+
           |                      |                      |
           v                      v                      v
+------------------+   +------------------+   +------------------+
| Native Backend   |   | Wasm Backend     |   | Test Backend     |
| wgpu / Vello     |   | WebGPU / Canvas  |   | snapshots / IR   |
| window / IME     |   | browser host     |   | golden tests     |
+------------------+   +------------------+   +------------------+
```

Go and Rust frameworks then become language bindings over the same Rust kernel:

```text
Go User API
  -> Go generated component glue
  -> C ABI / cgo / wasm import
  -> Rust Vugra Kernel

Rust User API
  -> Rust generated component glue
  -> direct Rust crate calls
  -> Rust Vugra Kernel
```

In practice:

```text
Go app
  State struct / signal.Int / methods
       |
       v
  generated Go adapter
       |
       v
  stable ABI
       |
       v
  Rust kernel

Rust app
  State struct / Signal<T> / impl methods
       |
       v
  generated Rust adapter
       |
       v
  direct kernel API
       |
       v
  Rust kernel
```

## Suggested Crate Layout

The Rust foundation should be split into small crates with clear ownership:

```text
crates/
  vugra-ir/
    Component IR schema
    template node model
    binding/event/style metadata
    source spans

  vugra-core/
    runtime tree
    state slots
    signals/effects
    scheduler
    lifecycle
    component instance model

  vugra-layout/
    style resolution
    Taffy integration
    layout boxes

  vugra-text/
    font discovery
    shaping
    glyph runs
    selection/caret metrics

  vugra-scene/
    display list
    retained scene
    clip/scroll nodes
    dirty regions

  vugra-render/
    renderer traits
    render commands

  vugra-render-vello/
    Vello backend

  vugra-render-wgpu/
    wgpu/WebGPU backend

  vugra-host-native/
    window/input/IME/accessibility bridge

  vugra-host-wasm/
    wasm host
    canvas/WebGPU/browser events/accessibility mirror

  vugra-abi/
    C ABI / stable binary ABI
    value representation
    state/method/signal calls

  vugra-go/
    Go binding support
    ABI wrappers
    generated Go adapter contract

  vugra-rs/
    Rust user-facing framework API
    macros/codegen helpers

  vugra-rs-codegen/
    Rust component contract and state adapter glue generation
```

## ABI Boundary

The ABI is the most important part of this architecture.

Go should not directly touch Rust-owned complex objects, Rust generics, trait
objects, `Vec`, `String`, or Rust struct layout. The cross-language boundary
should expose stable handles and plain data structs only.

Conceptual ABI types:

```c
typedef uint64_t VugraAppHandle;
typedef uint64_t VugraComponentHandle;
typedef uint64_t VugraStateHandle;
typedef uint32_t VugraSignalId;
typedef uint32_t VugraMethodId;

typedef struct {
  uint32_t kind;
  uint32_t flags;
  double number;
  uint64_t string_ptr;
  uint64_t string_len;
} VugraValue;

typedef struct {
  uint32_t kind;
  uint32_t key;
  float x;
  float y;
  float delta_x;
  float delta_y;
  uint32_t modifiers;
} VugraEvent;

typedef struct {
  uint64_t commands_len;
  uint64_t pixels_ptr;
  uint64_t pixels_len;
  uint64_t drawn_pixels;
} VugraNativeFrameView;
```

Go should see lightweight wrapper handles:

```go
type App struct {
    handle C.VugraAppHandle
}
```

Rust users can call the kernel directly through rich Rust types:

```rust
pub struct App {
    inner: vugra_core::App,
}
```

This gives Rust the faster path and consistent core memory layout, while Go gets
the same runtime semantics through the stable ABI.

## Component Contract

Language frontends should expose component state through one shared contract:

```text
Component Contract
  - create_state() -> StateHandle
  - destroy_state(StateHandle)
  - get_signal(StateHandle, SignalId) -> Value
  - set_signal(StateHandle, SignalId, Value)
  - call_method(StateHandle, MethodId)
  - call_event_method(StateHandle, MethodId, Event)
  - subscribe_signal(StateHandle, SignalId, callback)
```

Go components implement the contract through generated Go glue:

```text
Go State
  -> generated Go adapter
  -> ABI callbacks
  -> Rust kernel calls back into Go methods/signals
```

Rust components implement the same contract directly:

```text
Rust State
  -> generated Rust adapter
  -> direct trait impl
  -> Rust kernel calls directly
```

The Rust kernel can model the internal interface as a trait:

```rust
pub trait ComponentState {
    fn get_signal(&self, id: SignalId) -> Value;
    fn set_signal(&mut self, id: SignalId, value: Value);
    fn call_method(&mut self, id: MethodId);
    fn call_event_method(&mut self, id: MethodId, event: Event);
}
```

The Go binding becomes an FFI adapter for that trait-like behavior.

## Compile Pipeline

The desired compiler flow is:

```text
.vue file
  -> SFC parser
  -> template parser
  -> script frontend:
       Go analyzer
       Rust analyzer
  -> language-neutral Component IR
  -> codegen:
       Go adapter code
       Rust adapter code
       component manifest
  -> runtime:
       Rust kernel
  -> backend:
       native / wasm / test
```

For a Go component:

```text
Counter.vue
  +- template
  |    -> Vugra template IR
  |
  +- style
  |    -> Vugra style IR
  |
  +- script lang="go"
       -> Go metadata
       -> generated Go adapter

Both produce:
  component.vugrair
```

For a Rust component:

```text
Counter.vue
  +- template
  |    -> Vugra template IR
  |
  +- style
  |    -> Vugra style IR
  |
  +- script lang="rust"
       -> Rust metadata
       -> generated Rust adapter

Both produce:
  component.vugrair
```

## Constraints To Preserve

Avoid this shape:

```text
Go runtime
  -> Rust runtime
  -> Go renderer
  -> Rust renderer
```

That creates two overlapping runtimes and makes ownership unclear.

Also avoid this:

```text
Go compiler 生成 Go runtime 行为
Rust compiler 生成 Rust runtime 行为
```

That would let Go and Rust component semantics drift over time.

The healthier rule is:

```text
Only IR defines component semantics.
Only Rust kernel defines runtime semantics.
Language frontends only expose state/method metadata.
```

## Recommended First Steps

Before implementing a Rust runtime, define the stable foundation:

```text
1. vugra-ir schema
2. vugra-state/component contract
3. vugra-abi value/event/layout/display-list structs
```

Once these are stable, the Rust kernel and Go binding can be developed without
turning the project into two separate frameworks.

## Current Executable Slice

The repository now contains a first executable version of this architecture:

```text
crates/vugra-ir
  Finder Lite component contract:
  signals / methods / row bindings

crates/vugra-core
  ComponentState trait
  App dispatch
  renderer-neutral frame model

crates/vugra-abi
  C-shaped VugraValue / VugraEvent structs
  stable App / Component / State handles
  exported C ABI functions for state, method, render, native frame,
  and native event dispatch calls
  ABI adapter implementing ComponentState

crates/vugra-go
  Go-shaped wrapper contract over stable ABI handles
  proof that Go bindings can mount, render native frames, and dispatch
  native pointer/key/text events without Rust object ownership

crates/vugra-layout
  platform-neutral layout boxes from core frames
  overflow and scroll metadata for scene clip/scroll construction

crates/vugra-text
  text run boundary with metrics and temporary bitmap glyph pixels
  fixed metrics test implementation while real shaping is pending

crates/vugra-scene
  retained display-list construction from layout boxes
  display items with clip ids
  clip nodes and scroll nodes
  retained hit-test tree from interactive layout boxes

crates/vugra-render
  renderer trait
  render commands
  test renderer snapshots

crates/vugra-render-vello
  Vello backend boundary over renderer-neutral scenes
  tested lowering from render commands to Vello-like fill/text/end operations

crates/vugra-render-wgpu
  wgpu/WebGPU backend boundary over renderer-neutral scenes
  tested lowering from render commands to WebGPU-style quad/text/end pass ops

crates/vugra-host-native
  native/test host entry point for the Rust pipeline
  render-command driven software native window backend for Rust Finder Lite

crates/vugra-host-wasm
  wasm host boundary with canvas identity and shared render path

crates/vugra-rs
  Rust user-facing App API
  direct Rust components mount without ABI overhead

crates/vugra-rs-codegen
  tested Rust contract generation from Vugra IR
  Finder Lite generated fixture compiles back to the same IR
  tested ComponentState adapter generation for Rust bindings

internal/rustanalysis
  minimal Rust SFC metadata extractor for State fields and methods

internal/rustcodegen
  Go-side Rust adapter source generator from compiler metadata
  generic binding-trait adapter generation
  Finder Lite contract adapter generation

examples/finder-rust
  direct Rust component state
  ABI-backed component state
  both rendered through the same Rust kernel pipeline

examples/finder-rust-sfc
  <script lang="rust"> Finder Lite SFC input
  generated Rust adapter/native window smoke path
```

The current verified runtime path is:

```text
Finder Lite state
  -> vugra-core App
  -> vugra-layout layout boxes
  -> vugra-scene display list
  -> vugra-render commands
  -> vugra-host-native test output or native software window
```

Two Finder Lite implementations exercise the language-binding boundary:

```text
direct
  Rust state struct
  -> ComponentState trait
  -> vugra-rs App
  -> Rust kernel

abi
  ABI-shaped signal/method table
  -> ComponentState adapter
  -> Rust kernel

go-shaped
  lightweight App / Component / State wrappers
  -> stable ABI handles
  -> native frame / pointer / key / text ABI calls
  -> Rust kernel
```

The Go CLI can start the Rust Finder Lite variants and a Go runtime smoke:

```sh
go run ./cmd/vugra go-finder-lite smoke
go run ./cmd/vugra go-finder-lite native-window-smoke
go run ./cmd/vugra go-finder-lite native
go run ./cmd/vugra go-finder-lite run
go run ./cmd/vugra gui-runtime-smoke
go run ./cmd/vugra gui-runtime-smoke window
go run ./cmd/vugra rust-sfc-smoke
go run ./cmd/vugra rust-finder-sfc native
go run ./cmd/vugra rust-finder-sfc native-software
go run ./cmd/vugra rust-finder-sfc native-vello
go run ./cmd/vugra rust-finder-sfc native-wgpu
go run ./cmd/vugra rust-finder-sfc native-window-smoke
go run ./cmd/vugra rust-finder-sfc native-software-window-smoke
go run ./cmd/vugra rust-finder-sfc native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite direct
go run ./cmd/vugra rust-finder-lite abi
go run ./cmd/vugra rust-finder-lite native
go run ./cmd/vugra rust-finder-lite native-software
go run ./cmd/vugra rust-finder-lite native-vello
go run ./cmd/vugra rust-finder-lite native-wgpu
go run ./cmd/vugra rust-finder-lite abi-native
go run ./cmd/vugra rust-finder-lite abi-native-software
go run ./cmd/vugra rust-finder-lite abi-native-vello
go run ./cmd/vugra rust-finder-lite abi-native-wgpu
go run ./cmd/vugra rust-finder-lite native-smoke
go run ./cmd/vugra rust-finder-lite native-window-smoke
go run ./cmd/vugra rust-finder-lite native-software-window-smoke
go run ./cmd/vugra rust-finder-lite native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite abi-window-smoke
go run ./cmd/vugra rust-finder-lite abi-software-window-smoke
go run ./cmd/vugra rust-finder-lite abi-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite wgpu-device-smoke
# Requires rustc 1.88+ because Vello 0.9 / wgpu 29 need newer Rust:
cargo run -q -p finder-rust --features vugra-render-vello/vello-device -- vello-device-smoke
```

The native Rust Finder Lite path opens a window, paints from renderer-neutral
commands into a software pixel buffer, and dispatches row clicks back through the
Rust kernel before re-rendering. Row selection is represented in IR/core frame
state, flows through layout boxes into the retained scene hit-test tree, and is
used by native hit-testing instead of hard-coded coordinates or renderer command
callbacks. The Rust Finder Lite surface now includes toolbar, path field, search
field, sidebar favorites, column headers, file rows, selected-row highlight, and
status bar commands. Sidebar entries are also method-bound scene hit-test nodes,
so clicking Documents, Downloads, or Pictures updates the Rust state, active
sidebar item, path, rows, status, and selection before re-rendering. ArrowUp and
ArrowDown dispatch the same method-bound state path as pointer input, so keyboard
row navigation also re-renders through the Rust kernel instead of mutating
host-only view state. Text input in the native window is collected by the host
and dispatched as a renderer-neutral event method to the search binding;
Backspace dispatches the search delete method.
Both direct Rust state and ABI-shaped state use that same path to filter the
visible Finder rows before layout and rendering. Enter dispatches an
OpenSelected method through the same runtime path: folders replace the visible
row set and update the path, while files update the selected summary. When the
search query is empty, Backspace dispatches OpenParent so a native user can
return from a folder entered through OpenSelected; when search is non-empty it
continues to edit the search query.

The native host now has selectable Rust render backend modes:

```text
software
  render commands -> software rasterizer

vello
  render commands -> Vello-like fill/text/end ops -> native pixel buffer

wgpu
  render commands -> WebGPU-style quad/text/end pass ops -> native pixel buffer
```

The Vello and wgpu modes still present the final pixel buffer through minifb in
this slice, but their backend ops now rasterize pixels directly instead of
round-tripping back through render commands. Hit-testing uses the retained scene
hit-test tree, so row/sidebar method dispatch remains renderer-independent. The
scene hit-test nodes retain parent links and can now produce an event route with
explicit capture, target, and bubble node indexes. The scene layer also builds
an executable propagation plan and supports stop-after-handler semantics.
`LayoutBox` and `HitTestNode` carry distinct capture, target, and bubble handler
slots, while the older single `method` field is normalized into a target handler
for compatibility. The native host now dispatches pointer input through that
plan while only invoking the target phase for the current Finder Lite contract;
renderer backends do not participate in the propagation semantics.
The Rust scene layer also has a retained scene model and update cache. `Scene`
keeps the existing renderer command stream, plus display items carrying active
clip ids, clip nodes, and scroll nodes derived from layout overflow metadata.
Hit-test nodes retain the active clip rectangles from scene construction, and
native pointer dispatch rejects targets outside those clip bounds before handler
dispatch.
`RetainedScene` compares stable `Begin` and `Text` command ids against the
previous scene snapshot, reports visible dirty rectangles on the first update,
reports no dirty regions for unchanged updates, marks both old and new
rectangles when commands move or resize, marks the same rectangle when command
content or state changes, and marks old rectangles when commands disappear.
`NativeFrameState` holds that retained cache across native frame renders, and
the native window loop reuses it after pointer, key, and text events. Native
frames therefore carry dirty rectangles for initial, unchanged, and changed
scene updates. The native host keeps a pixel cache as well: unchanged frames
reuse the cached buffer, and changed frames update the cache from dirty
rectangles before presenting. The software backend now redraws only dirty
regions into that cache by clearing each dirty rectangle and clipping replayed
commands to the dirty bounds. The Vello-lowered and wgpu-lowered native paths do
the same at their lowered-op layer, clipping `VelloOp` and `WgpuPassOp` replay
to dirty bounds before presenting.
The native software bitmap edge preserves fractional layout rectangles until the
final raster step, where starts are floored and ends are ceiled. That keeps
subpixel geometry from being truncated into zero-width or zero-height pixels
before the renderer boundary.

The `wgpu-device` feature on `vugra-render-wgpu` adds a real headless wgpu
offscreen path that works on the current Rust 1.83 baseline. It lowers Finder
Lite to `WgpuPassOp`, submits solid and glyph quads to a real wgpu device,
renders into an RGBA texture, copies it back, and validates the pixel buffer,
title text pixels, and checksum:

```sh
go run ./cmd/vugra rust-finder-lite wgpu-device-smoke
```

This device smoke proves the wgpu device/readback path for Vugra's
renderer-neutral GUI output, including the Finder Lite title text. The temporary
8x8 glyph pixels live behind the `vugra-text::TextRun` boundary, which returns
metrics and paint pixels together and is shared by the native software present
path, the Vello-lowered native path, the wgpu-device path, and the
feature-gated Vello device path. `VelloOp::Text` carries the original text and
rect for diagnostics/round-tripping plus the unified `TextRun` for actual
painting. This keeps the replacement point for a real shaped text pipeline
explicit instead of letting each backend invent separate measurement and
painting data.

The `vello-device` feature on `vugra-render-vello` adds a real Vello/wgpu
offscreen path for the Rust runtime lowering output. It lowers Finder Lite to
`VelloOp`, builds a `vello::Scene`, paints text from the same `TextRun` glyph
pixel geometry consumed by the native software/Vello-lowered paths, renders it
through a headless wgpu device into an RGBA texture, and reads pixels back for
checksum validation. The command is intentionally feature-gated because Vello
0.9 / wgpu 29 currently require rustc 1.88+, while the default workspace remains
buildable on the older local toolchain:

```sh
cargo run -q -p finder-rust --features vugra-render-vello/vello-device -- vello-device-smoke
```

`native-smoke` is the non-window validation path for the native Rust GUI runtime
matrix. It renders direct Rust state and ABI-shaped state through the software,
Vello-lowered, and wgpu-lowered native frame paths, verifies that each frame has
commands, pixels, drawn content, and method-bound row metadata, and then checks
direct and ABI frame equality for each backend. It also drives the same native
host dispatch helpers used by the window loop for row pointer selection,
ArrowDown, search text input, Backspace, and Enter, checking after each step
that direct Rust state and ABI-shaped state still produce the same frame. It is
safe to run in CI because it does not open a blocking minifb window.

The `*window-smoke` variants are the short-lived validation path for the actual
native window host. They open a minifb window, present a small number of Finder
Lite frames, check command count, pixel count, and drawn pixels, then exit
without requiring the user to close a long-running window. Rust native GUI
commands default to the Vello-lowered backend: `native-window-smoke` and
`abi-window-smoke` are the default Vello paths, while `native-vello-window-smoke`
and `abi-vello-window-smoke` remain explicit aliases. Direct Rust state is also
covered by `native-software-window-smoke` and `native-wgpu-window-smoke`;
ABI-shaped state is covered by `abi-software-window-smoke` and
`abi-wgpu-window-smoke`. This proves the window creation/present path
separately from the non-window frame and device smokes across the Vello-lowered
default, software fallback, and wgpu-lowered native backends.
It still requires a usable local window system, so `gui-runtime-smoke` keeps the
non-window Rust checks as its default aggregate path. For local end-to-end
window validation, `gui-runtime-smoke window` appends the Go Finder Lite native
window smoke, the generated Rust SFC window smoke, and all direct/ABI
short-lived native backend window smokes.

The ABI window-smoke variants run the same short-lived native window path
through stable ABI handles. The ABI function catches native-window panics inside
the `extern "C"` boundary and returns an empty smoke result instead of unwinding
through FFI. The Rust unit tests keep window-opening cases ignored because the
test harness may not run them on the platform main thread; the CLI smoke is the
authoritative local window check.

`go-finder-lite smoke` is the matching non-window validation path for the Go
runtime. It compiles `examples/finder/FinderLite.vue`, mounts it with Go
Signal-backed Finder state, dispatches row selection and search text through the
runtime hit-test/input paths, then renders the final command frame through the
software pixel renderer. `go-finder-lite native-window-smoke` opens a
short-lived Cocoa native window for the same Go Finder Lite runtime, verifies
native command/pixel output plus a method-bound row click, then exits without
waiting for manual window close. `go-finder-lite native` opens the same Go
Finder Lite component through the native window command, and `go-finder-lite
run` opens the configured `examples/finder` project. Together,
`go-finder-lite smoke`, `go-finder-lite native-window-smoke`, and
`rust-finder-lite native-smoke` are the repeatable check that both Go and Rust
GUI runtime paths can execute without opening a blocking window. The aggregate
`gui-runtime-smoke` command runs the Go smoke, the Rust native-smoke matrix, and
the real wgpu device/readback smoke in sequence. `gui-runtime-smoke window`
adds the Go native window smoke, the generated Rust SFC window smoke, plus
direct and ABI native window-present checks for all native backend modes on
local machines with a usable window system.

The `abi-native*` variants mount the ABI-shaped Finder Lite state into the same
native host and backend paths as the direct Rust state. This proves the native
GUI path is no longer limited to direct Rust component state.

The Rust SFC path now accepts `<script lang="rust">`, extracts a narrow State
field/method metadata model, generates a Rust `ComponentState` adapter from the
compiler result, and verifies that adapter in temporary Cargo crates. The
generated Finder Lite SFC path uses the Finder Lite contract adapter and can
open a native window:

```sh
go run ./cmd/vugra rust-sfc-smoke
go run ./cmd/vugra rust-finder-sfc native
go run ./cmd/vugra rust-finder-sfc native-software
go run ./cmd/vugra rust-finder-sfc native-vello
go run ./cmd/vugra rust-finder-sfc native-wgpu
go run ./cmd/vugra rust-finder-sfc native-window-smoke
go run ./cmd/vugra rust-finder-sfc native-software-window-smoke
go run ./cmd/vugra rust-finder-sfc native-vello-window-smoke
go run ./cmd/vugra rust-finder-sfc native-wgpu-window-smoke
```

This is intentionally still a constrained SFC implementation. It proves the
language-neutral compiler metadata -> Rust adapter -> Rust kernel -> native host
path across the Vello-lowered default, software fallback, and wgpu-lowered
native backend modes,
but it is not yet a complete Rust source compiler for arbitrary component logic.

The `vugra-abi` and `vugra-go` crates also expose the native frame and native
dispatch path through stable handles. `vugra_app_render_native_frame` returns
renderer-neutral command counts plus a retained pixel view, while
`vugra_app_dispatch_native_pointer`, `vugra_app_dispatch_native_key`, and
`vugra_app_dispatch_native_text` call the same host helpers as the native window
loop. The Go-shaped Rust wrapper tests exercise those ABI functions across the
software, Vello-lowered, and wgpu-lowered backends. The wrapper also exposes the
short-lived native window smoke shape, while the actual window-present check is
kept in the main-thread CLI smoke.

The Rust glue/codegen slice now includes `vugra-rs-codegen`. It can generate a
Rust function that reconstructs a `vugra_ir::Component` contract from IR and a
Rust `ComponentState` adapter that maps SignalId/MethodId/Event calls onto a
snake_case binding trait. The Finder Lite generated fixtures are checked in as
compile-tested artifacts: generator output must match the fixtures, the
contract fixture must compile back to the same `finder_lite_contract()` IR, and
the adapter fixture must mount in `vugra_core::App`, render rows, dispatch row
selection, and dispatch search text input. This is still a narrow Rust glue
generation step, but it moves Rust toward generated component state adapters
instead of a purely hand-written Finder Lite contract.

Validation commands:

```sh
cargo test --workspace
env GOPROXY=https://goproxy.cn,direct go test ./cmd/vugra ./internal/ir ./internal/runtime ./internal/compiler
go run ./cmd/vugra go-finder-lite smoke
go run ./cmd/vugra go-finder-lite native-window-smoke
go run ./cmd/vugra go-finder-lite native
go run ./cmd/vugra go-finder-lite run
go run ./cmd/vugra gui-runtime-smoke
go run ./cmd/vugra gui-runtime-smoke window
go run ./cmd/vugra rust-sfc-smoke
go run ./cmd/vugra rust-finder-sfc native-window-smoke
go run ./cmd/vugra rust-finder-sfc native-software-window-smoke
go run ./cmd/vugra rust-finder-sfc native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite direct
go run ./cmd/vugra rust-finder-lite abi
go run ./cmd/vugra rust-finder-lite native-smoke
go run ./cmd/vugra rust-finder-lite native-window-smoke
go run ./cmd/vugra rust-finder-lite native-software-window-smoke
go run ./cmd/vugra rust-finder-lite native-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite abi-window-smoke
go run ./cmd/vugra rust-finder-lite abi-software-window-smoke
go run ./cmd/vugra rust-finder-lite abi-wgpu-window-smoke
go run ./cmd/vugra rust-finder-lite wgpu-device-smoke
# Optional real Vello/wgpu offscreen check on rustc 1.88+:
cargo run -q -p finder-rust --features vugra-render-vello/vello-device -- vello-device-smoke
# Opens a native Rust Finder Lite window and runs until the window is closed:
go run ./cmd/vugra rust-finder-lite native
go run ./cmd/vugra rust-finder-lite native-software
go run ./cmd/vugra rust-finder-lite native-vello
go run ./cmd/vugra rust-finder-lite native-wgpu
go run ./cmd/vugra rust-finder-lite abi-native
go run ./cmd/vugra rust-finder-lite abi-native-software
go run ./cmd/vugra rust-finder-lite abi-native-vello
go run ./cmd/vugra rust-finder-lite abi-native-wgpu
go run ./cmd/vugra rust-finder-sfc native
go run ./cmd/vugra rust-finder-sfc native-software
go run ./cmd/vugra rust-finder-sfc native-vello
go run ./cmd/vugra rust-finder-sfc native-wgpu
```

This is still an executable slice, not the whole final architecture. The next
steps are to deepen the IR schema beyond Finder Lite rows, add real
style/layout/text backends, move dirty-region rendering from the lowered native
pixel fallback into real retained Vello/wgpu surfaces, connect generated Go glue to the exported ABI
functions, expand Rust SFC analysis/codegen beyond the current State metadata
and binding-trait adapter shape, expose capture/bubble handler bindings through
the compiler-facing Rust IR instead of only the native retained scene structs,
expand the feature-gated Vello device path into the default supported toolchain
once the project baseline reaches rustc 1.88+, and replace the temporary
`vugra-text::TextRun` bitmap pixel implementation with a real shaped text
pipeline.
