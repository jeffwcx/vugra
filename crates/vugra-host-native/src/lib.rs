//! Native/test host entry points for the Rust kernel path.

use std::cell::RefCell;
use std::collections::HashMap;
use std::rc::Rc;
use std::time::{Duration, Instant};

use minifb::{InputCallback, Key, KeyRepeat, MouseButton, MouseMode, Window, WindowOptions};
use vugra_core::{App, ComponentState, Event, MethodId, Modifiers};
use vugra_layout::{layout_frame, Constraints, LayoutBox, LayoutTree, Overflow};
use vugra_render::{RenderCommand, Renderer, TestRenderer};
use vugra_render_vello::{Color as VelloColor, VelloOp, VelloRenderer};
use vugra_render_wgpu::{Color as WgpuColor, WgpuPassOp, WgpuRenderer};
use vugra_scene::{
    build_scene, dispatch_event_route, dispatch_event_route_for_kind, hit_test_route,
    hit_test_scroll_node, HitTestTree, RetainedScene, ScrollNode,
};

pub fn render_test_frame<S: ComponentState>(
    app: &App<S>,
    constraints: Constraints,
) -> TestRenderer {
    let frame = app.render_frame();
    let layout = layout_frame(&frame, constraints);
    let scene = build_scene(&layout);
    let mut renderer = TestRenderer::default();
    renderer.render(&scene);
    renderer
}

pub fn render_commands<S: ComponentState>(
    app: &App<S>,
    constraints: Constraints,
) -> Vec<RenderCommand> {
    render_test_frame(app, constraints).commands().to_vec()
}

pub fn render_commands_with_backend<S: ComponentState>(
    app: &App<S>,
    constraints: Constraints,
    backend: NativeRenderBackend,
) -> Vec<RenderCommand> {
    let frame = app.render_frame();
    let layout = layout_frame(&frame, constraints);
    let scene = build_scene(&layout);
    match backend {
        NativeRenderBackend::Software => {
            let mut renderer = TestRenderer::default();
            renderer.render(&scene);
            renderer.commands().to_vec()
        }
        NativeRenderBackend::Vello => {
            let mut renderer = VelloRenderer::default();
            renderer.render(&scene);
            commands_from_vello_ops(renderer.ops())
        }
        NativeRenderBackend::Wgpu => {
            let mut renderer = WgpuRenderer::default();
            renderer.render(&scene);
            commands_from_wgpu_pass(renderer.pass())
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct NativeFrame {
    pub commands: Vec<RenderCommand>,
    pub pixels: Vec<u32>,
    pub hit_test: HitTestTree,
    pub scrolls: Vec<ScrollNode>,
    pub dirty: Vec<vugra_layout::Rect>,
}

pub fn render_native_frame<S: ComponentState>(
    app: &App<S>,
    constraints: Constraints,
    backend: NativeRenderBackend,
    width: usize,
    height: usize,
) -> NativeFrame {
    let mut state = NativeFrameState::new(backend, width, height);
    state.render(app, constraints)
}

#[derive(Clone, Debug)]
pub struct NativeFrameState {
    backend: NativeRenderBackend,
    width: usize,
    height: usize,
    retained: RetainedScene,
    pixel_cache: Option<Vec<u32>>,
    scroll_offsets: HashMap<String, f32>,
    text_provider: vugra_text::SystemFontFallbackProvider,
}

impl NativeFrameState {
    pub fn new(backend: NativeRenderBackend, width: usize, height: usize) -> Self {
        Self {
            backend,
            width: width.max(1),
            height: height.max(1),
            retained: RetainedScene::new(),
            pixel_cache: None,
            scroll_offsets: HashMap::new(),
            text_provider: vugra_text::SystemFontTextMeasurer::new().fallback_provider(),
        }
    }

    pub fn render<S: ComponentState>(
        &mut self,
        app: &App<S>,
        constraints: Constraints,
    ) -> NativeFrame {
        render_native_frame_with_state(app, constraints, self)
    }

    pub fn scroll_offset(&self, id: &str) -> f32 {
        self.scroll_offsets.get(id).copied().unwrap_or(0.0)
    }

    pub fn apply_scroll_delta(&mut self, scroll: &ScrollNode, delta_y: f32) -> bool {
        let max_offset = (scroll.content_height - scroll.rect.height).max(0.0);
        if max_offset <= 0.0 {
            return false;
        }
        let current = self.scroll_offset(&scroll.id);
        let next = (current - delta_y * 32.0).clamp(0.0, max_offset);
        if (next - current).abs() < f32::EPSILON {
            return false;
        }
        self.scroll_offsets.insert(scroll.id.clone(), next);
        true
    }
}

pub fn render_native_frame_with_state<S: ComponentState>(
    app: &App<S>,
    constraints: Constraints,
    state: &mut NativeFrameState,
) -> NativeFrame {
    let frame = app.render_frame();
    let mut layout = layout_frame(&frame, constraints);
    apply_scroll_offsets(&mut layout, &state.scroll_offsets);
    let update = state.retained.update(&layout);
    let scene = update.scene;
    let mut metadata = TestRenderer::default();
    metadata.render(&scene);
    let commands = metadata.commands().to_vec();
    let paint_commands = lower_native_paint_commands(&commands, &state.text_provider);
    let pixels = match state.backend {
        NativeRenderBackend::Software => {
            state.render_software_pixels(&paint_commands, &update.dirty)
        }
        NativeRenderBackend::Vello => {
            let ops = native_paint_commands_to_vello_ops(&paint_commands);
            state.render_vello_ops(&ops, &update.dirty)
        }
        NativeRenderBackend::Wgpu => {
            let pass = native_paint_commands_to_wgpu_pass(&paint_commands);
            state.render_wgpu_pass(&pass, &update.dirty)
        }
    };
    NativeFrame {
        commands,
        pixels,
        hit_test: scene.hit_test,
        scrolls: scene.scrolls,
        dirty: update.dirty,
    }
}

fn apply_scroll_offsets(tree: &mut LayoutTree, offsets: &HashMap<String, f32>) {
    apply_scroll_offsets_to_box(&mut tree.root, offsets);
}

fn apply_scroll_offsets_to_box(node: &mut LayoutBox, offsets: &HashMap<String, f32>) {
    if node.overflow == Overflow::Scroll {
        node.scroll_y = offsets.get(&node.id).copied().unwrap_or(node.scroll_y);
    }
    for child in &mut node.children {
        apply_scroll_offsets_to_box(child, offsets);
    }
}

#[derive(Clone, Debug, PartialEq)]
enum NativePaintCommand {
    Element {
        id: String,
        role: String,
        rect: vugra_layout::Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
        method: Option<MethodId>,
    },
    Text {
        id: String,
        text: String,
        role: String,
        rect: vugra_layout::Rect,
        run: vugra_text::TextRun,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
    },
    End {
        id: String,
    },
}

fn lower_native_paint_commands(
    commands: &[RenderCommand],
    text_provider: &impl vugra_text::GlyphMetricsProvider,
) -> Vec<NativePaintCommand> {
    commands
        .iter()
        .map(|command| match command {
            RenderCommand::Element {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
            } => NativePaintCommand::Element {
                id: id.clone(),
                role: role.clone(),
                rect: *rect,
                selected: *selected,
                visual_state: *visual_state,
                method: *method,
            },
            RenderCommand::Text {
                id,
                text,
                role,
                rect,
                selected,
                visual_state,
            } => NativePaintCommand::Text {
                id: id.clone(),
                text: text.clone(),
                role: role.clone(),
                rect: *rect,
                run: layout_native_text_run(role, text, *rect, text_provider),
                selected: *selected,
                visual_state: *visual_state,
            },
            RenderCommand::End { id } => NativePaintCommand::End { id: id.clone() },
        })
        .collect()
}

fn layout_native_text_run(
    role: &str,
    text: &str,
    rect: vugra_layout::Rect,
    text_provider: &impl vugra_text::GlyphMetricsProvider,
) -> vugra_text::TextRun {
    let font_size = native_font_size_for_role(role);
    let font_weight = native_font_weight_for_role(role);
    let run = vugra_text::layout_text_run_with_provider_weighted(
        text,
        rect.x,
        rect.y,
        font_size,
        font_weight,
        Some(rect.width),
        text_provider,
    );
    align_native_text_run_for_role(role, rect, run, text_provider)
}

fn align_native_text_run_for_role(
    role: &str,
    rect: vugra_layout::Rect,
    run: vugra_text::TextRun,
    text_provider: &impl vugra_text::GlyphMetricsProvider,
) -> vugra_text::TextRun {
    if role != "row-size-cell" {
        return run;
    }
    let aligned_x = (rect.x + rect.width - run.advance()).max(rect.x);
    vugra_text::layout_text_run_with_provider_weighted(
        &run.text,
        aligned_x,
        rect.y,
        run.font_size,
        run.font_weight,
        Some(rect.width),
        text_provider,
    )
}

fn native_font_size_for_role(role: &str) -> f32 {
    match role {
        "dialog-title" => 15.0,
        _ => 13.0,
    }
}

fn native_font_weight_for_role(role: &str) -> u16 {
    match role {
        "column-header" | "sidebar-section-label" | "dialog-title" => 600,
        _ => 400,
    }
}

fn native_paint_commands_to_vello_ops(commands: &[NativePaintCommand]) -> Vec<VelloOp> {
    commands
        .iter()
        .filter_map(|command| match command {
            NativePaintCommand::Element {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
            } => vello_fill_for_role(role, *selected, *visual_state).map(|color| VelloOp::Fill {
                id: id.clone(),
                role: role.clone(),
                rect: *rect,
                selected: *selected,
                visual_state: *visual_state,
                method: *method,
                color,
            }),
            NativePaintCommand::Text {
                id,
                text,
                role,
                rect,
                run,
                selected,
                visual_state,
            } => Some(VelloOp::Text {
                id: id.clone(),
                text: text.clone(),
                role: role.clone(),
                rect: *rect,
                run: run.clone(),
                color: vello_text_color(role, text, *selected, *visual_state),
            }),
            NativePaintCommand::End { id } => Some(VelloOp::End { id: id.clone() }),
        })
        .collect()
}

fn native_paint_commands_to_wgpu_pass(commands: &[NativePaintCommand]) -> Vec<WgpuPassOp> {
    commands
        .iter()
        .filter_map(|command| match command {
            NativePaintCommand::Element {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
            } => wgpu_fill_for_role(role, *selected, *visual_state).map(|color| WgpuPassOp::Quad {
                id: id.clone(),
                role: role.clone(),
                pipeline: vugra_render_wgpu::Pipeline::Solid,
                rect: *rect,
                selected: *selected,
                visual_state: *visual_state,
                method: *method,
                color,
            }),
            NativePaintCommand::Text {
                id,
                text,
                role,
                rect,
                run,
                selected,
                visual_state,
            } => Some(WgpuPassOp::Text {
                id: id.clone(),
                rect: *rect,
                text: text.clone(),
                role: role.clone(),
                run: run.clone(),
                color: wgpu_text_color(role, text, *selected, *visual_state),
            }),
            NativePaintCommand::End { id } => Some(WgpuPassOp::End { id: id.clone() }),
        })
        .collect()
}

impl NativeFrameState {
    fn render_software_pixels(
        &mut self,
        commands: &[NativePaintCommand],
        dirty: &[vugra_layout::Rect],
    ) -> Vec<u32> {
        let expected_len = self.width * self.height;
        let needs_full_render = self
            .pixel_cache
            .as_ref()
            .is_none_or(|cache| cache.len() != expected_len)
            || dirty_covers_full_surface(dirty, self.width, self.height);
        if needs_full_render {
            let pixels = render_native_paint_pixels(commands, self.width, self.height);
            self.pixel_cache = Some(pixels.clone());
            return pixels;
        }

        let cache = self
            .pixel_cache
            .as_mut()
            .expect("pixel cache exists after needs_full_render check");
        if !dirty.is_empty() {
            render_native_paint_pixels_into_dirty(cache, commands, self.width, self.height, dirty);
        }
        cache.clone()
    }

    fn render_vello_ops(&mut self, ops: &[VelloOp], dirty: &[vugra_layout::Rect]) -> Vec<u32> {
        let expected_len = self.width * self.height;
        let needs_full_render = self
            .pixel_cache
            .as_ref()
            .is_none_or(|cache| cache.len() != expected_len)
            || dirty_covers_full_surface(dirty, self.width, self.height);
        if needs_full_render {
            let pixels = render_vello_pixels(ops, self.width, self.height);
            self.pixel_cache = Some(pixels.clone());
            return pixels;
        }

        let cache = self
            .pixel_cache
            .as_mut()
            .expect("pixel cache exists after needs_full_render check");
        if !dirty.is_empty() {
            render_vello_pixels_into_dirty(cache, ops, self.width, self.height, dirty);
        }
        cache.clone()
    }

    fn render_wgpu_pass(&mut self, pass: &[WgpuPassOp], dirty: &[vugra_layout::Rect]) -> Vec<u32> {
        let expected_len = self.width * self.height;
        let needs_full_render = self
            .pixel_cache
            .as_ref()
            .is_none_or(|cache| cache.len() != expected_len)
            || dirty_covers_full_surface(dirty, self.width, self.height);
        if needs_full_render {
            let pixels = render_wgpu_pixels(pass, self.width, self.height);
            self.pixel_cache = Some(pixels.clone());
            return pixels;
        }

        let cache = self
            .pixel_cache
            .as_mut()
            .expect("pixel cache exists after needs_full_render check");
        if !dirty.is_empty() {
            render_wgpu_pixels_into_dirty(cache, pass, self.width, self.height, dirty);
        }
        cache.clone()
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct NativeWindowConfig {
    pub title: String,
    pub width: usize,
    pub height: usize,
    pub backend: NativeRenderBackend,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum NativeRenderBackend {
    Software,
    Vello,
    Wgpu,
}

impl NativeRenderBackend {
    pub fn parse(value: &str) -> Option<Self> {
        match value {
            "software" => Some(Self::Software),
            "vello" => Some(Self::Vello),
            "wgpu" | "webgpu" => Some(Self::Wgpu),
            _ => None,
        }
    }

    pub fn as_str(self) -> &'static str {
        match self {
            Self::Software => "software",
            Self::Vello => "vello",
            Self::Wgpu => "wgpu",
        }
    }
}

impl Default for NativeWindowConfig {
    fn default() -> Self {
        Self {
            title: "Vugra Rust Finder Lite".to_string(),
            width: 800,
            height: 600,
            backend: NativeRenderBackend::Vello,
        }
    }
}

pub fn render_text_pixels(text: &str, width: usize, height: usize) -> Vec<u32> {
    let width = width.max(1);
    let height = height.max(1);
    let mut pixels = vec![0x00f7f7f8; width * height];
    fill_rect(&mut pixels, width, height, 0, 0, width, 54, 0x00f2f2f4);
    fill_rect(
        &mut pixels,
        width,
        height,
        0,
        54,
        220,
        height.saturating_sub(54),
        0x00ececf0,
    );
    fill_rect(
        &mut pixels,
        width,
        height,
        220,
        54,
        width.saturating_sub(220),
        height.saturating_sub(54),
        0x00ffffff,
    );

    let mut y = 16usize;
    for (index, line) in text.lines().enumerate() {
        let (x, color) = match index {
            0 => (18, 0x001f2328),
            1..=3 => (240, 0x00374151),
            _ => (250, 0x001f2937),
        };
        draw_text(&mut pixels, width, height, x as f32, y as f32, line, color);
        y += if index == 0 { 36 } else { 28 };
    }
    pixels
}

pub fn run_text_window(text: &str, config: NativeWindowConfig) -> Result<(), minifb::Error> {
    let width = config.width.max(1);
    let height = config.height.max(1);
    let pixels = render_text_pixels(text, width, height);
    let mut window = Window::new(&config.title, width, height, WindowOptions::default())?;
    window.set_target_fps(60);
    while window.is_open() && !window.is_key_down(Key::Escape) {
        window.update_with_buffer(&pixels, width, height)?;
    }
    Ok(())
}

pub fn render_command_pixels(commands: &[RenderCommand], width: usize, height: usize) -> Vec<u32> {
    let text_provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
    let paint_commands = lower_native_paint_commands(commands, &text_provider);
    render_native_paint_pixels(&paint_commands, width, height)
}

fn render_native_paint_pixels(
    commands: &[NativePaintCommand],
    width: usize,
    height: usize,
) -> Vec<u32> {
    let width = width.max(1);
    let height = height.max(1);
    let mut pixels = vec![0x00f7f7f8; width * height];
    render_native_paint_commands_into_pixels(&mut pixels, commands, width, height);
    pixels
}

fn render_native_paint_commands_into_pixels(
    pixels: &mut [u32],
    commands: &[NativePaintCommand],
    width: usize,
    height: usize,
) {
    for command in commands {
        render_native_paint_command_into_pixels(pixels, command, width, height);
    }
}

fn render_native_paint_command_into_pixels(
    pixels: &mut [u32],
    command: &NativePaintCommand,
    width: usize,
    height: usize,
) {
    match command {
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } if role == "row" => {
            let color = row_color_u32(*selected, *visual_state);
            paint_role_surface(
                pixels,
                width,
                height,
                role,
                *rect,
                color,
                *selected,
                *visual_state,
                None,
            );
        }
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } if role == "sidebar-item" => {
            let color = if *selected { 0x00d9e8ff } else { 0x00ececf0 };
            paint_role_surface(
                pixels,
                width,
                height,
                role,
                *rect,
                color,
                *selected,
                *visual_state,
                None,
            );
        }
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } => {
            if draw_role_icon(pixels, width, height, role, *rect, None) {
            } else if let Some(color) = role_fill(role) {
                paint_role_surface(
                    pixels,
                    width,
                    height,
                    role,
                    *rect,
                    color,
                    *selected,
                    *visual_state,
                    None,
                );
            }
        }
        NativePaintCommand::Text {
            role,
            text,
            rect,
            run,
            selected,
            visual_state,
            ..
        } => {
            let color = native_text_color(role, text, *selected, *visual_state);
            draw_text_run_clipped(pixels, width, height, run, color, *rect);
        }
        _ => {}
    }
}

fn render_native_paint_pixels_into_dirty(
    pixels: &mut [u32],
    commands: &[NativePaintCommand],
    width: usize,
    height: usize,
    dirty: &[vugra_layout::Rect],
) {
    let dirty = disjoint_dirty_rects(dirty);
    for rect in &dirty {
        fill_rect_from_rect(pixels, width, height, *rect, 0x00f7f7f8);
    }
    for command in commands {
        for rect in &dirty {
            if native_paint_command_intersects_rect(command, *rect) {
                render_native_paint_command_into_pixels_clipped(
                    pixels, command, width, height, *rect,
                );
            }
        }
    }
}

fn native_paint_command_intersects_rect(
    command: &NativePaintCommand,
    dirty_rect: vugra_layout::Rect,
) -> bool {
    let rect = match command {
        NativePaintCommand::Element { rect, .. } => *rect,
        NativePaintCommand::Text { rect, run, .. } => text_run_bounds(run, *rect),
        NativePaintCommand::End { .. } => return false,
    };
    rects_intersect(rect, dirty_rect)
}

fn render_native_paint_command_into_pixels_clipped(
    pixels: &mut [u32],
    command: &NativePaintCommand,
    width: usize,
    height: usize,
    clip: vugra_layout::Rect,
) {
    match command {
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } if role == "row" => {
            let color = row_color_u32(*selected, *visual_state);
            paint_role_surface(
                pixels,
                width,
                height,
                role,
                *rect,
                color,
                *selected,
                *visual_state,
                Some(clip),
            );
        }
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } if role == "sidebar-item" => {
            let color = if *selected { 0x00d9e8ff } else { 0x00ececf0 };
            paint_role_surface(
                pixels,
                width,
                height,
                role,
                *rect,
                color,
                *selected,
                *visual_state,
                Some(clip),
            );
        }
        NativePaintCommand::Element {
            role,
            rect,
            selected,
            visual_state,
            ..
        } => {
            if draw_role_icon(pixels, width, height, role, *rect, Some(clip)) {
            } else if let Some(color) = role_fill(role) {
                paint_role_surface(
                    pixels,
                    width,
                    height,
                    role,
                    *rect,
                    color,
                    *selected,
                    *visual_state,
                    Some(clip),
                );
            }
        }
        NativePaintCommand::Text {
            role,
            text,
            rect,
            run,
            selected,
            visual_state,
            ..
        } => {
            let color = native_text_color(role, text, *selected, *visual_state);
            if let Some(text_clip) = intersect_rect(*rect, clip) {
                draw_text_run_clipped(pixels, width, height, run, color, text_clip);
            }
        }
        _ => {}
    }
}

fn row_color_u32(selected: bool, visual_state: vugra_layout::RowVisualState) -> u32 {
    match visual_state {
        vugra_layout::RowVisualState::Selected if selected => 0x000a84ff,
        vugra_layout::RowVisualState::Selected => 0x000a84ff,
        vugra_layout::RowVisualState::Focus => 0x00edf6ff,
        vugra_layout::RowVisualState::Hover => 0x00f4f7fb,
        vugra_layout::RowVisualState::Editing => 0x00eef6ff,
        vugra_layout::RowVisualState::Normal if selected => 0x000a84ff,
        vugra_layout::RowVisualState::Normal => 0x00ffffff,
    }
}

fn paint_role_surface(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    role: &str,
    rect: vugra_layout::Rect,
    fill: u32,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
    clip: Option<vugra_layout::Rect>,
) {
    if let Some(radius) = role_corner_radius(role, visual_state) {
        fill_rounded_rect(pixels, width, height, rect, radius, fill, clip);
    } else {
        fill_icon_rect(pixels, width, height, rect, fill, clip);
    }
    if let Some(border) = role_border_color(role, selected, visual_state) {
        draw_role_border(pixels, width, height, rect, border, clip);
    }
}

pub fn native_role_corner_radius(
    role: &str,
    visual_state: vugra_layout::RowVisualState,
) -> Option<f32> {
    match role {
        "nav-button" | "path" | "search" | "sidebar-item" => Some(6.0),
        "row" if visual_state == vugra_layout::RowVisualState::Normal => Some(5.0),
        "row" => Some(5.0),
        "rename-inline" => Some(5.0),
        "menu" | "dialog" => Some(8.0),
        "primary-button" | "secondary-button" => Some(6.0),
        _ => None,
    }
}

fn role_corner_radius(role: &str, visual_state: vugra_layout::RowVisualState) -> Option<f32> {
    native_role_corner_radius(role, visual_state)
}

fn role_border_color(
    role: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> Option<u32> {
    match role {
        "toolbar" | "statusbar" => Some(0x00d8d8dc),
        "sidebar" | "sidebar-200" | "sidebar-280" | "sidebar-320" => Some(0x00d1d1d6),
        "overlay" => Some(0x00c7c7cc),
        "dialog-layer" => Some(0x00d8d8dc),
        "nav-button" | "search" | "menu" | "dialog" | "secondary-button" => Some(0x00c7c7cc),
        "primary-button" => Some(0x000a84ff),
        "path" => Some(0x00d8d8dc),
        "file-header" => Some(0x00e4e4e7),
        "row" if visual_state == vugra_layout::RowVisualState::Editing => Some(0x000a84ff),
        "row" if visual_state == vugra_layout::RowVisualState::Focus => Some(0x006aa8ff),
        "row" if selected => None,
        "rename-inline" => Some(0x000a84ff),
        _ => None,
    }
}

fn draw_role_border(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    color: u32,
    clip: Option<vugra_layout::Rect>,
) {
    let thickness = 1.0;
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y,
            width: rect.width,
            height: thickness,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y + rect.height - thickness,
            width: rect.width,
            height: thickness,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y,
            width: thickness,
            height: rect.height,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + rect.width - thickness,
            y: rect.y,
            width: thickness,
            height: rect.height,
        },
        color,
        clip,
    );
}

fn rects_intersect(a: vugra_layout::Rect, b: vugra_layout::Rect) -> bool {
    a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y
}

fn text_run_bounds(run: &vugra_text::TextRun, fallback: vugra_layout::Rect) -> vugra_layout::Rect {
    vugra_layout::Rect {
        x: run.x,
        y: run.y,
        width: run.advance().max(1.0),
        height: fallback.height.max(run.metrics.height).max(1.0),
    }
}

fn intersect_rect(a: vugra_layout::Rect, b: vugra_layout::Rect) -> Option<vugra_layout::Rect> {
    let x1 = a.x.max(b.x);
    let y1 = a.y.max(b.y);
    let x2 = (a.x + a.width).min(b.x + b.width);
    let y2 = (a.y + a.height).min(b.y + b.height);
    (x2 > x1 && y2 > y1).then_some(vugra_layout::Rect {
        x: x1,
        y: y1,
        width: x2 - x1,
        height: y2 - y1,
    })
}

fn disjoint_dirty_rects(rects: &[vugra_layout::Rect]) -> Vec<vugra_layout::Rect> {
    let mut out = Vec::new();
    for rect in rects
        .iter()
        .copied()
        .filter(|rect| rect.width > 0.0 && rect.height > 0.0)
    {
        let mut pieces = vec![rect];
        for existing in &out {
            let mut next = Vec::new();
            for piece in pieces {
                subtract_rect(piece, *existing, &mut next);
            }
            pieces = next;
            if pieces.is_empty() {
                break;
            }
        }
        out.extend(pieces);
    }
    out
}

fn subtract_rect(
    rect: vugra_layout::Rect,
    cut: vugra_layout::Rect,
    out: &mut Vec<vugra_layout::Rect>,
) {
    let Some(overlap) = intersect_rect(rect, cut) else {
        out.push(rect);
        return;
    };
    let rect_right = rect.x + rect.width;
    let rect_bottom = rect.y + rect.height;
    let overlap_right = overlap.x + overlap.width;
    let overlap_bottom = overlap.y + overlap.height;
    push_positive_rect(
        out,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y,
            width: rect.width,
            height: overlap.y - rect.y,
        },
    );
    push_positive_rect(
        out,
        vugra_layout::Rect {
            x: rect.x,
            y: overlap_bottom,
            width: rect.width,
            height: rect_bottom - overlap_bottom,
        },
    );
    push_positive_rect(
        out,
        vugra_layout::Rect {
            x: rect.x,
            y: overlap.y,
            width: overlap.x - rect.x,
            height: overlap.height,
        },
    );
    push_positive_rect(
        out,
        vugra_layout::Rect {
            x: overlap_right,
            y: overlap.y,
            width: rect_right - overlap_right,
            height: overlap.height,
        },
    );
}

fn push_positive_rect(out: &mut Vec<vugra_layout::Rect>, rect: vugra_layout::Rect) {
    if rect.width > 0.0 && rect.height > 0.0 {
        out.push(rect);
    }
}

fn role_fill(role: &str) -> Option<u32> {
    match role {
        "toolbar" => Some(0x00f2f2f4),
        "sidebar" | "sidebar-200" | "sidebar-280" | "sidebar-320" => Some(0x00ececf0),
        "splitter" => Some(0x00d1d1d6),
        "splitter-hover" => Some(0x008bb8f7),
        "file-pane" | "file-list" => Some(0x00ffffff),
        "file-header" => Some(0x00fbfbfc),
        "statusbar" => Some(0x00f5f5f7),
        "overlay" => Some(0x00f7f9fc),
        "dialog-layer" => Some(0x00eef2f7),
        "menu" | "dialog" | "secondary-button" => Some(0x00ffffff),
        "primary-button" => Some(0x000a84ff),
        "menu-item" => Some(0x00ffffff),
        "path" | "search" => Some(0x00ffffff),
        "nav-button" => Some(0x00ffffff),
        "back-icon" | "forward-icon" => Some(0x00374151),
        "sidebar-item" => Some(0x00ececf0),
        "sidebar-section" => Some(0x00ececf0),
        "folder-icon" => Some(0x006fb2ff),
        "file-icon" => Some(0x00ffffff),
        "download-icon" | "project-icon" => Some(0x002563eb),
        "picture-icon" => Some(0x0086efac),
        "chevron-down-icon" | "chevron-right-icon" => Some(0x006b7280),
        _ => None,
    }
}

fn vello_fill_for_role(
    role: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> Option<VelloColor> {
    match role {
        "window" => Some(VelloColor::WINDOW),
        "toolbar" => Some(VelloColor::TOOLBAR),
        "sidebar" | "sidebar-200" | "sidebar-280" | "sidebar-320" => Some(VelloColor::SIDEBAR),
        "splitter" => Some(VelloColor::SPLITTER),
        "splitter-hover" => Some(VelloColor::SPLITTER_HOVER),
        "file-pane" | "file-list" => Some(VelloColor::FILE_PANE),
        "file-header" => Some(VelloColor::COLUMN_HEADER),
        "statusbar" => Some(VelloColor::STATUSBAR),
        "overlay" => Some(VelloColor::OVERLAY),
        "dialog-layer" => Some(VelloColor::DIALOG_LAYER),
        "menu" | "dialog" | "menu-item" | "secondary-button" => Some(VelloColor::FIELD),
        "primary-button" => Some(VelloColor::ROW_SELECTED),
        "path" | "search" => Some(VelloColor::FIELD),
        "nav-button" => Some(VelloColor::NAV_BUTTON),
        "back-icon" | "forward-icon" => Some(VelloColor::NAV_ICON),
        "sidebar-item" if selected => Some(VelloColor::SIDEBAR_ACTIVE),
        "sidebar-item" => Some(VelloColor::SIDEBAR_ITEM),
        "sidebar-section" => Some(VelloColor::SIDEBAR),
        "row" if selected || visual_state == vugra_layout::RowVisualState::Selected => {
            Some(VelloColor::ROW_SELECTED)
        }
        "row" if visual_state == vugra_layout::RowVisualState::Focus => Some(VelloColor::ROW_FOCUS),
        "row" if visual_state == vugra_layout::RowVisualState::Hover => Some(VelloColor::ROW_HOVER),
        "row" if visual_state == vugra_layout::RowVisualState::Editing => {
            Some(VelloColor::ROW_EDITING)
        }
        "row" => Some(VelloColor::ROW),
        "folder-icon" => Some(VelloColor::FOLDER_ICON),
        "file-icon" => Some(VelloColor::FILE_ICON),
        "download-icon" => Some(VelloColor::DOWNLOAD_ICON),
        "picture-icon" => Some(VelloColor::PICTURE_ICON),
        "project-icon" => Some(VelloColor::PROJECT_ICON),
        "chevron-down-icon" | "chevron-right-icon" => Some(VelloColor::CHEVRON_ICON),
        _ => None,
    }
}

fn wgpu_fill_for_role(
    role: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> Option<WgpuColor> {
    match role {
        "window" => Some(WgpuColor::WINDOW),
        "toolbar" => Some(WgpuColor::TOOLBAR),
        "sidebar" | "sidebar-200" | "sidebar-280" | "sidebar-320" => Some(WgpuColor::SIDEBAR),
        "splitter" => Some(WgpuColor::SPLITTER),
        "splitter-hover" => Some(WgpuColor::SPLITTER_HOVER),
        "file-pane" | "file-list" => Some(WgpuColor::FILE_PANE),
        "file-header" => Some(WgpuColor::COLUMN_HEADER),
        "statusbar" => Some(WgpuColor::STATUSBAR),
        "overlay" => Some(WgpuColor::OVERLAY),
        "dialog-layer" => Some(WgpuColor::DIALOG_LAYER),
        "menu" | "dialog" | "menu-item" | "secondary-button" => Some(WgpuColor::FIELD),
        "primary-button" => Some(WgpuColor::ROW_SELECTED),
        "path" | "search" => Some(WgpuColor::FIELD),
        "nav-button" => Some(WgpuColor::NAV_BUTTON),
        "back-icon" | "forward-icon" => Some(WgpuColor::NAV_ICON),
        "sidebar-item" if selected => Some(WgpuColor::SIDEBAR_ACTIVE),
        "sidebar-item" => Some(WgpuColor::SIDEBAR_ITEM),
        "sidebar-section" => Some(WgpuColor::SIDEBAR),
        "row" if selected || visual_state == vugra_layout::RowVisualState::Selected => {
            Some(WgpuColor::ROW_SELECTED)
        }
        "row" if visual_state == vugra_layout::RowVisualState::Focus => Some(WgpuColor::ROW_FOCUS),
        "row" if visual_state == vugra_layout::RowVisualState::Hover => Some(WgpuColor::ROW_HOVER),
        "row" if visual_state == vugra_layout::RowVisualState::Editing => {
            Some(WgpuColor::ROW_EDITING)
        }
        "row" => Some(WgpuColor::ROW),
        "folder-icon" => Some(WgpuColor::FOLDER_ICON),
        "file-icon" => Some(WgpuColor::FILE_ICON),
        "download-icon" => Some(WgpuColor::DOWNLOAD_ICON),
        "picture-icon" => Some(WgpuColor::PICTURE_ICON),
        "project-icon" => Some(WgpuColor::PROJECT_ICON),
        "chevron-down-icon" | "chevron-right-icon" => Some(WgpuColor::CHEVRON_ICON),
        _ => None,
    }
}

fn native_text_color(
    role: &str,
    text: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> u32 {
    if role == "sidebar-item-label" && selected {
        0x000f3d74
    } else if matches!(role, "row-date-cell" | "row-size-cell") {
        0x004b5563
    } else if selected {
        0x00ffffff
    } else if role == "row-name-cell"
        && matches!(
            visual_state,
            vugra_layout::RowVisualState::Focus | vugra_layout::RowVisualState::Editing
        )
    {
        0x000f3d74
    } else if is_subtle_text_role(role) {
        0x006b7280
    } else if is_muted_text_role(role) {
        0x004b5563
    } else if role == "sidebar-item-label" {
        0x00374151
    } else if role == "dialog-title" {
        0x00111827
    } else if text == "FinderLite" {
        0x001f2328
    } else if text == "▣" {
        0x000a84ff
    } else if text == "□" {
        0x001f2937
    } else if text.starts_with("* ") {
        0x00ffffff
    } else if text.starts_with("- ") {
        0x001f2937
    } else if text == "Search" {
        0x009ca3af
    } else if role == "search" {
        0x000f172a
    } else {
        0x001f2937
    }
}

fn is_subtle_text_role(role: &str) -> bool {
    matches!(role, "column-header" | "sidebar-section-label")
}

fn is_muted_text_role(role: &str) -> bool {
    matches!(
        role,
        "statusbar" | "status-text" | "path" | "row-date-cell" | "row-size-cell" | "preview-copy"
    )
}

fn vello_text_color(
    role: &str,
    text: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> VelloColor {
    if role == "sidebar-item-label" && selected {
        VelloColor::ROW_ACCENT_TEXT
    } else if matches!(role, "row-date-cell" | "row-size-cell") {
        VelloColor::TEXT_MUTED
    } else if selected {
        VelloColor::TEXT_INVERTED
    } else if role == "row-name-cell"
        && matches!(
            visual_state,
            vugra_layout::RowVisualState::Focus | vugra_layout::RowVisualState::Editing
        )
    {
        VelloColor::ROW_ACCENT_TEXT
    } else if is_subtle_text_role(role) {
        VelloColor::TEXT_SUBTLE
    } else if is_muted_text_role(role) {
        VelloColor::TEXT_MUTED
    } else if role == "sidebar-item-label" {
        VelloColor::TEXT_SECONDARY
    } else if role == "dialog-title" {
        VelloColor::TITLE
    } else if text == "▣" {
        VelloColor::ROW_SELECTED
    } else if text == "□" {
        VelloColor::TEXT
    } else if text.starts_with("* ") {
        VelloColor::TEXT_INVERTED
    } else if text.starts_with("- ") {
        VelloColor::TEXT
    } else if text == "Search" {
        VelloColor::PLACEHOLDER
    } else if text == "FinderLite" {
        VelloColor::TITLE
    } else if role == "search" {
        VelloColor::INPUT_TEXT
    } else {
        VelloColor::TEXT
    }
}

fn wgpu_text_color(
    role: &str,
    text: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> WgpuColor {
    if role == "sidebar-item-label" && selected {
        WgpuColor::ROW_ACCENT_TEXT
    } else if matches!(role, "row-date-cell" | "row-size-cell") {
        WgpuColor::TEXT_MUTED
    } else if selected {
        WgpuColor::TEXT_INVERTED
    } else if role == "row-name-cell"
        && matches!(
            visual_state,
            vugra_layout::RowVisualState::Focus | vugra_layout::RowVisualState::Editing
        )
    {
        WgpuColor::ROW_ACCENT_TEXT
    } else if is_subtle_text_role(role) {
        WgpuColor::TEXT_SUBTLE
    } else if is_muted_text_role(role) {
        WgpuColor::TEXT_MUTED
    } else if role == "sidebar-item-label" {
        WgpuColor::TEXT_SECONDARY
    } else if role == "dialog-title" {
        WgpuColor::TITLE
    } else if text == "▣" {
        WgpuColor::ROW_SELECTED
    } else if text == "□" {
        WgpuColor::TEXT
    } else if text.starts_with("* ") {
        WgpuColor::TEXT_INVERTED
    } else if text.starts_with("- ") {
        WgpuColor::TEXT
    } else if text == "Search" {
        WgpuColor::PLACEHOLDER
    } else if text == "FinderLite" {
        WgpuColor::TITLE
    } else if role == "search" {
        WgpuColor::INPUT_TEXT
    } else {
        WgpuColor::TEXT
    }
}

pub fn render_vello_pixels(ops: &[VelloOp], width: usize, height: usize) -> Vec<u32> {
    let width = width.max(1);
    let height = height.max(1);
    let mut pixels = vec![0x00f7f7f8; width * height];

    for op in ops {
        match op {
            VelloOp::Fill {
                role,
                rect,
                selected,
                visual_state,
                color,
                ..
            } => {
                if draw_role_icon(&mut pixels, width, height, role, *rect, None) {
                } else {
                    paint_role_surface(
                        &mut pixels,
                        width,
                        height,
                        role,
                        *rect,
                        vello_color_u32(*color),
                        *selected,
                        *visual_state,
                        None,
                    );
                }
            }
            VelloOp::Text {
                rect, run, color, ..
            } => {
                draw_text_run_clipped(
                    &mut pixels,
                    width,
                    height,
                    run,
                    vello_color_u32(*color),
                    *rect,
                );
            }
            VelloOp::End { .. } => {}
        }
    }
    pixels
}

fn render_vello_pixels_into_dirty(
    pixels: &mut [u32],
    ops: &[VelloOp],
    width: usize,
    height: usize,
    dirty: &[vugra_layout::Rect],
) {
    let dirty = disjoint_dirty_rects(dirty);
    for rect in &dirty {
        fill_rect_from_rect(pixels, width, height, *rect, 0x00f7f7f8);
    }
    for op in ops {
        for rect in &dirty {
            if vello_op_intersects_rect(op, *rect) {
                render_vello_op_clipped(pixels, op, width, height, *rect);
            }
        }
    }
}

fn vello_op_intersects_rect(op: &VelloOp, dirty_rect: vugra_layout::Rect) -> bool {
    let rect = match op {
        VelloOp::Fill { rect, .. } => *rect,
        VelloOp::Text { rect, run, .. } => text_run_bounds(run, *rect),
        VelloOp::End { .. } => return false,
    };
    rects_intersect(rect, dirty_rect)
}

fn render_vello_op_clipped(
    pixels: &mut [u32],
    op: &VelloOp,
    width: usize,
    height: usize,
    clip: vugra_layout::Rect,
) {
    match op {
        VelloOp::Fill {
            role,
            rect,
            selected,
            visual_state,
            color,
            ..
        } => {
            if draw_role_icon(pixels, width, height, role, *rect, Some(clip)) {
            } else {
                paint_role_surface(
                    pixels,
                    width,
                    height,
                    role,
                    *rect,
                    vello_color_u32(*color),
                    *selected,
                    *visual_state,
                    Some(clip),
                );
            }
        }
        VelloOp::Text {
            rect, run, color, ..
        } => {
            if let Some(text_clip) = intersect_rect(*rect, clip) {
                draw_text_run_clipped(
                    pixels,
                    width,
                    height,
                    run,
                    vello_color_u32(*color),
                    text_clip,
                );
            }
        }
        VelloOp::End { .. } => {}
    }
}

pub fn render_wgpu_pixels(pass: &[WgpuPassOp], width: usize, height: usize) -> Vec<u32> {
    let width = width.max(1);
    let height = height.max(1);
    let mut pixels = vec![0x00f7f7f8; width * height];

    for op in pass {
        match op {
            WgpuPassOp::Quad {
                role,
                rect,
                selected,
                visual_state,
                color,
                ..
            } => {
                if draw_role_icon(&mut pixels, width, height, role, *rect, None) {
                } else {
                    paint_role_surface(
                        &mut pixels,
                        width,
                        height,
                        role,
                        *rect,
                        wgpu_color_u32(*color),
                        *selected,
                        *visual_state,
                        None,
                    );
                }
            }
            WgpuPassOp::Text {
                rect, run, color, ..
            } => {
                draw_text_run_clipped(
                    &mut pixels,
                    width,
                    height,
                    run,
                    wgpu_color_u32(*color),
                    *rect,
                );
            }
            WgpuPassOp::End { .. } => {}
        }
    }
    pixels
}

fn render_wgpu_pixels_into_dirty(
    pixels: &mut [u32],
    pass: &[WgpuPassOp],
    width: usize,
    height: usize,
    dirty: &[vugra_layout::Rect],
) {
    let dirty = disjoint_dirty_rects(dirty);
    for rect in &dirty {
        fill_rect_from_rect(pixels, width, height, *rect, 0x00f7f7f8);
    }
    for op in pass {
        for rect in &dirty {
            if wgpu_op_intersects_rect(op, *rect) {
                render_wgpu_op_clipped(pixels, op, width, height, *rect);
            }
        }
    }
}

fn wgpu_op_intersects_rect(op: &WgpuPassOp, dirty_rect: vugra_layout::Rect) -> bool {
    let rect = match op {
        WgpuPassOp::Quad { rect, .. } => *rect,
        WgpuPassOp::Text { rect, run, .. } => text_run_bounds(run, *rect),
        WgpuPassOp::End { .. } => return false,
    };
    rects_intersect(rect, dirty_rect)
}

fn render_wgpu_op_clipped(
    pixels: &mut [u32],
    op: &WgpuPassOp,
    width: usize,
    height: usize,
    clip: vugra_layout::Rect,
) {
    match op {
        WgpuPassOp::Quad {
            role,
            rect,
            selected,
            visual_state,
            color,
            ..
        } => {
            if draw_role_icon(pixels, width, height, role, *rect, Some(clip)) {
            } else {
                paint_role_surface(
                    pixels,
                    width,
                    height,
                    role,
                    *rect,
                    wgpu_color_u32(*color),
                    *selected,
                    *visual_state,
                    Some(clip),
                );
            }
        }
        WgpuPassOp::Text {
            rect, run, color, ..
        } => {
            if let Some(text_clip) = intersect_rect(*rect, clip) {
                draw_text_run_clipped(
                    pixels,
                    width,
                    height,
                    run,
                    wgpu_color_u32(*color),
                    text_clip,
                );
            }
        }
        WgpuPassOp::End { .. } => {}
    }
}

fn vello_color_u32(color: VelloColor) -> u32 {
    let VelloColor(red, green, blue, _) = color;
    ((red as u32) << 16) | ((green as u32) << 8) | blue as u32
}

fn wgpu_color_u32(color: WgpuColor) -> u32 {
    let WgpuColor(red, green, blue, _) = color;
    ((red as u32) << 16) | ((green as u32) << 8) | blue as u32
}

fn commands_from_vello_ops(ops: &[VelloOp]) -> Vec<RenderCommand> {
    ops.iter()
        .map(|op| match op {
            VelloOp::Fill {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
                ..
            } => RenderCommand::Element {
                id: id.clone(),
                role: role.clone(),
                rect: *rect,
                selected: *selected,
                visual_state: *visual_state,
                method: *method,
            },
            VelloOp::Text {
                id,
                text,
                role,
                rect,
                ..
            } => RenderCommand::Text {
                id: id.clone(),
                text: text.clone(),
                role: role.clone(),
                rect: *rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            },
            VelloOp::End { id } => RenderCommand::End { id: id.clone() },
        })
        .collect()
}

fn commands_from_wgpu_pass(pass: &[WgpuPassOp]) -> Vec<RenderCommand> {
    pass.iter()
        .map(|op| match op {
            WgpuPassOp::Quad {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
                ..
            } => RenderCommand::Element {
                id: id.clone(),
                role: role.clone(),
                rect: *rect,
                selected: *selected,
                visual_state: *visual_state,
                method: *method,
            },
            WgpuPassOp::Text {
                id,
                text,
                role,
                rect,
                ..
            } => RenderCommand::Text {
                id: id.clone(),
                text: text.clone(),
                role: role.clone(),
                rect: *rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            },
            WgpuPassOp::End { id } => RenderCommand::End { id: id.clone() },
        })
        .collect()
}

pub fn run_app_window<S: ComponentState>(
    app: &mut App<S>,
    config: NativeWindowConfig,
) -> Result<(), minifb::Error> {
    run_app_window_loop(app, config, None).map(|_| ())
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct NativeWindowSmoke {
    pub frames_presented: usize,
    pub commands: usize,
    pub pixels: usize,
    pub drawn_pixels: usize,
}

pub fn run_app_window_for_frames<S: ComponentState>(
    app: &mut App<S>,
    config: NativeWindowConfig,
    frames: usize,
) -> Result<NativeWindowSmoke, minifb::Error> {
    run_app_window_loop(app, config, Some(frames.max(1)))
}

fn run_app_window_loop<S: ComponentState>(
    app: &mut App<S>,
    config: NativeWindowConfig,
    max_frames: Option<usize>,
) -> Result<NativeWindowSmoke, minifb::Error> {
    let width = config.width.max(1);
    let height = config.height.max(1);
    let constraints = Constraints {
        width: width as f32,
        height: height as f32,
    };
    let mut frame_state = NativeFrameState::new(config.backend, width, height);
    let mut frame = frame_state.render(app, constraints);
    let mut window = Window::new(&config.title, width, height, WindowOptions::default())?;
    window.set_target_fps(60);
    let input = Rc::new(RefCell::new(Vec::new()));
    window.set_input_callback(Box::new(TextInput {
        chars: input.clone(),
    }));
    let mut mouse_was_down = false;
    let mut last_click: Option<(Instant, f32, f32)> = None;
    let mut context_was_down = false;
    let mut last_mouse_pos: Option<(f32, f32)> = None;
    let mut frames_presented = 0usize;
    while window.is_open() {
        if max_frames.is_some_and(|limit| frames_presented >= limit) {
            break;
        }
        let modifiers = window_modifiers(&window);
        if let Some(key) = window_native_key(&window, modifiers) {
            if dispatch_native_key(app, key) {
                frame = frame_state.render(app, constraints);
            }
        } else if window.is_key_pressed(Key::Down, KeyRepeat::Yes) {
            if dispatch_native_key(app, NativeKey::ArrowDown) {
                frame = frame_state.render(app, constraints);
            }
        } else if window.is_key_pressed(Key::Up, KeyRepeat::Yes) {
            if dispatch_native_key(app, NativeKey::ArrowUp) {
                frame = frame_state.render(app, constraints);
            }
        } else if window.is_key_pressed(Key::Enter, KeyRepeat::No)
            || window.is_key_pressed(Key::NumPadEnter, KeyRepeat::No)
        {
            if dispatch_native_key(app, NativeKey::Enter) {
                frame = frame_state.render(app, constraints);
            }
        } else if window.is_key_pressed(Key::Backspace, KeyRepeat::Yes)
            && dispatch_native_key(app, NativeKey::Backspace)
        {
            frame = frame_state.render(app, constraints);
        } else if window.is_key_pressed(Key::Escape, KeyRepeat::No) {
            if dispatch_native_key(app, NativeKey::Escape) {
                frame = frame_state.render(app, constraints);
            } else {
                break;
            }
        }
        let typed = drain_text_input(&input);
        if !typed.is_empty() && dispatch_native_text(app, typed) {
            frame = frame_state.render(app, constraints);
        }
        if let Some((x, y)) = window.get_mouse_pos(MouseMode::Discard) {
            if last_mouse_pos != Some((x, y)) {
                let modifiers = window_modifiers(&window);
                if mouse_was_down {
                    if let Some((last_x, last_y)) = last_mouse_pos {
                        if dispatch_native_drag_with_modifiers(
                            app,
                            &frame,
                            x,
                            y,
                            x - last_x,
                            y - last_y,
                            modifiers,
                        ) {
                            frame = frame_state.render(app, constraints);
                        }
                    }
                } else if dispatch_native_hover_with_modifiers(app, &frame, x, y, modifiers) {
                    frame = frame_state.render(app, constraints);
                }
                last_mouse_pos = Some((x, y));
            }
        }
        if let Some((_scroll_x, scroll_y)) = window.get_scroll_wheel() {
            if scroll_y != 0.0 {
                if let Some((x, y)) = window.get_mouse_pos(MouseMode::Discard) {
                    let scrolled = frame
                        .scroll_target(x, y)
                        .map(|scroll| frame_state.apply_scroll_delta(scroll, scroll_y))
                        .unwrap_or(false);
                    let dispatched = dispatch_native_scroll(app, &frame, x, y, scroll_y);
                    if scrolled || dispatched {
                        frame = frame_state.render(app, constraints);
                    }
                }
            }
        }
        let mouse_down = window.get_mouse_down(MouseButton::Left);
        if mouse_down && !mouse_was_down {
            if let Some((x, y)) = window.get_mouse_pos(MouseMode::Discard) {
                let now = Instant::now();
                let double_click = last_click.is_some_and(|(last, last_x, last_y)| {
                    now.duration_since(last) <= Duration::from_millis(500)
                        && (x - last_x).abs() <= 4.0
                        && (y - last_y).abs() <= 4.0
                });
                let handled = if double_click {
                    last_click = None;
                    dispatch_native_double_click_with_modifiers(
                        app,
                        &frame,
                        x,
                        y,
                        window_modifiers(&window),
                    )
                } else {
                    last_click = Some((now, x, y));
                    dispatch_native_pointer_with_modifiers(
                        app,
                        &frame,
                        x,
                        y,
                        window_modifiers(&window),
                    )
                };
                if handled {
                    frame = frame_state.render(app, constraints);
                }
            }
        }
        mouse_was_down = mouse_down;
        let context_down = window.get_mouse_down(MouseButton::Right);
        if context_down && !context_was_down {
            if let Some((x, y)) = window.get_mouse_pos(MouseMode::Discard) {
                if dispatch_native_context_menu_with_modifiers(
                    app,
                    &frame,
                    x,
                    y,
                    window_modifiers(&window),
                ) {
                    frame = frame_state.render(app, constraints);
                }
            }
        }
        context_was_down = context_down;
        window.update_with_buffer(&frame.pixels, width, height)?;
        frames_presented += 1;
    }
    Ok(NativeWindowSmoke {
        frames_presented,
        commands: frame.commands.len(),
        pixels: frame.pixels.len(),
        drawn_pixels: frame
            .pixels
            .iter()
            .filter(|pixel| **pixel != 0x00f7f7f8)
            .count(),
    })
}

fn window_modifiers(window: &Window) -> Modifiers {
    Modifiers {
        shift: window.is_key_down(Key::LeftShift) || window.is_key_down(Key::RightShift),
        ctrl: window.is_key_down(Key::LeftCtrl) || window.is_key_down(Key::RightCtrl),
        meta: window.is_key_down(Key::LeftSuper) || window.is_key_down(Key::RightSuper),
        alt: window.is_key_down(Key::LeftAlt) || window.is_key_down(Key::RightAlt),
    }
}

fn native_key_for_window_key(key: Key, modifiers: Modifiers) -> Option<NativeKey> {
    match key {
        Key::A if modifiers.meta || modifiers.ctrl => Some(NativeKey::SelectAll),
        Key::Delete => Some(NativeKey::Delete),
        _ => None,
    }
}

fn window_native_key(window: &Window, modifiers: Modifiers) -> Option<NativeKey> {
    if window.is_key_pressed(Key::A, KeyRepeat::No) {
        return native_key_for_window_key(Key::A, modifiers);
    }
    if window.is_key_pressed(Key::Delete, KeyRepeat::Yes) {
        return native_key_for_window_key(Key::Delete, modifiers);
    }
    None
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum NativeKey {
    ArrowDown,
    ArrowUp,
    Enter,
    Backspace,
    Escape,
    Delete,
    SelectAll,
}

pub fn dispatch_native_key<S: ComponentState>(app: &mut App<S>, key: NativeKey) -> bool {
    let frame = app.render_frame();
    if !frame.overlays.rename_text.is_empty() {
        return dispatch_rename_key(app, &frame, key);
    }
    let method = match key {
        NativeKey::ArrowDown => Some(MethodId(9)),
        NativeKey::ArrowUp => Some(MethodId(8)),
        NativeKey::Enter => Some(MethodId(13)),
        NativeKey::Backspace => {
            if frame.search_query.is_empty() {
                Some(MethodId(14))
            } else {
                frame.search_backspace_method
            }
        }
        NativeKey::Escape => frame.overlays.dismiss_overlay_method,
        NativeKey::Delete => frame.overlays.delete_selected_method,
        NativeKey::SelectAll => Some(MethodId(80)),
    };
    let Some(method) = method else {
        return false;
    };
    app.dispatch(method);
    true
}

pub fn dispatch_native_text<S: ComponentState>(app: &mut App<S>, text: String) -> bool {
    if text.is_empty() {
        return false;
    }
    let frame = app.render_frame();
    if !frame.overlays.rename_text.is_empty() {
        let Some(signal) = frame.overlays.rename_text_signal else {
            return false;
        };
        let mut next = frame.overlays.rename_text;
        next.push_str(&text);
        app.state_mut().set_signal(signal, next.into());
        return true;
    }
    let Some(method) = frame.search_input_method else {
        return false;
    };
    app.dispatch_event(
        method,
        Event {
            kind: "text".to_string(),
            text,
            ..Event::default()
        },
    );
    true
}

fn dispatch_rename_key<S: ComponentState>(
    app: &mut App<S>,
    frame: &vugra_core::Frame,
    key: NativeKey,
) -> bool {
    match key {
        NativeKey::Enter => {
            let Some(method) = frame.overlays.commit_rename_method else {
                return false;
            };
            app.dispatch(method);
            true
        }
        NativeKey::Escape => {
            let Some(method) = frame.overlays.cancel_rename_method else {
                return false;
            };
            app.dispatch(method);
            true
        }
        NativeKey::Backspace => {
            let Some(signal) = frame.overlays.rename_text_signal else {
                return false;
            };
            let mut chars: Vec<char> = frame.overlays.rename_text.chars().collect();
            if chars.is_empty() {
                return false;
            }
            chars.pop();
            app.state_mut()
                .set_signal(signal, chars.into_iter().collect::<String>().into());
            true
        }
        NativeKey::ArrowDown | NativeKey::ArrowUp | NativeKey::Delete | NativeKey::SelectAll => {
            false
        }
    }
}

pub fn dispatch_native_scroll<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    delta_y: f32,
) -> bool {
    let Some(scroll) = frame.scroll_target(x, y) else {
        return false;
    };
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route(&frame.hit_test, &route, |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "scroll".to_string(),
                x,
                y,
                delta_y,
                text: scroll.id.clone(),
                ..Event::default()
            },
        );
        true
    })
}

impl NativeFrame {
    pub fn scroll_target(&self, x: f32, y: f32) -> Option<&ScrollNode> {
        hit_test_scroll_node(&self.scrolls, x, y)
    }
}

pub fn dispatch_native_hover<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
) -> bool {
    dispatch_native_hover_with_modifiers(app, frame, x, y, Modifiers::default())
}

pub fn dispatch_native_hover_with_modifiers<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    modifiers: Modifiers,
) -> bool {
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route_for_kind(&frame.hit_test, &route, "hover", |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "hover".to_string(),
                x,
                y,
                modifiers,
                ..Event::default()
            },
        );
        true
    })
}

pub fn dispatch_native_drag<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    delta_x: f32,
    delta_y: f32,
) -> bool {
    dispatch_native_drag_with_modifiers(app, frame, x, y, delta_x, delta_y, Modifiers::default())
}

pub fn dispatch_native_drag_with_modifiers<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    delta_x: f32,
    delta_y: f32,
    modifiers: Modifiers,
) -> bool {
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route_for_kind(&frame.hit_test, &route, "drag", |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "drag".to_string(),
                x,
                y,
                delta_x,
                delta_y,
                modifiers,
                ..Event::default()
            },
        );
        true
    })
}

pub fn dispatch_native_pointer<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
) -> bool {
    dispatch_native_pointer_with_modifiers(app, frame, x, y, Modifiers::default())
}

pub fn dispatch_native_pointer_with_modifiers<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    modifiers: Modifiers,
) -> bool {
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route(&frame.hit_test, &route, |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "click".to_string(),
                x,
                y,
                modifiers,
                ..Event::default()
            },
        );
        true
    })
}

pub fn dispatch_native_double_click<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
) -> bool {
    dispatch_native_double_click_with_modifiers(app, frame, x, y, Modifiers::default())
}

pub fn dispatch_native_double_click_with_modifiers<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    modifiers: Modifiers,
) -> bool {
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route_for_kind(&frame.hit_test, &route, "dblclick", |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "dblclick".to_string(),
                x,
                y,
                modifiers,
                ..Event::default()
            },
        );
        true
    })
}

pub fn dispatch_native_context_menu<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
) -> bool {
    dispatch_native_context_menu_with_modifiers(app, frame, x, y, Modifiers::default())
}

pub fn dispatch_native_context_menu_with_modifiers<S: ComponentState>(
    app: &mut App<S>,
    frame: &NativeFrame,
    x: f32,
    y: f32,
    modifiers: Modifiers,
) -> bool {
    let Some(route) = hit_test_route(&frame.hit_test, x, y) else {
        return false;
    };
    dispatch_event_route_for_kind(&frame.hit_test, &route, "contextmenu", |step| {
        app.dispatch_event(
            step.method,
            Event {
                kind: "contextmenu".to_string(),
                x,
                y,
                modifiers,
                ..Event::default()
            },
        );
        true
    })
}

struct TextInput {
    chars: Rc<RefCell<Vec<u32>>>,
}

impl InputCallback for TextInput {
    fn add_char(&mut self, uni_char: u32) {
        self.chars.borrow_mut().push(uni_char);
    }
}

fn drain_text_input(input: &Rc<RefCell<Vec<u32>>>) -> String {
    let mut chars = input.borrow_mut();
    let text: String = chars
        .drain(..)
        .filter_map(char::from_u32)
        .filter(|ch| !ch.is_control())
        .collect();
    text
}

fn draw_text(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    x: f32,
    y: f32,
    text: &str,
    color: u32,
) {
    let run = vugra_text::layout_text_run(text, x, y, 13.0);
    for pixel in run.pixels {
        fill_rect_float(
            pixels,
            width,
            height,
            pixel.x,
            pixel.y,
            pixel.width,
            pixel.height,
            color,
            pixel.alpha,
        );
    }
}

fn draw_text_run_clipped(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    run: &vugra_text::TextRun,
    color: u32,
    clip: vugra_layout::Rect,
) {
    for pixel in &run.pixels {
        let pixel_rect = vugra_layout::Rect {
            x: pixel.x,
            y: pixel.y,
            width: pixel.width,
            height: pixel.height,
        };
        if let Some(clipped) = intersect_rect(pixel_rect, clip) {
            fill_rect_float(
                pixels,
                width,
                height,
                clipped.x,
                clipped.y,
                clipped.width,
                clipped.height,
                color,
                pixel.alpha,
            );
        }
    }
}

fn draw_folder_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    let tab = vugra_layout::Rect {
        x: rect.x + 1.0,
        y: rect.y + 3.0,
        width: rect.width * 0.46,
        height: 5.0,
    };
    let top = vugra_layout::Rect {
        x: rect.x + 3.0,
        y: rect.y + 5.0,
        width: rect.width - 4.0,
        height: 4.0,
    };
    let body = vugra_layout::Rect {
        x: rect.x + 1.0,
        y: rect.y + 7.0,
        width: rect.width - 2.0,
        height: rect.height - 6.0,
    };
    fill_icon_rect(pixels, width, height, tab, 0x0078b7ff, clip);
    fill_icon_rect(pixels, width, height, top, 0x004f9cf9, clip);
    fill_icon_rect(pixels, width, height, body, 0x006fb2ff, clip);
}

fn draw_role_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    role: &str,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) -> bool {
    match role {
        "folder-icon" => draw_folder_icon(pixels, width, height, rect, clip),
        "file-icon" => draw_file_icon(pixels, width, height, rect, clip),
        "download-icon" => draw_download_icon(pixels, width, height, rect, clip),
        "picture-icon" => draw_picture_icon(pixels, width, height, rect, clip),
        "project-icon" => draw_project_icon(pixels, width, height, rect, clip),
        "chevron-down-icon" => draw_chevron_down_icon(pixels, width, height, rect, clip),
        "chevron-right-icon" => draw_chevron_right_icon(pixels, width, height, rect, clip),
        "back-icon" => draw_back_icon(pixels, width, height, rect, clip),
        "forward-icon" => draw_forward_icon(pixels, width, height, rect, clip),
        _ => return false,
    }
    true
}

fn draw_file_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    let page = vugra_layout::Rect {
        x: rect.x + 3.0,
        y: rect.y + 1.0,
        width: rect.width - 6.0,
        height: rect.height - 2.0,
    };
    let fold = vugra_layout::Rect {
        x: rect.x + rect.width - 7.0,
        y: rect.y + 1.0,
        width: 4.0,
        height: 5.0,
    };
    fill_icon_rect(pixels, width, height, page, 0x00ffffff, clip);
    draw_icon_border(pixels, width, height, page, 0x009ca3af, clip);
    fill_icon_rect(pixels, width, height, fold, 0x00e5e7eb, clip);
    for offset in [8.0, 11.0, 14.0] {
        fill_icon_rect(
            pixels,
            width,
            height,
            vugra_layout::Rect {
                x: rect.x + 6.0,
                y: rect.y + offset,
                width: if offset == 14.0 { 5.0 } else { 7.0 },
                height: 1.0,
            },
            0x009ca3af,
            clip,
        );
    }
}

fn draw_download_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 3.0,
            width: 2.0,
            height: 9.0,
        },
        0x002563eb,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 10.0,
            width: 8.0,
            height: 2.0,
        },
        0x002563eb,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 12.0,
            width: 6.0,
            height: 2.0,
        },
        0x002563eb,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 3.0,
            y: rect.y + 16.0,
            width: 12.0,
            height: 2.0,
        },
        0x0064748b,
        clip,
    );
}

fn draw_picture_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    let frame = vugra_layout::Rect {
        x: rect.x + 2.0,
        y: rect.y + 3.0,
        width: rect.width - 4.0,
        height: rect.height - 5.0,
    };
    fill_icon_rect(pixels, width, height, frame, 0x00ffffff, clip);
    draw_icon_border(pixels, width, height, frame, 0x0094a3b8, clip);
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 7.0,
            width: 3.0,
            height: 3.0,
        },
        0x00f59e0b,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 4.0,
            y: rect.y + 13.0,
            width: 11.0,
            height: 3.0,
        },
        0x0086efac,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 11.0,
            width: 5.0,
            height: 3.0,
        },
        0x0022c55e,
        clip,
    );
}

fn draw_project_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    let page = vugra_layout::Rect {
        x: rect.x + 3.0,
        y: rect.y + 3.0,
        width: rect.width - 6.0,
        height: rect.height - 5.0,
    };
    fill_icon_rect(pixels, width, height, page, 0x00f8fafc, clip);
    draw_icon_border(pixels, width, height, page, 0x0064748b, clip);
    for offset in [7.0, 10.0, 13.0] {
        fill_icon_rect(
            pixels,
            width,
            height,
            vugra_layout::Rect {
                x: rect.x + 6.0,
                y: rect.y + offset,
                width: if offset == 13.0 { 6.0 } else { 8.0 },
                height: 1.0,
            },
            0x002563eb,
            clip,
        );
    }
}

fn draw_chevron_down_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 4.0,
            y: rect.y + 6.0,
            width: 8.0,
            height: 2.0,
        },
        0x006b7280,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 8.0,
            width: 4.0,
            height: 2.0,
        },
        0x006b7280,
        clip,
    );
}

fn draw_chevron_right_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 4.0,
            width: 2.0,
            height: 8.0,
        },
        0x006b7280,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 6.0,
            width: 2.0,
            height: 4.0,
        },
        0x006b7280,
        clip,
    );
}

fn draw_back_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 14.0,
            width: 8.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 12.0,
            width: 3.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 16.0,
            width: 3.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
}

fn draw_forward_icon(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 14.0,
            width: 8.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 10.0,
            y: rect.y + 12.0,
            width: 3.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + 10.0,
            y: rect.y + 16.0,
            width: 3.0,
            height: 2.0,
        },
        0x00374151,
        clip,
    );
}

fn draw_icon_border(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    color: u32,
    clip: Option<vugra_layout::Rect>,
) {
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y,
            width: rect.width,
            height: 1.0,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y + rect.height - 1.0,
            width: rect.width,
            height: 1.0,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x,
            y: rect.y,
            width: 1.0,
            height: rect.height,
        },
        color,
        clip,
    );
    fill_icon_rect(
        pixels,
        width,
        height,
        vugra_layout::Rect {
            x: rect.x + rect.width - 1.0,
            y: rect.y,
            width: 1.0,
            height: rect.height,
        },
        color,
        clip,
    );
}

fn fill_icon_rect(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    color: u32,
    clip: Option<vugra_layout::Rect>,
) {
    let rect = if let Some(clip) = clip {
        let Some(clipped) = intersect_rect(rect, clip) else {
            return;
        };
        clipped
    } else {
        rect
    };
    fill_rect_from_rect(pixels, width, height, rect, color);
}

fn fill_rounded_rect(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    radius: f32,
    color: u32,
    clip: Option<vugra_layout::Rect>,
) {
    let shape_rect = rect;
    let paint_rect = if let Some(clip) = clip {
        let Some(clipped) = intersect_rect(rect, clip) else {
            return;
        };
        clipped
    } else {
        rect
    };
    if paint_rect.width <= 0.0 || paint_rect.height <= 0.0 {
        return;
    }

    let radius = radius
        .min(shape_rect.width / 2.0)
        .min(shape_rect.height / 2.0);
    let radius_sq = radius * radius;
    let start_x = paint_rect.x.floor().max(0.0) as usize;
    let start_y = paint_rect.y.floor().max(0.0) as usize;
    let end_x = (paint_rect.x + paint_rect.width)
        .ceil()
        .max(0.0)
        .min(width as f32) as usize;
    let end_y = (paint_rect.y + paint_rect.height)
        .ceil()
        .max(0.0)
        .min(height as f32) as usize;
    for py in start_y..end_y {
        for px in start_x..end_x {
            let cx = px as f32 + 0.5;
            let cy = py as f32 + 0.5;
            if !point_in_rounded_rect(cx, cy, shape_rect, radius, radius_sq) {
                continue;
            }
            set_pixel(pixels, width, height, px, py, color);
        }
    }
}

fn point_in_rounded_rect(
    x: f32,
    y: f32,
    rect: vugra_layout::Rect,
    radius: f32,
    radius_sq: f32,
) -> bool {
    if radius <= 0.0 {
        return true;
    }
    let left = rect.x + radius;
    let right = rect.x + rect.width - radius;
    let top = rect.y + radius;
    let bottom = rect.y + rect.height - radius;
    let dx = if x < left {
        left - x
    } else if x > right {
        x - right
    } else {
        0.0
    };
    let dy = if y < top {
        top - y
    } else if y > bottom {
        y - bottom
    } else {
        0.0
    };
    dx * dx + dy * dy <= radius_sq
}

fn fill_rect(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    x: usize,
    y: usize,
    rect_width: usize,
    rect_height: usize,
    color: u32,
) {
    let end_y = (y + rect_height).min(height);
    let end_x = (x + rect_width).min(width);
    for py in y..end_y {
        for px in x..end_x {
            set_pixel(pixels, width, height, px, py, color);
        }
    }
}

fn fill_rect_from_rect(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    rect: vugra_layout::Rect,
    color: u32,
) {
    fill_rect_float(
        pixels,
        width,
        height,
        rect.x,
        rect.y,
        rect.width,
        rect.height,
        color,
        1.0,
    );
}

fn dirty_covers_full_surface(dirty: &[vugra_layout::Rect], width: usize, height: usize) -> bool {
    dirty.iter().any(|rect| {
        rect.x <= 0.0
            && rect.y <= 0.0
            && rect.x + rect.width >= width as f32
            && rect.y + rect.height >= height as f32
    })
}

fn fill_rect_float(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    x: f32,
    y: f32,
    rect_width: f32,
    rect_height: f32,
    color: u32,
    alpha: f32,
) {
    if rect_width <= 0.0 || rect_height <= 0.0 || alpha <= 0.0 {
        return;
    }
    let start_x = x.floor().max(0.0) as usize;
    let start_y = y.floor().max(0.0) as usize;
    let end_x = (x + rect_width).ceil().max(0.0).min(width as f32) as usize;
    let end_y = (y + rect_height).ceil().max(0.0).min(height as f32) as usize;
    for py in start_y..end_y {
        for px in start_x..end_x {
            blend_pixel(pixels, width, height, px, py, color, alpha);
        }
    }
}

fn blend_pixel(
    pixels: &mut [u32],
    width: usize,
    height: usize,
    x: usize,
    y: usize,
    color: u32,
    alpha: f32,
) {
    if alpha >= 1.0 {
        set_pixel(pixels, width, height, x, y, color);
        return;
    }
    if x >= width || y >= height {
        return;
    }
    let dst = pixels[y * width + x];
    let alpha = alpha.clamp(0.0, 1.0);
    let src_r = ((color >> 16) & 0xff) as f32;
    let src_g = ((color >> 8) & 0xff) as f32;
    let src_b = (color & 0xff) as f32;
    let dst_r = ((dst >> 16) & 0xff) as f32;
    let dst_g = ((dst >> 8) & 0xff) as f32;
    let dst_b = (dst & 0xff) as f32;
    let out_r = (src_r * alpha + dst_r * (1.0 - alpha)).round() as u32;
    let out_g = (src_g * alpha + dst_g * (1.0 - alpha)).round() as u32;
    let out_b = (src_b * alpha + dst_b * (1.0 - alpha)).round() as u32;
    pixels[y * width + x] = (out_r << 16) | (out_g << 8) | out_b;
}

fn set_pixel(pixels: &mut [u32], width: usize, height: usize, x: usize, y: usize, color: u32) {
    if x < width && y < height {
        pixels[y * width + x] = color;
    }
}

#[cfg(test)]
mod native_tests {
    use super::*;
    use std::collections::HashMap;
    use vugra_core::{finder_lite_contract, SignalId, Value};
    use vugra_layout::{EventHandlers, Rect};
    use vugra_scene::hit_test;

    #[test]
    fn text_pixels_include_drawn_content() {
        let pixels = render_text_pixels("FinderLite\npath: Documents", 320, 200);
        let changed = pixels.iter().filter(|pixel| **pixel == 0x001f2328).count();
        assert!(changed > 20, "expected title text pixels, got {changed}");
    }

    #[test]
    fn command_pixels_include_render_command_text() {
        let rect = vugra_layout::Rect {
            x: 12.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let pixels = render_command_pixels(
            &[RenderCommand::Text {
                id: "title:text".to_string(),
                text: "FinderLite".to_string(),
                role: "heading".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            320,
            200,
        );
        let changed = pixels.iter().filter(|pixel| **pixel == 0x001f2328).count();
        assert!(changed > 20, "expected command text pixels, got {changed}");
    }

    #[test]
    fn native_text_pixels_clip_to_text_rect_on_full_paint() {
        let rect = vugra_layout::Rect {
            x: 10.0,
            y: 10.0,
            width: 18.0,
            height: 13.0,
        };
        let text = "FinderLite FinderLite";
        let pixels = render_command_pixels(
            &[RenderCommand::Text {
                id: "label:text".to_string(),
                text: text.to_string(),
                role: "text".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            80,
            40,
        );
        assert!(
            pixels.iter().filter(|pixel| **pixel == 0x001f2937).count() > 0,
            "expected text pixels inside clipped text rect"
        );
        for y in 0..40 {
            for x in 28..80 {
                assert_eq!(
                    pixels[y * 80 + x],
                    0x00f7f7f8,
                    "leaked text pixel at {x},{y}"
                );
            }
        }
    }

    #[test]
    fn native_backend_text_pixels_clip_to_text_rect_consistently() {
        let rect = vugra_layout::Rect {
            x: 10.0,
            y: 10.0,
            width: 18.0,
            height: 13.0,
        };
        let text = "FinderLite FinderLite";
        let run = vugra_text::layout_text_run_wrapped(text, rect.x, rect.y, 13.0, Some(rect.width));
        let vello_pixels = render_vello_pixels(
            &[VelloOp::Text {
                id: "label:text".to_string(),
                text: text.to_string(),
                role: "text".to_string(),
                rect,
                run: run.clone(),
                color: VelloColor::TEXT,
            }],
            80,
            40,
        );
        let wgpu_pixels = render_wgpu_pixels(
            &[WgpuPassOp::Text {
                id: "label:text".to_string(),
                rect,
                text: text.to_string(),
                role: "text".to_string(),
                run,
                color: WgpuColor::TEXT,
            }],
            80,
            40,
        );
        assert_eq!(vello_pixels, wgpu_pixels);
        assert!(
            vello_pixels
                .iter()
                .filter(|pixel| **pixel == 0x001f2937)
                .count()
                > 0,
            "expected backend text pixels inside clipped text rect"
        );
        for pixels in [&vello_pixels, &wgpu_pixels] {
            for y in 0..40 {
                for x in 28..80 {
                    assert_eq!(
                        pixels[y * 80 + x],
                        0x00f7f7f8,
                        "leaked text pixel at {x},{y}"
                    );
                }
            }
        }
    }

    #[test]
    fn native_file_row_icons_render_as_shapes_not_text_glyphs() {
        let folder = vugra_layout::Rect {
            x: 8.0,
            y: 8.0,
            width: 18.0,
            height: 18.0,
        };
        let file = vugra_layout::Rect {
            x: 32.0,
            y: 8.0,
            width: 18.0,
            height: 18.0,
        };
        let commands = [
            RenderCommand::Element {
                id: "folder".to_string(),
                role: "folder-icon".to_string(),
                rect: folder,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "file".to_string(),
                role: "file-icon".to_string(),
                rect: file,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
        ];
        let software = render_command_pixels(&commands, 80, 40);
        let vello = render_vello_pixels(
            &[
                VelloOp::Fill {
                    id: "folder".to_string(),
                    role: "folder-icon".to_string(),
                    rect: folder,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: VelloColor::FOLDER_ICON,
                },
                VelloOp::Fill {
                    id: "file".to_string(),
                    role: "file-icon".to_string(),
                    rect: file,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: VelloColor::FILE_ICON,
                },
            ],
            80,
            40,
        );
        let wgpu = render_wgpu_pixels(
            &[
                WgpuPassOp::Quad {
                    id: "folder".to_string(),
                    role: "folder-icon".to_string(),
                    pipeline: vugra_render_wgpu::Pipeline::Solid,
                    rect: folder,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: WgpuColor::FOLDER_ICON,
                },
                WgpuPassOp::Quad {
                    id: "file".to_string(),
                    role: "file-icon".to_string(),
                    pipeline: vugra_render_wgpu::Pipeline::Solid,
                    rect: file,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: WgpuColor::FILE_ICON,
                },
            ],
            80,
            40,
        );

        assert_eq!(software, vello);
        assert_eq!(software, wgpu);
        assert_eq!(software[16 * 80 + 14], 0x006fb2ff);
        assert_eq!(software[10 * 80 + 36], 0x00ffffff);
        assert_eq!(software[9 * 80 + 35], 0x009ca3af);
        assert_eq!(software[12 * 80 + 49], 0x00f7f7f8);
    }

    #[test]
    fn native_sidebar_icons_render_as_role_shapes() {
        let commands = [
            RenderCommand::Element {
                id: "download".to_string(),
                role: "download-icon".to_string(),
                rect: vugra_layout::Rect {
                    x: 8.0,
                    y: 8.0,
                    width: 18.0,
                    height: 18.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "picture".to_string(),
                role: "picture-icon".to_string(),
                rect: vugra_layout::Rect {
                    x: 32.0,
                    y: 8.0,
                    width: 18.0,
                    height: 18.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "project".to_string(),
                role: "project-icon".to_string(),
                rect: vugra_layout::Rect {
                    x: 56.0,
                    y: 8.0,
                    width: 18.0,
                    height: 18.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "chevron".to_string(),
                role: "chevron-down-icon".to_string(),
                rect: vugra_layout::Rect {
                    x: 80.0,
                    y: 8.0,
                    width: 16.0,
                    height: 16.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
        ];
        let software = render_command_pixels(&commands, 110, 40);
        let vello = render_vello_pixels(
            &commands
                .iter()
                .filter_map(|command| match command {
                    RenderCommand::Element {
                        id,
                        role,
                        rect,
                        selected,
                        visual_state,
                        method,
                    } => vello_fill_for_role(role, *selected, *visual_state).map(|color| {
                        VelloOp::Fill {
                            id: id.clone(),
                            role: role.clone(),
                            rect: *rect,
                            selected: *selected,
                            visual_state: *visual_state,
                            method: *method,
                            color,
                        }
                    }),
                    _ => None,
                })
                .collect::<Vec<_>>(),
            110,
            40,
        );
        let wgpu = render_wgpu_pixels(
            &commands
                .iter()
                .filter_map(|command| match command {
                    RenderCommand::Element {
                        id,
                        role,
                        rect,
                        selected,
                        visual_state,
                        method,
                    } => wgpu_fill_for_role(role, *selected, *visual_state).map(|color| {
                        WgpuPassOp::Quad {
                            id: id.clone(),
                            role: role.clone(),
                            pipeline: vugra_render_wgpu::Pipeline::Solid,
                            rect: *rect,
                            selected: *selected,
                            visual_state: *visual_state,
                            method: *method,
                            color,
                        }
                    }),
                    _ => None,
                })
                .collect::<Vec<_>>(),
            110,
            40,
        );

        assert_eq!(software, vello);
        assert_eq!(software, wgpu);
        assert_eq!(software[11 * 110 + 16], 0x002563eb);
        assert_eq!(software[24 * 110 + 12], 0x0064748b);
        assert_eq!(software[21 * 110 + 24], 0x00f7f7f8);
        assert_eq!(software[21 * 110 + 43], 0x0022c55e);
        assert_eq!(software[15 * 110 + 64], 0x002563eb);
        assert_eq!(software[14 * 110 + 88], 0x006b7280);
    }

    #[test]
    fn native_surfaces_match_go_finder_css_role_colors() {
        let commands = [
            RenderCommand::Element {
                id: "toolbar".to_string(),
                role: "toolbar".to_string(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 30.0,
                    height: 20.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "sidebar".to_string(),
                role: "sidebar".to_string(),
                rect: Rect {
                    x: 30.0,
                    y: 0.0,
                    width: 30.0,
                    height: 20.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "statusbar".to_string(),
                role: "statusbar".to_string(),
                rect: Rect {
                    x: 60.0,
                    y: 0.0,
                    width: 30.0,
                    height: 20.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "sidebar-item".to_string(),
                role: "sidebar-item".to_string(),
                rect: Rect {
                    x: 90.0,
                    y: 0.0,
                    width: 30.0,
                    height: 20.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
        ];
        let software = render_command_pixels(&commands, 120, 24);
        assert_eq!(software[10 * 120 + 15], 0x00f2f2f4);
        assert_eq!(software[10 * 120 + 45], 0x00ececf0);
        assert_eq!(software[10 * 120 + 75], 0x00f5f5f7);
        assert_eq!(software[10 * 120 + 105], 0x00ececf0);

        let vello = render_vello_pixels(
            &commands
                .iter()
                .filter_map(|command| match command {
                    RenderCommand::Element {
                        id,
                        role,
                        rect,
                        selected,
                        visual_state,
                        method,
                    } => vello_fill_for_role(role, *selected, *visual_state).map(|color| {
                        VelloOp::Fill {
                            id: id.clone(),
                            role: role.clone(),
                            rect: *rect,
                            selected: *selected,
                            visual_state: *visual_state,
                            method: *method,
                            color,
                        }
                    }),
                    _ => None,
                })
                .collect::<Vec<_>>(),
            120,
            24,
        );
        let wgpu = render_wgpu_pixels(
            &commands
                .iter()
                .filter_map(|command| match command {
                    RenderCommand::Element {
                        id,
                        role,
                        rect,
                        selected,
                        visual_state,
                        method,
                    } => wgpu_fill_for_role(role, *selected, *visual_state).map(|color| {
                        WgpuPassOp::Quad {
                            id: id.clone(),
                            role: role.clone(),
                            pipeline: vugra_render_wgpu::Pipeline::Solid,
                            rect: *rect,
                            selected: *selected,
                            visual_state: *visual_state,
                            method: *method,
                            color,
                        }
                    }),
                    _ => None,
                })
                .collect::<Vec<_>>(),
            120,
            24,
        );
        assert_eq!(software, vello);
        assert_eq!(software, wgpu);
    }

    #[test]
    fn command_pixels_preserve_fractional_rect_coverage_at_bitmap_edge() {
        let rect = vugra_layout::Rect {
            x: 1.25,
            y: 2.25,
            width: 0.5,
            height: 0.5,
        };
        let pixels = render_command_pixels(
            &[RenderCommand::Element {
                id: "field".to_string(),
                role: "file-pane".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            }],
            8,
            8,
        );
        assert_eq!(pixels[2 * 8 + 1], 0x00ffffff);
        assert_eq!(pixels[2 * 8 + 2], 0x00f7f7f8);
    }

    #[test]
    fn command_pixels_cover_fractional_rect_end_with_ceil() {
        let rect = vugra_layout::Rect {
            x: 1.25,
            y: 2.25,
            width: 2.5,
            height: 1.5,
        };
        let pixels = render_command_pixels(
            &[RenderCommand::Element {
                id: "field".to_string(),
                role: "file-pane".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            }],
            8,
            8,
        );
        for y in 2..4 {
            for x in 1..4 {
                assert_eq!(pixels[y * 8 + x], 0x00ffffff, "pixel {x},{y}");
            }
        }
        assert_eq!(pixels[4 * 8 + 1], 0x00f7f7f8);
        assert_eq!(pixels[2 * 8 + 4], 0x00f7f7f8);
    }

    #[test]
    fn native_surfaces_render_go_finder_borders_and_rounded_corners() {
        let commands = [
            RenderCommand::Element {
                id: "toolbar".to_string(),
                role: "toolbar".to_string(),
                rect: Rect {
                    x: 0.0,
                    y: 0.0,
                    width: 120.0,
                    height: 52.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "search".to_string(),
                role: "search".to_string(),
                rect: Rect {
                    x: 10.0,
                    y: 10.0,
                    width: 40.0,
                    height: 30.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "nav".to_string(),
                role: "nav-button".to_string(),
                rect: Rect {
                    x: 60.0,
                    y: 10.0,
                    width: 34.0,
                    height: 30.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
        ];
        let pixels = render_command_pixels(&commands, 120, 60);
        assert_eq!(pixels[51 * 120 + 60], 0x00d8d8dc);
        assert_eq!(pixels[10 * 120 + 14], 0x00c7c7cc);
        assert_eq!(pixels[16 * 120 + 10], 0x00c7c7cc);
        assert_eq!(pixels[16 * 120 + 16], 0x00ffffff);
        assert_eq!(pixels[20 * 120 + 20], 0x00ffffff);
        assert_eq!(pixels[10 * 120 + 64], 0x00c7c7cc);
        assert_eq!(pixels[16 * 120 + 66], 0x00ffffff);
    }

    #[test]
    fn native_surfaces_render_go_finder_overlay_and_dialog_layers() {
        let commands = [
            RenderCommand::Element {
                id: "overlay".to_string(),
                role: "overlay".to_string(),
                rect: Rect {
                    x: 10.0,
                    y: 10.0,
                    width: 60.0,
                    height: 40.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "dialog-layer".to_string(),
                role: "dialog-layer".to_string(),
                rect: Rect {
                    x: 80.0,
                    y: 10.0,
                    width: 60.0,
                    height: 40.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            RenderCommand::Element {
                id: "primary".to_string(),
                role: "primary-button".to_string(),
                rect: Rect {
                    x: 20.0,
                    y: 60.0,
                    width: 40.0,
                    height: 24.0,
                },
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
        ];
        let pixels = render_command_pixels(&commands, 160, 100);
        assert_eq!(pixels[10 * 160 + 30], 0x00c7c7cc);
        assert_eq!(pixels[20 * 160 + 30], 0x00f7f9fc);
        assert_eq!(pixels[10 * 160 + 100], 0x00d8d8dc);
        assert_eq!(pixels[20 * 160 + 100], 0x00eef2f7);
        assert_eq!(pixels[60 * 160 + 30], 0x000a84ff);
    }

    #[test]
    fn native_backend_surface_pixels_match_for_bordered_controls() {
        let rect = Rect {
            x: 10.0,
            y: 10.0,
            width: 40.0,
            height: 30.0,
        };
        let command = RenderCommand::Element {
            id: "search".to_string(),
            role: "search".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
            method: None,
        };
        let software = render_command_pixels(&[command], 80, 60);
        let vello = render_vello_pixels(
            &[VelloOp::Fill {
                id: "search".to_string(),
                role: "search".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: VelloColor::FIELD,
            }],
            80,
            60,
        );
        let wgpu = render_wgpu_pixels(
            &[WgpuPassOp::Quad {
                id: "search".to_string(),
                role: "search".to_string(),
                pipeline: vugra_render_wgpu::Pipeline::Solid,
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: WgpuColor::FIELD,
            }],
            80,
            60,
        );
        assert_eq!(software, vello);
        assert_eq!(software, wgpu);
    }

    #[test]
    fn native_backend_surface_pixels_match_for_overlay_layers() {
        let rect = Rect {
            x: 10.0,
            y: 10.0,
            width: 60.0,
            height: 40.0,
        };
        let command = RenderCommand::Element {
            id: "overlay".to_string(),
            role: "overlay".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
            method: None,
        };
        let software = render_command_pixels(&[command], 90, 70);
        let vello = render_vello_pixels(
            &[VelloOp::Fill {
                id: "overlay".to_string(),
                role: "overlay".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: VelloColor::OVERLAY,
            }],
            90,
            70,
        );
        let wgpu = render_wgpu_pixels(
            &[WgpuPassOp::Quad {
                id: "overlay".to_string(),
                role: "overlay".to_string(),
                pipeline: vugra_render_wgpu::Pipeline::Solid,
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: WgpuColor::OVERLAY,
            }],
            90,
            70,
        );
        assert_eq!(software, vello);
        assert_eq!(software, wgpu);
    }

    #[test]
    fn text_input_drain_keeps_printable_chars_only() {
        let input = Rc::new(RefCell::new(vec!['r' as u32, 8, 'o' as u32]));
        assert_eq!(drain_text_input(&input), "ro");
        assert!(input.borrow().is_empty());
    }

    #[test]
    fn native_backend_commands_preserve_interactive_methods() {
        for backend in [
            NativeRenderBackend::Software,
            NativeRenderBackend::Vello,
            NativeRenderBackend::Wgpu,
        ] {
            let app = App::new(finder_lite_contract(), test_state());
            let commands = render_commands_with_backend(
                &app,
                Constraints {
                    width: 800.0,
                    height: 600.0,
                },
                backend,
            );
            assert!(commands.iter().any(|command| matches!(
                command,
                RenderCommand::Element {
                    role,
                    method: Some(MethodId(2)),
                    ..
                } if role == "row"
            )));
            assert!(commands.iter().any(|command| matches!(
                command,
                RenderCommand::Element {
                    role,
                    method: Some(MethodId(6)),
                    ..
                } if role == "sidebar-item"
            )));
        }
    }

    #[test]
    fn native_window_config_defaults_to_vello_backend() {
        assert_eq!(
            NativeWindowConfig::default().backend,
            NativeRenderBackend::Vello
        );
    }

    #[test]
    fn native_backend_pixels_match_software_reference() {
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        let software = render_native_frame(
            &App::new(finder_lite_contract(), test_state()),
            constraints,
            NativeRenderBackend::Software,
            800,
            600,
        );
        for backend in [NativeRenderBackend::Vello, NativeRenderBackend::Wgpu] {
            let frame = render_native_frame(
                &App::new(finder_lite_contract(), test_state()),
                constraints,
                backend,
                800,
                600,
            );
            if frame.pixels != software.pixels {
                let index = frame
                    .pixels
                    .iter()
                    .zip(software.pixels.iter())
                    .position(|(left, right)| left != right)
                    .expect("pixel mismatch");
                panic!(
                    "{backend:?} pixels differ at {},{}: backend={:#08x} software={:#08x}",
                    index % 800,
                    index / 800,
                    frame.pixels[index],
                    software.pixels[index]
                );
            }
            assert_eq!(hit_test(&frame.hit_test, 260.0, 104.0), Some(MethodId(2)));
        }
    }

    #[test]
    fn native_default_text_provider_feeds_loaded_font_runs() {
        let state = NativeFrameState::new(NativeRenderBackend::Software, 800, 600);
        let commands = vec![RenderCommand::Text {
            id: "label:text".to_string(),
            text: "Finder 文".to_string(),
            role: "text".to_string(),
            rect: Rect {
                x: 10.0,
                y: 12.0,
                width: 120.0,
                height: 40.0,
            },
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }];
        let paint = lower_native_paint_commands(&commands, &state.text_provider);
        let [NativePaintCommand::Text { run, .. }] = paint.as_slice() else {
            panic!("expected one text paint command, got {paint:?}");
        };
        assert!(run.pixels.len() > 20, "{run:?}");
        if !state.text_provider.is_empty() {
            assert!(run
                .positioned_glyphs
                .iter()
                .all(|glyph| glyph.source == vugra_text::GlyphSource::LoadedFont));
            assert!(run
                .positioned_glyphs
                .iter()
                .all(|glyph| glyph.font_key.is_some()));
        }
    }

    #[test]
    fn native_text_runs_use_go_finder_role_font_sizes() {
        let state = NativeFrameState::new(NativeRenderBackend::Software, 800, 600);
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 160.0,
            height: 24.0,
        };
        let paint = lower_native_paint_commands(
            &[
                RenderCommand::Text {
                    id: "preview-title:text".to_string(),
                    text: "Roadmap.md".to_string(),
                    role: "dialog-title".to_string(),
                    rect,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                },
                RenderCommand::Text {
                    id: "row1-name:text".to_string(),
                    text: "Roadmap.md".to_string(),
                    role: "row-name-cell".to_string(),
                    rect,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                },
            ],
            &state.text_provider,
        );
        let [NativePaintCommand::Text { run: title, .. }, NativePaintCommand::Text { run: row, .. }] =
            paint.as_slice()
        else {
            panic!("expected two text paint commands, got {paint:?}");
        };
        assert_eq!(title.font_size, 15.0);
        assert_eq!(row.font_size, 13.0);
        assert_eq!(title.font_weight, 600);
        assert_eq!(row.font_weight, 400);
    }

    #[test]
    fn native_row_focus_and_editing_text_use_go_finder_accent_color() {
        assert_eq!(
            native_text_color(
                "row-name-cell",
                "Roadmap.md",
                false,
                vugra_layout::RowVisualState::Focus
            ),
            0x000f3d74
        );
        assert_eq!(
            native_text_color(
                "row-name-cell",
                "Design",
                false,
                vugra_layout::RowVisualState::Editing
            ),
            0x000f3d74
        );
        assert_eq!(
            native_text_color(
                "row-name-cell",
                "Selected",
                true,
                vugra_layout::RowVisualState::Selected
            ),
            0x00ffffff
        );
    }

    #[test]
    fn native_selected_sidebar_label_uses_go_finder_active_text_color() {
        assert_eq!(
            native_text_color(
                "sidebar-item-label",
                "Documents",
                true,
                vugra_layout::RowVisualState::Normal
            ),
            0x000f3d74
        );
    }

    #[test]
    fn native_file_list_uses_go_finder_white_surface_fill() {
        assert_eq!(role_fill("file-list"), Some(0x00ffffff));
        assert_eq!(
            vello_fill_for_role("file-list", false, vugra_layout::RowVisualState::Normal),
            Some(VelloColor::FILE_PANE)
        );
        assert_eq!(
            wgpu_fill_for_role("file-list", false, vugra_layout::RowVisualState::Normal),
            Some(WgpuColor::FILE_PANE)
        );
    }

    #[test]
    fn native_search_text_uses_go_renderer_default_input_color() {
        assert_eq!(
            native_text_color(
                "search",
                "road",
                false,
                vugra_layout::RowVisualState::Normal
            ),
            0x000f172a
        );
    }

    #[test]
    fn vello_native_pixels_consume_text_run_geometry() {
        let rect = vugra_layout::Rect {
            x: 10.5,
            y: 12.25,
            width: 90.0,
            height: 20.0,
        };
        let run = vugra_text::layout_text_run("FinderLite", rect.x, rect.y, 13.0);
        let pixels = render_vello_pixels(
            &[VelloOp::Text {
                id: "title:text".to_string(),
                text: "FinderLite".to_string(),
                role: "heading".to_string(),
                rect,
                run: run.clone(),
                color: VelloColor::TITLE,
            }],
            160,
            80,
        );
        let first = run.pixels[0];
        let px = first.x.floor() as usize;
        let py = first.y.floor() as usize;
        assert_eq!(pixels[py * 160 + px], 0x001f2328);
    }

    #[test]
    fn native_row_state_colors_match_go_finder_css() {
        assert_eq!(
            row_color_u32(false, vugra_layout::RowVisualState::Normal),
            0x00ffffff
        );
        assert_eq!(
            row_color_u32(false, vugra_layout::RowVisualState::Hover),
            0x00f4f7fb
        );
        assert_eq!(
            row_color_u32(false, vugra_layout::RowVisualState::Focus),
            0x00edf6ff
        );
        assert_eq!(
            row_color_u32(false, vugra_layout::RowVisualState::Editing),
            0x00eef6ff
        );
        assert_eq!(
            row_color_u32(true, vugra_layout::RowVisualState::Normal),
            0x000a84ff
        );
    }

    #[test]
    fn native_window_key_mapping_includes_go_finder_list_shortcuts() {
        assert_eq!(
            native_key_for_window_key(
                Key::A,
                Modifiers {
                    meta: true,
                    ..Default::default()
                }
            ),
            Some(NativeKey::SelectAll)
        );
        assert_eq!(
            native_key_for_window_key(
                Key::A,
                Modifiers {
                    ctrl: true,
                    ..Default::default()
                }
            ),
            Some(NativeKey::SelectAll)
        );
        assert_eq!(
            native_key_for_window_key(Key::A, Modifiers::default()),
            None
        );
        assert_eq!(
            native_key_for_window_key(Key::Delete, Modifiers::default()),
            Some(NativeKey::Delete)
        );
    }

    #[test]
    fn native_frame_keeps_scene_hit_test_tree() {
        let frame = render_native_frame(
            &App::new(finder_lite_contract(), test_state()),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        assert_eq!(hit_test(&frame.hit_test, 260.0, 104.0), Some(MethodId(2)));
        assert_eq!(hit_test(&frame.hit_test, 24.0, 76.0), Some(MethodId(6)));
        assert_eq!(hit_test(&frame.hit_test, 700.0, 500.0), Some(MethodId(77)));
    }

    #[test]
    fn native_frame_state_reports_retained_dirty_regions() {
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        for backend in [
            NativeRenderBackend::Software,
            NativeRenderBackend::Vello,
            NativeRenderBackend::Wgpu,
        ] {
            let mut app = App::new(
                finder_lite_contract(),
                StatefulFinder {
                    selected: Some(0),
                    ..StatefulFinder::default()
                },
            );
            let mut state = NativeFrameState::new(backend, 800, 600);

            let first = state.render(&app, constraints);
            assert!(
                !first.dirty.is_empty(),
                "{backend:?} first dirty should cover initial scene"
            );

            let second = state.render(&app, constraints);
            assert!(
                second.dirty.is_empty(),
                "{backend:?} unchanged dirty = {:?}",
                second.dirty
            );

            assert!(dispatch_native_pointer(&mut app, &second, 260.0, 132.0));
            let changed = state.render(&app, constraints);
            assert!(
                !changed.dirty.is_empty(),
                "{backend:?} state change should produce retained dirty regions"
            );
            assert!(
                changed
                    .dirty
                    .iter()
                    .any(|rect| rect.x == 252.0 && rect.y == 92.0),
                "{backend:?} expected previous selected row dirty rect, got {:?}",
                changed.dirty
            );
            assert!(
                changed
                    .dirty
                    .iter()
                    .any(|rect| rect.x == 252.0 && rect.y == 122.0),
                "{backend:?} expected new selected row dirty rect, got {:?}",
                changed.dirty
            );
            let full = render_native_frame(&app, constraints, backend, 800, 600);
            if changed.pixels != full.pixels {
                let index = changed
                    .pixels
                    .iter()
                    .zip(full.pixels.iter())
                    .position(|(left, right)| left != right)
                    .expect("pixel mismatch");
                panic!(
                    "{backend:?} partial render differs from full render at {},{}: partial={:#08x} full={:#08x}",
                    index % 800,
                    index / 800,
                    changed.pixels[index],
                    full.pixels[index]
                );
            }
        }
    }

    #[test]
    fn native_frame_state_reuses_pixels_when_scene_is_unchanged() {
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        for backend in [
            NativeRenderBackend::Software,
            NativeRenderBackend::Vello,
            NativeRenderBackend::Wgpu,
        ] {
            let app = App::new(finder_lite_contract(), StatefulFinder::default());
            let mut state = NativeFrameState::new(backend, 800, 600);

            let first = state.render(&app, constraints);
            let second = state.render(&app, constraints);

            assert!(
                second.dirty.is_empty(),
                "{backend:?} dirty = {:?}",
                second.dirty
            );
            assert_eq!(second.pixels, first.pixels, "{backend:?}");
        }
    }

    #[test]
    fn native_dispatch_helpers_drive_component_state() {
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        let mut app = App::new(finder_lite_contract(), StatefulFinder::default());
        let frame = render_native_frame(&app, constraints, NativeRenderBackend::Software, 800, 600);

        assert!(dispatch_native_pointer(&mut app, &frame, 260.0, 104.0));
        assert_eq!(app.render_frame().selected_summary, "selected row 1");

        assert!(dispatch_native_context_menu(&mut app, &frame, 260.0, 104.0));
        let context_frame = app.render_frame();
        assert!(context_frame.overlays.item_menu_open);
        assert!(!context_frame.overlays.blank_menu_open);
        assert!(context_frame.rows[0].selected);

        assert!(dispatch_native_context_menu(&mut app, &frame, 700.0, 500.0));
        let blank_context_frame = app.render_frame();
        assert!(!blank_context_frame.overlays.item_menu_open);
        assert!(blank_context_frame.overlays.blank_menu_open);

        assert!(dispatch_native_pointer(&mut app, &frame, 700.0, 500.0));
        let cleared_frame = app.render_frame();
        assert_eq!(cleared_frame.selected_summary, "selected row 0");
        assert!(!cleared_frame.overlays.item_menu_open);
        assert!(!cleared_frame.overlays.blank_menu_open);

        assert!(dispatch_native_key(&mut app, NativeKey::ArrowDown));
        assert_eq!(app.render_frame().selected_summary, "selected row 2");

        assert!(dispatch_native_key(&mut app, NativeKey::SelectAll));
        let all_frame = app.render_frame();
        assert_eq!(all_frame.selected_summary, "selected all");
        assert!(all_frame.rows.iter().all(|row| row.selected));

        assert!(dispatch_native_key(&mut app, NativeKey::Delete));
        let deleted_frame = app.render_frame();
        assert_eq!(deleted_frame.selected_summary, "selected row 0");
        assert!(deleted_frame.rows.iter().all(|row| !row.selected));

        assert!(dispatch_native_text(&mut app, "road".to_string()));
        assert_eq!(app.render_frame().search_query, "road");

        assert!(dispatch_native_key(&mut app, NativeKey::Backspace));
        assert_eq!(app.render_frame().search_query, "roa");

        assert!(dispatch_native_key(&mut app, NativeKey::Enter));
        assert_eq!(app.render_frame().selected_summary, "opened selected");

        app.dispatch(MethodId(3));
        app.dispatch(MethodId(33));
        assert_eq!(app.render_frame().overlays.rename_text, "Roadmap.md");

        assert!(dispatch_native_text(&mut app, " Final".to_string()));
        assert_eq!(app.render_frame().overlays.rename_text, "Roadmap.md Final");

        assert!(dispatch_native_key(&mut app, NativeKey::Backspace));
        assert_eq!(app.render_frame().overlays.rename_text, "Roadmap.md Fina");

        assert!(dispatch_native_key(&mut app, NativeKey::Escape));
        assert!(app.render_frame().overlays.rename_text.is_empty());

        app.dispatch(MethodId(33));
        app.state_mut()
            .set_signal(SignalId(102), "Roadmap Final.md".into());
        assert!(dispatch_native_key(&mut app, NativeKey::Enter));
        assert_eq!(app.render_frame().rows[1].name, "Roadmap Final.md");
    }

    #[test]
    fn native_context_menu_dispatch_uses_dedicated_context_handler() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 24.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        target: Some(MethodId(2)),
                        context_menu: Some(MethodId(41)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                }],
            },
        };

        assert!(dispatch_native_context_menu(&mut app, &frame, 12.0, 12.0));
        assert_eq!(app.state().calls, vec![MethodId(41)]);
        assert_eq!(app.state().events.len(), 1);
        assert_eq!(app.state().events[0].kind, "contextmenu");
        assert_eq!(app.state().events[0].x, 12.0);
        assert_eq!(app.state().events[0].y, 12.0);
    }

    #[test]
    fn native_pointer_dispatch_preserves_event_modifiers() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 24.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        target: Some(MethodId(2)),
                        double_click: Some(MethodId(65)),
                        context_menu: Some(MethodId(41)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                }],
            },
        };
        let modifiers = Modifiers {
            shift: true,
            ctrl: true,
            meta: true,
            alt: true,
        };

        assert!(dispatch_native_pointer_with_modifiers(
            &mut app, &frame, 12.0, 12.0, modifiers
        ));
        assert!(dispatch_native_context_menu_with_modifiers(
            &mut app, &frame, 12.0, 12.0, modifiers
        ));
        assert!(dispatch_native_double_click_with_modifiers(
            &mut app, &frame, 12.0, 12.0, modifiers
        ));

        assert_eq!(
            app.state()
                .events
                .iter()
                .map(|event| (event.kind.as_str(), event.modifiers))
                .collect::<Vec<_>>(),
            vec![
                ("click", modifiers),
                ("contextmenu", modifiers),
                ("dblclick", modifiers),
            ]
        );
    }

    #[test]
    fn native_hover_dispatch_uses_dedicated_hover_handler() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 24.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        target: Some(MethodId(2)),
                        hover: Some(MethodId(53)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                }],
            },
        };

        assert!(dispatch_native_hover(&mut app, &frame, 12.0, 12.0));
        assert_eq!(app.state().calls, vec![MethodId(53)]);
        assert_eq!(app.state().events.len(), 1);
        assert_eq!(app.state().events[0].kind, "hover");
    }

    #[test]
    fn native_drag_dispatch_uses_dedicated_drag_handler() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "splitter".to_string(),
                    role: "splitter".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 8.0,
                        height: 80.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        drag: Some(MethodId(82)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                }],
            },
        };

        assert!(dispatch_native_drag(
            &mut app, &frame, 12.0, 20.0, 8.0, -2.0
        ));
        assert_eq!(app.state().calls, vec![MethodId(82)]);
        assert_eq!(app.state().events.len(), 1);
        assert_eq!(app.state().events[0].kind, "drag");
        assert_eq!(app.state().events[0].delta_x, 8.0);
        assert_eq!(app.state().events[0].delta_y, -2.0);
    }

    #[test]
    fn native_double_click_dispatch_uses_dedicated_open_handler() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 80.0,
                        height: 24.0,
                    },
                    method: None,
                    handlers: EventHandlers {
                        target: Some(MethodId(2)),
                        double_click: Some(MethodId(65)),
                        context_menu: Some(MethodId(41)),
                        ..EventHandlers::default()
                    },
                    parent: None,
                    clips: Vec::new(),
                }],
            },
        };

        assert!(dispatch_native_double_click(&mut app, &frame, 12.0, 12.0));
        assert_eq!(app.state().calls, vec![MethodId(65)]);
        assert_eq!(app.state().events.len(), 1);
        assert_eq!(app.state().events[0].kind, "dblclick");
        assert_eq!(app.state().events[0].x, 12.0);
        assert_eq!(app.state().events[0].y, 12.0);
    }

    #[test]
    fn native_pointer_dispatch_invokes_capture_target_and_bubble_route() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![
                    vugra_scene::HitTestNode {
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
                            capture: Some(MethodId(1)),
                            bubble: Some(MethodId(3)),
                            ..EventHandlers::default()
                        },
                        parent: None,
                        clips: Vec::new(),
                    },
                    vugra_scene::HitTestNode {
                        id: "button".to_string(),
                        role: "button".to_string(),
                        rect: Rect {
                            x: 10.0,
                            y: 10.0,
                            width: 20.0,
                            height: 20.0,
                        },
                        method: None,
                        handlers: EventHandlers::target(MethodId(2)),
                        parent: Some(0),
                        clips: Vec::new(),
                    },
                ],
            },
        };
        assert!(dispatch_native_pointer(&mut app, &frame, 12.0, 12.0));
        assert_eq!(
            app.state().calls,
            vec![MethodId(1), MethodId(2), MethodId(3)]
        );
        assert_eq!(
            app.state()
                .events
                .iter()
                .map(|event| (event.kind.as_str(), event.x, event.y))
                .collect::<Vec<_>>(),
            vec![
                ("click", 12.0, 12.0),
                ("click", 12.0, 12.0),
                ("click", 12.0, 12.0)
            ]
        );
    }

    #[test]
    fn native_pointer_dispatch_ignores_targets_outside_clip_rects() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: Vec::new(),
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 12.0,
                        y: 44.0,
                        width: 80.0,
                        height: 30.0,
                    },
                    method: None,
                    handlers: EventHandlers::target(MethodId(2)),
                    parent: None,
                    clips: vec![Rect {
                        x: 10.0,
                        y: 10.0,
                        width: 100.0,
                        height: 40.0,
                    }],
                }],
            },
        };

        assert!(dispatch_native_pointer(&mut app, &frame, 20.0, 46.0));
        assert!(!dispatch_native_pointer(&mut app, &frame, 20.0, 60.0));
        assert_eq!(app.state().calls, vec![MethodId(2)]);
    }

    #[test]
    fn native_scroll_dispatch_targets_scroll_container_child() {
        let mut app = App::new(finder_lite_contract(), RecordingState::default());
        let frame = NativeFrame {
            commands: Vec::new(),
            pixels: Vec::new(),
            dirty: Vec::new(),
            scrolls: vec![ScrollNode {
                id: "file-list".to_string(),
                rect: Rect {
                    x: 246.0,
                    y: 86.0,
                    width: 554.0,
                    height: 486.0,
                },
                offset_y: 0.0,
                content_height: 600.0,
                clip_id: "file-list:clip".to_string(),
            }],
            hit_test: HitTestTree {
                nodes: vec![vugra_scene::HitTestNode {
                    id: "row".to_string(),
                    role: "row".to_string(),
                    rect: Rect {
                        x: 252.0,
                        y: 92.0,
                        width: 542.0,
                        height: 30.0,
                    },
                    method: None,
                    handlers: EventHandlers::target(MethodId(2)),
                    parent: None,
                    clips: vec![Rect {
                        x: 246.0,
                        y: 86.0,
                        width: 554.0,
                        height: 486.0,
                    }],
                }],
            },
        };

        assert!(dispatch_native_scroll(&mut app, &frame, 260.0, 104.0, -1.0));
        assert!(!dispatch_native_scroll(&mut app, &frame, 20.0, 104.0, -1.0));
        assert_eq!(app.state().calls, vec![MethodId(2)]);
        assert_eq!(app.state().events.len(), 1);
        assert_eq!(app.state().events[0].kind, "scroll");
        assert_eq!(app.state().events[0].text, "file-list");
        assert_eq!(app.state().events[0].delta_y, -1.0);
    }

    #[test]
    fn native_frame_state_applies_scroll_offsets_to_next_frame() {
        let constraints = Constraints {
            width: 800.0,
            height: 140.0,
        };
        let app = App::new(finder_lite_contract(), test_state());
        let mut state = NativeFrameState::new(NativeRenderBackend::Software, 800, 140);
        let first = state.render(&app, constraints);
        let file_pane = first
            .scroll_target(260.0, 104.0)
            .expect("file-list scroll target")
            .clone();

        assert!(state.apply_scroll_delta(&file_pane, -1.0));
        assert_eq!(state.scroll_offset("file-list"), 32.0);
        let second = state.render(&app, constraints);

        let row1_y = second.commands.iter().find_map(|command| {
            if let RenderCommand::Element { id, rect, .. } = command {
                (id == "row1").then_some(rect.y)
            } else {
                None
            }
        });
        assert_eq!(row1_y, Some(60.0));
        assert_eq!(second.scrolls[1].id, "file-list");
        assert_eq!(second.scrolls[1].offset_y, 32.0);
        assert!(
            second
                .dirty
                .iter()
                .any(|rect| rect.x == 252.0 && rect.y == 92.0),
            "dirty = {:?}",
            second.dirty
        );
    }

    fn test_state() -> HashMapState {
        let mut values = HashMap::new();
        values.insert(SignalId(1), "Documents".into());
        values.insert(SignalId(2), "3 items · Current path: Documents".into());
        values.insert(SignalId(3), "1 items selected".into());
        values.insert(SignalId(13), "Documents".into());
        values.insert(SignalId(14), "Downloads".into());
        values.insert(SignalId(15), "Pictures".into());
        values.insert(SignalId(16), true.into());
        values.insert(SignalId(17), false.into());
        values.insert(SignalId(18), false.into());
        values.insert(SignalId(19), "".into());
        values.insert(SignalId(20), "Design".into());
        values.insert(SignalId(21), "folder".into());
        values.insert(SignalId(22), "--".into());
        values.insert(SignalId(23), "--".into());
        values.insert(SignalId(24), "file-row-selected".into());
        values.insert(SignalId(25), true.into());
        values.insert(SignalId(26), "Roadmap.md".into());
        values.insert(SignalId(27), "file".into());
        values.insert(SignalId(28), "--".into());
        values.insert(SignalId(29), "12 KB".into());
        values.insert(SignalId(30), "file-row".into());
        values.insert(SignalId(31), false.into());
        values.insert(SignalId(32), "Budget 2026.xlsx".into());
        values.insert(SignalId(33), "file".into());
        values.insert(SignalId(34), "--".into());
        values.insert(SignalId(35), "842 KB".into());
        values.insert(SignalId(36), "file-row".into());
        values.insert(SignalId(37), false.into());
        values.insert(SignalId(92), "Favorites".into());
        values.insert(SignalId(93), "Workspace".into());
        values.insert(SignalId(94), true.into());
        values.insert(SignalId(95), true.into());
        values.insert(SignalId(96), "Current Project".into());
        values.insert(SignalId(97), "Parent Folder".into());
        values.insert(SignalId(98), false.into());
        values.insert(SignalId(99), false.into());
        HashMapState { values }
    }

    struct HashMapState {
        values: HashMap<SignalId, Value>,
    }

    impl ComponentState for HashMapState {
        fn get_signal(&self, id: SignalId) -> Value {
            self.values.get(&id).cloned().unwrap_or(Value::None)
        }

        fn set_signal(&mut self, id: SignalId, value: Value) {
            self.values.insert(id, value);
        }

        fn call_method(&mut self, _: MethodId) {}
    }

    #[derive(Default)]
    struct RecordingState {
        calls: Vec<MethodId>,
        events: Vec<Event>,
    }

    impl ComponentState for RecordingState {
        fn get_signal(&self, _: SignalId) -> Value {
            Value::None
        }

        fn set_signal(&mut self, _: SignalId, _: Value) {}

        fn call_method(&mut self, id: MethodId) {
            self.calls.push(id);
        }

        fn call_event_method(&mut self, id: MethodId, event: Event) {
            self.calls.push(id);
            self.events.push(event);
        }
    }

    #[derive(Default)]
    struct StatefulFinder {
        selected: Option<usize>,
        selected_all: bool,
        search: String,
        opened: bool,
        item_menu_open: bool,
        blank_menu_open: bool,
        rename_text: String,
        row2_name: String,
    }

    impl ComponentState for StatefulFinder {
        fn get_signal(&self, id: SignalId) -> Value {
            match id.0 {
                1 => "Documents".into(),
                2 => "3 items · Current path: Documents".into(),
                3 if self.opened => "opened selected".into(),
                3 if self.selected_all => "selected all".into(),
                3 => self
                    .selected
                    .map(|selected| format!("selected row {}", selected + 1))
                    .unwrap_or_else(|| "selected row 0".to_string())
                    .into(),
                13 => "Documents".into(),
                14 => "Downloads".into(),
                15 => "Pictures".into(),
                16 => true.into(),
                17 => false.into(),
                18 => false.into(),
                19 => self.search.clone().into(),
                100 => self.item_menu_open.into(),
                101 => self.blank_menu_open.into(),
                102 => self.rename_text.clone().into(),
                20 => "Design".into(),
                21 => "folder".into(),
                22 => "--".into(),
                23 => "--".into(),
                24 if self.selected_all || self.selected == Some(0) => "file-row-selected".into(),
                24 => "file-row".into(),
                25 => Value::Bool(self.selected_all || self.selected == Some(0)),
                26 => {
                    if self.row2_name.is_empty() {
                        "Roadmap.md".into()
                    } else {
                        self.row2_name.clone().into()
                    }
                }
                27 => "file".into(),
                28 => "--".into(),
                29 => "12 KB".into(),
                30 if self.selected_all || self.selected == Some(1) => "file-row-selected".into(),
                30 => "file-row".into(),
                31 => Value::Bool(self.selected_all || self.selected == Some(1)),
                32 => "Budget 2026.xlsx".into(),
                33 => "file".into(),
                34 => "--".into(),
                35 => "842 KB".into(),
                36 if self.selected_all || self.selected == Some(2) => "file-row-selected".into(),
                36 => "file-row".into(),
                37 => Value::Bool(self.selected_all || self.selected == Some(2)),
                _ => Value::None,
            }
        }

        fn set_signal(&mut self, id: SignalId, value: Value) {
            match id.0 {
                19 => self.search = value.as_text(),
                102 => self.rename_text = value.as_text(),
                _ => {}
            }
        }

        fn call_method(&mut self, id: MethodId) {
            match id.0 {
                2 => {
                    self.selected = Some(0);
                    self.selected_all = false;
                }
                3 => {
                    self.selected = Some(1);
                    self.selected_all = false;
                }
                4 => {
                    self.selected = Some(2);
                    self.selected_all = false;
                }
                8 => {
                    self.selected = Some(self.selected.unwrap_or(0).saturating_sub(1));
                    self.selected_all = false;
                }
                9 => {
                    self.selected = Some((self.selected.unwrap_or(0) + 1).min(2));
                    self.selected_all = false;
                }
                11 => {
                    self.search.pop();
                }
                13 => self.opened = true,
                33 => {
                    self.rename_text = if self.selected == Some(1) {
                        if self.row2_name.is_empty() {
                            "Roadmap.md".to_string()
                        } else {
                            self.row2_name.clone()
                        }
                    } else {
                        "Design".to_string()
                    };
                    self.item_menu_open = false;
                    self.blank_menu_open = false;
                }
                34 => self.rename_text.clear(),
                35 => {
                    if self.selected == Some(1) && !self.rename_text.trim().is_empty() {
                        self.row2_name = self.rename_text.trim().to_string();
                    }
                    self.rename_text.clear();
                }
                39 => {
                    self.item_menu_open = false;
                    self.blank_menu_open = true;
                }
                77 => {
                    self.selected = None;
                    self.selected_all = false;
                    self.item_menu_open = false;
                    self.blank_menu_open = false;
                    self.rename_text.clear();
                }
                36 => {
                    self.selected = None;
                    self.selected_all = false;
                    self.item_menu_open = false;
                    self.blank_menu_open = false;
                }
                80 => {
                    self.selected = Some(0);
                    self.selected_all = true;
                }
                41 => {
                    self.selected = Some(0);
                    self.selected_all = false;
                    self.item_menu_open = true;
                    self.blank_menu_open = false;
                }
                42 => {
                    self.selected = Some(1);
                    self.selected_all = false;
                    self.item_menu_open = true;
                    self.blank_menu_open = false;
                }
                43 => {
                    self.selected = Some(2);
                    self.selected_all = false;
                    self.item_menu_open = true;
                    self.blank_menu_open = false;
                }
                _ => {}
            }
        }

        fn call_event_method(&mut self, id: MethodId, event: Event) {
            if id.0 == 10 {
                self.search.push_str(&event.text);
            } else {
                self.call_method(id);
            }
        }
    }
}
