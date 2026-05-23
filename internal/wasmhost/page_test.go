package wasmhost_test

import (
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/wasmhost"
)

func TestGenerateHostPage(t *testing.T) {
	page := wasmhost.GenerateHostPage(wasmhost.PageInput{
		WasmPath: "counter.wasm",
		Title:    "Counter",
		CanvasID: "counter-canvas",
		Width:    320,
		Height:   200,
	})
	for _, want := range []string{
		`canvas { display: block; width: 100vw; height: 100vh;`,
		`#status {`,
		`position: fixed;`,
		`pointer-events: none;`,
		`<canvas id="counter-canvas" tabindex="0"></canvas>`,
		"fetch(\"counter.wasm\")",
		"response.headers.get(\"Content-Type\") === \"application/wasm\"",
		"response.arrayBuffer()",
		"wasm_exec.js",
		`id="vugra-a11y"`,
		"Vugra accessibility tree",
		"<title>Counter</title>",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
	for _, blocked := range []string{
		`style="width: 320px; height: 200px;"`,
		`padding: 24px`,
		`border: 1px solid #94a3b8`,
	} {
		if strings.Contains(page, blocked) {
			t.Fatalf("page contains fixed-shell fragment %q:\n%s", blocked, page)
		}
	}
}

func TestGenerateHostPageEscapesProjectStrings(t *testing.T) {
	page := wasmhost.GenerateHostPage(wasmhost.PageInput{
		WasmPath: `dir/app "dev".wasm`,
		Title:    `A <B> & "C"`,
		CanvasID: `canvas" onclick="bad`,
	})
	for _, want := range []string{
		"<title>A &lt;B&gt; &amp; &#34;C&#34;</title>",
		`<canvas id="canvas&#34; onclick=&#34;bad"`,
		`fetch("dir/app \"dev\".wasm")`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing escaped %q:\n%s", want, page)
		}
	}
	for _, blocked := range []string{
		"<title>A <B>",
		`<canvas id="canvas" onclick="bad"`,
		`fetch("dir/app "dev".wasm")`,
	} {
		if strings.Contains(page, blocked) {
			t.Fatalf("page contains unescaped %q:\n%s", blocked, page)
		}
	}
}
