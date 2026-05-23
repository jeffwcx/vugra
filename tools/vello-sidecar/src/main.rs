use anyhow::{Context, Result};
use serde::Deserialize;
use std::collections::HashMap;
use std::fs::File;
use std::io::{self, BufWriter, Read};
use std::path::Path;
use ttf_parser::Face;
use vello::wgpu;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "PascalCase")]
struct Op {
    kind: String,
    rect: Rect,
    text: String,
    #[serde(default)]
    lines: Option<Vec<LineBox>>,
    #[serde(default)]
    glyphs: Option<Vec<GlyphRun>>,
    role: String,
    tag: String,
    props: Option<HashMap<String, String>>,
    #[serde(default)]
    style: Style,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "lowercase")]
struct Rect {
    x: f32,
    y: f32,
    width: f32,
    height: f32,
}

#[derive(Clone, Debug, Default, Deserialize)]
#[serde(rename_all = "lowercase")]
struct LineBox {
    text: String,
    x: f32,
    y: f32,
    width: f32,
    height: f32,
    baseline: f32,
}

#[derive(Debug, Default, Deserialize)]
#[serde(rename_all = "lowercase")]
struct GlyphRun {
    text: String,
    #[serde(default)]
    font: String,
    size: f32,
    x: f32,
    y: f32,
    advance: f32,
    baseline: f32,
}

#[derive(Debug, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
struct Style {
    #[serde(default)]
    font_size: f32,
    #[serde(default)]
    line_height: f32,
    #[serde(default)]
    background_color: String,
    #[serde(default)]
    opacity: f32,
    #[serde(default)]
    border_width: f32,
    #[serde(default)]
    border_color: String,
    #[serde(default)]
    border_radius: f32,
    #[serde(default)]
    color: String,
    #[serde(default)]
    overflow: String,
}

fn main() -> Result<()> {
    let mut input = String::new();
    io::stdin()
        .read_to_string(&mut input)
        .context("read Vugra Vello ops from stdin")?;
    let ops: Vec<Op> = serde_json::from_str(&input).context("parse Vugra Vello ops JSON")?;

    let mut scene = vello::Scene::new();
    let font = load_font();
    let mut text_ops = 0;
    for op in &ops {
        match op.kind.as_str() {
            "fill-rect" => {
                draw_control_fill(&mut scene, op);
            }
            "stroke-rect" => {
                draw_control_stroke(&mut scene, op);
            }
            "begin-clip" => {
                push_clip(&mut scene, op);
            }
            "end-clip" => {
                scene.pop_layer();
            }
            "text" => {
                if let Some(font) = font.as_ref() {
                    if draw_glyph_runs(&mut scene, font, op) == 0 {
                        draw_fallback_text(&mut scene, font, op);
                    }
                    text_ops += 1;
                }
            }
            _ => {}
        }
    }

    let output = Output::from_args(std::env::args().skip(1).collect());
    let render_status = match pollster::block_on(render_scene(&scene, output.width, output.height))
    {
        Ok(image) => {
            if let Some(path) = output.png.as_deref() {
                write_png(path, image.width, image.height, &image.pixels)
                    .with_context(|| format!("write Vello PNG {path}"))?;
            }
            if let Some(path) = output.raw.as_deref() {
                std::fs::write(path, &image.pixels)
                    .with_context(|| format!("write Vello raw RGBA {path}"))?;
            }
            match output.png.as_deref().or(output.raw.as_deref()) {
                Some(path) => format!(
                    ",\"render\":\"texture\",\"checksum\":{},\"output\":{:?}",
                    image.checksum, path
                ),
                None => format!(",\"render\":\"texture\",\"checksum\":{}", image.checksum),
            }
        }
        Err(err) => format!(
            ",\"render\":\"unavailable\",\"renderError\":{:?}",
            err.to_string()
        ),
    };

    println!(
        "{{\"backend\":\"vello\",\"ops\":{},\"textOps\":{},\"status\":\"scene-built\"{}}}",
        ops.len(),
        text_ops,
        render_status
    );
    Ok(())
}

fn draw_control_fill(scene: &mut vello::Scene, op: &Op) {
    if !should_draw_element(op) {
        return;
    }
    let rect = shape(op);
    let color = apply_opacity(
        parse_color(&op.style.background_color).unwrap_or_else(|| match op.role.as_str() {
            "button" => vello::peniko::Color::from_rgb8(238, 246, 255),
            "textbox" => vello::peniko::Color::from_rgb8(255, 255, 255),
            "checkbox" => {
                if prop(op, "checked").is_some_and(|value| value == "true") {
                    vello::peniko::Color::from_rgb8(37, 99, 235)
                } else {
                    vello::peniko::Color::from_rgb8(255, 255, 255)
                }
            }
            "listitem" => vello::peniko::Color::from_rgb8(255, 255, 255),
            _ => vello::peniko::Color::from_rgb8(248, 250, 252),
        }),
        op.style.opacity,
    );
    scene.fill(
        vello::peniko::Fill::NonZero,
        vello::kurbo::Affine::IDENTITY,
        color,
        None,
        &rect,
    );
}

fn draw_control_stroke(scene: &mut vello::Scene, op: &Op) {
    if !should_draw_element(op) {
        return;
    }
    let rect = shape(op);
    let color = apply_opacity(
        parse_color(&op.style.border_color).unwrap_or_else(|| match op.role.as_str() {
            "button" => vello::peniko::Color::from_rgb8(37, 99, 235),
            "textbox" | "checkbox" => vello::peniko::Color::from_rgb8(148, 163, 184),
            "listitem" => vello::peniko::Color::from_rgb8(203, 213, 225),
            _ => vello::peniko::Color::from_rgb8(226, 232, 240),
        }),
        op.style.opacity,
    );
    let border_width = element_border_width(op);
    if border_width <= 0.0 {
        return;
    }
    scene.stroke(
        &vello::kurbo::Stroke::new(border_width),
        vello::kurbo::Affine::IDENTITY,
        color,
        None,
        &rect,
    );
    if op.role == "checkbox" && prop(op, "checked").is_some_and(|value| value == "true") {
        let x = op.rect.x as f64;
        let y = op.rect.y as f64;
        let mut path = vello::kurbo::BezPath::new();
        path.move_to((x + 4.0, y + 10.0));
        path.line_to((x + 8.0, y + 14.0));
        path.line_to((x + 16.0, y + 5.0));
        scene.stroke(
            &vello::kurbo::Stroke::new(2.0),
            vello::kurbo::Affine::IDENTITY,
            vello::peniko::Color::from_rgb8(255, 255, 255),
            None,
            &path,
        );
    }
}

fn push_clip(scene: &mut vello::Scene, op: &Op) {
    if !clips_overflow(&op.style.overflow) {
        return;
    }
    let rect = shape(op);
    scene.push_clip_layer(
        vello::peniko::Fill::NonZero,
        vello::kurbo::Affine::IDENTITY,
        &rect,
    );
}

fn clips_overflow(overflow: &str) -> bool {
    overflow == "hidden" || overflow == "scroll"
}

fn should_draw_element(op: &Op) -> bool {
    if is_control_element(op) {
        return true;
    }
    !op.style.background_color.is_empty()
        || !op.style.border_color.is_empty()
        || op.style.border_width > 0.0
        || op.style.border_radius > 0.0
}

fn is_control_element(op: &Op) -> bool {
    matches!(op.role.as_str(), "button" | "textbox" | "checkbox")
        || matches!(op.tag.as_str(), "button" | "input")
}

fn prop<'a>(op: &'a Op, name: &str) -> Option<&'a str> {
    op.props
        .as_ref()
        .and_then(|props| props.get(name))
        .map(String::as_str)
}

fn text_color(op: &Op) -> vello::peniko::Color {
    let color = if let Some(color) = parse_color(&op.style.color) {
        color
    } else if op.role == "button" {
        vello::peniko::Color::from_rgb8(37, 99, 235)
    } else {
        vello::peniko::Color::from_rgb8(15, 23, 42)
    };
    apply_opacity(color, op.style.opacity)
}

fn draw_glyph_runs(scene: &mut vello::Scene, font: &LoadedFont, op: &Op) -> usize {
    let text_color = text_color(op);
    let mut count = 0usize;
    let glyphs = op.glyphs.as_deref().unwrap_or(&[]);
    for run in glyphs {
        let size = if run.size > 0.0 {
            run.size
        } else {
            font_size(op)
        };
        let y = run.y
            + if run.baseline > 0.0 {
                run.baseline
            } else {
                baseline(size, op.style.line_height)
            };
        let advance = if run.advance > 0.0 {
            run.advance / run.text.chars().count().max(1) as f32
        } else if !run.font.is_empty() {
            char_advance(size)
        } else {
            char_advance(size)
        };
        let glyphs = run.text.chars().enumerate().filter_map(|(index, ch)| {
            font.glyph_id(ch).map(|id| vello::Glyph {
                id,
                x: run.x + (index as f32 * advance),
                y,
            })
        });
        scene
            .draw_glyphs(&font.data)
            .font_size(size)
            .brush(text_color)
            .draw(vello::peniko::Fill::NonZero, glyphs);
        count += run.text.chars().count();
    }
    count
}

fn draw_fallback_text(scene: &mut vello::Scene, font: &LoadedFont, op: &Op) {
    let size = font_size(op);
    let text_color = text_color(op);
    let op_lines = op.lines.as_deref().unwrap_or(&[]);
    let lines: Vec<LineBox> = if op_lines.is_empty() {
        op.text
            .lines()
            .enumerate()
            .map(|(index, text)| LineBox {
                text: text.to_string(),
                x: op.rect.x,
                y: op.rect.y + index as f32 * line_height(op, size),
                width: op.rect.width,
                height: line_height(op, size),
                baseline: baseline(size, op.style.line_height),
            })
            .collect()
    } else {
        op_lines.to_vec()
    };
    for line in lines {
        let advance = if line.width > 0.0 {
            line.width / line.text.chars().count().max(1) as f32
        } else {
            char_advance(size)
        };
        let y = line.y
            + if line.baseline > 0.0 {
                line.baseline
            } else {
                baseline(size, op.style.line_height)
            };
        let line_size = if line.height > 0.0 {
            size.min(line.height)
        } else {
            size
        };
        let glyphs = line.text.chars().enumerate().filter_map(|(index, ch)| {
            font.glyph_id(ch).map(|id| vello::Glyph {
                id,
                x: line.x + (index as f32 * advance),
                y,
            })
        });
        scene
            .draw_glyphs(&font.data)
            .font_size(line_size)
            .brush(text_color)
            .draw(vello::peniko::Fill::NonZero, glyphs);
    }
}

fn parse_color(value: &str) -> Option<vello::peniko::Color> {
    let value = value.trim();
    if value.len() != 7 || !value.starts_with('#') {
        return None;
    }
    let rgb = u32::from_str_radix(&value[1..], 16).ok()?;
    Some(vello::peniko::Color::from_rgb8(
        ((rgb >> 16) & 0xff) as u8,
        ((rgb >> 8) & 0xff) as u8,
        (rgb & 0xff) as u8,
    ))
}

fn apply_opacity(color: vello::peniko::Color, opacity: f32) -> vello::peniko::Color {
    if opacity <= 0.0 || opacity >= 1.0 {
        return color;
    }
    color.multiply_alpha(opacity)
}

fn rect(op: &Op) -> vello::kurbo::Rect {
    vello::kurbo::Rect::new(
        op.rect.x as f64,
        op.rect.y as f64,
        (op.rect.x + op.rect.width) as f64,
        (op.rect.y + op.rect.height) as f64,
    )
}

fn shape(op: &Op) -> impl vello::kurbo::Shape {
    let rect = rect(op);
    if op.style.border_radius > 0.0 {
        rect.to_rounded_rect(op.style.border_radius as f64)
    } else {
        rect.to_rounded_rect(0.0)
    }
}

fn element_border_width(op: &Op) -> f64 {
    if op.style.border_width > 0.0 {
        return op.style.border_width as f64;
    }
    if !is_control_element(op) {
        if op.style.border_color.is_empty() {
            return 0.0;
        }
        return 1.0;
    }
    1.0
}

fn font_size(op: &Op) -> f32 {
    if op.style.font_size > 0.0 {
        return op.style.font_size;
    }
    match op.tag.as_str() {
        "h1" => 28.0,
        "h2" => 24.0,
        "h3" => 20.0,
        "h4" | "h5" | "h6" => 18.0,
        _ => 16.0,
    }
}

fn line_height(op: &Op, size: f32) -> f32 {
    if op.style.line_height > 0.0 {
        op.style.line_height
    } else if size >= 24.0 {
        size + 10.0
    } else if size >= 18.0 {
        size + 8.0
    } else {
        24.0
    }
}

fn baseline(size: f32, line_height: f32) -> f32 {
    if line_height > 0.0 {
        return ((line_height - size) * 0.5 + size * 0.82).round();
    }
    (size * 0.82).round()
}

fn char_advance(size: f32) -> f32 {
    (size * 0.52).max(7.0)
}

struct Output {
    png: Option<String>,
    raw: Option<String>,
    width: u32,
    height: u32,
}

impl Output {
    fn from_args(args: Vec<String>) -> Self {
        let mut output = Self {
            png: None,
            raw: None,
            width: 800,
            height: 600,
        };
        let mut i = 0;
        while i < args.len() {
            match args[i].as_str() {
                "--raw" if i + 1 < args.len() => {
                    output.raw = Some(args[i + 1].clone());
                    i += 2;
                }
                "--size" if i + 1 < args.len() => {
                    if let Some((width, height)) = parse_size(&args[i + 1]) {
                        output.width = width;
                        output.height = height;
                    }
                    i += 2;
                }
                path => {
                    output.png = Some(path.to_string());
                    i += 1;
                }
            }
        }
        output
    }
}

fn parse_size(value: &str) -> Option<(u32, u32)> {
    let (width, height) = value.split_once('x')?;
    let width = width.parse().ok()?;
    let height = height.parse().ok()?;
    if width == 0 || height == 0 {
        return None;
    }
    Some((width, height))
}

struct RenderedImage {
    width: u32,
    height: u32,
    pixels: Vec<u8>,
    checksum: u64,
}

async fn render_scene(scene: &vello::Scene, width: u32, height: u32) -> Result<RenderedImage> {
    let instance = wgpu::Instance::new(wgpu::InstanceDescriptor::new_without_display_handle());
    let adapter = instance
        .request_adapter(&wgpu::RequestAdapterOptions::default())
        .await
        .context("request wgpu adapter")?;
    let (device, queue) = adapter
        .request_device(&wgpu::DeviceDescriptor {
            label: Some("vugra-vello-sidecar-device"),
            required_features: wgpu::Features::empty(),
            required_limits: adapter.limits(),
            experimental_features: wgpu::ExperimentalFeatures::disabled(),
            memory_hints: wgpu::MemoryHints::Performance,
            trace: wgpu::Trace::Off,
        })
        .await
        .context("request wgpu device")?;
    let texture = device.create_texture(&wgpu::TextureDescriptor {
        label: Some("vugra-vello-sidecar-target"),
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
                base_color: vello::peniko::Color::from_rgb8(250, 250, 250),
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
        label: Some("vugra-vello-sidecar-readback"),
        size: output_buffer_size,
        usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
        mapped_at_creation: false,
    });
    let mut encoder = device.create_command_encoder(&wgpu::CommandEncoderDescriptor {
        label: Some("vugra-vello-sidecar-copy"),
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
    Ok(RenderedImage {
        width,
        height,
        pixels,
        checksum,
    })
}

fn write_png(path: &str, width: u32, height: u32, pixels: &[u8]) -> Result<()> {
    let file = File::create(path)?;
    let writer = BufWriter::new(file);
    let mut encoder = png::Encoder::new(writer, width, height);
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    let mut png_writer = encoder.write_header()?;
    png_writer.write_image_data(pixels)?;
    Ok(())
}

struct LoadedFont {
    bytes: Vec<u8>,
    data: vello::peniko::FontData,
    index: u32,
}

impl LoadedFont {
    fn glyph_id(&self, ch: char) -> Option<u32> {
        let face = Face::parse(&self.bytes, self.index).ok()?;
        face.glyph_index(ch).map(|id| u32::from(id.0))
    }
}

fn load_font() -> Option<LoadedFont> {
    for path in [
        "/System/Library/Fonts/Helvetica.ttc",
        "/System/Library/Fonts/SFNS.ttf",
        "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Supplemental/Courier New.ttf",
    ] {
        if Path::new(path).exists() {
            if let Ok(bytes) = std::fs::read(path) {
                for index in 0..8 {
                    if let Ok(face) = Face::parse(&bytes, index) {
                        if face.glyph_index('A').is_some() {
                            return Some(LoadedFont {
                                data: vello::peniko::FontData::new(bytes.clone().into(), index),
                                bytes,
                                index,
                            });
                        }
                    }
                }
            }
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_go_renderer_text_layout_fields() {
        let ops: Vec<Op> = serde_json::from_str(
            r#"[
              {
                "Kind":"text",
                "Rect":{"x":4,"y":5,"width":80,"height":24},
                "Text":"Finder",
                "Lines":[{"text":"Finder","x":4,"y":5,"width":52,"height":20,"baseline":15}],
                "Glyphs":[{"text":"Finder","font":"system","size":13,"x":4,"y":5,"advance":52,"baseline":15}],
                "Role":"text",
                "Tag":"span",
                "Style":{"fontSize":13,"lineHeight":20}
              }
            ]"#,
        )
        .expect("parse Go-renderer Vello ops");
        let op = &ops[0];
        assert_eq!(op.lines.as_ref().unwrap()[0].text, "Finder");
        assert_eq!(op.glyphs.as_ref().unwrap()[0].advance, 52.0);
    }
}
