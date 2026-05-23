# Vugra Project Config

Vugra projects are described by `vugra.config.json` at the project root. The
config is intentionally renderer-neutral at the component layer: project
configuration can select a host/runtime and request system GUI capabilities,
while `.vue` components consume the resulting layout tokens through `env(...)`.

## Layout

```text
my-app/
  vugra.config.json
  src/
    App.vue
    components/
      Toolbar.vue
```

`entry` is resolved relative to the config file directory.

## Minimal Config

```json
{
  "$schema": "https://vugra.dev/schemas/vugra.config.schema.json",
  "name": "my-app",
  "entry": "src/App.vue"
}
```

## Full Example

```json
{
  "$schema": "https://vugra.dev/schemas/vugra.config.schema.json",
  "name": "finder-lite",
  "version": "0.1.0",
  "entry": "FinderLite.vue",
  "app": {
    "title": "Finder Lite",
    "width": 800,
    "height": 600
  },
  "runtime": {
    "renderer": "vello-native",
    "layout": "go"
  },
  "scripts": {
    "onLaunch": [
      {
        "use": "window.setChrome",
        "titlebar": "hidden",
        "controls": {
          "x": 18,
          "y": 11,
          "width": 72,
          "height": 28
        }
      }
    ]
  }
}
```

## Schema

The JSON Schema lives at:

```text
docs/vugra.config.schema.json
```

The current schema covers:

- `name`: project name.
- `version`: optional project version.
- `entry`: project-relative `.vue` entry component.
- `app.title`: native window title.
- `app.width` / `app.height`: logical viewport size.
- `runtime.renderer`: `vello-native`, `vello`, or `software`.
- `runtime.layout`: `go` or `css`.
- `scripts.onLaunch`: declarative host/system actions.
- `system.gui.onLaunch`: legacy alias for declarative system GUI actions.

## Project Scripts

Project scripts are declarative host/system actions. They are not shell scripts
and they do not run arbitrary code from config. Hosts decide which actions they
support. This keeps the config suitable for native and wasm hosts without
turning project startup into unbounded command execution.

The first supported action is:

```json
{
  "use": "window.setChrome",
  "titlebar": "hidden",
  "controls": {
    "x": 18,
    "y": 11,
    "width": 72,
    "height": 28
  }
}
```

On macOS native-window this hides the titlebar, keeps the native traffic-light
buttons, and positions their group at the requested logical coordinates.

The host exposes the resulting area through system tokens:

```text
vugra-window-controls-x
vugra-window-controls-y
vugra-window-controls-width
vugra-window-controls-height
vugra-window-controls-left
vugra-window-controls-top
```

Components should avoid native chrome through CSS:

```css
.toolbar {
  padding-left: env(vugra-window-controls-left, 10px);
}
```

## CLI

Run a project from its config file or directory:

```sh
go run ./cmd/vugra run examples/finder
go run ./cmd/vugra run examples/finder/vugra.config.json
```

When no argument is passed, `vugra run` reads `./vugra.config.json`.

On macOS with cgo, `vugra run` auto-enables the native window backend when
invoked through `go run ./cmd/vugra`.
