//! Vello renderer boundary.

#[cfg(feature = "vello-device")]
use anyhow::{Context, Result};
#[cfg(feature = "vello-device")]
use vello::wgpu;
use vugra_render::{RenderCommand, Renderer, TestRenderer};
use vugra_scene::Scene;
use vugra_text::{GlyphMetricsProvider, TextRun};

#[derive(Default)]
pub struct VelloRenderer {
    recorded: TestRenderer,
    ops: Vec<VelloOp>,
}

#[derive(Clone, Debug, PartialEq)]
pub enum VelloOp {
    Fill {
        id: String,
        role: String,
        rect: vugra_layout::Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
        method: Option<vugra_ir::MethodId>,
        color: Color,
    },
    Text {
        id: String,
        text: String,
        role: String,
        rect: vugra_layout::Rect,
        run: TextRun,
        color: Color,
    },
    End {
        id: String,
    },
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Color(pub u8, pub u8, pub u8, pub u8);

impl Color {
    pub const WINDOW: Self = Self(247, 247, 248, 255);
    pub const TOOLBAR: Self = Self(242, 242, 244, 255);
    pub const SIDEBAR: Self = Self(236, 236, 240, 255);
    pub const FILE_PANE: Self = Self(255, 255, 255, 255);
    pub const SPLITTER: Self = Self(209, 209, 214, 255);
    pub const SPLITTER_HOVER: Self = Self(139, 184, 247, 255);
    pub const STATUSBAR: Self = Self(245, 245, 247, 255);
    pub const OVERLAY: Self = Self(247, 249, 252, 255);
    pub const DIALOG_LAYER: Self = Self(238, 242, 247, 255);
    pub const FIELD: Self = Self(255, 255, 255, 255);
    pub const NAV_BUTTON: Self = Self(255, 255, 255, 255);
    pub const NAV_ICON: Self = Self(55, 65, 81, 255);
    pub const SIDEBAR_ITEM: Self = Self(236, 236, 240, 255);
    pub const SIDEBAR_ACTIVE: Self = Self(217, 232, 255, 255);
    pub const ROW: Self = Self(255, 255, 255, 255);
    pub const ROW_HOVER: Self = Self(244, 247, 251, 255);
    pub const ROW_FOCUS: Self = Self(237, 246, 255, 255);
    pub const ROW_EDITING: Self = Self(238, 246, 255, 255);
    pub const ROW_SELECTED: Self = Self(10, 132, 255, 255);
    pub const COLUMN_HEADER: Self = Self(251, 251, 252, 255);
    pub const FOLDER_ICON: Self = Self(111, 178, 255, 255);
    pub const FOLDER_ICON_TOP: Self = Self(79, 156, 249, 255);
    pub const DOWNLOAD_ICON: Self = Self(37, 99, 235, 255);
    pub const PICTURE_ICON: Self = Self(134, 239, 172, 255);
    pub const PROJECT_ICON: Self = Self(37, 99, 235, 255);
    pub const CHEVRON_ICON: Self = Self(107, 114, 128, 255);
    pub const FILE_ICON: Self = Self(255, 255, 255, 255);
    pub const FILE_ICON_ACCENT: Self = Self(229, 231, 235, 255);
    pub const TITLE: Self = Self(31, 35, 40, 255);
    pub const TEXT: Self = Self(31, 41, 55, 255);
    pub const INPUT_TEXT: Self = Self(15, 23, 42, 255);
    pub const TEXT_SECONDARY: Self = Self(55, 65, 81, 255);
    pub const TEXT_MUTED: Self = Self(75, 85, 99, 255);
    pub const TEXT_SUBTLE: Self = Self(107, 114, 128, 255);
    pub const TEXT_INVERTED: Self = Self(255, 255, 255, 255);
    pub const ROW_ACCENT_TEXT: Self = Self(15, 61, 116, 255);
    pub const PLACEHOLDER: Self = Self(156, 163, 175, 255);
}

impl VelloRenderer {
    pub fn commands(&self) -> &[RenderCommand] {
        self.recorded.commands()
    }

    pub fn ops(&self) -> &[VelloOp] {
        &self.ops
    }
}

impl Renderer for VelloRenderer {
    fn render(&mut self, scene: &Scene) {
        self.recorded.render(scene);
        self.ops = lower_commands(self.recorded.commands());
    }
}

pub fn lower_commands(commands: &[RenderCommand]) -> Vec<VelloOp> {
    let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
    lower_commands_with_text_provider(commands, Some(&provider))
}

pub fn lower_commands_with_text_provider(
    commands: &[RenderCommand],
    text_provider: Option<&impl GlyphMetricsProvider>,
) -> Vec<VelloOp> {
    commands
        .iter()
        .filter_map(|command| match command {
            RenderCommand::Element {
                id,
                role,
                rect,
                selected,
                visual_state,
                method,
            } => fill_for_role_with_visual_state(role, *selected, *visual_state).map(|color| {
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
            RenderCommand::Text {
                id,
                text,
                role,
                rect,
                selected,
                visual_state,
            } => Some(VelloOp::Text {
                id: id.clone(),
                text: text.clone(),
                role: role.clone(),
                rect: *rect,
                run: layout_text_for_rect(role, text, *rect, text_provider),
                color: text_color(role, text, *selected, *visual_state),
            }),
            RenderCommand::End { id } => Some(VelloOp::End { id: id.clone() }),
        })
        .collect()
}

fn layout_text_for_rect(
    role: &str,
    text: &str,
    rect: vugra_layout::Rect,
    text_provider: Option<&impl GlyphMetricsProvider>,
) -> TextRun {
    let font_size = font_size_for_role(role);
    let font_weight = font_weight_for_role(role);
    if let Some(provider) = text_provider {
        let run = vugra_text::layout_text_run_with_provider_weighted(
            text,
            rect.x,
            rect.y,
            font_size,
            font_weight,
            Some(rect.width),
            provider,
        );
        align_text_run_for_role_with_provider(role, rect, run, provider)
    } else {
        let run = vugra_text::layout_text_run_wrapped_weighted(
            text,
            rect.x,
            rect.y,
            font_size,
            font_weight,
            Some(rect.width),
        );
        align_text_run_for_role(role, rect, run)
    }
}

fn align_text_run_for_role(role: &str, rect: vugra_layout::Rect, run: TextRun) -> TextRun {
    if role != "row-size-cell" {
        return run;
    }
    let aligned_x = (rect.x + rect.width - run.advance()).max(rect.x);
    vugra_text::layout_text_run_wrapped_weighted(
        &run.text,
        aligned_x,
        rect.y,
        run.font_size,
        run.font_weight,
        Some(rect.width),
    )
}

fn align_text_run_for_role_with_provider(
    role: &str,
    rect: vugra_layout::Rect,
    run: TextRun,
    provider: &impl GlyphMetricsProvider,
) -> TextRun {
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
        provider,
    )
}

fn font_size_for_role(role: &str) -> f32 {
    match role {
        "dialog-title" => 15.0,
        _ => 13.0,
    }
}

fn font_weight_for_role(role: &str) -> u16 {
    match role {
        "column-header" | "sidebar-section-label" | "dialog-title" => 600,
        _ => 400,
    }
}

fn fill_for_role_with_visual_state(
    role: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> Option<Color> {
    match role {
        "window" => Some(Color::WINDOW),
        "toolbar" => Some(Color::TOOLBAR),
        "sidebar" | "sidebar-200" | "sidebar-280" | "sidebar-320" => Some(Color::SIDEBAR),
        "splitter" => Some(Color::SPLITTER),
        "splitter-hover" => Some(Color::SPLITTER_HOVER),
        "file-pane" | "file-list" => Some(Color::FILE_PANE),
        "file-header" => Some(Color::COLUMN_HEADER),
        "statusbar" => Some(Color::STATUSBAR),
        "overlay" => Some(Color::OVERLAY),
        "dialog-layer" => Some(Color::DIALOG_LAYER),
        "menu" | "dialog" | "menu-item" | "secondary-button" => Some(Color::FIELD),
        "primary-button" => Some(Color::ROW_SELECTED),
        "path" | "search" => Some(Color::FIELD),
        "nav-button" => Some(Color::NAV_BUTTON),
        "back-icon" | "forward-icon" => Some(Color::NAV_ICON),
        "sidebar-item" if selected => Some(Color::SIDEBAR_ACTIVE),
        "sidebar-item" => Some(Color::SIDEBAR_ITEM),
        "sidebar-section" => Some(Color::SIDEBAR),
        "row" if selected || visual_state == vugra_layout::RowVisualState::Selected => {
            Some(Color::ROW_SELECTED)
        }
        "row" if visual_state == vugra_layout::RowVisualState::Focus => Some(Color::ROW_FOCUS),
        "row" if visual_state == vugra_layout::RowVisualState::Hover => Some(Color::ROW_HOVER),
        "row" if visual_state == vugra_layout::RowVisualState::Editing => Some(Color::ROW_EDITING),
        "row" => Some(Color::ROW),
        "folder-icon" => Some(Color::FOLDER_ICON),
        "file-icon" => Some(Color::FILE_ICON),
        "download-icon" => Some(Color::DOWNLOAD_ICON),
        "picture-icon" => Some(Color::PICTURE_ICON),
        "project-icon" => Some(Color::PROJECT_ICON),
        "chevron-down-icon" | "chevron-right-icon" => Some(Color::CHEVRON_ICON),
        _ => None,
    }
}

fn text_color(
    role: &str,
    text: &str,
    selected: bool,
    visual_state: vugra_layout::RowVisualState,
) -> Color {
    if role == "sidebar-item-label" && selected {
        Color::ROW_ACCENT_TEXT
    } else if role == "row-date-cell" || role == "row-size-cell" {
        Color::TEXT_MUTED
    } else if selected {
        Color::TEXT_INVERTED
    } else if role == "row-name-cell"
        && matches!(
            visual_state,
            vugra_layout::RowVisualState::Focus | vugra_layout::RowVisualState::Editing
        )
    {
        Color::ROW_ACCENT_TEXT
    } else if role == "column-header" || role == "sidebar-section-label" {
        Color::TEXT_SUBTLE
    } else if role == "statusbar"
        || role == "status-text"
        || role == "path"
        || role == "row-date-cell"
        || role == "row-size-cell"
        || role == "preview-copy"
    {
        Color::TEXT_MUTED
    } else if role == "sidebar-item-label" {
        Color::TEXT_SECONDARY
    } else if role == "dialog-title" {
        Color::TITLE
    } else if text == "▣" {
        Color::ROW_SELECTED
    } else if text == "□" {
        Color::TEXT
    } else if text.starts_with("* ") {
        Color::TEXT_INVERTED
    } else if text.starts_with("- ") {
        Color::TEXT
    } else if text == "Search" {
        Color::PLACEHOLDER
    } else if text == "FinderLite" {
        Color::TITLE
    } else if role == "search" {
        Color::INPUT_TEXT
    } else {
        Color::TEXT
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct OffscreenImage {
    pub width: u32,
    pub height: u32,
    pub pixels: Vec<u8>,
    pub checksum: u64,
}

#[cfg(feature = "vello-device")]
pub fn render_ops_offscreen(ops: &[VelloOp], width: u32, height: u32) -> Result<OffscreenImage> {
    let width = width.max(1);
    let height = height.max(1);
    let mut scene = vello::Scene::new();
    for op in ops {
        match op {
            VelloOp::Fill {
                role, rect, color, ..
            } => {
                if !draw_icon_parts(&mut scene, role, *rect) {
                    fill_scene_rect(&mut scene, *rect, *color);
                }
            }
            VelloOp::Text {
                rect, run, color, ..
            } => {
                draw_text_run_clipped(&mut scene, run, *color, *rect);
            }
            VelloOp::End { .. } => {}
        }
    }
    pollster::block_on(render_scene(&scene, width, height))
}

#[cfg(not(feature = "vello-device"))]
pub fn render_ops_offscreen(
    _ops: &[VelloOp],
    _width: u32,
    _height: u32,
) -> std::result::Result<OffscreenImage, String> {
    Err("vello-device feature is not enabled".to_string())
}

#[cfg(feature = "vello-device")]
fn draw_icon_parts(scene: &mut vello::Scene, role: &str, rect: vugra_layout::Rect) -> bool {
    let Some(parts) = icon_parts(role, rect) else {
        return false;
    };
    for (part, color) in parts {
        fill_scene_rect(scene, part, color);
    }
    true
}

#[cfg(feature = "vello-device")]
fn fill_scene_rect(scene: &mut vello::Scene, rect: vugra_layout::Rect, color: Color) {
    if rect.width <= 0.0 || rect.height <= 0.0 {
        return;
    }
    let rect = vello::kurbo::Rect::new(
        rect.x as f64,
        rect.y as f64,
        (rect.x + rect.width) as f64,
        (rect.y + rect.height) as f64,
    );
    scene.fill(
        vello::peniko::Fill::NonZero,
        vello::kurbo::Affine::IDENTITY,
        vello_color(color),
        None,
        &rect,
    );
}

#[cfg_attr(not(feature = "vello-device"), allow(dead_code))]
fn icon_parts(role: &str, rect: vugra_layout::Rect) -> Option<Vec<(vugra_layout::Rect, Color)>> {
    match role {
        "folder-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 1.0,
                    y: rect.y + 3.0,
                    width: rect.width * 0.46,
                    height: 5.0,
                },
                Color(120, 183, 255, 255),
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 3.0,
                    y: rect.y + 5.0,
                    width: rect.width - 4.0,
                    height: 4.0,
                },
                Color::FOLDER_ICON_TOP,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 1.0,
                    y: rect.y + 7.0,
                    width: rect.width - 2.0,
                    height: rect.height - 6.0,
                },
                Color::FOLDER_ICON,
            ),
        ]),
        "file-icon" => {
            let mut parts = vec![
                (
                    vugra_layout::Rect {
                        x: rect.x + 3.0,
                        y: rect.y + 1.0,
                        width: rect.width - 6.0,
                        height: rect.height - 2.0,
                    },
                    Color::FILE_ICON,
                ),
                (
                    vugra_layout::Rect {
                        x: rect.x + rect.width - 7.0,
                        y: rect.y + 1.0,
                        width: 4.0,
                        height: 5.0,
                    },
                    Color::FILE_ICON_ACCENT,
                ),
            ];
            for offset in [8.0, 11.0, 14.0] {
                parts.push((
                    vugra_layout::Rect {
                        x: rect.x + 6.0,
                        y: rect.y + offset,
                        width: if offset == 14.0 { 5.0 } else { 7.0 },
                        height: 1.0,
                    },
                    Color(156, 163, 175, 255),
                ));
            }
            Some(parts)
        }
        "download-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 8.0,
                    y: rect.y + 3.0,
                    width: 2.0,
                    height: 9.0,
                },
                Color::DOWNLOAD_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 10.0,
                    width: 8.0,
                    height: 2.0,
                },
                Color::DOWNLOAD_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 6.0,
                    y: rect.y + 12.0,
                    width: 6.0,
                    height: 2.0,
                },
                Color::DOWNLOAD_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 3.0,
                    y: rect.y + 16.0,
                    width: 12.0,
                    height: 2.0,
                },
                Color(100, 116, 139, 255),
            ),
        ]),
        "picture-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 2.0,
                    y: rect.y + 3.0,
                    width: rect.width - 4.0,
                    height: rect.height - 5.0,
                },
                Color::FILE_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 7.0,
                    width: 3.0,
                    height: 3.0,
                },
                Color(245, 158, 11, 255),
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 4.0,
                    y: rect.y + 13.0,
                    width: 11.0,
                    height: 3.0,
                },
                Color::PICTURE_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 8.0,
                    y: rect.y + 11.0,
                    width: 5.0,
                    height: 3.0,
                },
                Color(34, 197, 94, 255),
            ),
        ]),
        "project-icon" => {
            let mut parts = vec![(
                vugra_layout::Rect {
                    x: rect.x + 3.0,
                    y: rect.y + 3.0,
                    width: rect.width - 6.0,
                    height: rect.height - 5.0,
                },
                Color(248, 250, 252, 255),
            )];
            for offset in [7.0, 10.0, 13.0] {
                parts.push((
                    vugra_layout::Rect {
                        x: rect.x + 6.0,
                        y: rect.y + offset,
                        width: if offset == 13.0 { 6.0 } else { 8.0 },
                        height: 1.0,
                    },
                    Color::PROJECT_ICON,
                ));
            }
            Some(parts)
        }
        "chevron-down-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 4.0,
                    y: rect.y + 6.0,
                    width: 8.0,
                    height: 2.0,
                },
                Color::CHEVRON_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 6.0,
                    y: rect.y + 8.0,
                    width: 4.0,
                    height: 2.0,
                },
                Color::CHEVRON_ICON,
            ),
        ]),
        "chevron-right-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 6.0,
                    y: rect.y + 4.0,
                    width: 2.0,
                    height: 8.0,
                },
                Color::CHEVRON_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 8.0,
                    y: rect.y + 6.0,
                    width: 2.0,
                    height: 4.0,
                },
                Color::CHEVRON_ICON,
            ),
        ]),
        "back-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 14.0,
                    width: 8.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 12.0,
                    width: 3.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 16.0,
                    width: 3.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
        ]),
        "forward-icon" => Some(vec![
            (
                vugra_layout::Rect {
                    x: rect.x + 5.0,
                    y: rect.y + 14.0,
                    width: 8.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 10.0,
                    y: rect.y + 12.0,
                    width: 3.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
            (
                vugra_layout::Rect {
                    x: rect.x + 10.0,
                    y: rect.y + 16.0,
                    width: 3.0,
                    height: 2.0,
                },
                Color::NAV_ICON,
            ),
        ]),
        _ => None,
    }
}

#[cfg(feature = "vello-device")]
fn draw_text_run(scene: &mut vello::Scene, run: &TextRun, color: Color) {
    draw_text_run_with_clip(scene, run, color, None);
}

#[cfg(feature = "vello-device")]
fn draw_text_run_clipped(
    scene: &mut vello::Scene,
    run: &TextRun,
    color: Color,
    clip: vugra_layout::Rect,
) {
    draw_text_run_with_clip(scene, run, color, Some(clip));
}

#[cfg(feature = "vello-device")]
fn draw_text_run_with_clip(
    scene: &mut vello::Scene,
    run: &TextRun,
    color: Color,
    clip: Option<vugra_layout::Rect>,
) {
    if run.pixels.is_empty() {
        return;
    }
    for pixel in &run.pixels {
        if pixel.width <= 0.0 || pixel.height <= 0.0 {
            continue;
        }
        let mut pixel_rect = vugra_layout::Rect {
            x: pixel.x,
            y: pixel.y,
            width: pixel.width,
            height: pixel.height,
        };
        if let Some(clip) = clip {
            let Some(clipped) = intersect_rect(pixel_rect, clip) else {
                continue;
            };
            pixel_rect = clipped;
        }
        let mut color = color;
        color.3 = ((color.3 as f32) * pixel.alpha.clamp(0.0, 1.0)).round() as u8;
        let brush = vello_color(color);
        let rect = vello::kurbo::Rect::new(
            pixel_rect.x as f64,
            pixel_rect.y as f64,
            (pixel_rect.x + pixel_rect.width) as f64,
            (pixel_rect.y + pixel_rect.height) as f64,
        );
        scene.fill(
            vello::peniko::Fill::NonZero,
            vello::kurbo::Affine::IDENTITY,
            brush,
            None,
            &rect,
        );
    }
}

#[cfg(feature = "vello-device")]
fn intersect_rect(a: vugra_layout::Rect, b: vugra_layout::Rect) -> Option<vugra_layout::Rect> {
    let left = a.x.max(b.x);
    let top = a.y.max(b.y);
    let right = (a.x + a.width).min(b.x + b.width);
    let bottom = (a.y + a.height).min(b.y + b.height);
    (right > left && bottom > top).then_some(vugra_layout::Rect {
        x: left,
        y: top,
        width: right - left,
        height: bottom - top,
    })
}

#[cfg(feature = "vello-device")]
async fn render_scene(scene: &vello::Scene, width: u32, height: u32) -> Result<OffscreenImage> {
    let instance = wgpu::Instance::new(wgpu::InstanceDescriptor::new_without_display_handle());
    let adapter = instance
        .request_adapter(&wgpu::RequestAdapterOptions::default())
        .await
        .context("request wgpu adapter")?;
    let (device, queue) = adapter
        .request_device(&wgpu::DeviceDescriptor {
            label: Some("vugra-render-vello-device"),
            required_features: wgpu::Features::empty(),
            required_limits: adapter.limits(),
            experimental_features: wgpu::ExperimentalFeatures::disabled(),
            memory_hints: wgpu::MemoryHints::Performance,
            trace: wgpu::Trace::Off,
        })
        .await
        .context("request wgpu device")?;
    let texture = device.create_texture(&wgpu::TextureDescriptor {
        label: Some("vugra-render-vello-target"),
        size: wgpu::Extent3d {
            width,
            height,
            depth_or_array_layers: 1,
        },
        mip_level_count: 1,
        sample_count: 1,
        dimension: wgpu::TextureDimension::D2,
        format: wgpu::TextureFormat::Rgba8Unorm,
        usage: wgpu::TextureUsages::STORAGE_BINDING | wgpu::TextureUsages::COPY_SRC,
        view_formats: &[],
    });
    let view = texture.create_view(&wgpu::TextureViewDescriptor::default());
    let mut renderer = vello::Renderer::new(
        &device,
        vello::RendererOptions {
            use_cpu: false,
            antialiasing_support: [vello::AaConfig::Area].into_iter().collect(),
            num_init_threads: std::num::NonZeroUsize::new(1),
            pipeline_cache: None,
        },
    )
    .context("create Vello renderer")?;
    renderer
        .render_to_texture(
            &device,
            &queue,
            scene,
            &view,
            &vello::RenderParams {
                base_color: vello::peniko::Color::from_rgb8(247, 247, 248),
                width,
                height,
                antialiasing_method: vello::AaConfig::Area,
            },
        )
        .context("render Vello scene to texture")?;

    let bytes_per_pixel = 4u32;
    let unpadded_bytes_per_row = width * bytes_per_pixel;
    let align = wgpu::COPY_BYTES_PER_ROW_ALIGNMENT;
    let padded_bytes_per_row = unpadded_bytes_per_row.div_ceil(align) * align;
    let output_buffer_size = padded_bytes_per_row as u64 * height as u64;
    let output_buffer = device.create_buffer(&wgpu::BufferDescriptor {
        label: Some("vugra-render-vello-readback"),
        size: output_buffer_size,
        usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
        mapped_at_creation: false,
    });
    let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor {
        label: Some("vugra-render-vello-copy"),
    });
    encoder.copy_texture_to_buffer(
        texture.as_image_copy(),
        wgpu::TexelCopyBufferInfo {
            buffer: &output_buffer,
            layout: wgpu::TexelCopyBufferLayout {
                offset: 0,
                bytes_per_row: Some(padded_bytes_per_row),
                rows_per_image: Some(height),
            },
        },
        wgpu::Extent3d {
            width,
            height,
            depth_or_array_layers: 1,
        },
    );
    queue.submit(Some(encoder.finish()));

    let buffer_slice = output_buffer.slice(..);
    let (sender, receiver) = futures_intrusive::channel::shared::oneshot_channel();
    buffer_slice.map_async(wgpu::MapMode::Read, move |result| {
        sender.send(result).ok();
    });
    device
        .poll(wgpu::PollType::wait_indefinitely())
        .context("wait for readback")?;
    receiver
        .receive()
        .await
        .context("readback callback dropped")?
        .context("map Vello readback buffer")?;

    let data = buffer_slice.get_mapped_range();
    let mut checksum = 0u64;
    let mut pixels = Vec::with_capacity((width * height * 4) as usize);
    for row in 0..height as usize {
        let start = row * padded_bytes_per_row as usize;
        let end = start + unpadded_bytes_per_row as usize;
        let row_data = &data[start..end];
        pixels.extend_from_slice(row_data);
        for byte in row_data {
            checksum = checksum.wrapping_mul(16777619) ^ u64::from(*byte);
        }
    }
    drop(data);
    output_buffer.unmap();
    Ok(OffscreenImage {
        width,
        height,
        pixels,
        checksum,
    })
}

#[cfg(feature = "vello-device")]
fn vello_color(color: Color) -> vello::peniko::Color {
    vello::peniko::Color::from_rgba8(color.0, color.1, color.2, color.3)
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_layout::Rect;
    use vugra_scene::SceneCommand;

    #[test]
    fn vello_boundary_accepts_renderer_neutral_scene() {
        let scene = Scene::from_commands(vec![]);
        let mut renderer = VelloRenderer::default();
        renderer.render(&scene);
        assert!(renderer.commands().is_empty());
        assert!(renderer.ops().is_empty());
    }

    #[test]
    fn vello_lowering_projects_fills_and_text_ops() {
        let rect = Rect {
            x: 1.0,
            y: 2.0,
            width: 3.0,
            height: 4.0,
        };
        let scene = Scene::from_commands(vec![
            SceneCommand::Begin {
                id: "row1-icon".to_string(),
                role: "folder-icon".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            SceneCommand::End {
                id: "row1-icon".to_string(),
            },
        ]);
        let mut renderer = VelloRenderer::default();
        renderer.render(&scene);
        assert_eq!(
            renderer.ops(),
            &[
                VelloOp::Fill {
                    id: "row1-icon".to_string(),
                    role: "folder-icon".to_string(),
                    rect,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: Color::FOLDER_ICON,
                },
                VelloOp::End {
                    id: "row1-icon".to_string()
                },
            ]
        );
    }

    #[test]
    fn vello_file_list_lowers_to_go_finder_white_surface() {
        let rect = Rect {
            x: 10.0,
            y: 20.0,
            width: 120.0,
            height: 80.0,
        };
        let ops = lower_commands(&[RenderCommand::Element {
            id: "file-list".to_string(),
            role: "file-list".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
            method: None,
        }]);
        assert_eq!(
            ops.first(),
            Some(&VelloOp::Fill {
                id: "file-list".to_string(),
                role: "file-list".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: Color::FILE_PANE,
            })
        );
    }

    #[test]
    fn vello_device_icon_parts_expand_role_shapes() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 18.0,
            height: 18.0,
        };
        let parts = icon_parts("folder-icon", rect).expect("folder icon parts");
        assert_eq!(parts.len(), 3);
        assert_eq!(
            parts[0],
            (
                Rect {
                    x: 11.0,
                    y: 15.0,
                    width: 8.28,
                    height: 5.0
                },
                Color(120, 183, 255, 255)
            )
        );
        assert_eq!(
            parts[1],
            (
                Rect {
                    x: 13.0,
                    y: 17.0,
                    width: 14.0,
                    height: 4.0
                },
                Color::FOLDER_ICON_TOP
            )
        );
        assert_eq!(
            parts[2],
            (
                Rect {
                    x: 11.0,
                    y: 19.0,
                    width: 16.0,
                    height: 12.0
                },
                Color::FOLDER_ICON
            )
        );
        assert_eq!(icon_parts("row", rect), None);
    }

    #[test]
    fn vello_text_ops_carry_unified_text_run() {
        let rect = Rect {
            x: 10.5,
            y: 12.25,
            width: 90.0,
            height: 20.0,
        };
        let ops = lower_commands(&[RenderCommand::Text {
            id: "title:text".to_string(),
            text: "FinderLite".to_string(),
            role: "heading".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        let [VelloOp::Text {
            text,
            rect: op_rect,
            run,
            ..
        }] = ops.as_slice()
        else {
            panic!("expected one text op, got {ops:?}");
        };
        assert_eq!(text, "FinderLite");
        assert_eq!(*op_rect, rect);
        assert_eq!(run.text, "FinderLite");
        assert_eq!(run.x, rect.x);
        assert_eq!(run.y, rect.y);
        assert_eq!(run.metrics.width, run.advance());
        assert!(run.pixels.len() > 20);
        assert!(run.pixels.iter().all(|pixel| pixel.x >= rect.x));
        assert!(run.pixels.iter().all(|pixel| pixel.y >= rect.y));
    }

    #[test]
    fn vello_text_ops_use_role_colors_from_go_finder_css() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let ops = lower_commands(&[
            RenderCommand::Text {
                id: "header-name:text".to_string(),
                text: "Name".to_string(),
                role: "column-header".to_string(),
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
        ]);
        assert!(matches!(
            &ops[0],
            VelloOp::Text {
                color: Color::TEXT_SUBTLE,
                ..
            }
        ));
        assert!(matches!(
            &ops[1],
            VelloOp::Text {
                color: Color::TEXT,
                ..
            }
        ));
    }

    #[test]
    fn vello_row_focus_and_editing_text_use_go_finder_accent_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let ops = lower_commands(&[
            RenderCommand::Text {
                id: "row1-name:text".to_string(),
                text: "Roadmap.md".to_string(),
                role: "row-name-cell".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Focus,
            },
            RenderCommand::Text {
                id: "row2-name:text".to_string(),
                text: "Design".to_string(),
                role: "row-name-cell".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Editing,
            },
            RenderCommand::Text {
                id: "row3-name:text".to_string(),
                text: "Selected".to_string(),
                role: "row-name-cell".to_string(),
                rect,
                selected: true,
                visual_state: vugra_layout::RowVisualState::Selected,
            },
        ]);
        assert!(matches!(
            &ops[0],
            VelloOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
        assert!(matches!(
            &ops[1],
            VelloOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
        assert!(matches!(
            &ops[2],
            VelloOp::Text {
                color: Color::TEXT_INVERTED,
                ..
            }
        ));
    }

    #[test]
    fn vello_selected_sidebar_label_uses_go_finder_active_text_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let ops = lower_commands(&[RenderCommand::Text {
            id: "sidebar-documents-label:text".to_string(),
            text: "Documents".to_string(),
            role: "sidebar-item-label".to_string(),
            rect,
            selected: true,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        assert!(matches!(
            &ops[0],
            VelloOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
    }

    #[test]
    fn vello_search_text_uses_go_renderer_default_input_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let ops = lower_commands(&[RenderCommand::Text {
            id: "search:text".to_string(),
            text: "road".to_string(),
            role: "search".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        assert!(matches!(
            &ops[0],
            VelloOp::Text {
                color: Color::INPUT_TEXT,
                ..
            }
        ));
    }

    #[test]
    fn vello_text_ops_use_role_font_sizes_from_go_finder_css() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 160.0,
            height: 24.0,
        };
        let ops = lower_commands(&[
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
        ]);
        let [VelloOp::Text { run: title, .. }, VelloOp::Text { run: row, .. }] = ops.as_slice()
        else {
            panic!("expected two text ops, got {ops:?}");
        };
        assert_eq!(title.font_size, 15.0);
        assert_eq!(row.font_size, 13.0);
        assert_eq!(title.font_weight, 600);
        assert_eq!(row.font_weight, 400);
        assert!(title.metrics.height >= row.metrics.height);
    }

    #[test]
    fn vello_text_ops_preserve_cjk_fallback_glyph_runs() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 90.0,
            height: 20.0,
        };
        let ops = lower_commands_with_text_provider(
            &[RenderCommand::Text {
                id: "label:text".to_string(),
                text: "文a".to_string(),
                role: "text".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            None::<&vugra_text::LoadedFontTextMeasurer>,
        );
        let [VelloOp::Text { run, .. }] = ops.as_slice() else {
            panic!("expected one text op, got {ops:?}");
        };
        assert_eq!(
            run.glyphs,
            vec![
                vugra_text::GlyphRunGlyph {
                    ch: '文',
                    source: vugra_text::GlyphSource::FallbackBox,
                },
                vugra_text::GlyphRunGlyph {
                    ch: 'a',
                    source: vugra_text::GlyphSource::Font8x8,
                },
            ]
        );
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.x >= rect.x && pixel.x < rect.x + 8.0));
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.x >= rect.x + 9.0 && pixel.x < rect.x + 17.0));
        assert_eq!(
            run.positioned_glyphs[0].source,
            vugra_text::GlyphSource::FallbackBox
        );
        assert_eq!(run.positioned_glyphs[0].advance, 10.0);
        assert_eq!(
            run.positioned_glyphs[1].source,
            vugra_text::GlyphSource::Font8x8
        );
        assert_eq!(run.positioned_glyphs[1].x, rect.x + 10.0);
    }

    #[test]
    fn vello_default_text_ops_use_system_font_provider_when_available() {
        let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for vello default provider test");
            return;
        }
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let ops = lower_commands(&[RenderCommand::Text {
            id: "label:text".to_string(),
            text: "Finder 文".to_string(),
            role: "text".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        let [VelloOp::Text { run, .. }] = ops.as_slice() else {
            panic!("expected one text op, got {ops:?}");
        };
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.source == vugra_text::GlyphSource::LoadedFont));
        assert!(run.pixels.iter().any(|pixel| pixel.alpha < 1.0));
    }

    #[test]
    fn vello_text_ops_use_rect_width_for_unified_line_boxes() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 18.0,
            height: 40.0,
        };
        let ops = lower_commands_with_text_provider(
            &[RenderCommand::Text {
                id: "label:text".to_string(),
                text: "abcd".to_string(),
                role: "text".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            None::<&vugra_text::LoadedFontTextMeasurer>,
        );
        let [VelloOp::Text { run, .. }] = ops.as_slice() else {
            panic!("expected one text op, got {ops:?}");
        };
        assert_eq!(run.lines.len(), 2);
        assert_eq!(run.lines[0].text, "ab");
        assert_eq!(run.lines[1].text, "cd");
        assert_eq!(run.metrics.width, 18.0);
        assert_eq!(run.metrics.height, 26.0);
        assert_eq!(run.positioned_glyphs[2].line_index, 1);
        assert_eq!(run.positioned_glyphs[2].x, rect.x);
        assert_eq!(run.positioned_glyphs[2].y, rect.y + 13.0);
    }

    #[test]
    fn vello_text_ops_can_use_loaded_font_provider() {
        let Some(font) = vugra_text::SystemFontTextMeasurer::new().load_for_text("Finder 文")
        else {
            eprintln!("no system font available for vello loaded font text test");
            return;
        };
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let ops = lower_commands_with_text_provider(
            &[RenderCommand::Text {
                id: "label:text".to_string(),
                text: "Finder 文".to_string(),
                role: "text".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            Some(&font),
        );
        let [VelloOp::Text { run, .. }] = ops.as_slice() else {
            panic!("expected one text op, got {ops:?}");
        };
        assert!(run.pixels.len() > 20);
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.source == vugra_text::GlyphSource::LoadedFont));
        assert!(run.metrics.width > 40.0);
    }

    #[test]
    fn vello_text_ops_can_use_system_fallback_provider() {
        let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for vello fallback provider test");
            return;
        }
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let ops = lower_commands_with_text_provider(
            &[RenderCommand::Text {
                id: "label:text".to_string(),
                text: "Finder 文".to_string(),
                role: "text".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
            }],
            Some(&provider),
        );
        let [VelloOp::Text { run, .. }] = ops.as_slice() else {
            panic!("expected one text op, got {ops:?}");
        };
        assert!(run.pixels.len() > 20);
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.font_key.is_some()));
    }
}
