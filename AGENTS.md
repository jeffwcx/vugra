# Vugra Agent Guidelines

This repository is for Vugra: a Vue-like single-file component system with standard HTML template syntax, Go component logic, Signal-based reactivity, and wasm/native rendering targets.

## Product Boundary

- Treat Vugra as a new cross-platform UI framework, not as a fork of Vue or a wrapper around Vue runtime.
- Template syntax should stay close to standard HTML. Vue-like directives may be added where they compile cleanly to Vugra IR.
- `<script lang="go">` is Go source, not JavaScript with Go-like syntax.
- The compiler should produce framework-owned IR before targeting renderers, LSP metadata, or generated code.
- Native and wasm targets are first-class. Do not design APIs that only work with browser DOM semantics.

## Architecture

Preferred pipeline:

```text
.vue source
  -> SFC parser
  -> HTML template AST + Go script AST/metadata + style AST
  -> Vugra component IR
  -> runtime state/effects/layout tree
  -> renderer backend
```

Keep these layers separate:

- Parsing: read `.vue` files and preserve source ranges.
- Analysis: resolve component state, methods, props, events, and template bindings.
- Reactivity: use Go Signals and explicit effects; do not emulate JavaScript Proxy.
- Layout: produce platform-neutral layout boxes before painting.
- Rendering: convert layout/render commands to Vello, wgpu, WebGPU, or future backends.
- Tooling: expose virtual files and metadata to LSP without coupling editor support to the renderer.

## Template Rules

- Use standard HTML parsing rules where possible.
- Supported semantics are a deliberate subset of browser behavior.
- Document clearly that Vugra templates use HTML syntax, not full browser DOM semantics.
- Start with common tags such as `div`, `span`, `p`, `button`, `input`, `img`, `label`, headings, and lists.
- Keep Vue-like directives minimal at first: `{{ expr }}`, `v-if`, `v-for`, `:prop`, `@event`.
- Avoid promising full CSS cascade, table layout, DOM APIs, contenteditable, iframe, media elements, or full form behavior in early stages.

## Go And Reactivity

- Prefer explicit `signal.Signal[T]` or concrete signal types over implicit field mutation.
- First-class state should be discoverable from Go types and tags.
- Template bindings should map to Go state/method metadata.
- Do not rely on source rewriting such as `count++` until the explicit Signal API is stable.
- Effects should batch invalidations and schedule work through the runtime.

## Renderer Targets

- Vello and wgpu are acceptable renderer backends, but do not let backend details leak into component syntax.
- Vello is a 2D rendering backend, not a complete UI runtime. The project still needs layout, text, input, focus, events, accessibility, and scheduling layers.
- Use WebGPU-aware wasm design, but preserve a path for native backends.

## LSP

- Prefer Volar.js as the embedded-language framework for `.vue` tooling.
- Use virtual documents for template, Go script, style, and compiler metadata.
- Bridge `<script lang="go">` to `gopls`; do not expect Volar's TypeScript service to understand Go.
- Diagnostics and completions must map back to original `.vue` source ranges.
- Template intelligence should be driven by compiler metadata from Go state/method analysis.

## Engineering Practices

- Keep MVPs narrow and executable.
- Favor source maps and stable IR over ad hoc string transformations.
- Add tests around parser ranges, IR generation, diagnostics, and backend-independent behavior.
- Avoid broad refactors unless they directly protect the compiler/runtime boundary.
- When uncertain, preserve native/wasm parity over browser-only convenience.
