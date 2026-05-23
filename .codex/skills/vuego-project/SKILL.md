---
name: vuego-project
description: Use when designing, implementing, reviewing, or planning Vuego, a Vue-like SFC framework with standard HTML templates, Go script blocks, Signal-based reactivity, Volar.js/gopls tooling, and wasm/native renderers such as Vello or wgpu.
---

# Vuego Project

Use this skill for Vuego architecture, compiler, runtime, renderer, component model, `.vuego` file format, and LSP work.

## Core Stance

- Vuego is a new cross-platform UI framework, not a Vue fork.
- Template syntax is standard HTML plus a small Vue-like directive subset.
- `<script lang="go">` contains real Go code.
- Go Signals are the first reactivity model. Do not emulate JavaScript Proxy.
- Renderer backends are replaceable; component syntax must not depend on DOM, Vello, or wgpu internals.
- LSP support should use Volar.js for embedded-language mapping and `gopls` for Go semantics.

## Default Workflow

1. Read `AGENTS.md` first if it exists in the repository root.
2. Identify which layer the task touches: SFC parsing, template analysis, Go analysis, IR, reactivity, layout, renderer, wasm/native build, or LSP.
3. Keep edits within that layer unless the user asks for integration work.
4. Preserve source ranges whenever parsing or generating diagnostics.
5. Add focused tests for compiler/runtime behavior before adding broader examples.

## Design Defaults

- Compile `.vuego` source into Vuego-owned component IR before generating backend code.
- Treat HTML as syntax and a tag vocabulary, not as full browser DOM behavior.
- Template expressions should be Go-oriented or mapped explicitly to Go metadata.
- State and methods should be discoverable from Go types, tags, and exported method sets.
- Runtime invalidation should be explicit and batched through Signals/effects.
- Layout should produce backend-neutral boxes before painting.

## When More Detail Is Needed

- For architecture boundaries and MVP sequencing, read `references/architecture.md`.
- For editor support and virtual-file design, read `references/lsp.md`.

## Review Checklist

- Does the change keep native and wasm targets viable?
- Does it avoid coupling compiler IR to a specific renderer?
- Does it avoid depending on full DOM/CSS/browser behavior?
- Does Go code remain valid Go that `gopls` can understand?
- Are diagnostics mapped to original `.vuego` source locations?
- Are Signal updates explicit, batched, and testable?
