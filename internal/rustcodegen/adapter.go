package rustcodegen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vugra/vugra/internal/compiler"
)

type AdapterSource struct {
	Source      string
	Signals     []Binding
	Methods     []Binding
	Diagnostics []Diagnostic
}

type Binding struct {
	ID    int
	Name  string
	Rust  string
	Event bool
}

type Diagnostic struct {
	Code    string
	Message string
}

type Options struct {
	AdapterName string
	Contract    Contract
}

type Contract string

const (
	ContractAuto       Contract = ""
	ContractFinderLite Contract = "finder-lite"
)

func GenerateStateAdapter(result *compiler.Result, adapterName string) AdapterSource {
	return GenerateStateAdapterWithOptions(result, Options{AdapterName: adapterName})
}

func GenerateStateAdapterWithOptions(result *compiler.Result, options Options) AdapterSource {
	var out AdapterSource
	adapterName := options.AdapterName
	if adapterName == "" {
		adapterName = "GeneratedStateAdapter"
	}
	if result == nil || result.SFC == nil || result.SFC.Script == nil || result.SFC.Script.Lang != "rust" {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{
			Code:    "rustcodegen.not_rust_sfc",
			Message: "component is not a <script lang=\"rust\"> SFC",
		})
		return out
	}
	if result.Rust.State == nil {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{
			Code:    "rustcodegen.missing_state",
			Message: "missing Rust State metadata",
		})
		return out
	}
	if options.Contract == ContractFinderLite {
		out.Signals = finderLiteSignalBindings(result)
		out.Methods = finderLiteMethodBindings(result)
	} else {
		for index, field := range result.Rust.State.Fields {
			out.Signals = append(out.Signals, Binding{
				ID:   index + 1,
				Name: field.Alias,
				Rust: field.Name,
			})
		}
		for index, method := range exportedMethods(result) {
			out.Methods = append(out.Methods, Binding{
				ID:    index + 1,
				Name:  method.name,
				Rust:  method.name,
				Event: method.eventArg,
			})
		}
	}
	out.Source = renderAdapterSource(result, adapterName, out.Signals, out.Methods, options.Contract)
	return out
}

func renderAdapterSource(result *compiler.Result, adapterName string, signals, methods []Binding, contract Contract) string {
	var b strings.Builder
	fmt.Fprintln(&b, "pub fn generated_component_contract() -> vugra_ir::Component {")
	if contract == ContractFinderLite {
		fmt.Fprintln(&b, "    vugra_ir::finder_lite_contract()")
	} else {
		fmt.Fprintf(&b, "    let mut component = vugra_ir::Component::new(%q);\n", componentName(result))
		fmt.Fprintln(&b, "    component.signals = vec![")
		for _, signal := range signals {
			fmt.Fprintln(&b, "        vugra_ir::SignalDef {")
			fmt.Fprintf(&b, "            id: vugra_ir::SignalId(%d),\n", signal.ID)
			fmt.Fprintf(&b, "            name: %q.to_string(),\n", signal.Name)
			fmt.Fprintln(&b, "            kind: vugra_ir::ValueKind::String,")
			fmt.Fprintln(&b, "        },")
		}
		fmt.Fprintln(&b, "    ];")
		fmt.Fprintln(&b, "    component.methods = vec![")
		for _, method := range methods {
			fmt.Fprintln(&b, "        vugra_ir::MethodDef {")
			fmt.Fprintf(&b, "            id: vugra_ir::MethodId(%d),\n", method.ID)
			fmt.Fprintf(&b, "            name: %q.to_string(),\n", method.Name)
			fmt.Fprintln(&b, "        },")
		}
		fmt.Fprintln(&b, "    ];")
		fmt.Fprintln(&b, "    component")
	}
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "pub struct %s<T> {\n", adapterName)
	fmt.Fprintln(&b, "    inner: T,")
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "impl<T> %s<T> {\n", adapterName)
	fmt.Fprintln(&b, "    pub fn new(inner: T) -> Self {")
	fmt.Fprintln(&b, "        Self { inner }")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "    pub fn inner(&self) -> &T {")
	fmt.Fprintln(&b, "        &self.inner")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "    pub fn inner_mut(&mut self) -> &mut T {")
	fmt.Fprintln(&b, "        &mut self.inner")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "impl<T> vugra_core::ComponentState for %s<T>\n", adapterName)
	fmt.Fprintln(&b, "where")
	fmt.Fprintln(&b, "    T: RustSFCBindings,")
	fmt.Fprintln(&b, "{")
	fmt.Fprintln(&b, "    fn get_signal(&self, id: vugra_core::SignalId) -> vugra_core::Value {")
	fmt.Fprintln(&b, "        match id.0 {")
	for _, signal := range signals {
		fmt.Fprintf(&b, "            %d => self.inner.%s(),\n", signal.ID, signal.Rust)
	}
	fmt.Fprintln(&b, "            _ => vugra_core::Value::None,")
	fmt.Fprintln(&b, "        }")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "    fn set_signal(&mut self, id: vugra_core::SignalId, value: vugra_core::Value) {")
	fmt.Fprintln(&b, "        match id.0 {")
	for _, signal := range signals {
		fmt.Fprintf(&b, "            %d => self.inner.set_%s(value),\n", signal.ID, signal.Rust)
	}
	fmt.Fprintln(&b, "            _ => {}")
	fmt.Fprintln(&b, "        }")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "    fn call_method(&mut self, id: vugra_core::MethodId) {")
	fmt.Fprintln(&b, "        match id.0 {")
	for _, method := range methods {
		if method.Event {
			continue
		}
		fmt.Fprintf(&b, "            %d => self.inner.%s(),\n", method.ID, method.Rust)
	}
	fmt.Fprintln(&b, "            _ => {}")
	fmt.Fprintln(&b, "        }")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "    fn call_event_method(&mut self, id: vugra_core::MethodId, event: vugra_core::Event) {")
	fmt.Fprintln(&b, "        match id.0 {")
	for _, method := range methods {
		if method.Event {
			fmt.Fprintf(&b, "            %d => self.inner.%s(event),\n", method.ID, method.Rust)
		} else {
			fmt.Fprintf(&b, "            %d => self.inner.%s(),\n", method.ID, method.Rust)
		}
	}
	fmt.Fprintln(&b, "            _ => {}")
	fmt.Fprintln(&b, "        }")
	fmt.Fprintln(&b, "    }")
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "#[allow(non_snake_case)]")
	fmt.Fprintln(&b, "pub trait RustSFCBindings {")
	for _, signal := range signals {
		fmt.Fprintf(&b, "    fn %s(&self) -> vugra_core::Value;\n", signal.Rust)
		fmt.Fprintf(&b, "    fn set_%s(&mut self, value: vugra_core::Value);\n", signal.Rust)
	}
	for _, method := range methods {
		if method.Event {
			fmt.Fprintf(&b, "    fn %s(&mut self, event: vugra_core::Event);\n", method.Rust)
		} else {
			fmt.Fprintf(&b, "    fn %s(&mut self);\n", method.Rust)
		}
	}
	fmt.Fprintln(&b, "}")
	return b.String()
}

func componentName(result *compiler.Result) string {
	if result == nil || result.SFC == nil {
		return "Component"
	}
	base := filepath.Base(result.SFC.Path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "" || base == "." {
		return "Component"
	}
	return base
}

func finderLiteSignalBindings(result *compiler.Result) []Binding {
	stateFields := map[string]string{}
	for _, field := range result.Rust.State.Fields {
		stateFields[field.Alias] = field.Name
		stateFields[field.Name] = field.Name
	}
	defs := []struct {
		id   int
		name string
	}{
		{1, "path"},
		{2, "status"},
		{3, "selectedSummary"},
		{13, "documentsLabel"},
		{14, "downloadsLabel"},
		{15, "picturesLabel"},
		{16, "documentsActive"},
		{17, "downloadsActive"},
		{18, "picturesActive"},
		{19, "searchQuery"},
		{92, "favoritesLabel"},
		{93, "workspaceLabel"},
		{94, "favoritesOpen"},
		{95, "workspaceOpen"},
		{96, "projectALabel"},
		{97, "projectBLabel"},
		{98, "projectAActive"},
		{99, "projectBActive"},
		{100, "itemMenuOpen"},
		{101, "blankMenuOpen"},
		{102, "renameText"},
		{103, "previewOpen"},
		{104, "previewTitle"},
		{105, "previewBody"},
		{106, "sidebarClass"},
		{107, "splitterClass"},
	}
	for row := 1; row <= 12; row++ {
		base := 20 + (row-1)*6
		defs = append(defs,
			struct {
				id   int
				name string
			}{base, fmt.Sprintf("row%dName", row)},
			struct {
				id   int
				name string
			}{base + 1, fmt.Sprintf("row%dKind", row)},
			struct {
				id   int
				name string
			}{base + 2, fmt.Sprintf("row%dModified", row)},
			struct {
				id   int
				name string
			}{base + 3, fmt.Sprintf("row%dSize", row)},
			struct {
				id   int
				name string
			}{base + 4, fmt.Sprintf("row%dClass", row)},
			struct {
				id   int
				name string
			}{base + 5, fmt.Sprintf("row%dSelected", row)},
		)
	}
	out := make([]Binding, 0, len(defs))
	for _, def := range defs {
		rust := stateFields[def.name]
		if rust == "" {
			rust = def.name
		}
		out = append(out, Binding{ID: def.id, Name: def.name, Rust: rust})
	}
	return out
}

func finderLiteMethodBindings(result *compiler.Result) []Binding {
	methods := map[string]string{}
	eventMethods := map[string]bool{}
	for _, method := range result.Rust.Methods {
		if method.Exported {
			methods[method.Name] = method.Name
			eventMethods[method.Name] = method.EventArg
		}
	}
	defs := []struct {
		id   int
		name string
	}{
		{1, "Back"},
		{2, "SelectRow1"},
		{3, "SelectRow2"},
		{4, "SelectRow3"},
		{5, "OpenDocuments"},
		{6, "OpenDownloads"},
		{7, "OpenPictures"},
		{8, "SelectPrevious"},
		{9, "SelectNext"},
		{10, "SearchInput"},
		{11, "SearchBackspace"},
		{12, "SearchClear"},
		{13, "OpenSelected"},
		{14, "OpenParent"},
		{15, "ToggleFavorites"},
		{16, "ToggleWorkspace"},
		{17, "OpenProjectA"},
		{18, "OpenProjectB"},
		{19, "DismissOverlay"},
		{20, "Forward"},
		{33, "BeginRename"},
		{34, "CancelRename"},
		{35, "CommitRename"},
		{36, "DeleteSelected"},
		{37, "DuplicateSelected"},
		{38, "NewFolder"},
		{39, "ShowBlankMenu"},
		{40, "ClosePreview"},
		{77, "ClearSelection"},
		{78, "Paste"},
		{79, "Refresh"},
		{80, "SelectAll"},
		{81, "HoverSplitter"},
		{82, "ResizeSidebar"},
	}
	for row := 4; row <= 12; row++ {
		defs = append(defs, struct {
			id   int
			name string
		}{20 + row, fmt.Sprintf("SelectRow%d", row)})
	}
	for row := 1; row <= 12; row++ {
		defs = append(defs, struct {
			id   int
			name string
		}{40 + row, fmt.Sprintf("ShowRow%dMenu", row)})
	}
	for row := 1; row <= 12; row++ {
		defs = append(defs, struct {
			id   int
			name string
		}{52 + row, fmt.Sprintf("HoverRow%d", row)})
	}
	for row := 1; row <= 12; row++ {
		defs = append(defs, struct {
			id   int
			name string
		}{64 + row, fmt.Sprintf("OpenRow%d", row)})
	}
	out := make([]Binding, 0, len(defs))
	for _, def := range defs {
		rust := methods[def.name]
		if rust == "" {
			rust = def.name
		}
		out = append(out, Binding{
			ID:    def.id,
			Name:  def.name,
			Rust:  rust,
			Event: eventMethods[def.name] || finderLiteEventMethod(def.name),
		})
	}
	return out
}

func finderLiteEventMethod(name string) bool {
	return name == "SearchInput" || name == "ResizeSidebar"
}

func exportedMethods(result *compiler.Result) []compilerRustMethod {
	methods := make([]compilerRustMethod, 0, len(result.Rust.Methods))
	for _, method := range result.Rust.Methods {
		if method.Exported {
			methods = append(methods, compilerRustMethod{name: method.Name, eventArg: method.EventArg})
		}
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].name < methods[j].name
	})
	return methods
}

type compilerRustMethod struct {
	name     string
	eventArg bool
}
