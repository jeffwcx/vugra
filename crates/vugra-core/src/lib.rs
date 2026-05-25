//! Minimal Rust Vugra kernel slice.

use std::fmt;

pub use vugra_ir::{finder_lite_contract, Component, MethodId, SidebarItemKind, SignalId};

#[derive(Clone, Debug, PartialEq)]
pub enum Value {
    Bool(bool),
    Number(f64),
    String(String),
    None,
}

impl Value {
    pub fn as_text(&self) -> String {
        match self {
            Value::Bool(value) => value.to_string(),
            Value::Number(value) => value.to_string(),
            Value::String(value) => value.clone(),
            Value::None => String::new(),
        }
    }
}

impl From<&str> for Value {
    fn from(value: &str) -> Self {
        Value::String(value.to_string())
    }
}

impl From<String> for Value {
    fn from(value: String) -> Self {
        Value::String(value)
    }
}

impl From<bool> for Value {
    fn from(value: bool) -> Self {
        Value::Bool(value)
    }
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct Event {
    pub kind: String,
    pub key: String,
    pub text: String,
    pub x: f32,
    pub y: f32,
    pub delta_x: f32,
    pub delta_y: f32,
    pub modifiers: Modifiers,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Modifiers {
    pub shift: bool,
    pub ctrl: bool,
    pub meta: bool,
    pub alt: bool,
}

pub trait ComponentState {
    fn get_signal(&self, id: SignalId) -> Value;
    fn set_signal(&mut self, id: SignalId, value: Value);
    fn call_method(&mut self, id: MethodId);
    fn call_event_method(&mut self, id: MethodId, event: Event) {
        let _ = event;
        self.call_method(id);
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Frame {
    pub title: String,
    pub path: String,
    pub values: Vec<FrameValue>,
    pub search_query: String,
    pub search_input_method: Option<MethodId>,
    pub search_backspace_method: Option<MethodId>,
    pub search_clear_method: Option<MethodId>,
    pub toolbar: Option<FrameToolbar>,
    pub status: String,
    pub selected_summary: String,
    pub sidebar: Vec<FrameSidebarItem>,
    pub sidebar_sections: Vec<FrameSidebarSection>,
    pub splitter: Option<FrameSplitter>,
    pub rows: Vec<FrameRow>,
    pub overlays: FrameOverlays,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameValue {
    pub name: String,
    pub value: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameToolbar {
    pub back_method: MethodId,
    pub forward_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameSidebarItem {
    pub label: String,
    pub kind: SidebarItemKind,
    pub active: bool,
    pub open_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameSidebarSection {
    pub label: String,
    pub open: bool,
    pub toggle_method: MethodId,
    pub items: Vec<FrameSidebarItem>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameSplitter {
    pub sidebar_class: String,
    pub splitter_class: String,
    pub hover_method: MethodId,
    pub drag_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FrameRow {
    pub name: String,
    pub kind: String,
    pub modified: String,
    pub size: String,
    pub class: String,
    pub selected: bool,
    pub visual_state: RowVisualState,
    pub select_method: MethodId,
    pub hover_method: MethodId,
    pub open_method: MethodId,
    pub context_menu_method: MethodId,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum RowVisualState {
    #[default]
    Normal,
    Hover,
    Focus,
    Editing,
    Selected,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct FrameOverlays {
    pub item_menu_open: bool,
    pub blank_menu_open: bool,
    pub rename_text: String,
    pub preview_open: bool,
    pub preview_title: String,
    pub preview_body: String,
    pub rename_text_signal: Option<SignalId>,
    pub open_selected_method: Option<MethodId>,
    pub begin_rename_method: Option<MethodId>,
    pub cancel_rename_method: Option<MethodId>,
    pub commit_rename_method: Option<MethodId>,
    pub delete_selected_method: Option<MethodId>,
    pub duplicate_selected_method: Option<MethodId>,
    pub new_folder_method: Option<MethodId>,
    pub dismiss_overlay_method: Option<MethodId>,
    pub clear_selection_method: Option<MethodId>,
    pub show_blank_menu_method: Option<MethodId>,
    pub paste_method: Option<MethodId>,
    pub refresh_method: Option<MethodId>,
    pub close_preview_method: Option<MethodId>,
}

impl fmt::Display for Frame {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "{}", self.title)?;
        writeln!(f, "path: {}", self.path)?;
        writeln!(f, "status: {}", self.status)?;
        writeln!(f, "selected: {}", self.selected_summary)?;
        for row in &self.rows {
            let marker = if row.selected { "*" } else { "-" };
            writeln!(f, "{marker} {} ({})", row.name, row.kind)?;
        }
        Ok(())
    }
}

pub struct App<S> {
    component: Component,
    state: S,
}

impl<S: ComponentState> App<S> {
    pub fn new(component: Component, state: S) -> Self {
        Self { component, state }
    }

    pub fn state(&self) -> &S {
        &self.state
    }

    pub fn state_mut(&mut self) -> &mut S {
        &mut self.state
    }

    pub fn dispatch(&mut self, method: MethodId) {
        self.state.call_method(method);
    }

    pub fn dispatch_event(&mut self, method: MethodId, event: Event) {
        self.state.call_event_method(method, event);
    }

    pub fn render_frame(&self) -> Frame {
        let signal = |name: &str| {
            self.component
                .signal_id(name)
                .map(|id| self.state.get_signal(id).as_text())
                .unwrap_or_default()
        };
        let values = self
            .component
            .signals
            .iter()
            .map(|signal| FrameValue {
                name: signal.name.clone(),
                value: self.state.get_signal(signal.id).as_text(),
            })
            .filter(|value| !value.value.is_empty())
            .collect();
        let rows = self
            .component
            .rows
            .iter()
            .map(|row| FrameRow {
                name: self.state.get_signal(row.name).as_text(),
                kind: self.state.get_signal(row.kind).as_text(),
                modified: self.state.get_signal(row.modified).as_text(),
                size: self.state.get_signal(row.size).as_text(),
                class: self.state.get_signal(row.class).as_text(),
                selected: matches!(self.state.get_signal(row.selected), Value::Bool(true)),
                visual_state: RowVisualState::from_class(
                    &self.state.get_signal(row.class).as_text(),
                ),
                select_method: row.select_method,
                hover_method: row.hover_method,
                open_method: row.open_method,
                context_menu_method: row.context_menu_method,
            })
            .filter(|row| !row.name.is_empty())
            .collect();
        let sidebar = self
            .component
            .sidebar
            .iter()
            .map(|item| FrameSidebarItem {
                label: self.state.get_signal(item.label).as_text(),
                kind: item.kind,
                active: matches!(self.state.get_signal(item.active), Value::Bool(true)),
                open_method: item.open_method,
            })
            .filter(|item| !item.label.is_empty())
            .collect();
        let sidebar_sections = self
            .component
            .sidebar_sections
            .iter()
            .map(|section| FrameSidebarSection {
                label: self.state.get_signal(section.label).as_text(),
                open: matches!(self.state.get_signal(section.open), Value::Bool(true)),
                toggle_method: section.toggle_method,
                items: section
                    .items
                    .iter()
                    .map(|item| FrameSidebarItem {
                        label: self.state.get_signal(item.label).as_text(),
                        kind: item.kind,
                        active: matches!(self.state.get_signal(item.active), Value::Bool(true)),
                        open_method: item.open_method,
                    })
                    .filter(|item| !item.label.is_empty())
                    .collect(),
            })
            .filter(|section| !section.label.is_empty())
            .collect();
        let search = self.component.search.as_ref();
        let toolbar = self.component.toolbar.as_ref().map(|toolbar| FrameToolbar {
            back_method: toolbar.back_method,
            forward_method: toolbar.forward_method,
        });
        let splitter = self
            .component
            .splitter
            .as_ref()
            .map(|splitter| FrameSplitter {
                sidebar_class: self.state.get_signal(splitter.sidebar_class).as_text(),
                splitter_class: self.state.get_signal(splitter.splitter_class).as_text(),
                hover_method: splitter.hover_method,
                drag_method: splitter.drag_method,
            });
        let overlays = self
            .component
            .overlays
            .as_ref()
            .map(|overlays| FrameOverlays {
                item_menu_open: matches!(
                    self.state.get_signal(overlays.item_menu_open),
                    Value::Bool(true)
                ),
                blank_menu_open: matches!(
                    self.state.get_signal(overlays.blank_menu_open),
                    Value::Bool(true)
                ),
                rename_text: self.state.get_signal(overlays.rename_text).as_text(),
                preview_open: matches!(
                    self.state.get_signal(overlays.preview_open),
                    Value::Bool(true)
                ),
                preview_title: self.state.get_signal(overlays.preview_title).as_text(),
                preview_body: self.state.get_signal(overlays.preview_body).as_text(),
                rename_text_signal: Some(overlays.rename_text),
                open_selected_method: Some(overlays.open_selected_method),
                begin_rename_method: Some(overlays.begin_rename_method),
                cancel_rename_method: Some(overlays.cancel_rename_method),
                commit_rename_method: Some(overlays.commit_rename_method),
                delete_selected_method: Some(overlays.delete_selected_method),
                duplicate_selected_method: Some(overlays.duplicate_selected_method),
                new_folder_method: Some(overlays.new_folder_method),
                dismiss_overlay_method: Some(overlays.dismiss_overlay_method),
                clear_selection_method: Some(overlays.clear_selection_method),
                show_blank_menu_method: Some(overlays.show_blank_menu_method),
                paste_method: Some(overlays.paste_method),
                refresh_method: Some(overlays.refresh_method),
                close_preview_method: Some(overlays.close_preview_method),
            })
            .unwrap_or_default();

        Frame {
            title: self.component.name.clone(),
            path: signal("path"),
            values,
            search_query: search
                .map(|search| self.state.get_signal(search.query).as_text())
                .unwrap_or_default(),
            search_input_method: search.map(|search| search.input_method),
            search_backspace_method: search.map(|search| search.backspace_method),
            search_clear_method: search.map(|search| search.clear_method),
            toolbar,
            status: signal("status"),
            selected_summary: signal("selectedSummary"),
            sidebar,
            sidebar_sections,
            splitter,
            rows,
            overlays,
        }
    }
}

impl RowVisualState {
    pub fn from_class(class: &str) -> Self {
        if class
            .split_whitespace()
            .any(|part| part == "file-row-editing")
        {
            Self::Editing
        } else if class
            .split_whitespace()
            .any(|part| part == "file-row-selected")
        {
            Self::Selected
        } else if class
            .split_whitespace()
            .any(|part| part == "file-row-focus")
        {
            Self::Focus
        } else if class
            .split_whitespace()
            .any(|part| part == "file-row-hover")
        {
            Self::Hover
        } else {
            Self::Normal
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    struct CounterState {
        value: i32,
    }

    impl ComponentState for CounterState {
        fn get_signal(&self, id: SignalId) -> Value {
            match id.0 {
                1 => self.value.to_string().into(),
                _ => Value::None,
            }
        }

        fn set_signal(&mut self, id: SignalId, value: Value) {
            if id.0 == 1 {
                self.value = value.as_text().parse().unwrap();
            }
        }

        fn call_method(&mut self, id: MethodId) {
            if id.0 == 1 {
                self.value += 1;
            }
        }
    }

    #[test]
    fn dispatches_component_state_contract() {
        let mut component = Component::new("Counter");
        component.signals.push(vugra_ir::SignalDef {
            id: SignalId(1),
            name: "path".to_string(),
            kind: vugra_ir::ValueKind::String,
        });
        component.methods.push(vugra_ir::MethodDef {
            id: MethodId(1),
            name: "Inc".to_string(),
        });
        let mut app = App::new(component, CounterState { value: 41 });
        app.dispatch(MethodId(1));
        assert_eq!(app.render_frame().path, "42");
    }
}
