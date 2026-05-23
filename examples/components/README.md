# Component Import Demo

This example exercises Vugra's current component model.

Supported component syntax:

- Go script imports of relative `.vue` files:
  `import Badge "./Badge.vue"`
- PascalCase component tags resolved from those imports.
- Kebab-case component tags resolved back to PascalCase imports, such as `<plain-card>` for `PlainCard`.
- Static and bound props, with prop metadata derived from child `State` fields.
- Required prop diagnostics for explicitly tagged child state fields.
- Optional/default props through tags such as `vugra:"label,optional,default=Fallback"`.
- Fallthrough root attributes such as `class`, `id`, `style`, `data-*`, and `aria-*`.
- Default slots with `<slot />`.
- Named slots with `<template #name>`, `v-slot:name`, and `slot="name"`.
- Scoped slots with `<template #item="item">` and component-level `v-slot="item"`.
- Component `v-model` and `v-model:name` lowered to prop plus `update:name` listener metadata.
- Component emits declared in child Go methods with `vugra.Emit("event")`, then listened to by the parent with `@event`.
- Event listener methods can accept `vugra.Event` and are generated into runtime `EventMethods`.
- Dynamic `<component :is="current">` selection across resolved imported components. The binding value must match an import alias.
- Basic provide/inject metadata with `vugra:"key,provide"` and `vugra:"key,inject"`, expanded as implicit component props during compilation.
- Root lifecycle hooks for `BeforeMount`, `Mounted`, `BeforeUpdate`, `Updated`, `BeforeUnmount`, and `Unmounted`.
- Component lifecycle listeners such as `<Child @mounted="OnChildMounted">` on expanded component instances.
- Runtime component instances can own independent child state when the `Component` carries a `NewState` factory. Props are synced into child signals before layout, and child events resolve against child methods before declared emits call parent listeners.
- `vugra gen` emits renamed child state types, per-child runtime state factories, and attaches those factories to imported component IR.

Not supported yet:

- Async components. `async`, `v-async`, and `:async` on component tags produce diagnostics instead of falling through silently.
- Full Vue-compatible component semantics. Vugra currently implements the native/wasm-safe subset above; browser-DOM-only behavior and async component loading are intentionally outside this pass.

Run:

```sh
go run ./cmd/vugra check examples/components/Parent.vue
go run ./cmd/vugra frame examples/components/Parent.vue
```
