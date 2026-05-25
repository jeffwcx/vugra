//! wgpu/WebGPU renderer boundary.

#[cfg(feature = "wgpu-device")]
use anyhow::{Context, Result};
use vugra_render::{RenderCommand, Renderer, TestRenderer};
use vugra_scene::Scene;
use vugra_text::{GlyphMetricsProvider, TextRun};

#[derive(Default)]
pub struct WgpuRenderer {
    recorded: TestRenderer,
    pass: Vec<WgpuPassOp>,
}

#[derive(Clone, Debug, PartialEq)]
pub enum WgpuPassOp {
    Quad {
        id: String,
        role: String,
        pipeline: Pipeline,
        rect: vugra_layout::Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
        method: Option<vugra_ir::MethodId>,
        color: Color,
    },
    Text {
        id: String,
        rect: vugra_layout::Rect,
        text: String,
        role: String,
        run: TextRun,
        color: Color,
    },
    End {
        id: String,
    },
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Pipeline {
    Solid,
    Text,
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

impl WgpuRenderer {
    pub fn commands(&self) -> &[RenderCommand] {
        self.recorded.commands()
    }

    pub fn pass(&self) -> &[WgpuPassOp] {
        &self.pass
    }
}

impl Renderer for WgpuRenderer {
    fn render(&mut self, scene: &Scene) {
        self.recorded.render(scene);
        self.pass = lower_commands(self.recorded.commands());
    }
}

pub fn lower_commands(commands: &[RenderCommand]) -> Vec<WgpuPassOp> {
    let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
    lower_commands_with_text_provider(commands, Some(&provider))
}

pub fn lower_commands_with_text_provider(
    commands: &[RenderCommand],
    text_provider: Option<&impl GlyphMetricsProvider>,
) -> Vec<WgpuPassOp> {
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
                WgpuPassOp::Quad {
                    id: id.clone(),
                    role: role.clone(),
                    pipeline: Pipeline::Solid,
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
            } => Some(WgpuPassOp::Text {
                id: id.clone(),
                rect: *rect,
                text: text.clone(),
                role: role.clone(),
                run: layout_text_for_rect(role, text, *rect, text_provider),
                color: text_color(role, text, *selected, *visual_state),
            }),
            RenderCommand::End { id } => Some(WgpuPassOp::End { id: id.clone() }),
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

#[cfg(feature = "wgpu-device")]
#[repr(C)]
#[derive(Clone, Copy, Debug, bytemuck::Pod, bytemuck::Zeroable)]
struct QuadVertex {
    position: [f32; 2],
    color: [f32; 4],
}

#[cfg(feature = "wgpu-device")]
pub fn render_pass_offscreen(
    pass: &[WgpuPassOp],
    width: u32,
    height: u32,
) -> Result<OffscreenImage> {
    pollster::block_on(render_pass_offscreen_async(
        pass,
        width.max(1),
        height.max(1),
    ))
}

#[cfg(not(feature = "wgpu-device"))]
pub fn render_pass_offscreen(
    _pass: &[WgpuPassOp],
    _width: u32,
    _height: u32,
) -> std::result::Result<OffscreenImage, String> {
    Err("wgpu-device feature is not enabled".to_string())
}

#[cfg(feature = "wgpu-device")]
async fn render_pass_offscreen_async(
    pass: &[WgpuPassOp],
    width: u32,
    height: u32,
) -> Result<OffscreenImage> {
    let instance = wgpu::Instance::new(wgpu::InstanceDescriptor::default());
    let adapter = instance
        .request_adapter(&wgpu::RequestAdapterOptions::default())
        .await
        .context("request wgpu adapter")?;
    let (device, queue) = adapter
        .request_device(
            &wgpu::DeviceDescriptor {
                label: Some("vugra-render-wgpu-device"),
                required_features: wgpu::Features::empty(),
                required_limits: wgpu::Limits::downlevel_defaults(),
                memory_hints: wgpu::MemoryHints::Performance,
            },
            None,
        )
        .await
        .context("request wgpu device")?;

    let texture = device.create_texture(&wgpu::TextureDescriptor {
        label: Some("vugra-render-wgpu-target"),
        size: wgpu::Extent3d {
            width,
            height,
            depth_or_array_layers: 1,
        },
        mip_level_count: 1,
        sample_count: 1,
        dimension: wgpu::TextureDimension::D2,
        format: wgpu::TextureFormat::Rgba8Unorm,
        usage: wgpu::TextureUsages::RENDER_ATTACHMENT | wgpu::TextureUsages::COPY_SRC,
        view_formats: &[],
    });
    let view = texture.create_view(&wgpu::TextureViewDescriptor::default());
    let shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
        label: Some("vugra-render-wgpu-quad-shader"),
        source: wgpu::ShaderSource::Wgsl(QUAD_SHADER.into()),
    });
    let pipeline_layout = device.create_pipeline_layout(&wgpu::PipelineLayoutDescriptor {
        label: Some("vugra-render-wgpu-pipeline-layout"),
        bind_group_layouts: &[],
        push_constant_ranges: &[],
    });
    let pipeline = device.create_render_pipeline(&wgpu::RenderPipelineDescriptor {
        label: Some("vugra-render-wgpu-quad-pipeline"),
        layout: Some(&pipeline_layout),
        vertex: wgpu::VertexState {
            module: &shader,
            entry_point: "vs_main",
            compilation_options: wgpu::PipelineCompilationOptions::default(),
            buffers: &[wgpu::VertexBufferLayout {
                array_stride: std::mem::size_of::<QuadVertex>() as wgpu::BufferAddress,
                step_mode: wgpu::VertexStepMode::Vertex,
                attributes: &[
                    wgpu::VertexAttribute {
                        offset: 0,
                        shader_location: 0,
                        format: wgpu::VertexFormat::Float32x2,
                    },
                    wgpu::VertexAttribute {
                        offset: std::mem::size_of::<[f32; 2]>() as wgpu::BufferAddress,
                        shader_location: 1,
                        format: wgpu::VertexFormat::Float32x4,
                    },
                ],
            }],
        },
        primitive: wgpu::PrimitiveState::default(),
        depth_stencil: None,
        multisample: wgpu::MultisampleState::default(),
        fragment: Some(wgpu::FragmentState {
            module: &shader,
            entry_point: "fs_main",
            compilation_options: wgpu::PipelineCompilationOptions::default(),
            targets: &[Some(wgpu::ColorTargetState {
                format: wgpu::TextureFormat::Rgba8Unorm,
                blend: Some(wgpu::BlendState::ALPHA_BLENDING),
                write_mask: wgpu::ColorWrites::ALL,
            })],
        }),
        multiview: None,
        cache: None,
    });
    let vertices = quad_vertices(pass, width, height);
    let vertex_buffer = device.create_buffer(&wgpu::BufferDescriptor {
        label: Some("vugra-render-wgpu-vertices"),
        size: (vertices.len() * std::mem::size_of::<QuadVertex>()) as u64,
        usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
        mapped_at_creation: false,
    });
    if !vertices.is_empty() {
        queue.write_buffer(&vertex_buffer, 0, bytemuck::cast_slice(&vertices));
    }

    let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor {
        label: Some("vugra-render-wgpu-encoder"),
    });
    {
        let mut render_pass = encoder.begin_render_pass(&wgpu::RenderPassDescriptor {
            label: Some("vugra-render-wgpu-pass"),
            color_attachments: &[Some(wgpu::RenderPassColorAttachment {
                view: &view,
                resolve_target: None,
                ops: wgpu::Operations {
                    load: wgpu::LoadOp::Clear(wgpu::Color {
                        r: 247.0 / 255.0,
                        g: 247.0 / 255.0,
                        b: 248.0 / 255.0,
                        a: 1.0,
                    }),
                    store: wgpu::StoreOp::Store,
                },
            })],
            depth_stencil_attachment: None,
            timestamp_writes: None,
            occlusion_query_set: None,
        });
        if !vertices.is_empty() {
            render_pass.set_pipeline(&pipeline);
            render_pass.set_vertex_buffer(0, vertex_buffer.slice(..));
            render_pass.draw(0..vertices.len() as u32, 0..1);
        }
    }

    let bytes_per_pixel = 4u32;
    let unpadded_bytes_per_row = width * bytes_per_pixel;
    let align = wgpu::COPY_BYTES_PER_ROW_ALIGNMENT;
    let padded_bytes_per_row = unpadded_bytes_per_row.div_ceil(align) * align;
    let output_buffer_size = padded_bytes_per_row as u64 * height as u64;
    let output_buffer = device.create_buffer(&wgpu::BufferDescriptor {
        label: Some("vugra-render-wgpu-readback"),
        size: output_buffer_size,
        usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
        mapped_at_creation: false,
    });
    encoder.copy_texture_to_buffer(
        texture.as_image_copy(),
        wgpu::ImageCopyBuffer {
            buffer: &output_buffer,
            layout: wgpu::ImageDataLayout {
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
    device.poll(wgpu::Maintain::Wait);
    receiver
        .receive()
        .await
        .context("readback callback dropped")?
        .context("map wgpu readback buffer")?;

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

#[cfg(feature = "wgpu-device")]
fn quad_vertices(pass: &[WgpuPassOp], width: u32, height: u32) -> Vec<QuadVertex> {
    let mut vertices = Vec::new();
    for op in pass {
        match op {
            WgpuPassOp::Quad {
                role, rect, color, ..
            } => {
                if !push_icon_vertices(&mut vertices, role, *rect, width, height) {
                    push_rect_vertices(&mut vertices, *rect, *color, width, height);
                }
            }
            WgpuPassOp::Text {
                rect, run, color, ..
            } => {
                push_text_vertices(&mut vertices, run, *color, width, height, Some(*rect));
            }
            WgpuPassOp::End { .. } => {}
        }
    }
    vertices
}

#[cfg(feature = "wgpu-device")]
fn push_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    role: &str,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) -> bool {
    match role {
        "folder-icon" => push_folder_icon_vertices(vertices, rect, width, height),
        "file-icon" => push_file_icon_vertices(vertices, rect, width, height),
        "download-icon" => push_download_icon_vertices(vertices, rect, width, height),
        "picture-icon" => push_picture_icon_vertices(vertices, rect, width, height),
        "project-icon" => push_project_icon_vertices(vertices, rect, width, height),
        "chevron-down-icon" => push_chevron_down_vertices(vertices, rect, width, height),
        "chevron-right-icon" => push_chevron_right_vertices(vertices, rect, width, height),
        "back-icon" => push_back_icon_vertices(vertices, rect, width, height),
        "forward-icon" => push_forward_icon_vertices(vertices, rect, width, height),
        _ => return false,
    }
    true
}

#[cfg(feature = "wgpu-device")]
fn push_folder_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 1.0,
            y: rect.y + 3.0,
            width: rect.width * 0.46,
            height: 5.0,
        },
        Color(120, 183, 255, 255),
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 3.0,
            y: rect.y + 5.0,
            width: rect.width - 4.0,
            height: 4.0,
        },
        Color::FOLDER_ICON_TOP,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 1.0,
            y: rect.y + 7.0,
            width: rect.width - 2.0,
            height: rect.height - 6.0,
        },
        Color::FOLDER_ICON,
        width,
        height,
    );
}

#[cfg(feature = "wgpu-device")]
fn push_file_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 3.0,
            y: rect.y + 1.0,
            width: rect.width - 6.0,
            height: rect.height - 2.0,
        },
        Color::FILE_ICON,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + rect.width - 7.0,
            y: rect.y + 1.0,
            width: 4.0,
            height: 5.0,
        },
        Color::FILE_ICON_ACCENT,
        width,
        height,
    );
    for offset in [8.0, 11.0, 14.0] {
        push_rect_vertices(
            vertices,
            vugra_layout::Rect {
                x: rect.x + 6.0,
                y: rect.y + offset,
                width: if offset == 14.0 { 5.0 } else { 7.0 },
                height: 1.0,
            },
            Color(156, 163, 175, 255),
            width,
            height,
        );
    }
}

#[cfg(feature = "wgpu-device")]
fn push_download_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    for icon_rect in [
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 3.0,
            width: 2.0,
            height: 9.0,
        },
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 10.0,
            width: 8.0,
            height: 2.0,
        },
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 12.0,
            width: 6.0,
            height: 2.0,
        },
    ] {
        push_rect_vertices(vertices, icon_rect, Color::DOWNLOAD_ICON, width, height);
    }
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 3.0,
            y: rect.y + 16.0,
            width: 12.0,
            height: 2.0,
        },
        Color(100, 116, 139, 255),
        width,
        height,
    );
}

#[cfg(feature = "wgpu-device")]
fn push_picture_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 2.0,
            y: rect.y + 3.0,
            width: rect.width - 4.0,
            height: rect.height - 5.0,
        },
        Color::FILE_ICON,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 7.0,
            width: 3.0,
            height: 3.0,
        },
        Color(245, 158, 11, 255),
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 4.0,
            y: rect.y + 13.0,
            width: 11.0,
            height: 3.0,
        },
        Color::PICTURE_ICON,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 11.0,
            width: 5.0,
            height: 3.0,
        },
        Color(34, 197, 94, 255),
        width,
        height,
    );
}

#[cfg(feature = "wgpu-device")]
fn push_project_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 3.0,
            y: rect.y + 3.0,
            width: rect.width - 6.0,
            height: rect.height - 5.0,
        },
        Color(248, 250, 252, 255),
        width,
        height,
    );
    for offset in [7.0, 10.0, 13.0] {
        push_rect_vertices(
            vertices,
            vugra_layout::Rect {
                x: rect.x + 6.0,
                y: rect.y + offset,
                width: if offset == 13.0 { 6.0 } else { 8.0 },
                height: 1.0,
            },
            Color::PROJECT_ICON,
            width,
            height,
        );
    }
}

#[cfg(feature = "wgpu-device")]
fn push_chevron_down_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 4.0,
            y: rect.y + 6.0,
            width: 8.0,
            height: 2.0,
        },
        Color::CHEVRON_ICON,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 8.0,
            width: 4.0,
            height: 2.0,
        },
        Color::CHEVRON_ICON,
        width,
        height,
    );
}

#[cfg(feature = "wgpu-device")]
fn push_chevron_right_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 6.0,
            y: rect.y + 4.0,
            width: 2.0,
            height: 8.0,
        },
        Color::CHEVRON_ICON,
        width,
        height,
    );
    push_rect_vertices(
        vertices,
        vugra_layout::Rect {
            x: rect.x + 8.0,
            y: rect.y + 6.0,
            width: 2.0,
            height: 4.0,
        },
        Color::CHEVRON_ICON,
        width,
        height,
    );
}

#[cfg(feature = "wgpu-device")]
fn push_back_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    for icon_rect in [
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 14.0,
            width: 8.0,
            height: 2.0,
        },
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 12.0,
            width: 3.0,
            height: 2.0,
        },
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 16.0,
            width: 3.0,
            height: 2.0,
        },
    ] {
        push_rect_vertices(vertices, icon_rect, Color::NAV_ICON, width, height);
    }
}

#[cfg(feature = "wgpu-device")]
fn push_forward_icon_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    width: u32,
    height: u32,
) {
    for icon_rect in [
        vugra_layout::Rect {
            x: rect.x + 5.0,
            y: rect.y + 14.0,
            width: 8.0,
            height: 2.0,
        },
        vugra_layout::Rect {
            x: rect.x + 10.0,
            y: rect.y + 12.0,
            width: 3.0,
            height: 2.0,
        },
        vugra_layout::Rect {
            x: rect.x + 10.0,
            y: rect.y + 16.0,
            width: 3.0,
            height: 2.0,
        },
    ] {
        push_rect_vertices(vertices, icon_rect, Color::NAV_ICON, width, height);
    }
}

#[cfg(feature = "wgpu-device")]
fn push_text_vertices(
    vertices: &mut Vec<QuadVertex>,
    run: &TextRun,
    color: Color,
    width: u32,
    height: u32,
    clip: Option<vugra_layout::Rect>,
) {
    for pixel in &run.pixels {
        let mut rect = vugra_layout::Rect {
            x: pixel.x,
            y: pixel.y,
            width: pixel.width,
            height: pixel.height,
        };
        if let Some(clip) = clip {
            let Some(clipped) = intersect_rect(rect, clip) else {
                continue;
            };
            rect = clipped;
        }
        let mut color = color;
        color.3 = ((color.3 as f32) * pixel.alpha.clamp(0.0, 1.0)).round() as u8;
        push_rect_vertices(vertices, rect, color, width, height);
    }
}

#[cfg(feature = "wgpu-device")]
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

#[cfg(feature = "wgpu-device")]
fn push_rect_vertices(
    vertices: &mut Vec<QuadVertex>,
    rect: vugra_layout::Rect,
    color: Color,
    width: u32,
    height: u32,
) {
    if rect.width <= 0.0 || rect.height <= 0.0 {
        return;
    }
    let x0 = ndc_x(rect.x, width);
    let y0 = ndc_y(rect.y, height);
    let x1 = ndc_x(rect.x + rect.width, width);
    let y1 = ndc_y(rect.y + rect.height, height);
    let color = color_f32(color);
    vertices.extend_from_slice(&[
        QuadVertex {
            position: [x0, y0],
            color,
        },
        QuadVertex {
            position: [x1, y0],
            color,
        },
        QuadVertex {
            position: [x0, y1],
            color,
        },
        QuadVertex {
            position: [x0, y1],
            color,
        },
        QuadVertex {
            position: [x1, y0],
            color,
        },
        QuadVertex {
            position: [x1, y1],
            color,
        },
    ]);
}

#[cfg(feature = "wgpu-device")]
fn ndc_x(x: f32, width: u32) -> f32 {
    (x / width as f32) * 2.0 - 1.0
}

#[cfg(feature = "wgpu-device")]
fn ndc_y(y: f32, height: u32) -> f32 {
    1.0 - (y / height as f32) * 2.0
}

#[cfg(feature = "wgpu-device")]
fn color_f32(color: Color) -> [f32; 4] {
    [
        color.0 as f32 / 255.0,
        color.1 as f32 / 255.0,
        color.2 as f32 / 255.0,
        color.3 as f32 / 255.0,
    ]
}

#[cfg(feature = "wgpu-device")]
const QUAD_SHADER: &str = r#"
struct VertexOut {
    @builtin(position) position: vec4<f32>,
    @location(0) color: vec4<f32>,
};

@vertex
fn vs_main(@location(0) position: vec2<f32>, @location(1) color: vec4<f32>) -> VertexOut {
    var out: VertexOut;
    out.position = vec4<f32>(position, 0.0, 1.0);
    out.color = color;
    return out;
}

@fragment
fn fs_main(in: VertexOut) -> @location(0) vec4<f32> {
    return in.color;
}
"#;

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_layout::Rect;
    use vugra_scene::SceneCommand;

    #[test]
    fn wgpu_boundary_accepts_renderer_neutral_scene() {
        let scene = Scene::from_commands(vec![]);
        let mut renderer = WgpuRenderer::default();
        renderer.render(&scene);
        assert!(renderer.commands().is_empty());
        assert!(renderer.pass().is_empty());
    }

    #[test]
    fn wgpu_lowering_projects_render_pass_ops() {
        let rect = Rect {
            x: 5.0,
            y: 6.0,
            width: 7.0,
            height: 8.0,
        };
        let scene = Scene::from_commands(vec![
            SceneCommand::Begin {
                id: "row1-icon".to_string(),
                role: "file-icon".to_string(),
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
            },
            SceneCommand::End {
                id: "row1-icon".to_string(),
            },
        ]);
        let mut renderer = WgpuRenderer::default();
        renderer.render(&scene);
        assert_eq!(
            renderer.pass(),
            &[
                WgpuPassOp::Quad {
                    id: "row1-icon".to_string(),
                    role: "file-icon".to_string(),
                    pipeline: Pipeline::Solid,
                    rect,
                    selected: false,
                    visual_state: vugra_layout::RowVisualState::Normal,
                    method: None,
                    color: Color::FILE_ICON,
                },
                WgpuPassOp::End {
                    id: "row1-icon".to_string()
                },
            ]
        );
    }

    #[test]
    fn wgpu_file_list_lowers_to_go_finder_white_surface() {
        let rect = Rect {
            x: 10.0,
            y: 20.0,
            width: 120.0,
            height: 80.0,
        };
        let pass = lower_commands(&[RenderCommand::Element {
            id: "file-list".to_string(),
            role: "file-list".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
            method: None,
        }]);
        assert_eq!(
            pass.first(),
            Some(&WgpuPassOp::Quad {
                id: "file-list".to_string(),
                role: "file-list".to_string(),
                pipeline: Pipeline::Solid,
                rect,
                selected: false,
                visual_state: vugra_layout::RowVisualState::Normal,
                method: None,
                color: Color::FILE_PANE,
            })
        );
    }

    #[cfg(feature = "wgpu-device")]
    #[test]
    fn wgpu_device_text_vertices_are_submitted_as_quads() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 80.0,
            height: 20.0,
        };
        let pass = vec![WgpuPassOp::Text {
            id: "title:text".to_string(),
            rect,
            text: "FinderLite".to_string(),
            role: "heading".to_string(),
            run: vugra_text::layout_text_run("FinderLite", rect.x, rect.y, 13.0),
            color: Color::TITLE,
        }];
        let vertices = quad_vertices(&pass, 200, 100);
        assert_eq!(vertices.len() % 6, 0);
        assert!(
            vertices.len() > 6 * 8,
            "expected glyph pixels to become GPU quads, got {} vertices",
            vertices.len()
        );
    }

    #[cfg(feature = "wgpu-device")]
    #[test]
    fn wgpu_device_icon_vertices_expand_role_shapes() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 18.0,
            height: 18.0,
        };
        let pass = vec![WgpuPassOp::Quad {
            id: "folder".to_string(),
            role: "folder-icon".to_string(),
            pipeline: Pipeline::Solid,
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
            method: None,
            color: Color::FOLDER_ICON,
        }];
        let vertices = quad_vertices(&pass, 100, 80);
        assert_eq!(vertices.len(), 18);
        let colors = vertices
            .chunks_exact(6)
            .map(|quad| quad[0].color)
            .collect::<Vec<_>>();
        assert_eq!(colors[0], color_f32(Color(120, 183, 255, 255)));
        assert_eq!(colors[1], color_f32(Color::FOLDER_ICON_TOP));
        assert_eq!(colors[2], color_f32(Color::FOLDER_ICON));
    }

    #[test]
    fn wgpu_text_ops_carry_unified_text_run() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 90.0,
            height: 20.0,
        };
        let pass = lower_commands_with_text_provider(
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
        let [WgpuPassOp::Text { run, .. }] = pass.as_slice() else {
            panic!("expected one text op, got {pass:?}");
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
    fn wgpu_default_text_ops_use_system_font_provider_when_available() {
        let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for wgpu default provider test");
            return;
        }
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let pass = lower_commands(&[RenderCommand::Text {
            id: "label:text".to_string(),
            text: "Finder 文".to_string(),
            role: "text".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        let [WgpuPassOp::Text { run, .. }] = pass.as_slice() else {
            panic!("expected one text op, got {pass:?}");
        };
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.source == vugra_text::GlyphSource::LoadedFont));
        assert!(run.pixels.iter().any(|pixel| pixel.alpha < 1.0));
    }

    #[test]
    fn wgpu_text_ops_use_role_colors_from_go_finder_css() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let pass = lower_commands(&[
            RenderCommand::Text {
                id: "row1-modified:text".to_string(),
                text: "May 24".to_string(),
                role: "row-date-cell".to_string(),
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
            &pass[0],
            WgpuPassOp::Text {
                color: Color::TEXT_MUTED,
                ..
            }
        ));
        assert!(matches!(
            &pass[1],
            WgpuPassOp::Text {
                color: Color::TEXT,
                ..
            }
        ));
    }

    #[test]
    fn wgpu_row_focus_and_editing_text_use_go_finder_accent_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let pass = lower_commands(&[
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
            &pass[0],
            WgpuPassOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
        assert!(matches!(
            &pass[1],
            WgpuPassOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
        assert!(matches!(
            &pass[2],
            WgpuPassOp::Text {
                color: Color::TEXT_INVERTED,
                ..
            }
        ));
    }

    #[test]
    fn wgpu_selected_sidebar_label_uses_go_finder_active_text_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let pass = lower_commands(&[RenderCommand::Text {
            id: "sidebar-documents-label:text".to_string(),
            text: "Documents".to_string(),
            role: "sidebar-item-label".to_string(),
            rect,
            selected: true,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        assert!(matches!(
            &pass[0],
            WgpuPassOp::Text {
                color: Color::ROW_ACCENT_TEXT,
                ..
            }
        ));
    }

    #[test]
    fn wgpu_search_text_uses_go_renderer_default_input_color() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 20.0,
        };
        let pass = lower_commands(&[RenderCommand::Text {
            id: "search:text".to_string(),
            text: "road".to_string(),
            role: "search".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        assert!(matches!(
            &pass[0],
            WgpuPassOp::Text {
                color: Color::INPUT_TEXT,
                ..
            }
        ));
    }

    #[test]
    fn wgpu_text_ops_use_role_font_sizes_from_go_finder_css() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 160.0,
            height: 24.0,
        };
        let pass = lower_commands(&[
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
        let [WgpuPassOp::Text { run: title, .. }, WgpuPassOp::Text { run: row, .. }] =
            pass.as_slice()
        else {
            panic!("expected two text ops, got {pass:?}");
        };
        assert_eq!(title.font_size, 15.0);
        assert_eq!(row.font_size, 13.0);
        assert_eq!(title.font_weight, 600);
        assert_eq!(row.font_weight, 400);
        assert!(title.metrics.height >= row.metrics.height);
    }

    #[test]
    fn wgpu_text_ops_use_rect_width_for_unified_line_boxes() {
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 18.0,
            height: 40.0,
        };
        let pass = lower_commands_with_text_provider(
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
        let [WgpuPassOp::Text { run, .. }] = pass.as_slice() else {
            panic!("expected one text op, got {pass:?}");
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
    fn wgpu_text_ops_can_use_loaded_font_provider() {
        let Some(font) = vugra_text::SystemFontTextMeasurer::new().load_for_text("Finder 文")
        else {
            eprintln!("no system font available for wgpu loaded font text test");
            return;
        };
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let pass = lower_commands_with_text_provider(
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
        let [WgpuPassOp::Text { run, .. }] = pass.as_slice() else {
            panic!("expected one text op, got {pass:?}");
        };
        assert!(run.pixels.len() > 20);
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.source == vugra_text::GlyphSource::LoadedFont));
        assert!(run.metrics.width > 40.0);
    }

    #[test]
    fn wgpu_text_ops_can_use_system_fallback_provider() {
        let provider = vugra_text::SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for wgpu fallback provider test");
            return;
        }
        let rect = Rect {
            x: 10.0,
            y: 12.0,
            width: 120.0,
            height: 40.0,
        };
        let pass = lower_commands_with_text_provider(
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
        let [WgpuPassOp::Text { run, .. }] = pass.as_slice() else {
            panic!("expected one text op, got {pass:?}");
        };
        assert!(run.pixels.len() > 20);
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.font_key.is_some()));
    }
}
