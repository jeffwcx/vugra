//! Rust glue/codegen helpers for Vugra component contracts.

use std::fmt::Write;

use vugra_ir::{
    Component, MethodDef, OverlayBinding, RowBinding, SearchBinding, SidebarBinding,
    SidebarSectionBinding, SignalDef, SplitterBinding, ToolbarBinding,
};

pub fn generate_component_contract_fn(function_name: &str, component: &Component) -> String {
    let mut out = String::new();
    writeln!(
        &mut out,
        "pub fn {function_name}() -> vugra_ir::Component {{"
    )
    .unwrap();
    writeln!(
        &mut out,
        "    let mut component = vugra_ir::Component::new({:?});",
        component.name
    )
    .unwrap();
    write_signals(&mut out, &component.signals);
    write_methods(&mut out, &component.methods);
    write_rows(&mut out, &component.rows);
    write_sidebar(&mut out, &component.sidebar);
    write_sidebar_sections(&mut out, &component.sidebar_sections);
    write_toolbar(&mut out, component.toolbar.as_ref());
    write_splitter(&mut out, component.splitter.as_ref());
    write_search(&mut out, component.search.as_ref());
    write_overlays(&mut out, component.overlays.as_ref());
    writeln!(&mut out, "    component").unwrap();
    writeln!(&mut out, "}}").unwrap();
    out
}

pub struct StateAdapterInput<'a> {
    pub trait_name: &'a str,
    pub adapter_name: &'a str,
    pub component: &'a Component,
}

pub fn generate_finder_lite_state_adapter(input: StateAdapterInput<'_>) -> String {
    let mut out = String::new();
    writeln!(&mut out, "pub trait {} {{", input.trait_name).unwrap();
    for signal in input
        .component
        .signals
        .iter()
        .filter(|signal| !is_row_signal(signal))
    {
        writeln!(
            &mut out,
            "    fn {}(&self) -> vugra_core::Value;",
            rust_method_name(&signal.name)
        )
        .unwrap();
    }
    writeln!(
        &mut out,
        "    fn row_value(&self, index: usize, field: &str) -> vugra_core::Value;"
    )
    .unwrap();
    for method in &input.component.methods {
        let method_name = rust_method_name(&method.name);
        if is_row_select_method(method) {
            continue;
        }
        if is_event_method(input.component, method) {
            writeln!(
                &mut out,
                "    fn {method_name}(&mut self, event: vugra_core::Event);"
            )
            .unwrap();
        } else {
            writeln!(&mut out, "    fn {method_name}(&mut self);").unwrap();
        }
    }
    writeln!(&mut out, "    fn select_row(&mut self, index: usize);").unwrap();
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "pub struct {}<T> {{", input.adapter_name).unwrap();
    writeln!(&mut out, "    inner: T,").unwrap();
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "impl<T> {}<T> {{", input.adapter_name).unwrap();
    writeln!(&mut out, "    pub fn new(inner: T) -> Self {{").unwrap();
    writeln!(&mut out, "        Self {{ inner }}").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "    pub fn inner(&self) -> &T {{").unwrap();
    writeln!(&mut out, "        &self.inner").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "    pub fn inner_mut(&mut self) -> &mut T {{").unwrap();
    writeln!(&mut out, "        &mut self.inner").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(
        &mut out,
        "impl<T: {}> vugra_core::ComponentState for {}<T> {{",
        input.trait_name, input.adapter_name
    )
    .unwrap();
    write_finder_adapter_get_signal(&mut out, &input);
    write_adapter_set_signal(&mut out);
    write_finder_adapter_call_method(&mut out, &input);
    write_finder_adapter_call_event_method(&mut out, &input);
    out
}

pub fn generate_state_adapter(input: StateAdapterInput<'_>) -> String {
    let mut out = String::new();
    writeln!(&mut out, "pub trait {} {{", input.trait_name).unwrap();
    for signal in &input.component.signals {
        writeln!(
            &mut out,
            "    fn {}(&self) -> vugra_core::Value;",
            rust_method_name(&signal.name)
        )
        .unwrap();
    }
    for method in &input.component.methods {
        if is_event_method(input.component, method) {
            writeln!(
                &mut out,
                "    fn {}(&mut self, event: vugra_core::Event);",
                rust_method_name(&method.name)
            )
            .unwrap();
        } else {
            writeln!(
                &mut out,
                "    fn {}(&mut self);",
                rust_method_name(&method.name)
            )
            .unwrap();
        }
    }
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "pub struct {}<T> {{", input.adapter_name).unwrap();
    writeln!(&mut out, "    inner: T,").unwrap();
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "impl<T> {}<T> {{", input.adapter_name).unwrap();
    writeln!(&mut out, "    pub fn new(inner: T) -> Self {{").unwrap();
    writeln!(&mut out, "        Self {{ inner }}").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "    pub fn inner(&self) -> &T {{").unwrap();
    writeln!(&mut out, "        &self.inner").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(&mut out, "    pub fn inner_mut(&mut self) -> &mut T {{").unwrap();
    writeln!(&mut out, "        &mut self.inner").unwrap();
    writeln!(&mut out, "    }}").unwrap();
    writeln!(&mut out, "}}").unwrap();
    writeln!(&mut out).unwrap();
    writeln!(
        &mut out,
        "impl<T: {}> vugra_core::ComponentState for {}<T> {{",
        input.trait_name, input.adapter_name
    )
    .unwrap();
    write_adapter_get_signal(&mut out, &input);
    write_adapter_set_signal(&mut out);
    write_adapter_call_method(&mut out, &input);
    write_adapter_call_event_method(&mut out, &input);
    writeln!(&mut out, "}}").unwrap();
    out
}

fn write_finder_adapter_get_signal(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn get_signal(&self, id: vugra_core::SignalId) -> vugra_core::Value {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for signal in input
        .component
        .signals
        .iter()
        .filter(|signal| !is_row_signal(signal))
    {
        writeln!(
            &mut *out,
            "            {} => self.inner.{}(),",
            signal.id.0,
            rust_method_name(&signal.name)
        )
        .unwrap();
    }
    writeln!(
        &mut *out,
        "            20..=91 => row_signal_field(id.0).map(|(index, field)| self.inner.row_value(index, field)).unwrap_or(vugra_core::Value::None),"
    )
    .unwrap();
    writeln!(&mut *out, "            _ => vugra_core::Value::None,").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
}

fn write_finder_adapter_call_method(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn call_method(&mut self, id: vugra_core::MethodId) {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for method in &input.component.methods {
        if is_event_method(input.component, method) {
            continue;
        }
        if is_row_select_method(method) {
            writeln!(
                &mut *out,
                "            {} => self.inner.select_row({}),",
                method.id.0,
                row_select_index(method).unwrap()
            )
            .unwrap();
        } else {
            writeln!(
                &mut *out,
                "            {} => self.inner.{}(),",
                method.id.0,
                rust_method_name(&method.name)
            )
            .unwrap();
        }
    }
    writeln!(&mut *out, "            _ => {{}}").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
    writeln!(&mut *out).unwrap();
}

fn write_finder_adapter_call_event_method(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn call_event_method(&mut self, id: vugra_core::MethodId, event: vugra_core::Event) {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for method in &input.component.methods {
        if is_row_select_method(method) {
            writeln!(
                &mut *out,
                "            {} => self.inner.select_row({}),",
                method.id.0,
                row_select_index(method).unwrap()
            )
            .unwrap();
        } else if is_event_method(input.component, method) {
            writeln!(
                &mut *out,
                "            {} => self.inner.{}(event),",
                method.id.0,
                rust_method_name(&method.name)
            )
            .unwrap();
        } else {
            writeln!(
                &mut *out,
                "            {} => self.inner.{}(),",
                method.id.0,
                rust_method_name(&method.name)
            )
            .unwrap();
        }
    }
    writeln!(&mut *out, "            _ => {{}}").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
    writeln!(&mut *out).unwrap();
    writeln!(
        &mut *out,
        "}}\n\nfn row_signal_field(id: u32) -> Option<(usize, &'static str)> {{\n    if id < 20 {{\n        return None;\n    }}\n    let offset = id - 20;\n    let index = (offset / 6) as usize;\n    if index >= 12 {{\n        return None;\n    }}\n    let field = match offset % 6 {{\n        0 => \"name\",\n        1 => \"kind\",\n        2 => \"modified\",\n        3 => \"size\",\n        4 => \"class\",\n        5 => \"selected\",\n        _ => return None,\n    }};\n    Some((index, field))\n}}\n"
    )
    .unwrap();
}

fn is_row_signal(signal: &SignalDef) -> bool {
    (20..=91).contains(&signal.id.0)
}

fn is_row_select_method(method: &MethodDef) -> bool {
    row_select_index(method).is_some()
}

fn row_select_index(method: &MethodDef) -> Option<usize> {
    match method.id.0 {
        2 => Some(0),
        3 => Some(1),
        4 => Some(2),
        24..=32 => Some((method.id.0 - 21) as usize),
        _ => None,
    }
}

fn write_adapter_get_signal(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn get_signal(&self, id: vugra_core::SignalId) -> vugra_core::Value {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for signal in &input.component.signals {
        writeln!(
            &mut *out,
            "            {} => self.inner.{}(),",
            signal.id.0,
            rust_method_name(&signal.name)
        )
        .unwrap();
    }
    writeln!(&mut *out, "            _ => vugra_core::Value::None,").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
    writeln!(&mut *out).unwrap();
}

fn write_adapter_set_signal(out: &mut String) {
    writeln!(
        &mut *out,
        "    fn set_signal(&mut self, _: vugra_core::SignalId, _: vugra_core::Value) {{}}"
    )
    .unwrap();
    writeln!(&mut *out).unwrap();
}

fn write_adapter_call_method(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn call_method(&mut self, id: vugra_core::MethodId) {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for method in &input.component.methods {
        if is_event_method(input.component, method) {
            continue;
        }
        writeln!(
            &mut *out,
            "            {} => self.inner.{}(),",
            method.id.0,
            rust_method_name(&method.name)
        )
        .unwrap();
    }
    writeln!(&mut *out, "            _ => {{}}").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
    writeln!(&mut *out).unwrap();
}

fn write_adapter_call_event_method(out: &mut String, input: &StateAdapterInput<'_>) {
    writeln!(
        &mut *out,
        "    fn call_event_method(&mut self, id: vugra_core::MethodId, event: vugra_core::Event) {{"
    )
    .unwrap();
    writeln!(&mut *out, "        match id.0 {{").unwrap();
    for method in &input.component.methods {
        if is_event_method(input.component, method) {
            writeln!(
                &mut *out,
                "            {} => self.inner.{}(event),",
                method.id.0,
                rust_method_name(&method.name)
            )
            .unwrap();
        } else {
            writeln!(
                &mut *out,
                "            {} => self.inner.{}(),",
                method.id.0,
                rust_method_name(&method.name)
            )
            .unwrap();
        }
    }
    writeln!(&mut *out, "            _ => {{}}").unwrap();
    writeln!(&mut *out, "        }}").unwrap();
    writeln!(&mut *out, "    }}").unwrap();
}

fn is_event_method(component: &Component, method: &MethodDef) -> bool {
    component
        .search
        .as_ref()
        .is_some_and(|search| search.input_method == method.id)
        || component
            .splitter
            .as_ref()
            .is_some_and(|splitter| splitter.drag_method == method.id)
}

fn rust_method_name(name: &str) -> String {
    let mut out = String::new();
    let chars: Vec<char> = name.chars().collect();
    for (index, ch) in chars.iter().copied().enumerate() {
        if ch.is_ascii_uppercase() {
            let previous = index.checked_sub(1).and_then(|index| chars.get(index));
            let next = chars.get(index + 1);
            let split_before = previous.is_some_and(|previous| {
                previous.is_ascii_lowercase()
                    || previous.is_ascii_digit()
                    || (previous.is_ascii_uppercase()
                        && next.is_some_and(|next| next.is_ascii_lowercase()))
            });
            if !out.is_empty() && split_before {
                out.push('_');
            }
            out.push(ch.to_ascii_lowercase());
        } else if ch.is_ascii_alphanumeric() {
            out.push(ch.to_ascii_lowercase());
        } else if !out.ends_with('_') {
            out.push('_');
        }
    }
    if out.is_empty() {
        "binding".to_string()
    } else {
        out
    }
}

fn write_signals(out: &mut String, signals: &[SignalDef]) {
    writeln!(&mut *out, "    component.signals = vec![").unwrap();
    for signal in signals {
        writeln!(&mut *out, "        vugra_ir::SignalDef {{").unwrap();
        writeln!(
            &mut *out,
            "            id: vugra_ir::SignalId({}),",
            signal.id.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            name: {:?}.to_string(),",
            signal.name
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            kind: {},",
            value_kind_expr(signal.kind)
        )
        .unwrap();
        writeln!(&mut *out, "        }},").unwrap();
    }
    writeln!(&mut *out, "    ];").unwrap();
}

fn write_methods(out: &mut String, methods: &[MethodDef]) {
    writeln!(&mut *out, "    component.methods = vec![").unwrap();
    for method in methods {
        writeln!(&mut *out, "        vugra_ir::MethodDef {{").unwrap();
        writeln!(
            &mut *out,
            "            id: vugra_ir::MethodId({}),",
            method.id.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            name: {:?}.to_string(),",
            method.name
        )
        .unwrap();
        writeln!(&mut *out, "        }},").unwrap();
    }
    writeln!(&mut *out, "    ];").unwrap();
}

fn write_rows(out: &mut String, rows: &[RowBinding]) {
    writeln!(&mut *out, "    component.rows = vec![").unwrap();
    for row in rows {
        writeln!(&mut *out, "        vugra_ir::RowBinding {{").unwrap();
        writeln!(
            &mut *out,
            "            name: vugra_ir::SignalId({}),",
            row.name.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            kind: vugra_ir::SignalId({}),",
            row.kind.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            modified: vugra_ir::SignalId({}),",
            row.modified.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            size: vugra_ir::SignalId({}),",
            row.size.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            class: vugra_ir::SignalId({}),",
            row.class.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            selected: vugra_ir::SignalId({}),",
            row.selected.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            select_method: vugra_ir::MethodId({}),",
            row.select_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            hover_method: vugra_ir::MethodId({}),",
            row.hover_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            open_method: vugra_ir::MethodId({}),",
            row.open_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            context_menu_method: vugra_ir::MethodId({}),",
            row.context_menu_method.0
        )
        .unwrap();
        writeln!(&mut *out, "        }},").unwrap();
    }
    writeln!(&mut *out, "    ];").unwrap();
}

fn write_sidebar(out: &mut String, sidebar: &[SidebarBinding]) {
    writeln!(&mut *out, "    component.sidebar = vec![").unwrap();
    for item in sidebar {
        writeln!(&mut *out, "        vugra_ir::SidebarBinding {{").unwrap();
        writeln!(
            &mut *out,
            "            label: vugra_ir::SignalId({}),",
            item.label.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            kind: {},",
            sidebar_item_kind_expr(item.kind)
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            active: vugra_ir::SignalId({}),",
            item.active.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            open_method: vugra_ir::MethodId({}),",
            item.open_method.0
        )
        .unwrap();
        writeln!(&mut *out, "        }},").unwrap();
    }
    writeln!(&mut *out, "    ];").unwrap();
}

fn write_sidebar_sections(out: &mut String, sections: &[SidebarSectionBinding]) {
    writeln!(&mut *out, "    component.sidebar_sections = vec![").unwrap();
    for section in sections {
        writeln!(&mut *out, "        vugra_ir::SidebarSectionBinding {{").unwrap();
        writeln!(
            &mut *out,
            "            label: vugra_ir::SignalId({}),",
            section.label.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            open: vugra_ir::SignalId({}),",
            section.open.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "            toggle_method: vugra_ir::MethodId({}),",
            section.toggle_method.0
        )
        .unwrap();
        writeln!(&mut *out, "            items: vec![").unwrap();
        for item in &section.items {
            writeln!(&mut *out, "                vugra_ir::SidebarBinding {{").unwrap();
            writeln!(
                &mut *out,
                "                    label: vugra_ir::SignalId({}),",
                item.label.0
            )
            .unwrap();
            writeln!(
                &mut *out,
                "                    kind: {},",
                sidebar_item_kind_expr(item.kind)
            )
            .unwrap();
            writeln!(
                &mut *out,
                "                    active: vugra_ir::SignalId({}),",
                item.active.0
            )
            .unwrap();
            writeln!(
                &mut *out,
                "                    open_method: vugra_ir::MethodId({}),",
                item.open_method.0
            )
            .unwrap();
            writeln!(&mut *out, "                }},").unwrap();
        }
        writeln!(&mut *out, "            ],").unwrap();
        writeln!(&mut *out, "        }},").unwrap();
    }
    writeln!(&mut *out, "    ];").unwrap();
}

fn write_search(out: &mut String, search: Option<&SearchBinding>) {
    if let Some(search) = search {
        writeln!(
            &mut *out,
            "    component.search = Some(vugra_ir::SearchBinding {{"
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        query: vugra_ir::SignalId({}),",
            search.query.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        input_method: vugra_ir::MethodId({}),",
            search.input_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        backspace_method: vugra_ir::MethodId({}),",
            search.backspace_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        clear_method: vugra_ir::MethodId({}),",
            search.clear_method.0
        )
        .unwrap();
        writeln!(&mut *out, "    }});").unwrap();
    } else {
        writeln!(&mut *out, "    component.search = None;").unwrap();
    }
}

fn write_splitter(out: &mut String, splitter: Option<&SplitterBinding>) {
    if let Some(splitter) = splitter {
        writeln!(
            &mut *out,
            "    component.splitter = Some(vugra_ir::SplitterBinding {{"
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        sidebar_class: vugra_ir::SignalId({}),",
            splitter.sidebar_class.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        splitter_class: vugra_ir::SignalId({}),",
            splitter.splitter_class.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        hover_method: vugra_ir::MethodId({}),",
            splitter.hover_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        drag_method: vugra_ir::MethodId({}),",
            splitter.drag_method.0
        )
        .unwrap();
        writeln!(&mut *out, "    }});").unwrap();
    } else {
        writeln!(&mut *out, "    component.splitter = None;").unwrap();
    }
}

fn write_toolbar(out: &mut String, toolbar: Option<&ToolbarBinding>) {
    if let Some(toolbar) = toolbar {
        writeln!(
            &mut *out,
            "    component.toolbar = Some(vugra_ir::ToolbarBinding {{"
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        back_method: vugra_ir::MethodId({}),",
            toolbar.back_method.0
        )
        .unwrap();
        writeln!(
            &mut *out,
            "        forward_method: vugra_ir::MethodId({}),",
            toolbar.forward_method.0
        )
        .unwrap();
        writeln!(&mut *out, "    }});").unwrap();
    } else {
        writeln!(&mut *out, "    component.toolbar = None;").unwrap();
    }
}

fn write_overlays(out: &mut String, overlays: Option<&OverlayBinding>) {
    if let Some(overlays) = overlays {
        writeln!(
            &mut *out,
            "    component.overlays = Some(vugra_ir::OverlayBinding {{"
        )
        .unwrap();
        for (field, id) in [
            ("item_menu_open", overlays.item_menu_open.0),
            ("blank_menu_open", overlays.blank_menu_open.0),
            ("rename_text", overlays.rename_text.0),
            ("preview_open", overlays.preview_open.0),
            ("preview_title", overlays.preview_title.0),
            ("preview_body", overlays.preview_body.0),
        ] {
            writeln!(&mut *out, "        {field}: vugra_ir::SignalId({id}),").unwrap();
        }
        for (field, id) in [
            ("open_selected_method", overlays.open_selected_method.0),
            ("begin_rename_method", overlays.begin_rename_method.0),
            ("cancel_rename_method", overlays.cancel_rename_method.0),
            ("commit_rename_method", overlays.commit_rename_method.0),
            ("delete_selected_method", overlays.delete_selected_method.0),
            (
                "duplicate_selected_method",
                overlays.duplicate_selected_method.0,
            ),
            ("new_folder_method", overlays.new_folder_method.0),
            ("dismiss_overlay_method", overlays.dismiss_overlay_method.0),
            ("clear_selection_method", overlays.clear_selection_method.0),
            ("show_blank_menu_method", overlays.show_blank_menu_method.0),
            ("paste_method", overlays.paste_method.0),
            ("refresh_method", overlays.refresh_method.0),
            ("close_preview_method", overlays.close_preview_method.0),
        ] {
            writeln!(&mut *out, "        {field}: vugra_ir::MethodId({id}),").unwrap();
        }
        writeln!(&mut *out, "    }});").unwrap();
    } else {
        writeln!(&mut *out, "    component.overlays = None;").unwrap();
    }
}

fn value_kind_expr(kind: vugra_ir::ValueKind) -> &'static str {
    match kind {
        vugra_ir::ValueKind::Bool => "vugra_ir::ValueKind::Bool",
        vugra_ir::ValueKind::Number => "vugra_ir::ValueKind::Number",
        vugra_ir::ValueKind::String => "vugra_ir::ValueKind::String",
    }
}

fn sidebar_item_kind_expr(kind: vugra_ir::SidebarItemKind) -> &'static str {
    match kind {
        vugra_ir::SidebarItemKind::Folder => "vugra_ir::SidebarItemKind::Folder",
        vugra_ir::SidebarItemKind::Download => "vugra_ir::SidebarItemKind::Download",
        vugra_ir::SidebarItemKind::Picture => "vugra_ir::SidebarItemKind::Picture",
        vugra_ir::SidebarItemKind::Project => "vugra_ir::SidebarItemKind::Project",
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generates_finder_lite_contract_source() {
        let source = generate_component_contract_fn(
            "generated_finder_lite_contract",
            &vugra_ir::finder_lite_contract(),
        );

        assert!(source.contains("pub fn generated_finder_lite_contract() -> vugra_ir::Component"));
        assert!(source.contains("vugra_ir::Component::new(\"FinderLite\")"));
        assert!(source.contains("vugra_ir::SignalId(19)"));
        assert!(source.contains("vugra_ir::MethodId(14)"));
        assert!(source.contains("component.search = Some"));
        assert!(source.contains("component.toolbar = Some"));
        assert!(source.contains("component.splitter = Some"));
        assert!(source.contains("component.overlays = Some"));
        assert!(source.contains("clear_selection_method: vugra_ir::MethodId(77)"));
        assert!(source.contains("paste_method: vugra_ir::MethodId(78)"));
        assert!(source.contains("refresh_method: vugra_ir::MethodId(79)"));
        assert!(source.contains("show_blank_menu_method: vugra_ir::MethodId(39)"));
        assert!(source.contains("drag_method: vugra_ir::MethodId(82)"));
        assert!(source.contains("kind: vugra_ir::SidebarItemKind::Download"));
        assert!(source.contains("forward_method: vugra_ir::MethodId(20)"));
    }

    #[test]
    fn generated_finder_lite_contract_describes_expanded_rows() {
        let source = generate_component_contract_fn(
            "generated_finder_lite_contract",
            &vugra_ir::finder_lite_contract(),
        );
        assert!(source.contains("name: \"row12Name\".to_string()"));
        assert!(source.contains("modified: vugra_ir::SignalId(82)"));
        assert!(source.contains("class: vugra_ir::SignalId(84)"));
        assert!(source.contains("select_method: vugra_ir::MethodId(32)"));
    }

    #[test]
    fn generated_finder_lite_adapter_describes_expanded_rows() {
        let source = generate_state_adapter(StateAdapterInput {
            trait_name: "FinderLiteBindings",
            adapter_name: "FinderLiteAdapter",
            component: &vugra_ir::finder_lite_contract(),
        });
        assert!(source.contains("fn row12_name(&self) -> vugra_core::Value;"));
        assert!(source.contains("fn row12_modified(&self) -> vugra_core::Value;"));
        assert!(source.contains("fn select_row12(&mut self);"));
        assert!(source.contains("fn sidebar_class(&self) -> vugra_core::Value;"));
        assert!(source.contains("fn resize_sidebar(&mut self, event: vugra_core::Event);"));
        assert!(source.contains("32 => self.inner.select_row12(),"));
        assert!(source.contains("82 => self.inner.resize_sidebar(event),"));
    }

    #[test]
    fn rust_method_names_use_snake_case() {
        assert_eq!(rust_method_name("selectedSummary"), "selected_summary");
        assert_eq!(rust_method_name("OpenDocuments"), "open_documents");
        assert_eq!(rust_method_name("projectALabel"), "project_a_label");
        assert_eq!(rust_method_name("row1Name"), "row1_name");
    }
}
