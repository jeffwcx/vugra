package rustcodegen_test

import (
	"strings"
	"testing"

	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/rustcodegen"
)

func TestGenerateStateAdapterFromRustSFC(t *testing.T) {
	result := compiler.Compile("Counter.vue", []byte(`
<template>
  <button @click="Inc">{{ count }}</button>
</template>
<script lang="rust">
pub struct State {
    pub count: Signal<i32>,
}

impl State {
    pub fn Inc(&mut self) {}
}
</script>
`))
	generated := rustcodegen.GenerateStateAdapter(result, "CounterAdapter")
	if len(generated.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", generated.Diagnostics)
	}
	if len(generated.Signals) != 1 || generated.Signals[0].Name != "count" || generated.Signals[0].ID != 1 {
		t.Fatalf("signals = %+v", generated.Signals)
	}
	if len(generated.Methods) != 1 || generated.Methods[0].Name != "Inc" || generated.Methods[0].ID != 1 {
		t.Fatalf("methods = %+v", generated.Methods)
	}
	for _, want := range []string{
		"pub fn generated_component_contract() -> vugra_ir::Component",
		"vugra_ir::Component::new(\"Counter\")",
		"id: vugra_ir::SignalId(1)",
		"name: \"count\".to_string()",
		"id: vugra_ir::MethodId(1)",
		"pub struct CounterAdapter<T>",
		"impl<T> vugra_core::ComponentState for CounterAdapter<T>",
		"1 => self.inner.count()",
		"1 => self.inner.set_count(value)",
		"1 => self.inner.Inc()",
		"pub trait RustSFCBindings",
	} {
		if !strings.Contains(generated.Source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, generated.Source)
		}
	}
}

func TestGenerateStateAdapterRejectsNonRustSFC(t *testing.T) {
	result := compiler.Compile("Counter.vue", []byte(`
<template><div></div></template>
<script lang="go">type State struct{}</script>
`))
	generated := rustcodegen.GenerateStateAdapter(result, "CounterAdapter")
	if len(generated.Diagnostics) == 0 || generated.Diagnostics[0].Code != "rustcodegen.not_rust_sfc" {
		t.Fatalf("diagnostics = %+v", generated.Diagnostics)
	}
}

func TestGenerateFinderLiteStateAdapterUsesFinderContractIDs(t *testing.T) {
	result := compiler.Compile("FinderLite.vue", []byte(`
<template>
  <button @click="OpenDocuments">{{ path }}</button>
  <p>{{ row1Name }}</p>
</template>
<script lang="rust">
pub struct State {
    pub path: String,
    pub status: String,
    pub selectedSummary: String,
    pub row1Name: String,
    pub row1Kind: String,
    pub row2Name: String,
    pub row2Kind: String,
    pub row3Name: String,
    pub row3Kind: String,
    pub row1Selected: bool,
    pub row2Selected: bool,
    pub row3Selected: bool,
    pub documentsLabel: String,
    pub downloadsLabel: String,
    pub picturesLabel: String,
    pub documentsActive: bool,
    pub downloadsActive: bool,
    pub picturesActive: bool,
    pub searchQuery: String,
    pub itemMenuOpen: bool,
    pub blankMenuOpen: bool,
    pub renameText: String,
    pub previewOpen: bool,
    pub previewTitle: String,
    pub previewBody: String,
    pub sidebarClass: String,
    pub splitterClass: String,
}

impl State {
    pub fn OpenDocuments(&mut self) {}
    pub fn SelectRow1(&mut self) {}
    pub fn OpenRow1(&mut self) {}
    pub fn SearchInput(&mut self, event: Event) {}
    pub fn BeginRename(&mut self) {}
    pub fn DeleteSelected(&mut self) {}
    pub fn ClearSelection(&mut self) {}
    pub fn Paste(&mut self) {}
    pub fn Refresh(&mut self) {}
    pub fn SelectAll(&mut self) {}
    pub fn HoverSplitter(&mut self) {}
    pub fn ResizeSidebar(&mut self, event: Event) {}
}
</script>
`))
	generated := rustcodegen.GenerateStateAdapterWithOptions(result, rustcodegen.Options{
		AdapterName: "FinderLiteSFCAdapter",
		Contract:    rustcodegen.ContractFinderLite,
	})
	if len(generated.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", generated.Diagnostics)
	}
	if len(generated.Signals) != 98 || generated.Signals[0].Name != "path" || generated.Signals[9].Name != "searchQuery" || !hasBinding(generated.Signals, 100, "itemMenuOpen") || !hasBinding(generated.Signals, 91, "row12Selected") || !hasBinding(generated.Signals, 106, "sidebarClass") || !hasBinding(generated.Signals, 107, "splitterClass") {
		t.Fatalf("signals = %+v", generated.Signals)
	}
	if len(generated.Methods) != 79 || generated.Methods[0].Name != "Back" || generated.Methods[1].Name != "SelectRow1" || generated.Methods[4].Name != "OpenDocuments" || !hasBinding(generated.Methods, 20, "Forward") || !hasBinding(generated.Methods, 32, "SelectRow12") || !hasBinding(generated.Methods, 40, "ClosePreview") || !hasBinding(generated.Methods, 41, "ShowRow1Menu") || !hasBinding(generated.Methods, 52, "ShowRow12Menu") || !hasBinding(generated.Methods, 53, "HoverRow1") || !hasBinding(generated.Methods, 64, "HoverRow12") || !hasBinding(generated.Methods, 65, "OpenRow1") || !hasBinding(generated.Methods, 76, "OpenRow12") || !hasBinding(generated.Methods, 77, "ClearSelection") || !hasBinding(generated.Methods, 78, "Paste") || !hasBinding(generated.Methods, 79, "Refresh") || !hasBinding(generated.Methods, 80, "SelectAll") || !hasBinding(generated.Methods, 81, "HoverSplitter") || !hasBinding(generated.Methods, 82, "ResizeSidebar") {
		t.Fatalf("methods = %+v", generated.Methods)
	}
	for _, want := range []string{
		"vugra_ir::finder_lite_contract()",
		"1 => self.inner.path()",
		"19 => self.inner.searchQuery()",
		"92 => self.inner.favoritesLabel()",
		"99 => self.inner.projectBActive()",
		"100 => self.inner.itemMenuOpen()",
		"105 => self.inner.previewBody()",
		"106 => self.inner.sidebarClass()",
		"107 => self.inner.splitterClass()",
		"20 => self.inner.row1Name()",
		"31 => self.inner.row2Selected()",
		"91 => self.inner.row12Selected()",
		"1 => self.inner.Back()",
		"2 => self.inner.SelectRow1()",
		"5 => self.inner.OpenDocuments()",
		"10 => self.inner.SearchInput(event)",
		"15 => self.inner.ToggleFavorites()",
		"18 => self.inner.OpenProjectB()",
		"20 => self.inner.Forward()",
		"33 => self.inner.BeginRename()",
		"36 => self.inner.DeleteSelected()",
		"40 => self.inner.ClosePreview()",
		"32 => self.inner.SelectRow12()",
		"41 => self.inner.ShowRow1Menu()",
		"52 => self.inner.ShowRow12Menu()",
		"53 => self.inner.HoverRow1()",
		"64 => self.inner.HoverRow12()",
		"65 => self.inner.OpenRow1()",
		"76 => self.inner.OpenRow12()",
		"77 => self.inner.ClearSelection()",
		"78 => self.inner.Paste()",
		"79 => self.inner.Refresh()",
		"80 => self.inner.SelectAll()",
		"81 => self.inner.HoverSplitter()",
		"82 => self.inner.ResizeSidebar(event)",
		"fn SearchInput(&mut self, event: vugra_core::Event);",
		"fn ResizeSidebar(&mut self, event: vugra_core::Event);",
	} {
		if !strings.Contains(generated.Source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, generated.Source)
		}
	}
}

func hasBinding(bindings []rustcodegen.Binding, id int, name string) bool {
	for _, binding := range bindings {
		if binding.ID == id && binding.Name == name {
			return true
		}
	}
	return false
}
