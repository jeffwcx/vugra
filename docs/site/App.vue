<template>
  <div class="docs-shell">
    <div class="sidebar">
      <h1 class="brand">Vugra</h1>
      <p class="tagline">A Vue-like SFC framework with Go state and native/wasm render targets.</p>
      <p class="nav-title">Docs</p>
      <button :class="overviewButtonClass" @click="ShowOverview">Overview</button>
      <button :class="manualButtonClass" @click="ShowManual">Manual</button>
      <button :class="apiButtonClass" @click="ShowAPI">API</button>
      <button :class="wasmButtonClass" @click="ShowWasm">Wasm</button>
      <p class="nav-title">References</p>
      <p class="reference">docs/project-config.md</p>
      <p class="reference">docs/css-profile-v1.md</p>
      <p class="reference">README.md</p>
    </div>

    <div class="content">
      <p class="eyebrow">{{ section }}</p>
      <h1 class="title">{{ title }}</h1>
      <p class="lead">{{ lead }}</p>

      <div class="hero-grid">
        <div class="panel">
          <h3 class="panel-title">{{ cardOneTitle }}</h3>
          <p class="panel-copy">{{ cardOneBody }}</p>
        </div>
        <div class="panel">
          <h3 class="panel-title">{{ cardTwoTitle }}</h3>
          <p class="panel-copy">{{ cardTwoBody }}</p>
        </div>
        <div class="panel">
          <h3 class="panel-title">{{ cardThreeTitle }}</h3>
          <p class="panel-copy">{{ cardThreeBody }}</p>
        </div>
      </div>

      <div class="body">
        <h2 class="section-heading">{{ primaryHeading }}</h2>
        <p class="body-copy" v-for="item in primaryItems">{{ item }}</p>

        <h2 class="section-heading">{{ secondaryHeading }}</h2>
        <p class="body-copy" v-for="item in secondaryItems">{{ item }}</p>

        <h2 class="section-heading">{{ commandHeading }}</h2>
        <p class="code-line" v-for="item in commandItems">{{ item }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Section signal.String `vugra:"section"`
    Title signal.String `vugra:"title"`
    Lead signal.String `vugra:"lead"`
    OverviewButtonClass signal.String `vugra:"overviewButtonClass"`
    ManualButtonClass signal.String `vugra:"manualButtonClass"`
    APIButtonClass signal.String `vugra:"apiButtonClass"`
    WasmButtonClass signal.String `vugra:"wasmButtonClass"`
    CardOneTitle signal.String `vugra:"cardOneTitle"`
    CardOneBody signal.String `vugra:"cardOneBody"`
    CardTwoTitle signal.String `vugra:"cardTwoTitle"`
    CardTwoBody signal.String `vugra:"cardTwoBody"`
    CardThreeTitle signal.String `vugra:"cardThreeTitle"`
    CardThreeBody signal.String `vugra:"cardThreeBody"`
    PrimaryHeading signal.String `vugra:"primaryHeading"`
    PrimaryItems signal.Signal[[]string] `vugra:"primaryItems"`
    SecondaryHeading signal.String `vugra:"secondaryHeading"`
    SecondaryItems signal.Signal[[]string] `vugra:"secondaryItems"`
    CommandHeading signal.String `vugra:"commandHeading"`
    CommandItems signal.Signal[[]string] `vugra:"commandItems"`
}

func (s *State) BeforeMount() {
    s.ShowOverview()
}

func (s *State) ShowOverview() {
    s.setActive("overview")
    s.Section.Set("Documentation Site")
    s.Title.Set("Build Vugra components with standard HTML templates and Go logic.")
    s.Lead.Set("This documentation site is implemented as a Vugra component, compiled to framework IR, and bundled for the wasm canvas host.")
    s.CardOneTitle.Set("Manual")
    s.CardOneBody.Set("SFC shape, template syntax, Go state, CSS profile, project config, and CLI workflow.")
    s.CardTwoTitle.Set("API")
    s.CardTwoBody.Set("Public facade packages, signals, system APIs, and host-level event bridges.")
    s.CardThreeTitle.Set("Wasm")
    s.CardThreeBody.Set("Browser bundles with index.html, app.wasm, wasm_exec.js, canvas rendering, and accessibility sync.")
    s.PrimaryHeading.Set("Current Status")
    s.PrimaryItems.Set([]string{
        "Available now: SFC parsing, HTML template parsing, Go metadata extraction, IR validation, Signal runtime updates, component imports, layout, event routing, native window support, software rendering, Vello native rendering, and wasm canvas bundles.",
        "Not complete yet: WebGPU wasm rendering, full browser DOM behavior, full CSS cascade, richer widgets, platform accessibility bridges, and production LSP integration.",
    })
    s.SecondaryHeading.Set("Design Boundary")
    s.SecondaryItems.Set([]string{
        "Vugra uses HTML syntax, but it is not a wrapper around browser DOM semantics.",
        "Components compile into Vugra IR first, then runtime, layout, and render layers decide how to present the same component on native or wasm targets.",
    })
    s.CommandHeading.Set("Quick Start")
    s.CommandItems.Set([]string{
        "go test ./...",
        "go run ./cmd/vugra check examples/counter/Counter.vue",
        "go run ./cmd/vugra wasm docs/site /tmp/vugra-docs-site",
        "go run ./cmd/vugra wasm-serve /tmp/vugra-docs-site",
    })
}

func (s *State) ShowManual() {
    s.setActive("manual")
    s.Section.Set("Manual")
    s.Title.Set("Authoring Vugra components")
    s.Lead.Set("A Vugra component is a .vue single-file component with standard HTML template syntax, real Go in script lang=go, and a deliberately small renderer-neutral CSS profile.")
    s.CardOneTitle.Set("SFC Format")
    s.CardOneBody.Set("Vugra recognizes template, script lang=go, and style blocks while preserving source ranges for diagnostics.")
    s.CardTwoTitle.Set("Template")
    s.CardTwoBody.Set("Supported tags and directives are deliberate: common tags, interpolation, v-if, v-for, bound props, and events.")
    s.CardThreeTitle.Set("Go State")
    s.CardThreeBody.Set("State is discovered from a State type, signal fields, vugra struct tags, and methods on *State.")
    s.PrimaryHeading.Set("Template Semantics")
    s.PrimaryItems.Set([]string{
        "Templates use HTML parsing rules where possible, but supported semantics are a renderer-neutral subset.",
        "Common tags include div, span, p, button, input, img, label, headings, lists, and inline or referenced SVG subsets.",
        "Full DOM APIs, contenteditable, iframe, media elements, and full form behavior are intentionally outside the current profile.",
    })
    s.SecondaryHeading.Set("Go And Reactivity")
    s.SecondaryItems.Set([]string{
        "Use explicit signal.Signal[T], signal.Int, signal.Bool, or signal.String.",
        "Direct field mutation is not the reactivity model. Signal updates are batched and scheduled through the runtime.",
        "Components can import other .vue files from Go script imports; imported components are analyzed and lowered into parent IR before code generation.",
    })
    s.CommandHeading.Set("CLI")
    s.CommandItems.Set([]string{
        "vugra check <file>",
        "vugra ir <file>",
        "vugra frame <file>",
        "vugra png <file> <out.png>",
        "vugra wasm <file-or-project> <out-dir>",
        "vugra wasm-run <file-or-project> [addr]",
        "vugra wasm-serve <bundle-dir> [addr]",
    })
}

func (s *State) ShowAPI() {
    s.setActive("api")
    s.Section.Set("API")
    s.Title.Set("Stable public surface")
    s.Lead.Set("Application code should target the public facade packages instead of internal compiler, runtime, layout, or renderer packages.")
    s.CardOneTitle.Set("pkg/vugra")
    s.CardOneBody.Set("Mount(component, state, target, options), renderer facades, layout constraints, events, text selection, and wasm host helpers.")
    s.CardTwoTitle.Set("pkg/signal")
    s.CardTwoBody.Set("signal.Signal[T], signal.Int, signal.Bool, and signal.String for explicit reactive state.")
    s.CardThreeTitle.Set("pkg/system")
    s.CardThreeBody.Set("FileSystem and WindowChrome boundaries, including native OS and wasm Web Storage backends.")
    s.PrimaryHeading.Set("Runtime Facade")
    s.PrimaryItems.Set([]string{
        "Mount(component, state, target, options) owns rendering, effects, focus, events, accessibility, and batched invalidation.",
        "TextSelection, TextSelectionFor, SetTextSelection, CollapseTextSelection, CollapseTextSelectionFor, SelectedText, SelectedTextFor, DocumentTextSelection, SelectedDocumentText, and ClearDocumentTextSelection expose renderer-neutral text selection.",
        "NewCanvasMeasurer, SyncAccessibility, InstallAccessibilityEvents, and InstallAccessibilityEventHandlers bridge wasm canvas and accessibility behavior.",
        "pkg/system uses a WebFileSystem backend on wasm so ReadDir, Stat, Mkdir, Rename, Remove, and Duplicate operate in a browser sandbox.",
        "Focus, click, key, and text actions share the same runtime event path across native and wasm hosts.",
    })
    s.SecondaryHeading.Set("Input And Accessibility")
    s.SecondaryItems.Set([]string{
        "The runtime tracks focused state, Tab and Shift+Tab focus traversal, IME composition commits, and plain text paste events.",
        "Text editing supports Backspace, Delete, ArrowLeft, ArrowRight, Home, End, Mod+A editing, public input selection queries, and drag selection over rendered document text.",
        "System APIs stay explicit and portable; simple path commands avoid exposing browser-only DOM behavior.",
    })
    s.CommandHeading.Set("Public CLI Surface")
    s.CommandItems.Set([]string{
        "wasm-run <file-or-project> [addr]",
        "wasm-serve <bundle-dir> [addr]",
        "native-window <file>",
        "run [config-or-dir]",
        "native-app <binary> <out.app> [args...]",
    })
}

func (s *State) ShowWasm() {
    s.setActive("wasm")
    s.Section.Set("Wasm")
    s.Title.Set("Browser bundles for Vugra components")
    s.Lead.Set("The wasm target packages a Vugra component into a browser canvas host with index.html, app.wasm, and Go's wasm_exec.js.")
    s.CardOneTitle.Set("Bundle")
    s.CardOneBody.Set("vugra wasm compiles a file or project directory and writes index.html, app.wasm, and wasm_exec.js.")
    s.CardTwoTitle.Set("Preview")
    s.CardTwoBody.Set("vugra wasm-run creates a temporary bundle and serves it locally with the correct application/wasm MIME type.")
    s.CardThreeTitle.Set("QA")
    s.CardThreeBody.Set("tools/wasm-smoke and tools/wasm-browser-check verify startup, pixels, events, focus, text, and accessibility.")
    s.PrimaryHeading.Set("Build And Serve")
    s.PrimaryItems.Set([]string{
        "Project inputs use app.title, app.width, app.height, runtime.layout, and config-relative entry resolution.",
        "The generated host page loads app.wasm with fetch and needs an HTTP server; file URLs are not enough.",
        "Canvas-backed wasm text measurement, devicePixelRatio scaling, and accessibility sync run through the Vugra wasm host.",
    })
    s.SecondaryHeading.Set("Browser QA Coverage")
    s.SecondaryItems.Set([]string{
        "Chrome/CDP browser QA covers pointer clicks, a11y clicks, a11y focus, keyboard activation, Tab focus traversal, text input, checkboxes, disabled controls, scroll, styled boxes, opacity, SVG, hover, drag, double click, and context menu.",
        "Native and wasm hosts share the same runtime key names and focus/click/key/text action bridges.",
        "Smoke checks include runtime caret movement, IME composition, caret/selection editing, paste, checkmarks, fills, strokes, rounded borders, clipping, arcs, and SVG paths.",
    })
    s.CommandHeading.Set("Commands")
    s.CommandItems.Set([]string{
        "go run ./cmd/vugra wasm-run examples/counter/Counter.vue",
        "go run ./cmd/vugra wasm examples/finder /tmp/vugra-finder-wasm",
        "go run ./cmd/vugra wasm-serve /tmp/vugra-counter-wasm",
        "node tools/wasm-browser-check/run.mjs /tmp/vugra-counter-wasm --click 30,60 --expect-text 1 --expect-a11y button,+",
        "node tools/wasm-browser-check/run.mjs /tmp/vugra-textinput-wasm --click 20,20 --text abc --key ArrowLeft",
        "node tools/wasm-smoke/run.mjs /tmp/vugra-focus-wasm --key Tab --key Tab --key Shift+Tab --expect-a11y-focused button,First",
    })
}

func (s *State) setActive(active string) {
    s.OverviewButtonClass.Set(navClass(active == "overview"))
    s.ManualButtonClass.Set(navClass(active == "manual"))
    s.APIButtonClass.Set(navClass(active == "api"))
    s.WasmButtonClass.Set(navClass(active == "wasm"))
}

func navClass(active bool) string {
    if active {
        return "nav-button nav-active"
    }
    return "nav-button"
}
</script>

<style>
.docs-shell {
  display: flex;
  flex-direction: row;
  width: 100%;
  height: 100%;
  background-color: #f7f8fb;
  color: #172033;
}

.sidebar {
  width: 280px;
  height: 100%;
  background-color: #263142;
  color: #eef4ff;
  padding: 26px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.brand {
  color: #ffffff;
  font-size: 34px;
  line-height: 40px;
  margin: 0;
}

.tagline {
  color: #c5d0df;
  font-size: 14px;
  line-height: 20px;
  margin: 0;
}

.nav-title {
  color: #99a9bd;
  font-size: 12px;
  line-height: 18px;
  margin: 14px;
}

.nav-button {
  width: 228px;
  height: 38px;
  background-color: #263142;
  color: #eef4ff;
  border: 1px solid #3d4a60;
  border-radius: 6px;
  font-size: 15px;
  text-align: left;
  padding-left: 12px;
}

.nav-active {
  background-color: #0f766e;
  border-color: #7dd3c7;
  color: #ffffff;
}

.reference {
  color: #c5d0df;
  font-size: 13px;
  line-height: 19px;
  margin: 0;
}

.content {
  flex: 1;
  height: 100%;
  padding: 42px;
  overflow: scroll;
}

.eyebrow {
  color: #0f766e;
  font-size: 13px;
  line-height: 18px;
  margin: 0;
}

.title {
  color: #172033;
  font-size: 38px;
  line-height: 46px;
  margin: 0;
}

.lead {
  color: #5c667a;
  font-size: 19px;
  line-height: 29px;
  margin: 0;
}

.hero-grid {
  display: flex;
  flex-direction: row;
  gap: 14px;
  margin: 24px;
}

.panel {
  width: 240px;
  min-height: 128px;
  background-color: #ffffff;
  border: 1px solid #d8deea;
  border-radius: 8px;
  padding: 16px;
}

.panel-title {
  color: #172033;
  font-size: 20px;
  line-height: 25px;
  margin: 0;
}

.panel-copy {
  color: #5c667a;
  font-size: 14px;
  line-height: 20px;
  margin: 0;
}

.body {
  width: 100%;
}

.section-heading {
  color: #172033;
  font-size: 26px;
  line-height: 34px;
  margin: 28px;
}

.body-copy {
  color: #2d3748;
  font-size: 16px;
  line-height: 24px;
  margin: 0;
}

.code-line {
  background-color: #101828;
  color: #e6edf7;
  border-radius: 6px;
  padding: 8px;
  font-size: 14px;
  line-height: 20px;
}
</style>
