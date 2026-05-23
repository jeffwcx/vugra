#!/bin/sh
set -eu

site_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$site_dir/../.." && pwd)

for file in App.vue vugra.config.json index.html manual.html api.html wasm.html assets/site.css; do
  if [ ! -s "$site_dir/$file" ]; then
    echo "missing or empty docs site file: $file" >&2
    exit 1
  fi
done

grep -q '"name": "vugra-docs-site"' "$site_dir/vugra.config.json"
grep -q '"entry": "App.vue"' "$site_dir/vugra.config.json"
grep -q '"title": "Vugra Documentation"' "$site_dir/vugra.config.json"

grep -q '<template>' "$site_dir/App.vue"
grep -q '<script lang="go">' "$site_dir/App.vue"
grep -q '<style>' "$site_dir/App.vue"
grep -q 'type State struct' "$site_dir/App.vue"
grep -q 'func (s \*State) ShowOverview' "$site_dir/App.vue"
grep -q 'func (s \*State) ShowManual' "$site_dir/App.vue"
grep -q 'func (s \*State) ShowAPI' "$site_dir/App.vue"
grep -q 'func (s \*State) ShowWasm' "$site_dir/App.vue"
grep -q 'Build Vugra components with standard HTML templates and Go logic' "$site_dir/App.vue"
grep -q 'Authoring Vugra components' "$site_dir/App.vue"
grep -q 'Stable public surface' "$site_dir/App.vue"
grep -q 'Browser bundles for Vugra components' "$site_dir/App.vue"
grep -q 'vugra wasm-run <file-or-project> \[addr\]' "$site_dir/App.vue"
grep -q 'tools/wasm-browser-check/run.mjs' "$site_dir/App.vue"

for page in index.html manual.html api.html wasm.html; do
  if ! grep -q 'href="assets/site.css"' "$site_dir/$page"; then
    echo "$page does not load shared CSS" >&2
    exit 1
  fi
done

(cd "$repo_root" && go run ./cmd/vugra check docs/site/App.vue) >/dev/null
(cd "$repo_root" && go run ./cmd/vugra ir docs/site/App.vue | grep -q '"hook": "beforeMount"')

out_dir=$(mktemp -d "${TMPDIR:-/tmp}/vugra-docs-site-verify-XXXXXX")
trap 'rm -rf "$out_dir"' EXIT
(cd "$repo_root" && go run ./cmd/vugra wasm docs/site "$out_dir") >/dev/null

for file in index.html app.wasm wasm_exec.js; do
  if [ ! -s "$out_dir/$file" ]; then
    echo "generated bundle missing or empty: $file" >&2
    exit 1
  fi
done

grep -q '<title>Vugra Documentation</title>' "$out_dir/index.html"
grep -q '<canvas id="vugra-canvas" tabindex="0"></canvas>' "$out_dir/index.html"
grep -q 'width: 100vw; height: 100vh;' "$out_dir/index.html"
grep -q 'fetch("app.wasm")' "$out_dir/index.html"

echo "docs site ok"
