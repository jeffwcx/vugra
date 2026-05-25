//! Language-neutral Vugra component IR shared by Rust kernel experiments.

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Component {
    pub name: String,
    pub signals: Vec<SignalDef>,
    pub methods: Vec<MethodDef>,
    pub rows: Vec<RowBinding>,
    pub sidebar: Vec<SidebarBinding>,
    pub sidebar_sections: Vec<SidebarSectionBinding>,
    pub toolbar: Option<ToolbarBinding>,
    pub splitter: Option<SplitterBinding>,
    pub search: Option<SearchBinding>,
    pub overlays: Option<OverlayBinding>,
}

impl Component {
    pub fn new(name: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            signals: Vec::new(),
            methods: Vec::new(),
            rows: Vec::new(),
            sidebar: Vec::new(),
            sidebar_sections: Vec::new(),
            toolbar: None,
            splitter: None,
            search: None,
            overlays: None,
        }
    }

    pub fn signal_id(&self, name: &str) -> Option<SignalId> {
        self.signals
            .iter()
            .find(|signal| signal.name == name)
            .map(|signal| signal.id)
    }

    pub fn method_id(&self, name: &str) -> Option<MethodId> {
        self.methods
            .iter()
            .find(|method| method.name == name)
            .map(|method| method.id)
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct SignalId(pub u32);

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct MethodId(pub u32);

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SignalDef {
    pub id: SignalId,
    pub name: String,
    pub kind: ValueKind,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct MethodDef {
    pub id: MethodId,
    pub name: String,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ValueKind {
    Bool,
    Number,
    String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct RowBinding {
    pub name: SignalId,
    pub kind: SignalId,
    pub modified: SignalId,
    pub size: SignalId,
    pub class: SignalId,
    pub selected: SignalId,
    pub select_method: MethodId,
    pub hover_method: MethodId,
    pub open_method: MethodId,
    pub context_menu_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SidebarBinding {
    pub label: SignalId,
    pub kind: SidebarItemKind,
    pub active: SignalId,
    pub open_method: MethodId,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SidebarItemKind {
    Folder,
    Download,
    Picture,
    Project,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SidebarSectionBinding {
    pub label: SignalId,
    pub open: SignalId,
    pub toggle_method: MethodId,
    pub items: Vec<SidebarBinding>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ToolbarBinding {
    pub back_method: MethodId,
    pub forward_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SplitterBinding {
    pub sidebar_class: SignalId,
    pub splitter_class: SignalId,
    pub hover_method: MethodId,
    pub drag_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SearchBinding {
    pub query: SignalId,
    pub input_method: MethodId,
    pub backspace_method: MethodId,
    pub clear_method: MethodId,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct OverlayBinding {
    pub item_menu_open: SignalId,
    pub blank_menu_open: SignalId,
    pub rename_text: SignalId,
    pub preview_open: SignalId,
    pub preview_title: SignalId,
    pub preview_body: SignalId,
    pub open_selected_method: MethodId,
    pub begin_rename_method: MethodId,
    pub cancel_rename_method: MethodId,
    pub commit_rename_method: MethodId,
    pub delete_selected_method: MethodId,
    pub duplicate_selected_method: MethodId,
    pub new_folder_method: MethodId,
    pub dismiss_overlay_method: MethodId,
    pub clear_selection_method: MethodId,
    pub show_blank_menu_method: MethodId,
    pub paste_method: MethodId,
    pub refresh_method: MethodId,
    pub close_preview_method: MethodId,
}

pub fn finder_lite_contract() -> Component {
    let mut component = Component::new("FinderLite");
    let mut signals = vec![
        signal(1, "path", ValueKind::String),
        signal(2, "status", ValueKind::String),
        signal(3, "selectedSummary", ValueKind::String),
        signal(13, "documentsLabel", ValueKind::String),
        signal(14, "downloadsLabel", ValueKind::String),
        signal(15, "picturesLabel", ValueKind::String),
        signal(16, "documentsActive", ValueKind::Bool),
        signal(17, "downloadsActive", ValueKind::Bool),
        signal(18, "picturesActive", ValueKind::Bool),
        signal(19, "searchQuery", ValueKind::String),
        signal(92, "favoritesLabel", ValueKind::String),
        signal(93, "workspaceLabel", ValueKind::String),
        signal(94, "favoritesOpen", ValueKind::Bool),
        signal(95, "workspaceOpen", ValueKind::Bool),
        signal(96, "projectALabel", ValueKind::String),
        signal(97, "projectBLabel", ValueKind::String),
        signal(98, "projectAActive", ValueKind::Bool),
        signal(99, "projectBActive", ValueKind::Bool),
        signal(100, "itemMenuOpen", ValueKind::Bool),
        signal(101, "blankMenuOpen", ValueKind::Bool),
        signal(102, "renameText", ValueKind::String),
        signal(103, "previewOpen", ValueKind::Bool),
        signal(104, "previewTitle", ValueKind::String),
        signal(105, "previewBody", ValueKind::String),
        signal(106, "sidebarClass", ValueKind::String),
        signal(107, "splitterClass", ValueKind::String),
    ];
    for row in 1..=12 {
        let base = row_signal_base(row);
        signals.push(signal(base, &format!("row{row}Name"), ValueKind::String));
        signals.push(signal(
            base + 1,
            &format!("row{row}Kind"),
            ValueKind::String,
        ));
        signals.push(signal(
            base + 2,
            &format!("row{row}Modified"),
            ValueKind::String,
        ));
        signals.push(signal(
            base + 3,
            &format!("row{row}Size"),
            ValueKind::String,
        ));
        signals.push(signal(
            base + 4,
            &format!("row{row}Class"),
            ValueKind::String,
        ));
        signals.push(signal(
            base + 5,
            &format!("row{row}Selected"),
            ValueKind::Bool,
        ));
    }
    component.signals = signals;
    let mut methods = vec![
        method(1, "Back"),
        method(2, "SelectRow1"),
        method(3, "SelectRow2"),
        method(4, "SelectRow3"),
        method(5, "OpenDocuments"),
        method(6, "OpenDownloads"),
        method(7, "OpenPictures"),
        method(8, "SelectPrevious"),
        method(9, "SelectNext"),
        method(10, "SearchInput"),
        method(11, "SearchBackspace"),
        method(12, "SearchClear"),
        method(13, "OpenSelected"),
        method(14, "OpenParent"),
        method(15, "ToggleFavorites"),
        method(16, "ToggleWorkspace"),
        method(17, "OpenProjectA"),
        method(18, "OpenProjectB"),
        method(19, "DismissOverlay"),
        method(20, "Forward"),
        method(33, "BeginRename"),
        method(34, "CancelRename"),
        method(35, "CommitRename"),
        method(36, "DeleteSelected"),
        method(37, "DuplicateSelected"),
        method(38, "NewFolder"),
        method(39, "ShowBlankMenu"),
        method(40, "ClosePreview"),
        method(77, "ClearSelection"),
        method(78, "Paste"),
        method(79, "Refresh"),
        method(80, "SelectAll"),
        method(81, "HoverSplitter"),
        method(82, "ResizeSidebar"),
    ];
    for row in 4..=12 {
        methods.push(method(row_select_method(row).0, &format!("SelectRow{row}")));
    }
    for row in 1..=12 {
        methods.push(method(
            row_context_menu_method(row).0,
            &format!("ShowRow{row}Menu"),
        ));
    }
    for row in 1..=12 {
        methods.push(method(row_hover_method(row).0, &format!("HoverRow{row}")));
    }
    for row in 1..=12 {
        methods.push(method(row_open_method(row).0, &format!("OpenRow{row}")));
    }
    component.methods = methods;
    component.rows = (1..=12)
        .map(|row| {
            let base = row_signal_base(row);
            RowBinding {
                name: SignalId(base),
                kind: SignalId(base + 1),
                modified: SignalId(base + 2),
                size: SignalId(base + 3),
                class: SignalId(base + 4),
                selected: SignalId(base + 5),
                select_method: row_select_method(row),
                hover_method: row_hover_method(row),
                open_method: row_open_method(row),
                context_menu_method: row_context_menu_method(row),
            }
        })
        .collect();
    component.sidebar = vec![
        SidebarBinding {
            label: SignalId(13),
            kind: SidebarItemKind::Folder,
            active: SignalId(16),
            open_method: MethodId(5),
        },
        SidebarBinding {
            label: SignalId(14),
            kind: SidebarItemKind::Download,
            active: SignalId(17),
            open_method: MethodId(6),
        },
        SidebarBinding {
            label: SignalId(15),
            kind: SidebarItemKind::Picture,
            active: SignalId(18),
            open_method: MethodId(7),
        },
    ];
    component.sidebar_sections = vec![
        SidebarSectionBinding {
            label: SignalId(92),
            open: SignalId(94),
            toggle_method: MethodId(15),
            items: vec![
                SidebarBinding {
                    label: SignalId(13),
                    kind: SidebarItemKind::Folder,
                    active: SignalId(16),
                    open_method: MethodId(5),
                },
                SidebarBinding {
                    label: SignalId(14),
                    kind: SidebarItemKind::Download,
                    active: SignalId(17),
                    open_method: MethodId(6),
                },
                SidebarBinding {
                    label: SignalId(15),
                    kind: SidebarItemKind::Picture,
                    active: SignalId(18),
                    open_method: MethodId(7),
                },
            ],
        },
        SidebarSectionBinding {
            label: SignalId(93),
            open: SignalId(95),
            toggle_method: MethodId(16),
            items: vec![
                SidebarBinding {
                    label: SignalId(96),
                    kind: SidebarItemKind::Project,
                    active: SignalId(98),
                    open_method: MethodId(17),
                },
                SidebarBinding {
                    label: SignalId(97),
                    kind: SidebarItemKind::Folder,
                    active: SignalId(99),
                    open_method: MethodId(18),
                },
            ],
        },
    ];
    component.search = Some(SearchBinding {
        query: SignalId(19),
        input_method: MethodId(10),
        backspace_method: MethodId(11),
        clear_method: MethodId(12),
    });
    component.toolbar = Some(ToolbarBinding {
        back_method: MethodId(1),
        forward_method: MethodId(20),
    });
    component.splitter = Some(SplitterBinding {
        sidebar_class: SignalId(106),
        splitter_class: SignalId(107),
        hover_method: MethodId(81),
        drag_method: MethodId(82),
    });
    component.overlays = Some(OverlayBinding {
        item_menu_open: SignalId(100),
        blank_menu_open: SignalId(101),
        rename_text: SignalId(102),
        preview_open: SignalId(103),
        preview_title: SignalId(104),
        preview_body: SignalId(105),
        open_selected_method: MethodId(13),
        begin_rename_method: MethodId(33),
        cancel_rename_method: MethodId(34),
        commit_rename_method: MethodId(35),
        delete_selected_method: MethodId(36),
        duplicate_selected_method: MethodId(37),
        new_folder_method: MethodId(38),
        dismiss_overlay_method: MethodId(19),
        clear_selection_method: MethodId(77),
        show_blank_menu_method: MethodId(39),
        paste_method: MethodId(78),
        refresh_method: MethodId(79),
        close_preview_method: MethodId(40),
    });
    component
}

fn row_signal_base(row: u32) -> u32 {
    20 + (row - 1) * 6
}

fn row_select_method(row: u32) -> MethodId {
    match row {
        1 => MethodId(2),
        2 => MethodId(3),
        3 => MethodId(4),
        _ => MethodId(20 + row),
    }
}

fn row_context_menu_method(row: u32) -> MethodId {
    MethodId(40 + row)
}

fn row_hover_method(row: u32) -> MethodId {
    MethodId(52 + row)
}

fn row_open_method(row: u32) -> MethodId {
    MethodId(64 + row)
}

fn signal(id: u32, name: &str, kind: ValueKind) -> SignalDef {
    SignalDef {
        id: SignalId(id),
        name: name.to_string(),
        kind,
    }
}

fn method(id: u32, name: &str) -> MethodDef {
    MethodDef {
        id: MethodId(id),
        name: name.to_string(),
    }
}
