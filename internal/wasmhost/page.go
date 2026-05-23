package wasmhost

import (
	"fmt"
	"html"
)

type PageInput struct {
	WasmPath string
	Title    string
	CanvasID string
	Width    int
	Height   int
}

func GenerateHostPage(input PageInput) string {
	wasmPath := input.WasmPath
	if wasmPath == "" {
		wasmPath = "app.wasm"
	}
	title := input.Title
	if title == "" {
		title = "Vugra"
	}
	canvasID := input.CanvasID
	if canvasID == "" {
		canvasID = "vugra-canvas"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    html, body { margin: 0; width: 100%%; height: 100%%; overflow: hidden; background: #0f172a; font-family: system-ui, sans-serif; }
    body { min-height: 100%%; }
    canvas { display: block; width: 100vw; height: 100vh; background: #ffffff; outline: none; }
    #status {
      position: fixed;
      top: 12px;
      right: 12px;
      z-index: 10;
      max-width: min(420px, calc(100vw - 24px));
      padding: 6px 10px;
      border: 1px solid rgba(148, 163, 184, 0.45);
      border-radius: 6px;
      background: rgba(15, 23, 42, 0.72);
      color: #e2e8f0;
      font-size: 12px;
      line-height: 18px;
      pointer-events: none;
    }
    #vugra-a11y { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%%); white-space: nowrap; }
  </style>
</head>
<body>
  <div id="status">Loading Vugra wasm...</div>
  <canvas id="%s" tabindex="0"></canvas>
  <div id="vugra-a11y" aria-label="Vugra accessibility tree"></div>
  <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    async function instantiateVugraWasm() {
      const response = await fetch(%q);
      if (WebAssembly.instantiateStreaming && response.headers.get("Content-Type") === "application/wasm") {
        try {
          return await WebAssembly.instantiateStreaming(Promise.resolve(response.clone()), go.importObject);
        } catch (error) {
          console.warn("instantiateStreaming failed; falling back to arrayBuffer", error);
        }
      }
      const bytes = await response.arrayBuffer();
      return WebAssembly.instantiate(bytes, go.importObject);
    }
    const instantiate = instantiateVugraWasm();
    instantiate.then((result) => {
        document.getElementById("status").textContent = "Running";
        go.run(result.instance);
      })
      .catch((error) => {
        document.getElementById("status").textContent = error.message;
        console.error(error);
      });
  </script>
</body>
</html>
`, escapeHTML(title), escapeHTML(canvasID), wasmPath)
}

func escapeHTML(value string) string {
	return html.EscapeString(value)
}
