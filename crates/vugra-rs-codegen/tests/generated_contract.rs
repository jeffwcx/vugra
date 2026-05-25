include!("generated_finder_lite_contract.rs");
include!("generated_finder_lite_adapter.rs");

#[derive(Default)]
struct FinderState {
    selected: usize,
    search: String,
    sidebar_class: String,
    splitter_class: String,
}

impl FinderLiteBindings for FinderState {
    fn path(&self) -> vugra_core::Value {
        "Documents".into()
    }

    fn status(&self) -> vugra_core::Value {
        "3 items · Current path: Documents".into()
    }

    fn selected_summary(&self) -> vugra_core::Value {
        "1 items selected".into()
    }

    fn row_value(&self, index: usize, field: &str) -> vugra_core::Value {
        let rows = [
            ("Design", "folder"),
            ("Roadmap.md", "file"),
            ("Budget 2026.xlsx", "file"),
        ];
        let Some((name, kind)) = rows.get(index) else {
            return vugra_core::Value::None;
        };
        match field {
            "name" => (*name).into(),
            "kind" => (*kind).into(),
            "modified" => "--".into(),
            "size" => "--".into(),
            "class" if self.selected == index => "file-row-selected".into(),
            "class" => "file-row".into(),
            "selected" => vugra_core::Value::Bool(self.selected == index),
            _ => vugra_core::Value::None,
        }
    }

    fn documents_label(&self) -> vugra_core::Value {
        "Documents".into()
    }

    fn downloads_label(&self) -> vugra_core::Value {
        "Downloads".into()
    }

    fn pictures_label(&self) -> vugra_core::Value {
        "Pictures".into()
    }

    fn documents_active(&self) -> vugra_core::Value {
        true.into()
    }

    fn downloads_active(&self) -> vugra_core::Value {
        false.into()
    }

    fn pictures_active(&self) -> vugra_core::Value {
        false.into()
    }

    fn search_query(&self) -> vugra_core::Value {
        self.search.clone().into()
    }

    fn favorites_label(&self) -> vugra_core::Value {
        "Favorites".into()
    }

    fn workspace_label(&self) -> vugra_core::Value {
        "Workspace".into()
    }

    fn favorites_open(&self) -> vugra_core::Value {
        true.into()
    }

    fn workspace_open(&self) -> vugra_core::Value {
        true.into()
    }

    fn project_a_label(&self) -> vugra_core::Value {
        "Current Project".into()
    }

    fn project_b_label(&self) -> vugra_core::Value {
        "Parent Folder".into()
    }

    fn project_a_active(&self) -> vugra_core::Value {
        false.into()
    }

    fn project_b_active(&self) -> vugra_core::Value {
        false.into()
    }

    fn item_menu_open(&self) -> vugra_core::Value {
        false.into()
    }

    fn blank_menu_open(&self) -> vugra_core::Value {
        false.into()
    }

    fn rename_text(&self) -> vugra_core::Value {
        "".into()
    }

    fn preview_open(&self) -> vugra_core::Value {
        false.into()
    }

    fn preview_title(&self) -> vugra_core::Value {
        "".into()
    }

    fn preview_body(&self) -> vugra_core::Value {
        "".into()
    }

    fn sidebar_class(&self) -> vugra_core::Value {
        if self.sidebar_class.is_empty() {
            "sidebar".into()
        } else {
            self.sidebar_class.clone().into()
        }
    }

    fn splitter_class(&self) -> vugra_core::Value {
        if self.splitter_class.is_empty() {
            "splitter".into()
        } else {
            self.splitter_class.clone().into()
        }
    }

    fn back(&mut self) {}

    fn forward(&mut self) {}

    fn select_row(&mut self, index: usize) {
        self.selected = index.min(2);
    }

    fn open_documents(&mut self) {}

    fn open_downloads(&mut self) {}

    fn open_pictures(&mut self) {}

    fn select_previous(&mut self) {
        self.selected = self.selected.saturating_sub(1);
    }

    fn select_next(&mut self) {
        self.selected = (self.selected + 1).min(2);
    }

    fn search_input(&mut self, event: vugra_core::Event) {
        self.search.push_str(&event.text);
    }

    fn search_backspace(&mut self) {
        self.search.pop();
    }

    fn search_clear(&mut self) {
        self.search.clear();
    }

    fn open_selected(&mut self) {}

    fn open_parent(&mut self) {}

    fn toggle_favorites(&mut self) {}

    fn toggle_workspace(&mut self) {}

    fn open_project_a(&mut self) {}

    fn open_project_b(&mut self) {}

    fn dismiss_overlay(&mut self) {}

    fn begin_rename(&mut self) {}

    fn cancel_rename(&mut self) {}

    fn commit_rename(&mut self) {}

    fn delete_selected(&mut self) {}

    fn duplicate_selected(&mut self) {}

    fn new_folder(&mut self) {}

    fn clear_selection(&mut self) {}

    fn show_blank_menu(&mut self) {}

    fn paste(&mut self) {}

    fn refresh(&mut self) {}

    fn select_all(&mut self) {}

    fn hover_splitter(&mut self) {
        self.splitter_class = "splitter-hover".to_string();
    }

    fn resize_sidebar(&mut self, _event: vugra_core::Event) {
        self.sidebar_class = "sidebar-200".to_string();
        self.splitter_class = "splitter".to_string();
    }

    fn close_preview(&mut self) {}

    fn show_row1_menu(&mut self) {
        self.selected = 0;
    }
    fn show_row2_menu(&mut self) {
        self.selected = 1;
    }
    fn show_row3_menu(&mut self) {
        self.selected = 2;
    }
    fn show_row4_menu(&mut self) {}
    fn show_row5_menu(&mut self) {}
    fn show_row6_menu(&mut self) {}
    fn show_row7_menu(&mut self) {}
    fn show_row8_menu(&mut self) {}
    fn show_row9_menu(&mut self) {}
    fn show_row10_menu(&mut self) {}
    fn show_row11_menu(&mut self) {}
    fn show_row12_menu(&mut self) {}
    fn hover_row1(&mut self) {}
    fn hover_row2(&mut self) {}
    fn hover_row3(&mut self) {}
    fn hover_row4(&mut self) {}
    fn hover_row5(&mut self) {}
    fn hover_row6(&mut self) {}
    fn hover_row7(&mut self) {}
    fn hover_row8(&mut self) {}
    fn hover_row9(&mut self) {}
    fn hover_row10(&mut self) {}
    fn hover_row11(&mut self) {}
    fn hover_row12(&mut self) {}
    fn open_row1(&mut self) {
        self.selected = 0;
    }
    fn open_row2(&mut self) {
        self.selected = 1;
    }
    fn open_row3(&mut self) {
        self.selected = 2;
    }
    fn open_row4(&mut self) {}
    fn open_row5(&mut self) {}
    fn open_row6(&mut self) {}
    fn open_row7(&mut self) {}
    fn open_row8(&mut self) {}
    fn open_row9(&mut self) {}
    fn open_row10(&mut self) {}
    fn open_row11(&mut self) {}
    fn open_row12(&mut self) {}
}

#[test]
fn generated_finder_lite_contract_compiles_and_matches_ir() {
    assert_eq!(
        generated_finder_lite_contract(),
        vugra_ir::finder_lite_contract()
    );
}

#[test]
fn generated_finder_lite_adapter_mounts_and_dispatches_state() {
    let mut app = vugra_core::App::new(
        generated_finder_lite_contract(),
        FinderLiteAdapter::new(FinderState::default()),
    );

    app.dispatch(vugra_core::MethodId(3));
    let frame = app.render_frame();
    assert_eq!(frame.rows[1].name, "Roadmap.md");
    assert!(frame.rows[1].selected);

    app.dispatch(vugra_core::MethodId(65));
    let frame = app.render_frame();
    assert_eq!(frame.rows[0].name, "Design");
    assert!(frame.rows[0].selected);

    app.dispatch_event(
        vugra_core::MethodId(10),
        vugra_core::Event {
            text: "road".to_string(),
            ..vugra_core::Event::default()
        },
    );
    assert_eq!(app.render_frame().search_query, "road");
}
