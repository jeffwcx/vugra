pub trait FinderLiteBindings {
    fn path(&self) -> vugra_core::Value;
    fn status(&self) -> vugra_core::Value;
    fn selected_summary(&self) -> vugra_core::Value;
    fn documents_label(&self) -> vugra_core::Value;
    fn downloads_label(&self) -> vugra_core::Value;
    fn pictures_label(&self) -> vugra_core::Value;
    fn documents_active(&self) -> vugra_core::Value;
    fn downloads_active(&self) -> vugra_core::Value;
    fn pictures_active(&self) -> vugra_core::Value;
    fn search_query(&self) -> vugra_core::Value;
    fn favorites_label(&self) -> vugra_core::Value;
    fn workspace_label(&self) -> vugra_core::Value;
    fn favorites_open(&self) -> vugra_core::Value;
    fn workspace_open(&self) -> vugra_core::Value;
    fn project_a_label(&self) -> vugra_core::Value;
    fn project_b_label(&self) -> vugra_core::Value;
    fn project_a_active(&self) -> vugra_core::Value;
    fn project_b_active(&self) -> vugra_core::Value;
    fn item_menu_open(&self) -> vugra_core::Value;
    fn blank_menu_open(&self) -> vugra_core::Value;
    fn rename_text(&self) -> vugra_core::Value;
    fn preview_open(&self) -> vugra_core::Value;
    fn preview_title(&self) -> vugra_core::Value;
    fn preview_body(&self) -> vugra_core::Value;
    fn sidebar_class(&self) -> vugra_core::Value;
    fn splitter_class(&self) -> vugra_core::Value;
    fn row_value(&self, index: usize, field: &str) -> vugra_core::Value;
    fn back(&mut self);
    fn open_documents(&mut self);
    fn open_downloads(&mut self);
    fn open_pictures(&mut self);
    fn select_previous(&mut self);
    fn select_next(&mut self);
    fn search_input(&mut self, event: vugra_core::Event);
    fn search_backspace(&mut self);
    fn search_clear(&mut self);
    fn open_selected(&mut self);
    fn open_parent(&mut self);
    fn toggle_favorites(&mut self);
    fn toggle_workspace(&mut self);
    fn open_project_a(&mut self);
    fn open_project_b(&mut self);
    fn dismiss_overlay(&mut self);
    fn forward(&mut self);
    fn begin_rename(&mut self);
    fn cancel_rename(&mut self);
    fn commit_rename(&mut self);
    fn delete_selected(&mut self);
    fn duplicate_selected(&mut self);
    fn new_folder(&mut self);
    fn show_blank_menu(&mut self);
    fn close_preview(&mut self);
    fn clear_selection(&mut self);
    fn paste(&mut self);
    fn refresh(&mut self);
    fn select_all(&mut self);
    fn hover_splitter(&mut self);
    fn resize_sidebar(&mut self, event: vugra_core::Event);
    fn show_row1_menu(&mut self);
    fn show_row2_menu(&mut self);
    fn show_row3_menu(&mut self);
    fn show_row4_menu(&mut self);
    fn show_row5_menu(&mut self);
    fn show_row6_menu(&mut self);
    fn show_row7_menu(&mut self);
    fn show_row8_menu(&mut self);
    fn show_row9_menu(&mut self);
    fn show_row10_menu(&mut self);
    fn show_row11_menu(&mut self);
    fn show_row12_menu(&mut self);
    fn hover_row1(&mut self);
    fn hover_row2(&mut self);
    fn hover_row3(&mut self);
    fn hover_row4(&mut self);
    fn hover_row5(&mut self);
    fn hover_row6(&mut self);
    fn hover_row7(&mut self);
    fn hover_row8(&mut self);
    fn hover_row9(&mut self);
    fn hover_row10(&mut self);
    fn hover_row11(&mut self);
    fn hover_row12(&mut self);
    fn open_row1(&mut self);
    fn open_row2(&mut self);
    fn open_row3(&mut self);
    fn open_row4(&mut self);
    fn open_row5(&mut self);
    fn open_row6(&mut self);
    fn open_row7(&mut self);
    fn open_row8(&mut self);
    fn open_row9(&mut self);
    fn open_row10(&mut self);
    fn open_row11(&mut self);
    fn open_row12(&mut self);
    fn select_row(&mut self, index: usize);
}

pub struct FinderLiteAdapter<T> {
    inner: T,
}

impl<T> FinderLiteAdapter<T> {
    pub fn new(inner: T) -> Self {
        Self { inner }
    }

    pub fn inner(&self) -> &T {
        &self.inner
    }

    pub fn inner_mut(&mut self) -> &mut T {
        &mut self.inner
    }
}

impl<T: FinderLiteBindings> vugra_core::ComponentState for FinderLiteAdapter<T> {
    fn get_signal(&self, id: vugra_core::SignalId) -> vugra_core::Value {
        match id.0 {
            1 => self.inner.path(),
            2 => self.inner.status(),
            3 => self.inner.selected_summary(),
            13 => self.inner.documents_label(),
            14 => self.inner.downloads_label(),
            15 => self.inner.pictures_label(),
            16 => self.inner.documents_active(),
            17 => self.inner.downloads_active(),
            18 => self.inner.pictures_active(),
            19 => self.inner.search_query(),
            92 => self.inner.favorites_label(),
            93 => self.inner.workspace_label(),
            94 => self.inner.favorites_open(),
            95 => self.inner.workspace_open(),
            96 => self.inner.project_a_label(),
            97 => self.inner.project_b_label(),
            98 => self.inner.project_a_active(),
            99 => self.inner.project_b_active(),
            100 => self.inner.item_menu_open(),
            101 => self.inner.blank_menu_open(),
            102 => self.inner.rename_text(),
            103 => self.inner.preview_open(),
            104 => self.inner.preview_title(),
            105 => self.inner.preview_body(),
            106 => self.inner.sidebar_class(),
            107 => self.inner.splitter_class(),
            20..=91 => row_signal_field(id.0)
                .map(|(index, field)| self.inner.row_value(index, field))
                .unwrap_or(vugra_core::Value::None),
            _ => vugra_core::Value::None,
        }
    }
    fn set_signal(&mut self, _: vugra_core::SignalId, _: vugra_core::Value) {}

    fn call_method(&mut self, id: vugra_core::MethodId) {
        match id.0 {
            1 => self.inner.back(),
            2 => self.inner.select_row(0),
            3 => self.inner.select_row(1),
            4 => self.inner.select_row(2),
            5 => self.inner.open_documents(),
            6 => self.inner.open_downloads(),
            7 => self.inner.open_pictures(),
            8 => self.inner.select_previous(),
            9 => self.inner.select_next(),
            11 => self.inner.search_backspace(),
            12 => self.inner.search_clear(),
            13 => self.inner.open_selected(),
            14 => self.inner.open_parent(),
            15 => self.inner.toggle_favorites(),
            16 => self.inner.toggle_workspace(),
            17 => self.inner.open_project_a(),
            18 => self.inner.open_project_b(),
            19 => self.inner.dismiss_overlay(),
            20 => self.inner.forward(),
            33 => self.inner.begin_rename(),
            34 => self.inner.cancel_rename(),
            35 => self.inner.commit_rename(),
            36 => self.inner.delete_selected(),
            37 => self.inner.duplicate_selected(),
            38 => self.inner.new_folder(),
            39 => self.inner.show_blank_menu(),
            40 => self.inner.close_preview(),
            77 => self.inner.clear_selection(),
            78 => self.inner.paste(),
            79 => self.inner.refresh(),
            80 => self.inner.select_all(),
            81 => self.inner.hover_splitter(),
            24 => self.inner.select_row(3),
            25 => self.inner.select_row(4),
            26 => self.inner.select_row(5),
            27 => self.inner.select_row(6),
            28 => self.inner.select_row(7),
            29 => self.inner.select_row(8),
            30 => self.inner.select_row(9),
            31 => self.inner.select_row(10),
            32 => self.inner.select_row(11),
            41 => self.inner.show_row1_menu(),
            42 => self.inner.show_row2_menu(),
            43 => self.inner.show_row3_menu(),
            44 => self.inner.show_row4_menu(),
            45 => self.inner.show_row5_menu(),
            46 => self.inner.show_row6_menu(),
            47 => self.inner.show_row7_menu(),
            48 => self.inner.show_row8_menu(),
            49 => self.inner.show_row9_menu(),
            50 => self.inner.show_row10_menu(),
            51 => self.inner.show_row11_menu(),
            52 => self.inner.show_row12_menu(),
            53 => self.inner.hover_row1(),
            54 => self.inner.hover_row2(),
            55 => self.inner.hover_row3(),
            56 => self.inner.hover_row4(),
            57 => self.inner.hover_row5(),
            58 => self.inner.hover_row6(),
            59 => self.inner.hover_row7(),
            60 => self.inner.hover_row8(),
            61 => self.inner.hover_row9(),
            62 => self.inner.hover_row10(),
            63 => self.inner.hover_row11(),
            64 => self.inner.hover_row12(),
            65 => self.inner.open_row1(),
            66 => self.inner.open_row2(),
            67 => self.inner.open_row3(),
            68 => self.inner.open_row4(),
            69 => self.inner.open_row5(),
            70 => self.inner.open_row6(),
            71 => self.inner.open_row7(),
            72 => self.inner.open_row8(),
            73 => self.inner.open_row9(),
            74 => self.inner.open_row10(),
            75 => self.inner.open_row11(),
            76 => self.inner.open_row12(),
            _ => {}
        }
    }

    fn call_event_method(&mut self, id: vugra_core::MethodId, event: vugra_core::Event) {
        match id.0 {
            1 => self.inner.back(),
            2 => self.inner.select_row(0),
            3 => self.inner.select_row(1),
            4 => self.inner.select_row(2),
            5 => self.inner.open_documents(),
            6 => self.inner.open_downloads(),
            7 => self.inner.open_pictures(),
            8 => self.inner.select_previous(),
            9 => self.inner.select_next(),
            10 => self.inner.search_input(event),
            11 => self.inner.search_backspace(),
            12 => self.inner.search_clear(),
            13 => self.inner.open_selected(),
            14 => self.inner.open_parent(),
            15 => self.inner.toggle_favorites(),
            16 => self.inner.toggle_workspace(),
            17 => self.inner.open_project_a(),
            18 => self.inner.open_project_b(),
            19 => self.inner.dismiss_overlay(),
            20 => self.inner.forward(),
            33 => self.inner.begin_rename(),
            34 => self.inner.cancel_rename(),
            35 => self.inner.commit_rename(),
            36 => self.inner.delete_selected(),
            37 => self.inner.duplicate_selected(),
            38 => self.inner.new_folder(),
            39 => self.inner.show_blank_menu(),
            40 => self.inner.close_preview(),
            77 => self.inner.clear_selection(),
            78 => self.inner.paste(),
            79 => self.inner.refresh(),
            80 => self.inner.select_all(),
            81 => self.inner.hover_splitter(),
            82 => self.inner.resize_sidebar(event),
            24 => self.inner.select_row(3),
            25 => self.inner.select_row(4),
            26 => self.inner.select_row(5),
            27 => self.inner.select_row(6),
            28 => self.inner.select_row(7),
            29 => self.inner.select_row(8),
            30 => self.inner.select_row(9),
            31 => self.inner.select_row(10),
            32 => self.inner.select_row(11),
            41 => self.inner.show_row1_menu(),
            42 => self.inner.show_row2_menu(),
            43 => self.inner.show_row3_menu(),
            44 => self.inner.show_row4_menu(),
            45 => self.inner.show_row5_menu(),
            46 => self.inner.show_row6_menu(),
            47 => self.inner.show_row7_menu(),
            48 => self.inner.show_row8_menu(),
            49 => self.inner.show_row9_menu(),
            50 => self.inner.show_row10_menu(),
            51 => self.inner.show_row11_menu(),
            52 => self.inner.show_row12_menu(),
            53 => self.inner.hover_row1(),
            54 => self.inner.hover_row2(),
            55 => self.inner.hover_row3(),
            56 => self.inner.hover_row4(),
            57 => self.inner.hover_row5(),
            58 => self.inner.hover_row6(),
            59 => self.inner.hover_row7(),
            60 => self.inner.hover_row8(),
            61 => self.inner.hover_row9(),
            62 => self.inner.hover_row10(),
            63 => self.inner.hover_row11(),
            64 => self.inner.hover_row12(),
            65 => self.inner.open_row1(),
            66 => self.inner.open_row2(),
            67 => self.inner.open_row3(),
            68 => self.inner.open_row4(),
            69 => self.inner.open_row5(),
            70 => self.inner.open_row6(),
            71 => self.inner.open_row7(),
            72 => self.inner.open_row8(),
            73 => self.inner.open_row9(),
            74 => self.inner.open_row10(),
            75 => self.inner.open_row11(),
            76 => self.inner.open_row12(),
            _ => {}
        }
    }
}

fn row_signal_field(id: u32) -> Option<(usize, &'static str)> {
    if id < 20 {
        return None;
    }
    let offset = id - 20;
    let index = (offset / 6) as usize;
    if index >= 12 {
        return None;
    }
    let field = match offset % 6 {
        0 => "name",
        1 => "kind",
        2 => "modified",
        3 => "size",
        4 => "class",
        5 => "selected",
        _ => return None,
    };
    Some((index, field))
}
