//! Platform-neutral layout boxes for the Rust kernel path.

use vugra_core::Frame;
pub use vugra_core::RowVisualState;

const TOOLBAR_HEIGHT: f32 = 52.0;
const STATUSBAR_HEIGHT: f32 = 28.0;
const SIDEBAR_WIDTH: f32 = 240.0;
const SPLITTER_WIDTH: f32 = 6.0;
const FILE_LIST_PADDING: f32 = 6.0;
const FILE_HEADER_Y: f32 = TOOLBAR_HEIGHT;
const FILE_HEADER_HEIGHT: f32 = 34.0;
const FILE_HEADER_PADDING: f32 = 8.0;
const FILE_ROW_START_Y: f32 = FILE_HEADER_Y + FILE_HEADER_HEIGHT + FILE_LIST_PADDING;
const FILE_ROW_HEIGHT: f32 = 30.0;
const FILE_ROW_GAP: f32 = 0.0;
const FILE_ROW_PADDING: f32 = 6.0;
const FILE_ROW_GAP_X: f32 = 10.0;
const FILE_DATE_WIDTH: f32 = 150.0;
const FILE_SIZE_WIDTH: f32 = 90.0;
const FILE_ICON_WIDTH: f32 = 18.0;
const FILE_ICON_NAME_GAP: f32 = 8.0;
const SIDEBAR_PADDING: f32 = 12.0;
const SIDEBAR_SECTION_HEIGHT: f32 = 28.0;
const SIDEBAR_SECTION_GAP: f32 = 6.0;
const SIDEBAR_TREE_GAP: f32 = 3.0;
const SIDEBAR_ITEM_HEIGHT: f32 = 28.0;
const SIDEBAR_ITEM_PADDING: f32 = 6.0;
const SIDEBAR_ICON_WIDTH: f32 = 18.0;
const SIDEBAR_ICON_TEXT_GAP: f32 = 8.0;
const TOOLBAR_PADDING_X: f32 = 10.0;
const TOOLBAR_PADDING_RIGHT: f32 = 10.0;
const NAV_BUTTON_WIDTH: f32 = 34.0;
const NAV_BUTTON_HEIGHT: f32 = 30.0;
const NAV_BUTTON_GAP: f32 = 8.0;
const SEARCH_WIDTH: f32 = 220.0;
const OVERLAY_PADDING: f32 = 14.0;
const MENU_WIDTH: f32 = 180.0;
const MENU_PADDING: f32 = 6.0;
const MENU_ITEM_HEIGHT: f32 = 30.0;
const MENU_ITEM_WIDTH: f32 = MENU_WIDTH - MENU_PADDING * 2.0;
const DIALOG_WIDTH: f32 = 360.0;
const DIALOG_HEIGHT: f32 = 160.0;
const DIALOG_PADDING: f32 = 14.0;
const PRIMARY_BUTTON_WIDTH: f32 = 90.0;
const PRIMARY_BUTTON_HEIGHT: f32 = 32.0;
const FIXED_TEXT_CHAR_WIDTH: f32 = 8.0;

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Constraints {
    pub width: f32,
    pub height: f32,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Rect {
    pub x: f32,
    pub y: f32,
    pub width: f32,
    pub height: f32,
}

#[derive(Clone, Debug, PartialEq)]
pub struct LayoutBox {
    pub id: String,
    pub role: String,
    pub text: String,
    pub rect: Rect,
    pub overflow: Overflow,
    pub scroll_y: f32,
    pub selected: bool,
    pub visual_state: RowVisualState,
    pub method: Option<vugra_core::MethodId>,
    pub handlers: EventHandlers,
    pub children: Vec<LayoutBox>,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum Overflow {
    #[default]
    Visible,
    Hidden,
    Scroll,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct EventHandlers {
    pub capture: Option<vugra_core::MethodId>,
    pub target: Option<vugra_core::MethodId>,
    pub bubble: Option<vugra_core::MethodId>,
    pub hover: Option<vugra_core::MethodId>,
    pub drag: Option<vugra_core::MethodId>,
    pub double_click: Option<vugra_core::MethodId>,
    pub context_menu: Option<vugra_core::MethodId>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct LayoutTree {
    pub root: LayoutBox,
}

pub fn layout_frame(frame: &Frame, constraints: Constraints) -> LayoutTree {
    let search_text = frame.search_query.clone();
    let sidebar_width = sidebar_width_for_frame(frame);
    let file_pane_x = sidebar_width + SPLITTER_WIDTH;
    let mut sidebar = scroll_box_node(
        "sidebar",
        sidebar_role_for_frame(frame),
        "",
        0.0,
        TOOLBAR_HEIGHT,
        sidebar_width,
        constraints.height - TOOLBAR_HEIGHT - STATUSBAR_HEIGHT,
    );
    let mut file_pane = scroll_box_node(
        "file-pane",
        "file-pane",
        "",
        file_pane_x,
        TOOLBAR_HEIGHT,
        file_pane_width(constraints.width, file_pane_x),
        constraints.height - TOOLBAR_HEIGHT - STATUSBAR_HEIGHT,
    );
    file_pane.overflow = Overflow::Hidden;
    let mut file_list = scroll_box_node(
        "file-list",
        "file-list",
        "",
        file_pane_x,
        FILE_ROW_START_Y - FILE_LIST_PADDING,
        file_pane_width(constraints.width, file_pane_x),
        constraints.height - TOOLBAR_HEIGHT - STATUSBAR_HEIGHT - FILE_HEADER_HEIGHT,
    );

    let mut toolbar = box_node(
        "toolbar",
        "toolbar",
        "",
        0.0,
        0.0,
        constraints.width,
        TOOLBAR_HEIGHT,
    );
    let nav_y = 11.0;
    let back_x = TOOLBAR_PADDING_X;
    let forward_x = back_x + NAV_BUTTON_WIDTH + NAV_BUTTON_GAP;
    if let Some(toolbar_binding) = &frame.toolbar {
        toolbar.children.push(nav_button_box(
            "nav-back",
            "back-icon",
            back_x,
            nav_y,
            toolbar_binding.back_method,
        ));
        toolbar.children.push(nav_button_box(
            "nav-forward",
            "forward-icon",
            forward_x,
            nav_y,
            toolbar_binding.forward_method,
        ));
    }
    let path_x = if frame.toolbar.is_some() {
        forward_x + NAV_BUTTON_WIDTH + NAV_BUTTON_GAP
    } else {
        236.0
    };
    let search_x = constraints.width - SEARCH_WIDTH - TOOLBAR_PADDING_RIGHT;
    let path_width = (search_x - path_x - NAV_BUTTON_GAP).max(0.0);
    let statusbar_y = constraints.height - STATUSBAR_HEIGHT;
    let mut statusbar = box_node(
        "statusbar",
        "statusbar",
        "",
        0.0,
        statusbar_y,
        constraints.width,
        STATUSBAR_HEIGHT,
    );
    statusbar.children.push(box_node(
        "status-text",
        "status-text",
        &frame.status,
        6.0,
        statusbar_y + 6.0,
        (constraints.width * 0.5 - 12.0).max(0.0),
        18.0,
    ));
    statusbar.children.push(box_node(
        "selected-summary",
        "status-text",
        &frame.selected_summary,
        selected_summary_x(constraints.width, &frame.selected_summary),
        statusbar_y + 6.0,
        fixed_text_width(&frame.selected_summary),
        18.0,
    ));

    let mut children = vec![
        toolbar,
        statusbar,
        splitter_box(
            frame,
            sidebar_width,
            constraints.height - TOOLBAR_HEIGHT - STATUSBAR_HEIGHT,
        ),
    ];
    children[0].children.push(box_node(
        "path",
        "path",
        &frame.path,
        path_x,
        nav_y,
        path_width,
        NAV_BUTTON_HEIGHT,
    ));
    children[0].children.push(box_node(
        "search",
        "search",
        &search_text,
        search_x,
        nav_y,
        SEARCH_WIDTH,
        NAV_BUTTON_HEIGHT,
    ));
    if let Some(method) = frame.search_input_method {
        if let Some(search) = children[0]
            .children
            .iter_mut()
            .find(|child| child.id == "search")
        {
            search.method = Some(method);
            search.handlers.target = Some(method);
        }
    }
    if let Some(method) = frame.overlays.clear_selection_method {
        file_pane.method = Some(method);
        file_pane.handlers.target = Some(method);
    }
    if let Some(method) = frame.overlays.show_blank_menu_method {
        file_pane.handlers.context_menu = Some(method);
    }

    if frame.sidebar_sections.is_empty() {
        for (index, item) in frame.sidebar.iter().enumerate() {
            sidebar.children.push(sidebar_item_box(
                item,
                SIDEBAR_PADDING + index as f32 * (SIDEBAR_ITEM_HEIGHT + SIDEBAR_TREE_GAP),
            ));
        }
    } else {
        let mut y = SIDEBAR_PADDING;
        for section in &frame.sidebar_sections {
            let mut section_node = LayoutBox {
                id: format!("sidebar-section-{}", slug(&section.label)),
                role: "sidebar-section".to_string(),
                text: String::new(),
                rect: Rect {
                    x: SIDEBAR_PADDING,
                    y,
                    width: (sidebar_width - SIDEBAR_PADDING * 2.0).max(0.0),
                    height: SIDEBAR_SECTION_HEIGHT,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: RowVisualState::Normal,
                method: Some(section.toggle_method),
                handlers: EventHandlers::target(section.toggle_method),
                children: Vec::new(),
            };
            section_node.children.push(box_node(
                &format!("{}-chevron", section_node.id),
                if section.open {
                    "chevron-down-icon"
                } else {
                    "chevron-right-icon"
                },
                "",
                SIDEBAR_PADDING,
                y + (SIDEBAR_SECTION_HEIGHT - 14.0) / 2.0,
                14.0,
                14.0,
            ));
            section_node.children.push(box_node(
                &format!("{}-label", section_node.id),
                "sidebar-section-label",
                &section.label,
                SIDEBAR_PADDING + 14.0 + SIDEBAR_SECTION_GAP,
                y + (SIDEBAR_SECTION_HEIGHT - 18.0) / 2.0,
                (sidebar_width - SIDEBAR_PADDING * 2.0 - 14.0 - SIDEBAR_SECTION_GAP).max(0.0),
                18.0,
            ));
            sidebar.children.push(section_node);
            y += SIDEBAR_SECTION_HEIGHT;
            if section.open {
                for (index, item) in section.items.iter().enumerate() {
                    if index > 0 {
                        y += SIDEBAR_TREE_GAP;
                    }
                    sidebar
                        .children
                        .push(sidebar_item_box_for_width(item, y, sidebar_width));
                    y += SIDEBAR_ITEM_HEIGHT;
                }
            }
        }
    }

    file_pane
        .children
        .push(file_header_box(constraints.width, file_pane_x));

    let row_width = file_row_width(constraints.width, file_pane_x);
    let mut list_index = 0;
    for (index, row) in frame.rows.iter().enumerate() {
        let row_y = file_row_y(list_index);
        file_list
            .children
            .push(row_box(index, row, row_width, file_pane_x, row_y));
        list_index += 1;
        if row.visual_state == RowVisualState::Editing && !frame.overlays.rename_text.is_empty() {
            let mut rename = rename_inline_box(
                &frame.overlays.rename_text,
                row_width,
                file_pane_x,
                file_row_y(list_index),
            );
            if let Some(method) = frame.overlays.commit_rename_method {
                rename.method = Some(method);
                rename.handlers.target = Some(method);
            }
            file_list.children.push(rename);
            list_index += 1;
        }
    }

    if frame.rows.is_empty() {
        for (index, value) in frame.values.iter().enumerate() {
            file_list.children.push(LayoutBox {
                id: format!("value{}", index + 1),
                role: "value".to_string(),
                text: format!("{}: {}", value.name, value.value),
                rect: Rect {
                    x: file_pane_x + 18.0,
                    y: 100.0 + index as f32 * 28.0,
                    width: 420.0,
                    height: 24.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: Vec::new(),
            });
        }
    }

    file_pane.children.push(file_list);
    children.push(sidebar);
    children.push(file_pane);
    append_overlay_boxes(frame, constraints, &mut children);

    LayoutTree {
        root: LayoutBox {
            id: "finder-lite".to_string(),
            role: "window".to_string(),
            text: String::new(),
            rect: Rect {
                x: 0.0,
                y: 0.0,
                width: constraints.width,
                height: constraints.height,
            },
            overflow: Overflow::Visible,
            scroll_y: 0.0,
            selected: false,
            visual_state: RowVisualState::Normal,
            method: None,
            handlers: EventHandlers::default(),
            children,
        },
    }
}

fn append_overlay_boxes(frame: &Frame, constraints: Constraints, children: &mut Vec<LayoutBox>) {
    let overlay_rect = file_pane_rect(frame, constraints);
    if frame.overlays.item_menu_open {
        let mut overlay = overlay_box(
            "item-overlay",
            overlay_rect,
            frame.overlays.dismiss_overlay_method,
        );
        let menu_x = overlay_rect.x + OVERLAY_PADDING;
        let menu_y = overlay_rect.y + OVERLAY_PADDING;
        let mut menu = box_node(
            "item-menu",
            "menu",
            "",
            menu_x,
            menu_y,
            MENU_WIDTH,
            MENU_PADDING * 2.0 + MENU_ITEM_HEIGHT * 4.0,
        );
        let item_x = menu_x + MENU_PADDING;
        let item_y = menu_y + MENU_PADDING;
        if let Some(method) = frame.overlays.dismiss_overlay_method {
            overlay.method = Some(method);
        }
        push_menu_item(
            &mut menu,
            "item-menu-open",
            "Open",
            item_x,
            item_y,
            frame.overlays.open_selected_method,
        );
        push_menu_item(
            &mut menu,
            "item-menu-rename",
            "Rename",
            item_x,
            item_y + MENU_ITEM_HEIGHT,
            frame.overlays.begin_rename_method,
        );
        push_menu_item(
            &mut menu,
            "item-menu-delete",
            "Delete",
            item_x,
            item_y + MENU_ITEM_HEIGHT * 2.0,
            frame.overlays.delete_selected_method,
        );
        push_menu_item(
            &mut menu,
            "item-menu-duplicate",
            "Duplicate",
            item_x,
            item_y + MENU_ITEM_HEIGHT * 3.0,
            frame.overlays.duplicate_selected_method,
        );
        overlay.children.push(menu);
        children.push(overlay);
    }
    if frame.overlays.blank_menu_open {
        let mut overlay = overlay_box(
            "blank-overlay",
            overlay_rect,
            frame.overlays.dismiss_overlay_method,
        );
        let menu_x = overlay_rect.x + OVERLAY_PADDING;
        let menu_y = overlay_rect.y + OVERLAY_PADDING;
        let mut menu = box_node(
            "blank-menu",
            "menu",
            "",
            menu_x,
            menu_y,
            MENU_WIDTH,
            MENU_PADDING * 2.0 + MENU_ITEM_HEIGHT * 3.0,
        );
        let item_x = menu_x + MENU_PADDING;
        let item_y = menu_y + MENU_PADDING;
        if let Some(method) = frame.overlays.dismiss_overlay_method {
            overlay.method = Some(method);
        }
        push_menu_item(
            &mut menu,
            "blank-menu-new-folder",
            "New Folder",
            item_x,
            item_y,
            frame.overlays.new_folder_method,
        );
        push_menu_item(
            &mut menu,
            "blank-menu-paste",
            "Paste",
            item_x,
            item_y + MENU_ITEM_HEIGHT,
            frame.overlays.paste_method,
        );
        push_menu_item(
            &mut menu,
            "blank-menu-refresh",
            "Refresh",
            item_x,
            item_y + MENU_ITEM_HEIGHT * 2.0,
            frame.overlays.refresh_method,
        );
        overlay.children.push(menu);
        children.push(overlay);
    }
    if frame.overlays.preview_open {
        let mut layer = box_node(
            "preview-layer",
            "dialog-layer",
            "",
            overlay_rect.x,
            overlay_rect.y,
            overlay_rect.width,
            overlay_rect.height,
        );
        let dialog_x = overlay_rect.x + (overlay_rect.width - DIALOG_WIDTH).max(0.0) / 2.0;
        let dialog_y = overlay_rect.y + (overlay_rect.height - DIALOG_HEIGHT).max(0.0) / 2.0;
        let mut dialog = box_node(
            "preview-dialog",
            "dialog",
            "",
            dialog_x,
            dialog_y,
            DIALOG_WIDTH,
            DIALOG_HEIGHT,
        );
        dialog.children.push(box_node(
            "preview-title",
            "dialog-title",
            &frame.overlays.preview_title,
            dialog_x + DIALOG_PADDING,
            dialog_y + DIALOG_PADDING,
            DIALOG_WIDTH - DIALOG_PADDING * 2.0,
            24.0,
        ));
        dialog.children.push(box_node(
            "preview-body",
            "preview-copy",
            &frame.overlays.preview_body,
            dialog_x + DIALOG_PADDING,
            dialog_y + DIALOG_PADDING + 34.0,
            DIALOG_WIDTH - DIALOG_PADDING * 2.0,
            48.0,
        ));
        if let Some(method) = frame.overlays.close_preview_method {
            let mut done = box_node(
                "preview-done",
                "primary-button",
                "Done",
                dialog_x + DIALOG_WIDTH - DIALOG_PADDING - PRIMARY_BUTTON_WIDTH,
                dialog_y + DIALOG_HEIGHT - DIALOG_PADDING - PRIMARY_BUTTON_HEIGHT,
                PRIMARY_BUTTON_WIDTH,
                PRIMARY_BUTTON_HEIGHT,
            );
            done.method = Some(method);
            done.handlers.target = Some(method);
            dialog.children.push(done);
        }
        layer.children.push(dialog);
        children.push(layer);
    }
}

fn file_pane_rect(frame: &Frame, constraints: Constraints) -> Rect {
    let file_pane_x = sidebar_width_for_frame(frame) + SPLITTER_WIDTH;
    Rect {
        x: file_pane_x,
        y: TOOLBAR_HEIGHT,
        width: file_pane_width(constraints.width, file_pane_x),
        height: constraints.height - TOOLBAR_HEIGHT - STATUSBAR_HEIGHT,
    }
}

fn overlay_box(id: &str, rect: Rect, dismiss_method: Option<vugra_core::MethodId>) -> LayoutBox {
    let mut overlay = box_node(id, "overlay", "", rect.x, rect.y, rect.width, rect.height);
    if let Some(method) = dismiss_method {
        overlay.method = Some(method);
        overlay.handlers.target = Some(method);
        overlay.handlers.bubble = Some(method);
    }
    overlay
}

fn nav_button_box(
    id: &str,
    icon_role: &str,
    x: f32,
    y: f32,
    method: vugra_core::MethodId,
) -> LayoutBox {
    let mut node = box_node(
        id,
        "nav-button",
        "",
        x,
        y,
        NAV_BUTTON_WIDTH,
        NAV_BUTTON_HEIGHT,
    );
    node.method = Some(method);
    node.handlers = EventHandlers::target(method);
    node.children.push(box_node(
        &format!("{id}-icon"),
        icon_role,
        "",
        x + 8.0,
        y,
        18.0,
        NAV_BUTTON_HEIGHT,
    ));
    node
}

fn push_menu_item(
    parent: &mut LayoutBox,
    id: &str,
    label: &str,
    x: f32,
    y: f32,
    method: Option<vugra_core::MethodId>,
) {
    let mut item = box_node(
        id,
        "menu-item",
        label,
        x,
        y,
        MENU_ITEM_WIDTH,
        MENU_ITEM_HEIGHT,
    );
    if let Some(method) = method {
        item.method = Some(method);
        item.handlers.target = Some(method);
    }
    parent.children.push(item);
}

fn sidebar_role_for_frame(frame: &Frame) -> &str {
    frame
        .splitter
        .as_ref()
        .map(|splitter| splitter.sidebar_class.as_str())
        .filter(|class| !class.is_empty())
        .unwrap_or("sidebar")
}

fn sidebar_width_for_frame(frame: &Frame) -> f32 {
    match sidebar_role_for_frame(frame) {
        "sidebar-200" => 200.0,
        "sidebar-280" => 280.0,
        "sidebar-320" => 320.0,
        _ => SIDEBAR_WIDTH,
    }
}

fn splitter_box(frame: &Frame, x: f32, height: f32) -> LayoutBox {
    let mut splitter = box_node(
        "splitter",
        frame
            .splitter
            .as_ref()
            .map(|splitter| splitter.splitter_class.as_str())
            .filter(|class| !class.is_empty())
            .unwrap_or("splitter"),
        "",
        x,
        TOOLBAR_HEIGHT,
        SPLITTER_WIDTH,
        height,
    );
    if let Some(binding) = frame.splitter.as_ref() {
        splitter.handlers.hover = Some(binding.hover_method);
        splitter.handlers.drag = Some(binding.drag_method);
    }
    splitter
}

fn file_pane_width(root_width: f32, file_pane_x: f32) -> f32 {
    (root_width - file_pane_x).max(0.0)
}

fn file_row_width(root_width: f32, file_pane_x: f32) -> f32 {
    (file_pane_width(root_width, file_pane_x) - FILE_LIST_PADDING * 2.0).max(0.0)
}

fn header_inner_width(root_width: f32, file_pane_x: f32) -> f32 {
    (file_pane_width(root_width, file_pane_x) - FILE_HEADER_PADDING * 2.0).max(0.0)
}

fn header_name_width(root_width: f32, file_pane_x: f32) -> f32 {
    (header_inner_width(root_width, file_pane_x) - FILE_DATE_WIDTH - FILE_SIZE_WIDTH).max(0.0)
}

fn header_size_x(root_width: f32, file_pane_x: f32) -> f32 {
    file_pane_x + FILE_HEADER_PADDING + header_inner_width(root_width, file_pane_x)
        - FILE_SIZE_WIDTH
}

fn header_date_x(root_width: f32, file_pane_x: f32) -> f32 {
    header_size_x(root_width, file_pane_x) - FILE_DATE_WIDTH
}

fn file_header_box(root_width: f32, file_pane_x: f32) -> LayoutBox {
    let mut header = box_node(
        "file-header",
        "file-header",
        "",
        file_pane_x,
        FILE_HEADER_Y,
        file_pane_width(root_width, file_pane_x),
        FILE_HEADER_HEIGHT,
    );
    header.children.push(box_node(
        "header-name",
        "column-header",
        "Name",
        file_pane_x + FILE_HEADER_PADDING,
        FILE_HEADER_Y + FILE_HEADER_PADDING,
        header_name_width(root_width, file_pane_x),
        18.0,
    ));
    header.children.push(box_node(
        "header-kind",
        "column-header",
        "Modified",
        header_date_x(root_width, file_pane_x),
        FILE_HEADER_Y + FILE_HEADER_PADDING,
        FILE_DATE_WIDTH,
        18.0,
    ));
    header.children.push(box_node(
        "header-size",
        "column-header",
        "Size",
        header_size_x(root_width, file_pane_x),
        FILE_HEADER_Y + FILE_HEADER_PADDING,
        FILE_SIZE_WIDTH,
        18.0,
    ));
    header
}

fn row_box(
    index: usize,
    row: &vugra_core::FrameRow,
    row_width: f32,
    file_pane_x: f32,
    y: f32,
) -> LayoutBox {
    let id = format!("row{}", index + 1);
    let row_x = file_pane_x + FILE_LIST_PADDING;
    let content_x = row_x + FILE_ROW_PADDING;
    let size_x = row_x + row_width - FILE_ROW_PADDING - FILE_SIZE_WIDTH;
    let date_x = size_x - FILE_ROW_GAP_X - FILE_DATE_WIDTH;
    let name_x = content_x + FILE_ICON_WIDTH + FILE_ICON_NAME_GAP;
    let name_width = date_x - FILE_ROW_GAP_X - name_x;
    LayoutBox {
        id: id.clone(),
        role: "row".to_string(),
        text: String::new(),
        rect: Rect {
            x: row_x,
            y,
            width: row_width,
            height: FILE_ROW_HEIGHT,
        },
        overflow: Overflow::Visible,
        scroll_y: 0.0,
        selected: row.selected,
        visual_state: row.visual_state,
        method: Some(row.select_method),
        handlers: EventHandlers {
            target: Some(row.select_method),
            hover: Some(row.hover_method),
            double_click: Some(row.open_method),
            context_menu: Some(row.context_menu_method),
            ..EventHandlers::default()
        },
        children: vec![
            box_node(
                &format!("{id}-icon"),
                if row.kind == "folder" {
                    "folder-icon"
                } else {
                    "file-icon"
                },
                "",
                content_x,
                y + 6.0,
                FILE_ICON_WIDTH,
                18.0,
            ),
            box_node(
                &format!("{id}-name"),
                "row-name-cell",
                &row.name,
                name_x,
                y + 6.0,
                name_width,
                18.0,
            ),
            box_node(
                &format!("{id}-modified"),
                "row-date-cell",
                &row.modified,
                date_x,
                y + 6.0,
                FILE_DATE_WIDTH,
                18.0,
            ),
            box_node(
                &format!("{id}-size"),
                "row-size-cell",
                &row.size,
                size_x,
                y + 6.0,
                FILE_SIZE_WIDTH,
                18.0,
            ),
        ],
    }
}

fn file_row_y(index: usize) -> f32 {
    FILE_ROW_START_Y + index as f32 * (FILE_ROW_HEIGHT + FILE_ROW_GAP)
}

fn rename_inline_box(text: &str, row_width: f32, file_pane_x: f32, y: f32) -> LayoutBox {
    box_node(
        "rename-inline",
        "rename-inline",
        text,
        file_pane_x + FILE_LIST_PADDING,
        y,
        row_width,
        FILE_ROW_HEIGHT,
    )
}

fn sidebar_item_box(item: &vugra_core::FrameSidebarItem, y: f32) -> LayoutBox {
    sidebar_item_box_for_width(item, y, SIDEBAR_WIDTH)
}

fn sidebar_item_box_for_width(
    item: &vugra_core::FrameSidebarItem,
    y: f32,
    sidebar_width: f32,
) -> LayoutBox {
    let id = format!("sidebar-{}", slug(&item.label));
    let item_x = SIDEBAR_PADDING;
    let icon_x = item_x + SIDEBAR_ITEM_PADDING;
    let label_x = icon_x + SIDEBAR_ICON_WIDTH + SIDEBAR_ICON_TEXT_GAP;
    let mut node = LayoutBox {
        id: id.clone(),
        role: "sidebar-item".to_string(),
        text: String::new(),
        rect: Rect {
            x: item_x,
            y,
            width: (sidebar_width - SIDEBAR_PADDING * 2.0).max(0.0),
            height: SIDEBAR_ITEM_HEIGHT,
        },
        overflow: Overflow::Visible,
        scroll_y: 0.0,
        selected: item.active,
        visual_state: RowVisualState::Normal,
        method: Some(item.open_method),
        handlers: EventHandlers::target(item.open_method),
        children: Vec::new(),
    };
    node.children.push(box_node(
        &format!("{id}-icon"),
        sidebar_item_icon_role(item.kind),
        "",
        icon_x,
        y + (SIDEBAR_ITEM_HEIGHT - SIDEBAR_ICON_WIDTH) / 2.0,
        SIDEBAR_ICON_WIDTH,
        18.0,
    ));
    node.children.push(box_node(
        &format!("{id}-label"),
        "sidebar-item-label",
        &item.label,
        label_x,
        y + (SIDEBAR_ITEM_HEIGHT - 18.0) / 2.0,
        (sidebar_width - SIDEBAR_PADDING - label_x).max(0.0),
        18.0,
    ));
    node
}

fn sidebar_item_icon_role(kind: vugra_core::SidebarItemKind) -> &'static str {
    match kind {
        vugra_core::SidebarItemKind::Folder => "folder-icon",
        vugra_core::SidebarItemKind::Download => "download-icon",
        vugra_core::SidebarItemKind::Picture => "picture-icon",
        vugra_core::SidebarItemKind::Project => "project-icon",
    }
}

fn slug(value: &str) -> String {
    value
        .chars()
        .map(|ch| {
            if ch.is_ascii_alphanumeric() {
                ch.to_ascii_lowercase()
            } else {
                '-'
            }
        })
        .collect()
}

impl EventHandlers {
    pub fn target(method: vugra_core::MethodId) -> Self {
        Self {
            target: Some(method),
            ..Self::default()
        }
    }
}

fn box_node(
    id: &str,
    role: &str,
    text: &str,
    x: f32,
    y: f32,
    width: f32,
    height: f32,
) -> LayoutBox {
    LayoutBox {
        id: id.to_string(),
        role: role.to_string(),
        text: text.to_string(),
        rect: Rect {
            x,
            y,
            width,
            height,
        },
        overflow: Overflow::Visible,
        scroll_y: 0.0,
        selected: false,
        visual_state: RowVisualState::Normal,
        method: None,
        handlers: EventHandlers::default(),
        children: Vec::new(),
    }
}

fn scroll_box_node(
    id: &str,
    role: &str,
    text: &str,
    x: f32,
    y: f32,
    width: f32,
    height: f32,
) -> LayoutBox {
    let mut node = box_node(id, role, text, x, y, width, height);
    node.overflow = Overflow::Scroll;
    node
}

fn selected_summary_x(width: f32, text: &str) -> f32 {
    (width - 6.0 - fixed_text_width(text)).max(6.0)
}

fn fixed_text_width(text: &str) -> f32 {
    text.chars().count() as f32 * FIXED_TEXT_CHAR_WIDTH
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_core::{Frame, FrameOverlays, FrameRow, FrameSplitter};

    #[test]
    fn layout_preserves_frame_text_in_boxes() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: vec![vugra_core::FrameSidebarItem {
                    label: "Documents".to_string(),
                    kind: vugra_core::SidebarItemKind::Folder,
                    active: true,
                    open_method: vugra_core::MethodId(5),
                }],
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: vec![FrameRow {
                    name: "Design".to_string(),
                    kind: "folder".to_string(),
                    modified: "--".to_string(),
                    size: "--".to_string(),
                    class: "file-row-selected".to_string(),
                    selected: true,
                    visual_state: RowVisualState::Selected,
                    select_method: vugra_core::MethodId(2),
                    hover_method: vugra_core::MethodId(53),
                    open_method: vugra_core::MethodId(65),
                    context_menu_method: vugra_core::MethodId(41),
                }],
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );
        assert_eq!(tree.root.rect.width, 800.0);
        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        let file_list = file_pane
            .children
            .iter()
            .find(|child| child.id == "file-list")
            .expect("file-list");
        let row = file_list
            .children
            .iter()
            .find(|child| child.id == "row1")
            .expect("row1");
        assert_eq!(row.text, "");
        assert!(row.selected);
        assert!(row.children.iter().any(|child| {
            child.id == "row1-icon" && child.role == "folder-icon" && child.text.is_empty()
        }));
        assert!(row
            .children
            .iter()
            .any(|child| child.id == "row1-name" && child.text == "Design"));
        assert!(row
            .children
            .iter()
            .any(|child| child.id == "row1-modified" && child.text == "--"));
        assert!(row
            .children
            .iter()
            .any(|child| child.id == "row1-size" && child.text == "--"));
    }

    #[test]
    fn layout_matches_go_finder_file_row_grid_contract() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: vec![FrameRow {
                    name: "Roadmap.md".to_string(),
                    kind: "file".to_string(),
                    modified: "Today".to_string(),
                    size: "12 KB".to_string(),
                    class: "file-row".to_string(),
                    selected: false,
                    visual_state: RowVisualState::Normal,
                    select_method: vugra_core::MethodId(2),
                    hover_method: vugra_core::MethodId(53),
                    open_method: vugra_core::MethodId(65),
                    context_menu_method: vugra_core::MethodId(41),
                }],
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        assert_eq!(
            file_pane.rect,
            Rect {
                x: 246.0,
                y: 52.0,
                width: 554.0,
                height: 520.0,
            }
        );

        let file_header = file_pane
            .children
            .iter()
            .find(|child| child.id == "file-header")
            .expect("file-header");
        assert_eq!(
            file_header.rect,
            Rect {
                x: 246.0,
                y: 52.0,
                width: 554.0,
                height: 34.0,
            }
        );
        assert_eq!(file_header.role, "file-header");

        let header_name = file_header
            .children
            .iter()
            .find(|child| child.id == "header-name")
            .expect("header-name");
        let header_kind = file_header
            .children
            .iter()
            .find(|child| child.id == "header-kind")
            .expect("header-kind");
        let header_size = file_header
            .children
            .iter()
            .find(|child| child.id == "header-size")
            .expect("header-size");
        assert_eq!(
            header_name.rect,
            Rect {
                x: 254.0,
                y: 60.0,
                width: 298.0,
                height: 18.0
            }
        );
        assert_eq!(
            header_kind.rect,
            Rect {
                x: 552.0,
                y: 60.0,
                width: 150.0,
                height: 18.0
            }
        );
        assert_eq!(
            header_size.rect,
            Rect {
                x: 702.0,
                y: 60.0,
                width: 90.0,
                height: 18.0
            }
        );

        let file_list = file_pane
            .children
            .iter()
            .find(|child| child.id == "file-list")
            .expect("file-list");
        assert_eq!(
            file_list.rect,
            Rect {
                x: 246.0,
                y: 86.0,
                width: 554.0,
                height: 486.0,
            }
        );
        let row = file_list
            .children
            .iter()
            .find(|child| child.id == "row1")
            .expect("row1");
        assert_eq!(
            row.rect,
            Rect {
                x: 252.0,
                y: 92.0,
                width: 542.0,
                height: 30.0,
            }
        );
        let icon = row
            .children
            .iter()
            .find(|child| child.id == "row1-icon")
            .expect("row1-icon");
        let name = row
            .children
            .iter()
            .find(|child| child.id == "row1-name")
            .expect("row1-name");
        let modified = row
            .children
            .iter()
            .find(|child| child.id == "row1-modified")
            .expect("row1-modified");
        let size = row
            .children
            .iter()
            .find(|child| child.id == "row1-size")
            .expect("row1-size");
        assert_eq!(
            icon.rect,
            Rect {
                x: 258.0,
                y: 98.0,
                width: 18.0,
                height: 18.0
            }
        );
        assert_eq!(name.rect.x - (icon.rect.x + icon.rect.width), 8.0);
        assert_eq!(
            name.rect,
            Rect {
                x: 284.0,
                y: 98.0,
                width: 244.0,
                height: 18.0
            }
        );
        assert_eq!(
            modified.rect,
            Rect {
                x: 538.0,
                y: 98.0,
                width: 150.0,
                height: 18.0
            }
        );
        assert_eq!(
            size.rect,
            Rect {
                x: 698.0,
                y: 98.0,
                width: 90.0,
                height: 18.0
            }
        );
    }

    #[test]
    fn layout_places_rename_inline_after_editing_row_in_file_list() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "3 items".to_string(),
                selected_summary: "1 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: vec![
                    FrameRow {
                        name: "Archive".to_string(),
                        kind: "folder".to_string(),
                        modified: "Today".to_string(),
                        size: "--".to_string(),
                        class: "file-row".to_string(),
                        selected: false,
                        visual_state: RowVisualState::Normal,
                        select_method: vugra_core::MethodId(2),
                        hover_method: vugra_core::MethodId(53),
                        open_method: vugra_core::MethodId(65),
                        context_menu_method: vugra_core::MethodId(41),
                    },
                    FrameRow {
                        name: "Roadmap.md".to_string(),
                        kind: "file".to_string(),
                        modified: "Today".to_string(),
                        size: "12 KB".to_string(),
                        class: "file-row-editing".to_string(),
                        selected: true,
                        visual_state: RowVisualState::Editing,
                        select_method: vugra_core::MethodId(3),
                        hover_method: vugra_core::MethodId(54),
                        open_method: vugra_core::MethodId(66),
                        context_menu_method: vugra_core::MethodId(42),
                    },
                    FrameRow {
                        name: "Notes.txt".to_string(),
                        kind: "file".to_string(),
                        modified: "Yesterday".to_string(),
                        size: "4 KB".to_string(),
                        class: "file-row".to_string(),
                        selected: false,
                        visual_state: RowVisualState::Normal,
                        select_method: vugra_core::MethodId(4),
                        hover_method: vugra_core::MethodId(55),
                        open_method: vugra_core::MethodId(67),
                        context_menu_method: vugra_core::MethodId(43),
                    },
                ],
                overlays: FrameOverlays {
                    rename_text: "Roadmap draft.md".to_string(),
                    commit_rename_method: Some(vugra_core::MethodId(35)),
                    ..FrameOverlays::default()
                },
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let file_list = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .and_then(|pane| pane.children.iter().find(|child| child.id == "file-list"))
            .expect("file-list");
        assert_eq!(
            file_list
                .children
                .iter()
                .map(|child| child.id.as_str())
                .collect::<Vec<_>>(),
            vec!["row1", "row2", "rename-inline", "row3"]
        );
        let rename = file_list
            .children
            .iter()
            .find(|child| child.id == "rename-inline")
            .expect("rename-inline");
        assert_eq!(
            rename.rect,
            Rect {
                x: 252.0,
                y: 152.0,
                width: 542.0,
                height: 30.0,
            }
        );
        assert_eq!(rename.text, "Roadmap draft.md");
        assert_eq!(rename.method, Some(vugra_core::MethodId(35)));

        let row3 = file_list
            .children
            .iter()
            .find(|child| child.id == "row3")
            .expect("row3");
        assert_eq!(row3.rect.y, 182.0);
        assert!(tree
            .root
            .children
            .iter()
            .all(|child| child.id != "rename-inline"));
    }

    #[test]
    fn layout_matches_go_finder_scroll_container_structure() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );
        let sidebar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "sidebar")
            .expect("sidebar");
        assert_eq!(sidebar.overflow, Overflow::Scroll);
        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        assert_eq!(file_pane.overflow, Overflow::Hidden);
        let file_list = file_pane
            .children
            .iter()
            .find(|child| child.id == "file-list")
            .expect("file-list");
        assert_eq!(file_list.overflow, Overflow::Scroll);
    }

    #[test]
    fn layout_matches_go_finder_toolbar_and_statusbar_structure() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: "road".to_string(),
                search_input_method: Some(vugra_core::MethodId(9)),
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: Some(vugra_core::FrameToolbar {
                    back_method: vugra_core::MethodId(1),
                    forward_method: vugra_core::MethodId(20),
                }),
                status: "1 items · Current path: Documents".to_string(),
                selected_summary: "0 items selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        assert!(!tree
            .root
            .children
            .iter()
            .any(|child| child.id == "title" || child.text == "FinderLite"));

        let toolbar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "toolbar")
            .expect("toolbar");
        let path = toolbar
            .children
            .iter()
            .find(|child| child.id == "path")
            .expect("path");
        assert_eq!(path.role, "path");
        assert_eq!(path.text, "Documents");
        assert_eq!(
            path.rect,
            Rect {
                x: 94.0,
                y: 11.0,
                width: 468.0,
                height: 30.0
            }
        );
        let search = toolbar
            .children
            .iter()
            .find(|child| child.id == "search")
            .expect("search");
        assert_eq!(search.role, "search");
        assert_eq!(search.text, "road");
        assert_eq!(search.method, Some(vugra_core::MethodId(9)));
        assert_eq!(
            search.rect,
            Rect {
                x: 570.0,
                y: 11.0,
                width: 220.0,
                height: 30.0
            }
        );

        let statusbar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "statusbar")
            .expect("statusbar");
        assert_eq!(statusbar.text, "");
        assert_eq!(statusbar.rect.y, 572.0);
        assert_eq!(statusbar.rect.height, 28.0);
        assert!(statusbar
            .children
            .iter()
            .any(|child| child.id == "status-text"
                && child.role == "status-text"
                && child.text == "1 items · Current path: Documents"
                && child.rect.x == 6.0
                && child.rect.y == 578.0));
        assert!(statusbar
            .children
            .iter()
            .any(|child| child.id == "selected-summary"
                && child.role == "status-text"
                && child.text == "0 items selected"
                && child.rect.x == 666.0
                && child.rect.width == 128.0
                && child.rect.y == 578.0));
    }

    #[test]
    fn layout_separates_file_pane_click_and_context_menu_handlers() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "1 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays {
                    clear_selection_method: Some(vugra_core::MethodId(77)),
                    show_blank_menu_method: Some(vugra_core::MethodId(39)),
                    ..FrameOverlays::default()
                },
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        assert_eq!(file_pane.method, Some(vugra_core::MethodId(77)));
        assert_eq!(file_pane.handlers.target, Some(vugra_core::MethodId(77)));
        assert_eq!(
            file_pane.handlers.context_menu,
            Some(vugra_core::MethodId(39))
        );
    }

    #[test]
    fn layout_binds_row_hover_handlers_for_native_pointer_motion() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: vec![FrameRow {
                    name: "Design".to_string(),
                    kind: "folder".to_string(),
                    modified: "--".to_string(),
                    size: "--".to_string(),
                    class: "file-row".to_string(),
                    selected: false,
                    visual_state: RowVisualState::Normal,
                    select_method: vugra_core::MethodId(2),
                    hover_method: vugra_core::MethodId(53),
                    open_method: vugra_core::MethodId(65),
                    context_menu_method: vugra_core::MethodId(41),
                }],
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let row = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .and_then(|pane| pane.children.iter().find(|child| child.id == "file-list"))
            .and_then(|list| list.children.iter().find(|child| child.id == "row1"))
            .expect("row1");
        assert_eq!(row.handlers.hover, Some(vugra_core::MethodId(53)));
        assert_eq!(row.handlers.target, Some(vugra_core::MethodId(2)));
    }

    #[test]
    fn layout_binds_splitter_hover_drag_and_sidebar_width_state() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: Some(FrameSplitter {
                    sidebar_class: "sidebar-200".to_string(),
                    splitter_class: "splitter-hover".to_string(),
                    hover_method: vugra_core::MethodId(81),
                    drag_method: vugra_core::MethodId(82),
                }),
                rows: Vec::new(),
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let sidebar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "sidebar")
            .expect("sidebar");
        assert_eq!(sidebar.role, "sidebar-200");
        assert_eq!(sidebar.rect.width, 200.0);

        let splitter = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "splitter")
            .expect("splitter");
        assert_eq!(splitter.role, "splitter-hover");
        assert_eq!(splitter.rect.x, 200.0);
        assert_eq!(splitter.handlers.hover, Some(vugra_core::MethodId(81)));
        assert_eq!(splitter.handlers.drag, Some(vugra_core::MethodId(82)));

        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        assert_eq!(file_pane.rect.x, 206.0);
    }

    #[test]
    fn layout_nests_sidebar_and_rows_inside_scroll_containers() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: vec![vugra_core::FrameSidebarItem {
                    label: "Documents".to_string(),
                    kind: vugra_core::SidebarItemKind::Folder,
                    active: true,
                    open_method: vugra_core::MethodId(5),
                }],
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: vec![FrameRow {
                    name: "Design".to_string(),
                    kind: "folder".to_string(),
                    modified: "--".to_string(),
                    size: "--".to_string(),
                    class: "file-row-selected".to_string(),
                    selected: true,
                    visual_state: RowVisualState::Selected,
                    select_method: vugra_core::MethodId(2),
                    hover_method: vugra_core::MethodId(53),
                    open_method: vugra_core::MethodId(65),
                    context_menu_method: vugra_core::MethodId(41),
                }],
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let sidebar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "sidebar")
            .expect("sidebar");
        assert!(sidebar
            .children
            .iter()
            .any(|child| child.id == "sidebar-documents"));

        let file_pane = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "file-pane")
            .expect("file-pane");
        let file_list = file_pane
            .children
            .iter()
            .find(|child| child.id == "file-list")
            .expect("file-list");
        assert!(file_list.children.iter().any(|child| child.id == "row1"));
        assert!(tree.root.children.iter().all(|child| child.id != "row1"));
    }

    #[test]
    fn layout_renders_sidebar_sections_and_open_items() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: vec![vugra_core::FrameSidebarSection {
                    label: "Workspace".to_string(),
                    open: true,
                    toggle_method: vugra_core::MethodId(16),
                    items: vec![vugra_core::FrameSidebarItem {
                        label: "Current Project".to_string(),
                        kind: vugra_core::SidebarItemKind::Project,
                        active: true,
                        open_method: vugra_core::MethodId(17),
                    }],
                }],
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );
        let sidebar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "sidebar")
            .expect("sidebar");
        let section = sidebar
            .children
            .iter()
            .find(|child| child.id == "sidebar-section-workspace")
            .expect("workspace section");
        assert_eq!(section.role, "sidebar-section");
        assert_eq!(section.text, "");
        assert_eq!(section.method, Some(vugra_core::MethodId(16)));
        assert_eq!(
            section.rect,
            Rect {
                x: 12.0,
                y: 12.0,
                width: 216.0,
                height: 28.0
            }
        );
        assert!(section
            .children
            .iter()
            .any(|item| item.role == "chevron-down-icon"
                && item.rect.x == 12.0
                && item.rect.y == 19.0
                && item.rect.width == 14.0
                && item.rect.height == 14.0));
        assert!(section.children.iter().any(|item| {
            item.role == "sidebar-section-label"
                && item.text == "Workspace"
                && item.rect.x == 32.0
                && item.rect.y == 17.0
                && item.rect.height == 18.0
        }));

        let item = sidebar
            .children
            .iter()
            .find(|child| child.id == "sidebar-current-project")
            .expect("current project item");
        assert_eq!(
            item.rect,
            Rect {
                x: 12.0,
                y: 40.0,
                width: 216.0,
                height: 28.0
            }
        );
        assert!(item.selected);
        assert_eq!(item.method, Some(vugra_core::MethodId(17)));
        assert!(item.children.iter().any(|child| {
            child.role == "project-icon"
                && child.rect.x == 18.0
                && child.rect.y == 45.0
                && child.rect.width == 18.0
                && child.rect.height == 18.0
        }));
        assert!(item.children.iter().any(|child| {
            child.role == "sidebar-item-label"
                && child.text == "Current Project"
                && child.rect.x == 44.0
                && child.rect.y == 45.0
                && child.rect.height == 18.0
        }));
    }

    #[test]
    fn layout_renders_toolbar_nav_buttons_and_icons() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: Some(vugra_core::FrameToolbar {
                    back_method: vugra_core::MethodId(1),
                    forward_method: vugra_core::MethodId(20),
                }),
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays::default(),
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );
        let toolbar = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "toolbar")
            .expect("toolbar");
        assert!(toolbar.children.iter().any(|child| {
            child.id == "nav-back"
                && child.role == "nav-button"
                && child.method == Some(vugra_core::MethodId(1))
                && child.children.iter().any(|icon| icon.role == "back-icon")
        }));
        assert!(toolbar.children.iter().any(|child| {
            child.id == "nav-forward"
                && child.role == "nav-button"
                && child.method == Some(vugra_core::MethodId(20))
                && child
                    .children
                    .iter()
                    .any(|icon| icon.role == "forward-icon")
        }));
        let path = toolbar
            .children
            .iter()
            .find(|child| child.id == "path")
            .expect("path");
        assert_eq!(path.rect.x, 94.0);
        assert_eq!(path.rect.height, 30.0);
    }

    #[test]
    fn layout_renders_finder_overlays() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "1 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays {
                    item_menu_open: true,
                    preview_open: true,
                    preview_title: "Roadmap.md".to_string(),
                    preview_body: "System file · -- · 12 KB".to_string(),
                    open_selected_method: Some(vugra_core::MethodId(13)),
                    begin_rename_method: Some(vugra_core::MethodId(33)),
                    delete_selected_method: Some(vugra_core::MethodId(36)),
                    duplicate_selected_method: Some(vugra_core::MethodId(37)),
                    close_preview_method: Some(vugra_core::MethodId(40)),
                    ..FrameOverlays::default()
                },
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let item_overlay = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "item-overlay")
            .expect("item-overlay");
        assert_eq!(item_overlay.role, "overlay");
        assert_eq!(
            item_overlay.rect,
            Rect {
                x: 246.0,
                y: 52.0,
                width: 554.0,
                height: 520.0,
            }
        );
        assert!(item_overlay
            .children
            .iter()
            .any(|child| child.id == "item-menu" && child.role == "menu"));
        assert!(tree
            .root
            .children
            .iter()
            .all(|child| child.id != "rename-inline"));
        let preview_layer = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "preview-layer")
            .expect("preview-layer");
        assert_eq!(preview_layer.role, "dialog-layer");
        let preview_dialog = preview_layer
            .children
            .iter()
            .find(|child| child.id == "preview-dialog")
            .expect("preview-dialog");
        assert_eq!(
            preview_dialog.rect,
            Rect {
                x: 343.0,
                y: 232.0,
                width: 360.0,
                height: 160.0,
            }
        );
        assert!(preview_dialog
            .children
            .iter()
            .any(|item| item.id == "preview-title" && item.text == "Roadmap.md"));
    }

    #[test]
    fn layout_binds_blank_menu_paste_and_refresh_actions() {
        let tree = layout_frame(
            &Frame {
                title: "FinderLite".to_string(),
                path: "Documents".to_string(),
                values: Vec::new(),
                search_query: String::new(),
                search_input_method: None,
                search_backspace_method: None,
                search_clear_method: None,
                toolbar: None,
                status: "1 item".to_string(),
                selected_summary: "0 selected".to_string(),
                sidebar: Vec::new(),
                sidebar_sections: Vec::new(),
                splitter: None,
                rows: Vec::new(),
                overlays: FrameOverlays {
                    blank_menu_open: true,
                    new_folder_method: Some(vugra_core::MethodId(38)),
                    paste_method: Some(vugra_core::MethodId(78)),
                    refresh_method: Some(vugra_core::MethodId(79)),
                    ..FrameOverlays::default()
                },
            },
            Constraints {
                width: 800.0,
                height: 600.0,
            },
        );

        let overlay = tree
            .root
            .children
            .iter()
            .find(|child| child.id == "blank-overlay")
            .expect("blank-overlay");
        let menu = overlay
            .children
            .iter()
            .find(|child| child.id == "blank-menu")
            .expect("blank-menu");
        assert!(menu.children.iter().any(|item| {
            item.id == "blank-menu-new-folder" && item.method == Some(vugra_core::MethodId(38))
        }));
        assert!(menu.children.iter().any(|item| {
            item.id == "blank-menu-paste" && item.method == Some(vugra_core::MethodId(78))
        }));
        assert!(menu.children.iter().any(|item| {
            item.id == "blank-menu-refresh" && item.method == Some(vugra_core::MethodId(79))
        }));
    }
}
