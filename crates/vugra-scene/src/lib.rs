//! Retained scene and display-list construction for the Rust kernel path.

use std::collections::HashMap;

use vugra_layout::{EventHandlers, LayoutBox, LayoutTree, Overflow, Rect};

#[derive(Clone, Debug, PartialEq)]
pub struct Scene {
    pub commands: Vec<SceneCommand>,
    pub display_list: Vec<DisplayItem>,
    pub clips: Vec<ClipNode>,
    pub scrolls: Vec<ScrollNode>,
    pub hit_test: HitTestTree,
}

impl Scene {
    pub fn from_commands(commands: Vec<SceneCommand>) -> Self {
        let display_list = commands
            .iter()
            .enumerate()
            .filter_map(|(command_index, command)| {
                retained_command_key_rect(command)?;
                Some(DisplayItem {
                    command_index,
                    clip_id: None,
                })
            })
            .collect();
        Self {
            commands,
            display_list,
            clips: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree::default(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct DisplayItem {
    pub command_index: usize,
    pub clip_id: Option<String>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct ClipNode {
    pub id: String,
    pub rect: Rect,
    pub parent: Option<String>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct ScrollNode {
    pub id: String,
    pub rect: Rect,
    pub offset_y: f32,
    pub content_height: f32,
    pub clip_id: String,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct HitTestTree {
    pub nodes: Vec<HitTestNode>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct HitTestNode {
    pub id: String,
    pub role: String,
    pub rect: Rect,
    pub method: Option<vugra_ir::MethodId>,
    pub handlers: EventHandlers,
    pub parent: Option<usize>,
    pub clips: Vec<Rect>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct EventRoute {
    pub capture: Vec<usize>,
    pub target: usize,
    pub bubble: Vec<usize>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EventPhase {
    Capture,
    Target,
    Bubble,
}

#[derive(Clone, Debug, PartialEq)]
pub struct EventDispatch {
    pub phase: EventPhase,
    pub node: usize,
    pub method: vugra_ir::MethodId,
}

#[derive(Clone, Debug, PartialEq)]
pub enum SceneCommand {
    Begin {
        id: String,
        role: String,
        rect: Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
        method: Option<vugra_ir::MethodId>,
    },
    Text {
        id: String,
        text: String,
        role: String,
        rect: Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
    },
    End {
        id: String,
    },
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct RetainedScene {
    previous: HashMap<String, RetainedCommand>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct SceneUpdate {
    pub scene: Scene,
    pub dirty: Vec<Rect>,
}

#[derive(Clone, Debug, PartialEq)]
struct RetainedCommand {
    rect: Rect,
    command: SceneCommand,
}

impl RetainedScene {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn update(&mut self, tree: &LayoutTree) -> SceneUpdate {
        self.update_scene(build_scene(tree))
    }

    pub fn update_scene(&mut self, scene: Scene) -> SceneUpdate {
        let previous = std::mem::take(&mut self.previous);
        let mut next = HashMap::new();
        let mut dirty = Vec::new();

        for command in &scene.commands {
            let Some((key, rect)) = retained_command_key_rect(command) else {
                continue;
            };
            if !is_visible_rect(rect) {
                continue;
            }
            let retained = RetainedCommand {
                rect,
                command: command.clone(),
            };
            match previous.get(&key) {
                None => dirty.push(rect),
                Some(old) if old.command != *command => {
                    if old.rect != rect && is_visible_rect(old.rect) {
                        dirty.push(old.rect);
                    }
                    dirty.push(rect);
                }
                Some(_) => {}
            }
            next.insert(key, retained);
        }

        for (key, old) in previous {
            if !next.contains_key(&key) && is_visible_rect(old.rect) {
                dirty.push(old.rect);
            }
        }

        self.previous = next;
        SceneUpdate { scene, dirty }
    }
}

pub fn build_scene(tree: &LayoutTree) -> Scene {
    let mut commands = Vec::new();
    let mut display_list = Vec::new();
    let mut clips = Vec::new();
    let mut scrolls = Vec::new();
    let mut hit_test = HitTestTree::default();
    let mut clip_stack = Vec::new();
    push_box(
        &tree.root,
        &mut commands,
        &mut display_list,
        &mut clips,
        &mut scrolls,
        &mut hit_test,
        None,
        &mut clip_stack,
        0.0,
        false,
        vugra_layout::RowVisualState::Normal,
    );
    Scene {
        commands,
        display_list,
        clips,
        scrolls,
        hit_test,
    }
}

pub fn hit_test(tree: &HitTestTree, x: f32, y: f32) -> Option<vugra_ir::MethodId> {
    hit_test_route(tree, x, y).and_then(|route| {
        tree.nodes
            .get(route.target)
            .and_then(|node| node.handlers.target.or(node.method))
    })
}

pub fn hit_test_route(tree: &HitTestTree, x: f32, y: f32) -> Option<EventRoute> {
    let target = tree
        .nodes
        .iter()
        .enumerate()
        .rev()
        .find_map(|(index, node)| hit_test_node_contains(node, x, y).then_some(index))?;
    Some(event_route(tree, target))
}

pub fn hit_test_scroll_node(scrolls: &[ScrollNode], x: f32, y: f32) -> Option<&ScrollNode> {
    scrolls
        .iter()
        .rev()
        .find(|scroll| contains(scroll.rect, x, y))
}

pub fn event_route(tree: &HitTestTree, target: usize) -> EventRoute {
    let mut ancestors = Vec::new();
    let mut current = tree.nodes.get(target).and_then(|node| node.parent);
    while let Some(index) = current {
        ancestors.push(index);
        current = tree.nodes.get(index).and_then(|node| node.parent);
    }
    let capture = ancestors.iter().rev().copied().collect();
    EventRoute {
        capture,
        target,
        bubble: ancestors,
    }
}

pub fn route_target_method(tree: &HitTestTree, route: &EventRoute) -> Option<vugra_ir::MethodId> {
    tree.nodes
        .get(route.target)
        .and_then(|node| node.handlers.target.or(node.method))
}

pub fn event_dispatch_plan(tree: &HitTestTree, route: &EventRoute) -> Vec<EventDispatch> {
    event_dispatch_plan_for_kind(tree, route, "click")
}

pub fn event_dispatch_plan_for_kind(
    tree: &HitTestTree,
    route: &EventRoute,
    kind: &str,
) -> Vec<EventDispatch> {
    let mut plan = Vec::new();
    for node in &route.capture {
        if let Some(method) = tree.nodes.get(*node).and_then(|node| node.handlers.capture) {
            plan.push(EventDispatch {
                phase: EventPhase::Capture,
                node: *node,
                method,
            });
        }
    }
    if let Some(method) = tree.nodes.get(route.target).and_then(|node| {
        if kind == "contextmenu" {
            node.handlers
                .context_menu
                .or(node.handlers.target)
                .or(node.method)
        } else if kind == "dblclick" {
            node.handlers
                .double_click
                .or(node.handlers.target)
                .or(node.method)
        } else if kind == "hover" {
            node.handlers.hover
        } else if kind == "drag" {
            node.handlers.drag
        } else {
            node.handlers.target.or(node.method)
        }
    }) {
        plan.push(EventDispatch {
            phase: EventPhase::Target,
            node: route.target,
            method,
        });
    }
    for node in &route.bubble {
        if let Some(method) = tree.nodes.get(*node).and_then(|node| node.handlers.bubble) {
            plan.push(EventDispatch {
                phase: EventPhase::Bubble,
                node: *node,
                method,
            });
        }
    }
    plan
}

pub fn dispatch_event_route<F>(tree: &HitTestTree, route: &EventRoute, dispatch: F) -> bool
where
    F: FnMut(EventDispatch) -> bool,
{
    dispatch_event_route_for_kind(tree, route, "click", dispatch)
}

pub fn dispatch_event_route_for_kind<F>(
    tree: &HitTestTree,
    route: &EventRoute,
    kind: &str,
    mut dispatch: F,
) -> bool
where
    F: FnMut(EventDispatch) -> bool,
{
    let mut dispatched = false;
    for step in event_dispatch_plan_for_kind(tree, route, kind) {
        dispatched = true;
        if !dispatch(step) {
            break;
        }
    }
    dispatched
}

pub fn hit_test_legacy(tree: &HitTestTree, x: f32, y: f32) -> Option<vugra_ir::MethodId> {
    tree.nodes.iter().rev().find_map(|node| {
        contains(node.rect, x, y)
            .then_some(())
            .and_then(|_| node.handlers.target.or(node.method))
    })
}

fn push_box(
    node: &LayoutBox,
    commands: &mut Vec<SceneCommand>,
    display_list: &mut Vec<DisplayItem>,
    clips: &mut Vec<ClipNode>,
    scrolls: &mut Vec<ScrollNode>,
    hit_test: &mut HitTestTree,
    interactive_parent: Option<usize>,
    clip_stack: &mut Vec<String>,
    translate_y: f32,
    selected_text: bool,
    row_visual_state: vugra_layout::RowVisualState,
) {
    let rect = translate_rect(node.rect, translate_y);
    let selected_text =
        selected_text || ((node.role == "row" || node.role == "sidebar-item") && node.selected);
    let row_visual_state = if node.role == "row" {
        node.visual_state
    } else {
        row_visual_state
    };
    let command_index = commands.len();
    commands.push(SceneCommand::Begin {
        id: node.id.clone(),
        role: node.role.clone(),
        rect,
        selected: node.selected,
        visual_state: node.visual_state,
        method: node.method,
    });
    display_list.push(DisplayItem {
        command_index,
        clip_id: clip_stack.last().cloned(),
    });
    let clip_id = push_clip_if_needed(node, rect, clips, scrolls, clip_stack);
    let mut parent = interactive_parent;
    if has_event_handlers(node.method, &node.handlers) {
        let index = hit_test.nodes.len();
        hit_test.nodes.push(HitTestNode {
            id: node.id.clone(),
            role: node.role.clone(),
            rect,
            method: node.method,
            handlers: normalized_handlers(node.method, &node.handlers),
            parent: interactive_parent,
            clips: active_clip_rects(clips, clip_stack),
        });
        parent = Some(index);
    }
    if !node.text.is_empty() {
        let text_rect = text_rect_for_layout_box(node, rect);
        let command_index = commands.len();
        commands.push(SceneCommand::Text {
            id: format!("{}:text", node.id),
            text: node.text.clone(),
            role: node.role.clone(),
            rect: text_rect,
            selected: selected_text,
            visual_state: row_visual_state,
        });
        display_list.push(DisplayItem {
            command_index,
            clip_id: clip_stack.last().cloned(),
        });
    }
    let child_translate_y = if node.overflow == Overflow::Scroll {
        translate_y - node.scroll_y
    } else {
        translate_y
    };
    for child in &node.children {
        push_box(
            child,
            commands,
            display_list,
            clips,
            scrolls,
            hit_test,
            parent,
            clip_stack,
            child_translate_y,
            selected_text,
            row_visual_state,
        );
    }
    if clip_id.is_some() {
        clip_stack.pop();
    }
    commands.push(SceneCommand::End {
        id: node.id.clone(),
    });
}

fn translate_rect(rect: Rect, translate_y: f32) -> Rect {
    Rect {
        y: rect.y + translate_y,
        ..rect
    }
}

fn text_rect_for_layout_box(node: &LayoutBox, rect: Rect) -> Rect {
    match node.role.as_str() {
        "path" => inset_text_rect(rect, 6.0, 6.0),
        "search" => inset_text_rect(rect, 8.0, 7.0),
        _ => rect,
    }
}

fn inset_text_rect(rect: Rect, inset_x: f32, inset_y: f32) -> Rect {
    Rect {
        x: rect.x + inset_x,
        y: rect.y + inset_y,
        width: (rect.width - inset_x * 2.0).max(0.0),
        height: (rect.height - inset_y * 2.0).max(0.0),
    }
}

fn has_event_handlers(legacy_method: Option<vugra_ir::MethodId>, handlers: &EventHandlers) -> bool {
    legacy_method.is_some()
        || handlers.capture.is_some()
        || handlers.target.is_some()
        || handlers.bubble.is_some()
        || handlers.hover.is_some()
        || handlers.drag.is_some()
        || handlers.double_click.is_some()
        || handlers.context_menu.is_some()
}

fn normalized_handlers(
    legacy_method: Option<vugra_ir::MethodId>,
    handlers: &EventHandlers,
) -> EventHandlers {
    let mut normalized = handlers.clone();
    if normalized.target.is_none() {
        normalized.target = legacy_method;
    }
    normalized
}

fn contains(rect: Rect, x: f32, y: f32) -> bool {
    x >= rect.x && x <= rect.x + rect.width && y >= rect.y && y <= rect.y + rect.height
}

fn hit_test_node_contains(node: &HitTestNode, x: f32, y: f32) -> bool {
    contains(node.rect, x, y) && node.clips.iter().all(|clip| contains(*clip, x, y))
}

fn active_clip_rects(clips: &[ClipNode], clip_stack: &[String]) -> Vec<Rect> {
    clip_stack
        .iter()
        .filter_map(|id| {
            clips
                .iter()
                .rev()
                .find(|clip| clip.id == *id)
                .map(|clip| clip.rect)
        })
        .collect()
}

fn push_clip_if_needed(
    node: &LayoutBox,
    rect: Rect,
    clips: &mut Vec<ClipNode>,
    scrolls: &mut Vec<ScrollNode>,
    clip_stack: &mut Vec<String>,
) -> Option<String> {
    if !clips_overflow(node.overflow) {
        return None;
    }
    let clip_id = format!("{}:clip", node.id);
    let parent = clip_stack.last().cloned();
    clips.push(ClipNode {
        id: clip_id.clone(),
        rect,
        parent,
    });
    if node.overflow == Overflow::Scroll {
        scrolls.push(ScrollNode {
            id: node.id.clone(),
            rect,
            offset_y: node.scroll_y,
            content_height: scroll_content_height(node),
            clip_id: clip_id.clone(),
        });
    }
    clip_stack.push(clip_id.clone());
    Some(clip_id)
}

fn scroll_content_height(node: &LayoutBox) -> f32 {
    node.children
        .iter()
        .map(layout_box_bottom)
        .fold(node.rect.y + node.rect.height, f32::max)
        .max(node.rect.y)
        - node.rect.y
}

fn layout_box_bottom(node: &LayoutBox) -> f32 {
    node.children
        .iter()
        .map(layout_box_bottom)
        .fold(node.rect.y + node.rect.height, f32::max)
}

fn clips_overflow(overflow: Overflow) -> bool {
    matches!(overflow, Overflow::Hidden | Overflow::Scroll)
}

fn retained_command_key_rect(command: &SceneCommand) -> Option<(String, Rect)> {
    match command {
        SceneCommand::Begin { id, rect, .. } | SceneCommand::Text { id, rect, .. } => {
            Some((id.clone(), *rect))
        }
        SceneCommand::End { .. } => None,
    }
}

fn is_visible_rect(rect: Rect) -> bool {
    rect.width > 0.0 && rect.height > 0.0
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_layout::{LayoutBox, LayoutTree, Overflow, Rect};

    #[test]
    fn scene_keeps_display_list_order() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 10.0,
                    height: 10.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "title".to_string(),
                    role: "heading".to_string(),
                    text: "FinderLite".to_string(),
                    rect: Rect {
                        x: 1.0,
                        y: 1.0,
                        width: 8.0,
                        height: 2.0,
                    },
                    overflow: Overflow::Visible,
                    scroll_y: 0.0,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: Vec::new(),
                }],
            },
        });
        assert!(matches!(scene.commands[0], SceneCommand::Begin { .. }));
        assert!(matches!(scene.commands[2], SceneCommand::Text { .. }));
    }

    #[test]
    fn scene_builds_display_items_clips_and_scroll_nodes() {
        let tree = LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 200.0,
                    height: 160.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "viewport".to_string(),
                    role: "list".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 20.0,
                        width: 90.0,
                        height: 60.0,
                    },
                    overflow: Overflow::Scroll,
                    scroll_y: 7.5,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "row".to_string(),
                        role: "row".to_string(),
                        text: "Delta".to_string(),
                        rect: Rect {
                            x: 12.0,
                            y: 24.0,
                            width: 80.0,
                            height: 18.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    }],
                }],
            },
        };

        let scene = build_scene(&tree);

        assert_eq!(
            scene.clips,
            vec![ClipNode {
                id: "viewport:clip".to_string(),
                rect: Rect {
                    x: 10.0,
                    y: 20.0,
                    width: 90.0,
                    height: 60.0,
                },
                parent: None,
            }]
        );
        assert_eq!(
            scene.scrolls,
            vec![ScrollNode {
                id: "viewport".to_string(),
                rect: Rect {
                    x: 10.0,
                    y: 20.0,
                    width: 90.0,
                    height: 60.0,
                },
                offset_y: 7.5,
                content_height: 60.0,
                clip_id: "viewport:clip".to_string(),
            }]
        );
        let row_begin_index = scene
            .commands
            .iter()
            .position(|command| {
                matches!(
                    command,
                    SceneCommand::Begin { id, .. } if id == "row"
                )
            })
            .expect("row begin");
        assert!(
            scene.display_list.iter().any(|item| {
                item.command_index == row_begin_index
                    && item.clip_id.as_deref() == Some("viewport:clip")
            }),
            "display list = {:?}",
            scene.display_list
        );
    }

    #[test]
    fn scene_applies_scroll_offset_to_child_commands_and_hit_tests() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 200.0,
                    height: 160.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "viewport".to_string(),
                    role: "list".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 20.0,
                        width: 90.0,
                        height: 60.0,
                    },
                    overflow: Overflow::Scroll,
                    scroll_y: 14.0,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "row".to_string(),
                        role: "row".to_string(),
                        text: "Delta".to_string(),
                        rect: Rect {
                            x: 12.0,
                            y: 44.0,
                            width: 80.0,
                            height: 18.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: Some(vugra_ir::MethodId(22)),
                        handlers: EventHandlers::target(vugra_ir::MethodId(22)),
                        children: Vec::new(),
                    }],
                }],
            },
        });

        let row_rect = scene.commands.iter().find_map(|command| {
            if let SceneCommand::Begin { id, rect, .. } = command {
                (id == "row").then_some(*rect)
            } else {
                None
            }
        });
        assert_eq!(
            row_rect,
            Some(Rect {
                x: 12.0,
                y: 30.0,
                width: 80.0,
                height: 18.0,
            })
        );
        assert_eq!(
            hit_test(&scene.hit_test, 20.0, 32.0),
            Some(vugra_ir::MethodId(22))
        );
        assert_eq!(
            hit_test(&scene.hit_test, 20.0, 46.0),
            Some(vugra_ir::MethodId(22))
        );
        assert_eq!(hit_test(&scene.hit_test, 20.0, 82.0), None);
        assert_eq!(scene.scrolls[0].offset_y, 14.0);
    }

    #[test]
    fn scene_marks_text_inside_selected_rows_as_selected() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 200.0,
                    height: 160.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "row1".to_string(),
                    role: "row".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 20.0,
                        width: 90.0,
                        height: 24.0,
                    },
                    overflow: Overflow::Visible,
                    scroll_y: 0.0,
                    selected: true,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "row1-name".to_string(),
                        role: "row-name-cell".to_string(),
                        text: "Design".to_string(),
                        rect: Rect {
                            x: 18.0,
                            y: 22.0,
                            width: 60.0,
                            height: 20.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    }],
                }],
            },
        });

        assert!(scene.commands.iter().any(|command| matches!(
            command,
            SceneCommand::Text {
                id,
                selected: true,
                ..
            } if id == "row1-name:text"
        )));
    }

    #[test]
    fn scene_carries_parent_row_visual_state_to_text() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 120.0,
                    height: 80.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "row1".to_string(),
                    role: "row".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 20.0,
                        width: 90.0,
                        height: 24.0,
                    },
                    overflow: Overflow::Visible,
                    scroll_y: 0.0,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Focus,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "row1-name".to_string(),
                        role: "row-name-cell".to_string(),
                        text: "Design".to_string(),
                        rect: Rect {
                            x: 18.0,
                            y: 22.0,
                            width: 60.0,
                            height: 20.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    }],
                }],
            },
        });

        assert!(scene.commands.iter().any(|command| matches!(
            command,
            SceneCommand::Text {
                id,
                visual_state: vugra_layout::RowVisualState::Focus,
                ..
            } if id == "row1-name:text"
        )));
    }

    #[test]
    fn scene_carries_selected_sidebar_item_state_to_label_text() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 180.0,
                    height: 80.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "sidebar-documents".to_string(),
                    role: "sidebar-item".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 8.0,
                        y: 12.0,
                        width: 140.0,
                        height: 28.0,
                    },
                    overflow: Overflow::Visible,
                    scroll_y: 0.0,
                    selected: true,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "sidebar-documents-label".to_string(),
                        role: "sidebar-item-label".to_string(),
                        text: "Documents".to_string(),
                        rect: Rect {
                            x: 36.0,
                            y: 17.0,
                            width: 90.0,
                            height: 18.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    }],
                }],
            },
        });

        assert!(scene.commands.iter().any(|command| matches!(
            command,
            SceneCommand::Text {
                id,
                selected: true,
                ..
            } if id == "sidebar-documents-label:text"
        )));
    }

    #[test]
    fn hit_test_scroll_node_returns_topmost_scroll_container() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 200.0,
                    height: 160.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![
                    LayoutBox {
                        id: "back".to_string(),
                        role: "list".to_string(),
                        text: String::new(),
                        rect: Rect {
                            x: 10.0,
                            y: 10.0,
                            width: 100.0,
                            height: 100.0,
                        },
                        overflow: Overflow::Scroll,
                        scroll_y: 4.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    },
                    LayoutBox {
                        id: "front".to_string(),
                        role: "list".to_string(),
                        text: String::new(),
                        rect: Rect {
                            x: 20.0,
                            y: 20.0,
                            width: 80.0,
                            height: 80.0,
                        },
                        overflow: Overflow::Scroll,
                        scroll_y: 8.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::default(),
                        children: Vec::new(),
                    },
                ],
            },
        });

        let target = hit_test_scroll_node(&scene.scrolls, 24.0, 24.0).expect("scroll target");
        assert_eq!(target.id, "front");
        assert_eq!(target.offset_y, 8.0);
        assert_eq!(target.content_height, 80.0);
        assert_eq!(hit_test_scroll_node(&scene.scrolls, 180.0, 24.0), None);
    }

    #[test]
    fn hit_test_respects_active_clip_rects() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 200.0,
                    height: 200.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "viewport".to_string(),
                    role: "list".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 40.0,
                    },
                    overflow: Overflow::Scroll,
                    scroll_y: 0.0,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers::default(),
                    children: vec![LayoutBox {
                        id: "row".to_string(),
                        role: "row".to_string(),
                        text: "Overflowing row".to_string(),
                        rect: Rect {
                            x: 12.0,
                            y: 44.0,
                            width: 70.0,
                            height: 30.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: Some(vugra_ir::MethodId(22)),
                        handlers: EventHandlers::target(vugra_ir::MethodId(22)),
                        children: Vec::new(),
                    }],
                }],
            },
        });

        assert_eq!(
            hit_test(&scene.hit_test, 20.0, 46.0),
            Some(vugra_ir::MethodId(22))
        );
        assert_eq!(hit_test(&scene.hit_test, 20.0, 60.0), None);
        assert_eq!(
            scene.hit_test.nodes[0].clips,
            vec![Rect {
                x: 10.0,
                y: 10.0,
                width: 80.0,
                height: 40.0,
            }]
        );
    }

    #[test]
    fn retained_scene_marks_first_visible_commands_dirty() {
        let mut retained = RetainedScene::new();
        let update = retained.update(&sample_tree("Hello", 1.0, true));

        assert_eq!(
            update.dirty,
            vec![
                Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 100.0,
                    height: 80.0,
                },
                Rect {
                    x: 1.0,
                    y: 8.0,
                    width: 60.0,
                    height: 20.0,
                },
                Rect {
                    x: 1.0,
                    y: 8.0,
                    width: 60.0,
                    height: 20.0,
                },
            ]
        );
    }

    #[test]
    fn retained_scene_unchanged_update_has_no_dirty_rects() {
        let mut retained = RetainedScene::new();
        retained.update(&sample_tree("Hello", 1.0, true));

        let update = retained.update(&sample_tree("Hello", 1.0, true));

        assert!(update.dirty.is_empty(), "dirty = {:?}", update.dirty);
    }

    #[test]
    fn retained_scene_moved_command_marks_old_and_new_rects_dirty() {
        let mut retained = RetainedScene::new();
        retained.update(&sample_tree("Hello", 1.0, true));

        let update = retained.update(&sample_tree("Hello", 12.5, true));

        assert!(update.dirty.contains(&Rect {
            x: 1.0,
            y: 8.0,
            width: 60.0,
            height: 20.0,
        }));
        assert!(update.dirty.contains(&Rect {
            x: 12.5,
            y: 8.0,
            width: 60.0,
            height: 20.0,
        }));
    }

    #[test]
    fn retained_scene_text_change_marks_same_rect_dirty() {
        let mut retained = RetainedScene::new();
        retained.update(&sample_tree("Hello", 1.0, true));

        let update = retained.update(&sample_tree("World", 1.0, true));

        assert_eq!(
            update.dirty,
            vec![Rect {
                x: 1.0,
                y: 8.0,
                width: 60.0,
                height: 20.0,
            }]
        );
    }

    #[test]
    fn retained_scene_removed_command_marks_old_rect_dirty() {
        let mut retained = RetainedScene::new();
        retained.update(&sample_tree("Hello", 1.0, true));

        let update = retained.update(&sample_tree("Hello", 1.0, false));

        assert_eq!(
            update.dirty,
            vec![
                Rect {
                    x: 1.0,
                    y: 8.0,
                    width: 60.0,
                    height: 20.0,
                },
                Rect {
                    x: 1.0,
                    y: 8.0,
                    width: 60.0,
                    height: 20.0,
                },
            ]
        );
    }

    #[test]
    fn hit_test_tree_returns_topmost_bound_method() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 100.0,
                    height: 100.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![
                    LayoutBox {
                        id: "row1".to_string(),
                        role: "row".to_string(),
                        text: "A".to_string(),
                        rect: Rect {
                            x: 10.0,
                            y: 10.0,
                            width: 50.0,
                            height: 20.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: Some(vugra_ir::MethodId(2)),
                        handlers: EventHandlers::target(vugra_ir::MethodId(2)),
                        children: Vec::new(),
                    },
                    LayoutBox {
                        id: "row2".to_string(),
                        role: "row".to_string(),
                        text: "B".to_string(),
                        rect: Rect {
                            x: 10.0,
                            y: 10.0,
                            width: 50.0,
                            height: 20.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: Some(vugra_ir::MethodId(3)),
                        handlers: EventHandlers::target(vugra_ir::MethodId(3)),
                        children: Vec::new(),
                    },
                ],
            },
        });
        assert_eq!(
            hit_test(&scene.hit_test, 12.0, 12.0),
            Some(vugra_ir::MethodId(3))
        );
        assert_eq!(hit_test(&scene.hit_test, 80.0, 12.0), None);
    }

    #[test]
    fn hit_test_route_keeps_capture_target_and_bubble_path() {
        let scene = build_scene(&LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 100.0,
                    height: 100.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: vec![LayoutBox {
                    id: "panel".to_string(),
                    role: "panel".to_string(),
                    text: String::new(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 80.0,
                    },
                    overflow: Overflow::Visible,
                    scroll_y: 0.0,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    handlers: EventHandlers {
                        capture: Some(vugra_ir::MethodId(10)),
                        bubble: Some(vugra_ir::MethodId(11)),
                        ..EventHandlers::default()
                    },
                    children: vec![LayoutBox {
                        id: "button".to_string(),
                        role: "button".to_string(),
                        text: "Open".to_string(),
                        rect: Rect {
                            x: 20.0,
                            y: 20.0,
                            width: 30.0,
                            height: 20.0,
                        },
                        overflow: Overflow::Visible,
                        scroll_y: 0.0,
                        selected: false,
                        visual_state: vugra_layout::RowVisualState::Normal,
                        method: None,
                        handlers: EventHandlers::target(vugra_ir::MethodId(2)),
                        children: Vec::new(),
                    }],
                }],
            },
        });
        let route = hit_test_route(&scene.hit_test, 24.0, 24.0).expect("hit route");
        assert_eq!(route.capture, vec![0]);
        assert_eq!(route.target, 1);
        assert_eq!(route.bubble, vec![0]);
        assert_eq!(scene.hit_test.nodes[0].id, "panel");
        assert_eq!(scene.hit_test.nodes[1].id, "button");
        assert_eq!(
            route_target_method(&scene.hit_test, &route),
            Some(vugra_ir::MethodId(2))
        );
        assert_eq!(
            event_dispatch_plan(&scene.hit_test, &route),
            vec![
                EventDispatch {
                    phase: EventPhase::Capture,
                    node: 0,
                    method: vugra_ir::MethodId(10),
                },
                EventDispatch {
                    phase: EventPhase::Target,
                    node: 1,
                    method: vugra_ir::MethodId(2),
                },
                EventDispatch {
                    phase: EventPhase::Bubble,
                    node: 0,
                    method: vugra_ir::MethodId(11),
                },
            ]
        );
        assert_eq!(
            hit_test(&scene.hit_test, 24.0, 24.0),
            Some(vugra_ir::MethodId(2))
        );
    }

    #[test]
    fn event_route_dispatch_can_stop_after_capture() {
        let tree = HitTestTree {
            nodes: vec![
                HitTestNode {
                    id: "panel".to_string(),
                    role: "panel".to_string(),
                    rect: Rect {
                        x: 0.0,
                        y: 0.0,
                        width: 100.0,
                        height: 100.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        capture: Some(vugra_ir::MethodId(1)),
                        bubble: Some(vugra_ir::MethodId(3)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                },
                HitTestNode {
                    id: "button".to_string(),
                    role: "button".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 20.0,
                        height: 20.0,
                    },
                    method: Some(vugra_ir::MethodId(2)),
                    handlers: EventHandlers::target(vugra_ir::MethodId(2)),
                    parent: Some(0),
                    clips: Vec::new(),
                },
            ],
        };
        let route = event_route(&tree, 1);
        let mut calls = Vec::new();
        let dispatched = dispatch_event_route(&tree, &route, |step| {
            calls.push((step.phase, step.method));
            step.phase != EventPhase::Capture
        });
        assert!(dispatched);
        assert_eq!(calls, vec![(EventPhase::Capture, vugra_ir::MethodId(1))]);
    }

    fn sample_tree(text: &str, child_x: f32, include_child: bool) -> LayoutTree {
        let mut children = Vec::new();
        if include_child {
            children.push(LayoutBox {
                id: "label".to_string(),
                role: "text".to_string(),
                text: text.to_string(),
                rect: Rect {
                    x: child_x,
                    y: 8.0,
                    width: 60.0,
                    height: 20.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children: Vec::new(),
            });
        }
        LayoutTree {
            root: LayoutBox {
                id: "root".to_string(),
                role: "window".to_string(),
                text: String::new(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 100.0,
                    height: 80.0,
                },
                overflow: Overflow::Visible,
                scroll_y: 0.0,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                handlers: EventHandlers::default(),
                children,
            },
        }
    }
}
