# Vuego Architecture Reference

## Goal

Vuego combines familiar Vue-style single-file component ergonomics with Go component logic and a cross-platform renderer stack.

The stable mental model:

```text
.vuego
  -> parser
  -> template AST + Go metadata + style metadata
  -> component IR
  -> runtime tree
  -> layout tree
  -> render commands
  -> backend
```

## SFC Shape

```vue
<template>
  <div class="counter">
    <p>{{ count }}</p>
    <button @click="Inc">+</button>
  </div>
</template>

<script lang="go">
type State struct {
    Count signal.Int `vuego:"count"`
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}
</script>
```

## MVP Scope

Start with:

- SFC block parser with source ranges.
- HTML template parser.
- Tags: `div`, `span`, `p`, `button`, `input`, `img`, `label`, `h1`-`h6`, `ul`, `ol`, `li`.
- Directives: interpolation, `v-if`, `v-for`, `:prop`, `@event`.
- Go state metadata from `State` structs and `vuego` tags.
- Go method metadata from exported methods on `*State`.
- Signal primitives: `Signal[T]`, `Get`, `Set`, `Update`, `Subscribe`, batched effects.
- Backend-neutral layout/render IR.
- One native demo target before broadening the platform matrix.

Defer:

- Full CSS cascade.
- Full DOM event compatibility.
- Browser layout exactness.
- Table layout.
- Rich text editing and contenteditable.
- Complex forms, validation, and browser submission semantics.
- Slots, provide/inject, transitions, suspense, SSR hydration.
- Source rewriting for implicit Signal operations.

## Template Semantics

Use standard HTML syntax to reduce parser/tooling friction. Vuego semantics are narrower than browser semantics:

- `class` maps to style lookup.
- `style` may be rejected or limited early.
- Events map to Vuego event names.
- Boolean attributes should normalize into typed props.
- Unknown tags should either resolve as components or produce diagnostics.

Avoid letting users infer that browser APIs such as `document`, DOM nodes, or CSSOM exist in native targets.

## Reactivity

Signals are the first reactivity contract:

```go
type Signal[T any] interface {
    Get() T
    Set(T)
    Update(func(T) T)
    Subscribe(func()) func()
}
```

Rendering should run inside an effect. `Get()` records dependencies for the active effect. `Set()` and `Update()` mark effects dirty and schedule a batched update.

Do not implement JavaScript Proxy-like deep mutation for MVP. Collections should use explicit replacement or collection-specific Signal helpers later.

## Renderer Boundary

Vello and wgpu belong below layout/render IR:

```text
component tree
  -> layout tree
  -> display list / render commands
  -> Vello scene or wgpu commands
```

The runtime must own:

- Event loop integration.
- Focus and pointer routing.
- Keyboard and IME handling.
- Text measurement and shaping integration.
- Accessibility tree generation.
- Frame scheduling.

Vello can paint 2D output, but it should not define the component model.
