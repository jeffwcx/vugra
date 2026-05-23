# Vugra Implementation Plan

This document is the implementation plan for Vugra as a production-oriented
cross-platform UI framework.

Vugra is not a Vue runtime clone and is not a browser DOM wrapper. The long-term
goal is a Go-first UI framework with standard HTML-like template syntax, real Go
component logic, explicit Signal reactivity, a framework-owned IR, and native and
wasm renderers backed by a shared layout, text, event, and rendering pipeline.

## 1. Product Definition

Vugra uses:

- Standard HTML syntax for templates.
- A small Vue-like directive subset for binding and control flow.
- Real Go source in `<script lang="go">`.
- Explicit Signal-based reactivity instead of JavaScript Proxy semantics.
- A framework-owned component IR before runtime, layout, rendering, or tooling.
- Native and wasm targets as first-class platforms.
- Renderer backends such as Vello, wgpu, WebGPU, and a software test backend.
- Volar.js plus `gopls` for editor tooling.

The product should feel like a native cross-platform UI framework that happens to
use familiar template and CSS-like syntax. Browser compatibility is useful where
it reduces user friction, but browser DOM behavior must not define the internal
architecture.

## 2. Architecture

Target pipeline:

```text
.vue source
  -> SFC parser
  -> HTML template AST + Go metadata + Style AST
  -> Vugra component IR
  -> runtime tree
  -> layout tree
  -> display list / scene graph
  -> renderer backend
```

Renderer-specific work starts only after platform-neutral structures exist.
Vello, wgpu, WebGPU, and software rendering consume display lists or render
commands; they do not consume component nodes or template syntax.

Layer ownership:

- `sfc`: Reads `.vue` files, extracts blocks, and preserves source ranges.
- `template`: Parses standard HTML syntax and Vugra directives.
- `goanalysis`: Extracts state, methods, props, events, and Signal metadata from
  Go ASTs.
- `ir`: Defines platform-neutral component, binding, event, and style structures.
- `reactivity`: Implements Signal, effect tracking, subscriptions, and batched
  invalidation.
- `runtime`: Mounts components, evaluates bindings, owns focus/event state, and
  schedules commits.
- `style`: Parses supported CSS-like syntax into a Vugra style model.
- `layout`: Produces backend-neutral layout boxes with subpixel coordinates.
- `text`: Owns font discovery, fallback, shaping, line boxes, glyph runs, and text
  metrics.
- `scene`: Builds retained display lists, clip trees, scroll nodes, and dirty
  regions.
- `renderer`: Converts scenes to Vello, wgpu, WebGPU, or software output.
- `host`: Normalizes native and wasm windows, surfaces, input, timers, and IME.
- `lsp`: Builds virtual documents and maps diagnostics/completions back to
  `.vue` source ranges.

Hard boundaries:

- Component syntax must not depend on DOM, AppKit, Vello, wgpu, or WebGPU
  internals.
- CSS parsing must not be coupled to rendering.
- Layout must not paint.
- Text measurement and text painting must share one shaping source of truth.
- Events must dispatch through a framework hit-test tree, not ad hoc renderer
  callbacks.
- Native and wasm must share component/runtime/layout semantics.

## 3. Current Status

The repository already has working pieces of the vertical slice, but several are
prototype-grade and should be treated as such.

Implemented or prototyped:

- SFC parsing, template parsing, Go analysis, IR generation, runtime mounting,
  and test-renderer style flows.
- Native window support on macOS.
- Vello-native and Vello sidecar paths.
- Rust CSS/layout sidecar and FFI bridge using Lightning CSS, Taffy, and Parley.
- Basic CSS-like support for flex-oriented examples, border, border-radius, and
  overflow clipping/scroll experiments.
- Counter-style runtime examples.
- Event dispatch and repaint coalescing experiments.

Known gaps:

- Layout/render coordinates still need a full subpixel float pipeline.
- Text measurement and text painting are not yet one unified glyph-run pipeline.
- CSS support is a subset and is not consistently represented across parser,
  layout, runtime, and renderer.
- Events need a dedicated hit-test tree, propagation model, and scheduler
  integration.
- Rendering does not yet have a production retained scene/compositor model.
- Native Vello currently still has extra bridging/copying cost compared with a
  direct retained surface path.
- Software rendering is useful for tests and fallback, but it is not the default
  product direction.

## 4. Non-Goals For The First Production Slice

Do not attempt these before the core pipeline is stable:

- Full Vue runtime compatibility.
- Full browser DOM compatibility.
- Full CSS cascade and browser layout compatibility.
- JavaScript expression semantics in templates.
- Deep Proxy-like reactivity.
- Table layout.
- SSR hydration.
- Slots, transitions, suspense, provide/inject.
- Rich text editing and contenteditable.
- Complete accessibility support across every platform.
- Browser form behavior parity.

These can be added later only after compiler, runtime, layout, text, event, and
renderer boundaries are stable.

## 5. Production Principles

### 5.1 Vugra-owned IR first

The compiler must lower `.vue` into Vugra IR before runtime, renderer, wasm, or
LSP code consumes it. The IR should preserve source ranges and should describe
components, elements, text, bindings, conditionals, repeaters, props, events, and
style references without backend details.

### 5.2 CSS Profile instead of full CSS

Use existing CSS parsing technology, not a hand-written general parser. The
initial Rust path using Lightning CSS is the right direction, but Vugra should
define a deliberate CSS Profile v1 instead of promising browser CSS.

CSS Profile v1 should include:

- Selectors: element, class, id, and simple descendant selectors.
- Box model: `display`, `box-sizing`, `width`, `height`, `min-width`,
  `min-height`, `max-width`, `max-height`, `margin`, `padding`.
- Flex: `flex-direction`, `flex-wrap`, `flex`, `flex-grow`, `flex-shrink`,
  `flex-basis`, `align-items`, `align-self`, `justify-content`, `gap`,
  `row-gap`, `column-gap`.
- Visuals: `background`, `background-color`, `color`, `opacity`.
- Border: `border`, `border-width`, `border-color`, `border-style`,
  `border-radius`.
- Text: `font-family`, `font-size`, `font-weight`, `line-height`, `text-align`,
  `white-space`.
- Overflow: `overflow`, `overflow-x`, `overflow-y` with `visible`, `hidden`,
  `scroll`, and later `auto`.
- Positioning: `position: relative|absolute`, `left`, `top`, `right`, `bottom`,
  and `z-index` after retained scene support exists.

Grid, media queries, transforms, filters, animations, custom properties, and full
cascade behavior are later work.

### 5.3 Subpixel layout and rendering

All layout and render geometry should use float coordinates end to end:

```go
type Rect struct {
    X      float32
    Y      float32
    Width  float32
    Height float32
}
```

Taffy, Parley, layout boxes, hit testing, display lists, Vello ops, clip nodes,
scroll offsets, and native surface scaling should preserve subpixel values.
Integer rounding should happen only at the final software bitmap edge or where a
host platform API explicitly requires pixels.

### 5.4 Unified text system

Text cannot be production-grade if one path measures text and another unrelated
path paints it. Vugra should have one text engine that owns:

- Font discovery.
- Font fallback, including CJK fallback.
- Shaping.
- Glyph runs.
- Line boxes.
- Baselines.
- Selection and caret metrics later.
- Text painting data consumed by Vello/wgpu/WebGPU.

Parley and related Rust text crates are good candidates for this layer. The goal
is not just returning text width and height; the text layer must return paintable
glyph-run data.

### 5.5 Events as a framework subsystem

Native and wasm input should be normalized into Vugra events:

```text
host event
  -> normalized pointer/key/wheel/text event
  -> coordinate transform
  -> hit-test tree
  -> capture / target / bubble
  -> handler dispatch
  -> batched state invalidation
  -> scheduled frame commit
```

Events must not synchronously force full layout and rendering inside the native
input callback. The runtime should coalesce updates and render at frame cadence.

### 5.6 Retained scene before complex features

Production rendering needs more than immediate render commands. Vugra should
build a retained scene model with:

- Display items.
- Clip nodes.
- Scroll nodes.
- Stacking order.
- Dirty flags.
- Dirty regions.
- Optional retained Vello scene data.

This is required for responsive scrolling, lower input latency, efficient
repaint, z-index, overflow, and later animations.

### 5.7 Rust/Go boundary

Go should own framework semantics:

- SFC/compiler coordination.
- Go AST metadata.
- Component state.
- Signals and effects.
- Runtime tree.
- Event dispatch semantics.
- Public framework API.

Rust should own performance-critical rendering infrastructure:

- CSS parsing and style normalization.
- Taffy layout integration.
- Text shaping and glyph-run output.
- Display-list lowering.
- Vello/wgpu/WebGPU rendering.
- Native surface integration where practical.

Short-term JSON/FFI is acceptable for prototyping, but the long-term boundary
should move toward stable ABI structs or a compact binary display-list protocol.
Per-frame large JSON should not be the final architecture.

## 6. Milestones

### Milestone 0: Core compiler and runtime vertical slice

Scope:

- SFC parser.
- HTML template parser.
- Go metadata extraction.
- Vugra IR.
- Signal runtime.
- Test renderer.
- Counter example.

Acceptance criteria:

- `go test ./...` runs.
- Parser diagnostics preserve `.vue` source locations.
- Template bindings validate against Go metadata.
- A simulated click calls a Go method and updates Signal state.
- Test renderer output changes after state updates.

### Milestone 1: CSS Profile v1 and layout foundation

Scope:

- Use Lightning CSS or another maintained parser for style parsing.
- Normalize supported properties into a Vugra style model.
- Use Taffy for flex layout.
- Keep layout boxes backend-neutral.
- Preserve source ranges for style diagnostics where possible.

Acceptance criteria:

- CSS Profile v1 is documented.
- Unsupported properties produce clear diagnostics or explicit no-op behavior.
- Flex examples have deterministic layout tests.
- Border, border-radius, padding, margin, gap, and overflow are represented
  consistently from CSS parse through layout and render style.

### Milestone 2: Subpixel geometry pipeline

Scope:

- Convert layout boxes, render commands, Vello ops, clip rectangles, scroll
  offsets, and hit-test rectangles to float coordinates.
- Keep compatibility adapters for software tests where needed.
- Audit all int conversions in layout and renderer packages.

Acceptance criteria:

- Fractional layout positions survive from Taffy output to Vello ops.
- Hit testing works with fractional rectangles.
- Software renderer rounds only at the final bitmap boundary.
- Tests assert that fractional positions are not truncated during normal layout
  and rendering.

### Milestone 3: Unified text layout and painting

Scope:

- Move from text width/height measurement to line boxes and glyph runs.
- Implement font fallback for Latin and CJK text.
- Feed the same glyph-run data to Vello painting and layout measurement.
- Define text clipping behavior under overflow.

Acceptance criteria:

- Chinese and mixed Latin/CJK text render without `?` fallback in the default
  demo path.
- Text measurement and painted text agree within a small tolerance.
- Baselines are stable for buttons, inputs, headings, and list items.
- Tests cover clipping, missing glyph fallback, and repeated text updates.

### Milestone 4: Event system and scheduler

Scope:

- Build a retained hit-test tree from layout and scene data.
- Normalize mouse, pointer, wheel, keyboard, and text-input events.
- Implement target dispatch and then capture/bubble.
- Batch Signal invalidations and schedule frame commits.
- Keep native callbacks non-blocking.

Acceptance criteria:

- Button clicks, scroll wheel, and basic keyboard events dispatch reliably.
- Event dispatch does not synchronously render a full frame inside the native
  callback.
- Multiple state changes in one event produce one scheduled commit.
- Tests cover hit=false cases, clipped nodes, scrolled nodes, and bubbling order.

### Milestone 5: Retained scene and Vello-native default

Scope:

- Build a scene/display-list layer separate from runtime and renderer.
- Represent clips, scroll containers, borders, rounded rectangles, text, and
  stacking order.
- Keep Vello-native as the default native renderer.
- Keep software rendering as test/fallback infrastructure only.
- Reduce unnecessary bitmap copies and per-frame serialization.

Acceptance criteria:

- Native demos launch through the Vello-native path by default.
- Software must be opt-in for tests or debugging.
- Dirty scene updates avoid rebuilding and repainting unrelated subtrees.
- Scrolling a list does not trigger full component re-evaluation unless state
  changes.
- Renderer status clearly reports whether Vello-native, Vello sidecar, or
  software fallback is active.

### Milestone 6: Native host hardening

Scope:

- macOS main-thread ownership for AppKit.
- Window lifecycle.
- Resize handling.
- Surface scale factor.
- Cursor, focus, and keyboard focus state.
- Panic/error reporting around FFI boundaries.

Acceptance criteria:

- Native windows are always created on the main thread.
- Resize updates layout, scene, and renderer surface dimensions correctly.
- Device scale factor is applied consistently to layout, text, and render output.
- FFI failures return actionable errors instead of process crashes where
  possible.

### Milestone 7: Wasm/WebGPU path

Scope:

- Share component, runtime, style, layout, text, and scene semantics with native.
- Provide a wasm host page only as a loader.
- Use WebGPU where available, with a documented fallback plan.
- Keep browser DOM out of the component semantics.

Acceptance criteria:

- Counter and Finder Lite examples run in wasm using shared runtime code.
- Pointer and keyboard events normalize to the same Vugra event types as native.
- Layout and text metrics match native within documented tolerances.
- Renderer backend can be selected without changing component source.

### Milestone 8: LSP and diagnostics

Scope:

- Volar.js language plugin for `.vue`.
- Virtual documents:
  - `Component.template.html`
  - `Component.script.go`
  - `Component.style.css`
  - `Component.meta.json`
- Bridge Go virtual files to `gopls`.
- Map diagnostics and completions through source ranges.

Acceptance criteria:

- Go diagnostics in `<script lang="go">` map back to `.vue`.
- Unknown template binding diagnostics map to interpolation ranges.
- Unsupported CSS Profile properties produce style diagnostics.
- Completion offers supported tags, known state, known methods, and supported
  style properties.

## 7. First Production Slice

The first credible product slice should be a small but honest desktop-style
example, not a marketing page and not a renderer-only demo.

It should demonstrate:

- Real `.vue` SFC input.
- Go state and methods.
- Signal updates.
- Template bindings.
- Flex layout.
- Border and border-radius.
- Overflow scroll.
- CJK text.
- Button/input alignment.
- Pointer click.
- Wheel scroll.
- Vello-native rendering by default.

End-to-end acceptance criteria:

- `go test ./...` passes.
- `cargo test --manifest-path tools/css-layout/Cargo.toml` passes.
- `cargo test --manifest-path tools/vello-native/Cargo.toml` passes.
- `VUGRA_LAYOUT_ENGINE=css go run ./cmd/vugra native-window examples/counter/Counter.vue`
  opens a GUI using Vello-native by default.
- Clicking controls updates state with no visible delayed second click.
- CJK and Latin text render without widespread missing-glyph replacement.
- Borders and rounded corners render without unexpected debug outlines.
- The same component can produce deterministic test renderer output.

## 8. CLI and Developer Experience

Current and target commands:

```text
vugra parse <file>
vugra check <file>
vugra ir <file>
vugra frame <file>
vugra vello-ops <file>
vugra vello-png <file>
vugra native-window <file>
vugra wasm-host <wasm-path>
```

Future CLI work should add a higher-level wasm command once the wasm/WebGPU path
shares the production runtime, layout, text, and scene model:

```text
vugra wasm <file>
```

Acceptance criteria:

- `vugra check` runs parser, template analysis, Go analysis, style validation,
  and IR validation.
- CLI diagnostics use file, line, and column.
- Renderer selection is explicit in logs.
- Vello-native is the default native path.
- Software is opt-in and documented as experimental/test fallback.
- Examples have one-command GUI run paths.

## 9. Risk Register

Major risks:

- Text shaping, font fallback, IME, and selection can dominate runtime
  complexity.
- Full CSS compatibility is too large and can obscure the framework boundary.
- Per-frame JSON/FFI transfer can become a performance ceiling.
- Vello/wgpu/WebGPU backend maturity can affect wasm and native delivery.
- AppKit and other native hosts have strict thread and lifecycle constraints.
- `gopls` virtual document integration may be harder than basic Volar mapping.
- Source maps can become fragile if generated wrappers are not designed early.

Mitigations:

- Keep CSS Profile explicit and test-backed.
- Treat text as a first-class subsystem, not a renderer helper.
- Move geometry to float before adding more visual features.
- Use deterministic test renderers before backend-specific debugging.
- Preserve source ranges from the first parser milestone.
- Keep Go virtual files valid and wrapper generation minimal.
- Add source-map tests for every diagnostic-producing layer.
- Replace per-frame JSON with a stable ABI or binary protocol before optimizing
  renderer internals.

## 10. Suggested Execution Order From Current State

1. Document CSS Profile v1 and align parser/layout/render style structs.
2. Convert layout, render commands, Vello ops, clipping, scrolling, and hit-test
   rectangles to float coordinates.
3. Replace text measurement-only output with line boxes and glyph runs.
4. Make Vello-native consume the unified text and float geometry data.
5. Build a retained hit-test tree and event dispatch subsystem.
6. Add frame scheduler tests for batched invalidation and non-blocking events.
7. Introduce retained display lists, clips, scroll nodes, and dirty flags.
8. Harden native host lifecycle, main-thread behavior, resize, and scale factor.
9. Bring wasm/WebGPU up on the same runtime/layout/text/scene model.
10. Expand LSP diagnostics once compiler/style source mapping is stable.

The next checkpoint is not more CSS breadth. The next checkpoint is precision,
text correctness, event responsiveness, and retained Vello-native rendering for
the current native demo.
