# Vuego LSP Reference

## Direction

Use Volar.js as the embedded-language framework for `.vuego` files. Use `gopls` for Go semantics.

Volar should provide:

- `.vuego` file recognition.
- SFC block parsing.
- Virtual documents.
- Source maps.
- LSP request routing and mapping.

`gopls` should provide:

- Go diagnostics.
- Completion.
- Hover.
- Definition.
- Rename where practical.

## Virtual Documents

Recommended split:

```text
Component.vuego
  -> Component.template.html
  -> Component.script.go
  -> Component.style.css
  -> Component.meta.json
```

The generated Go virtual document should be valid Go, including package declarations and synthetic imports when needed:

```go
package components

import "example.com/vuego/signal"

type State struct {
    Count signal.Int `vuego:"count"`
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}
```

Every generated prefix, suffix, and wrapper must have source-map behavior defined. Diagnostics in synthetic regions should either be suppressed or remapped to the closest user-owned source.

## Template Intelligence

Template features should be driven by compiler metadata rather than TypeScript inference.

Metadata should include:

- State fields and template aliases.
- Signal value types.
- Methods and event-compatible signatures.
- Props.
- Emits.
- Imported or locally declared components.
- Supported intrinsic tags, props, and events.

Template diagnostics:

- Unknown tag.
- Unknown prop.
- Unknown event.
- Unknown state binding.
- Unknown method binding.
- Invalid `v-for` shape.
- Invalid `v-if` expression.
- Unsupported browser-only feature.

## Expression Strategy

Do not use JavaScript semantics for template expressions unless explicitly building a separate JS compatibility layer.

Preferred options:

- Small Go-oriented expression grammar.
- Alias-only expressions for MVP.
- Compiler-mapped shorthand such as `{{ count }}` -> `state.Count.Get()`.

Keep expression rules simple enough for both compiler and LSP to implement consistently.

## Volar Integration Notes

Implement a Volar language plugin that:

- Recognizes `.vuego`.
- Parses SFC blocks incrementally where possible.
- Creates embedded virtual code for template, Go, style, and metadata.
- Maintains mappings from virtual files to `.vuego`.
- Provides custom services for template completions and diagnostics.

Do not rely on Volar's TypeScript service for Go language semantics. Use a `gopls` bridge or a companion language server process for the virtual Go document.
