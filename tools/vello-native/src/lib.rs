use anyhow::{anyhow, Context, Result};
use cosmic_text::{Align, Attrs, Buffer, Family, FontSystem, Metrics, Shaping};
use serde::Deserialize;
use std::collections::HashMap;
use std::ffi::{c_char, CStr, CString};
use std::num::NonZeroUsize;
use std::os::raw::c_int;
use std::ptr;
use vello::peniko::{BlendMode, Color, Fill, Font};
use vello::wgpu;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "PascalCase")]
struct Op {
    kind: String,
    #[serde(default)]
    rect: Rect,
    #[serde(default)]
    text: String,
    #[serde(default, alias = "SVG")]
    svg: String,
    #[serde(default)]
    lines: Option<Vec<LineBox>>,
    #[serde(default)]
    glyphs: Option<Vec<GlyphRun>>,
    #[serde(default)]
    role: String,
    #[serde(default)]
    tag: String,
    props: Option<HashMap<String, String>>,
    #[serde(default)]
    style: Style,
}

#[derive(Debug, Default, Deserialize)]
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

#[repr(C)]
pub struct VugraTextMetrics {
    width: f32,
    height: f32,
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
    text_align: String,
    #[serde(default)]
    background_color: String,
    #[serde(default)]
    opacity: f32,
    #[serde(default)]
    border_width: f32,
    #[serde(default)]
    border_width_set: bool,
    #[serde(default)]
    border_color: String,
    #[serde(default)]
    border_radius: f32,
    #[serde(default)]
    color: String,
    #[serde(default)]
    overflow: String,
}

pub struct VugraNativeRenderer {
    width: u32,
    height: u32,
    device: wgpu::Device,
    queue: wgpu::Queue,
    renderer: vello::Renderer,
    font_system: FontSystem,
    font_cache: HashMap<(cosmic_text::fontdb::ID, cosmic_text::fontdb::Weight), Font>,
    pixels: Vec<u8>,
    status: CString,
}

impl VugraNativeRenderer {
    fn new(width: u32, height: u32) -> Result<Self> {
        let (device, queue) = pollster::block_on(create_device())?;
        let renderer = vello::Renderer::new(
            &device,
            vello::RendererOptions {
                surface_format: None,
                use_cpu: false,
                antialiasing_support: vello::AaSupport::area_only(),
                num_init_threads: NonZeroUsize::new(1),
            },
        )
        .context("create Vello renderer")?;
        Ok(Self {
            width: width.max(1),
            height: height.max(1),
            device,
            queue,
            renderer,
            font_system: FontSystem::new(),
            font_cache: HashMap::new(),
            pixels: Vec::new(),
            status: cstring("created"),
        })
    }

    fn resize(&mut self, width: u32, height: u32) {
        self.width = width.max(1);
        self.height = height.max(1);
    }

    fn render(&mut self, json: &[u8]) -> Result<()> {
        let ops: Vec<Op> = serde_json::from_slice(json).context("parse Vugra Vello ops JSON")?;
        let mut scene = vello::Scene::new();
        let mut text_ops = 0usize;
        let mut svg_ops = 0usize;
        let mut svg_draw_ops = 0usize;
        let mut glyph_ops = 0usize;

        for op in &ops {
            match op.kind.as_str() {
                "fill-rect" => draw_control_fill(&mut scene, op),
                "stroke-rect" => draw_control_stroke(&mut scene, op),
                "begin-clip" => push_clip(&mut scene, op),
                "end-clip" => scene.pop_layer(),
                "text" => {
                    text_ops += 1;
                    glyph_ops += self.draw_text(&mut scene, op)?;
                }
                "svg" => {
                    svg_ops += 1;
                    svg_draw_ops += draw_svg(&mut scene, op)?;
                }
                _ => {}
            }
        }

        self.pixels = pollster::block_on(self.render_scene(&scene))?;
        self.status = cstring(format!(
            "{{\"backend\":\"vello-native\",\"render\":\"texture\",\"ops\":{},\"textOps\":{},\"svgOps\":{},\"svgDrawOps\":{},\"glyphOps\":{},\"fonts\":{},\"pixels\":{}}}",
            ops.len(),
            text_ops,
            svg_ops,
            svg_draw_ops,
            glyph_ops,
            self.font_cache.len(),
            self.pixels.len()
        ));
        Ok(())
    }

    fn draw_text(&mut self, scene: &mut vello::Scene, op: &Op) -> Result<usize> {
        if op.text.is_empty() || op.rect.width <= 0.0 || op.rect.height <= 0.0 {
            return Ok(0);
        }

        if op.glyphs.as_ref().is_some_and(|glyphs| !glyphs.is_empty()) {
            return self.draw_text_from_runs(scene, op);
        }

        if op.lines.as_ref().is_some_and(|lines| !lines.is_empty()) {
            return self.draw_text_from_lines(scene, op);
        }

        let font_size = font_size(op);
        let line_height = line_height(op, font_size);
        let metrics = Metrics::new(font_size, line_height);
        let mut buffer = Buffer::new(&mut self.font_system, metrics);
        buffer.set_size(
            &mut self.font_system,
            Some(text_layout_width(op, font_size)),
            Some(op.rect.height),
        );
        let attrs = Attrs::new().family(Family::SansSerif);
        buffer.set_text(
            &mut self.font_system,
            &op.text,
            &attrs,
            Shaping::Advanced,
            text_align(op),
        );
        buffer.shape_until_scroll(&mut self.font_system, false);

        let color = text_color(op);
        let mut glyph_count = 0usize;
        let mut groups: HashMap<
            (cosmic_text::fontdb::ID, cosmic_text::fontdb::Weight),
            Vec<vello::Glyph>,
        > = HashMap::new();

        for run in buffer.layout_runs() {
            for glyph in run.glyphs {
                let key = (glyph.font_id, glyph.font_weight);
                groups.entry(key).or_default().push(vello::Glyph {
                    id: u32::from(glyph.glyph_id),
                    x: op.rect.x + glyph.x + glyph.font_size * glyph.x_offset,
                    y: op.rect.y + run.line_y + glyph.y - glyph.font_size * glyph.y_offset,
                });
                glyph_count += 1;
            }
        }

        for (key, glyphs) in groups {
            let font_data = self.font_data(key)?;
            scene
                .draw_glyphs(&font_data)
                .font_size(font_size)
                .brush(color)
                .hint(true)
                .draw(Fill::NonZero, glyphs.into_iter());
        }
        Ok(glyph_count)
    }

    fn draw_text_from_runs(&mut self, scene: &mut vello::Scene, op: &Op) -> Result<usize> {
        let attrs = Attrs::new().family(Family::SansSerif);
        let color = text_color(op);
        let mut glyph_count = 0usize;
        let mut groups: HashMap<
            (cosmic_text::fontdb::ID, cosmic_text::fontdb::Weight, u32),
            Vec<vello::Glyph>,
        > = HashMap::new();

        for run in op.glyphs.as_deref().unwrap_or(&[]) {
            let font_size = if run.size > 0.0 {
                run.size
            } else if !run.font.is_empty() {
                font_size(op)
            } else {
                font_size(op)
            };
            let line_height = line_height(op, font_size);
            let mut buffer =
                Buffer::new(&mut self.font_system, Metrics::new(font_size, line_height));
            buffer.set_size(
                &mut self.font_system,
                Some(run.advance.max(text_run_width(run, font_size))),
                Some(line_height),
            );
            buffer.set_text(
                &mut self.font_system,
                &run.text,
                &attrs,
                Shaping::Advanced,
                None,
            );
            buffer.shape_until_scroll(&mut self.font_system, false);
            for shaped in buffer.layout_runs() {
                for glyph in shaped.glyphs {
                    let key = (glyph.font_id, glyph.font_weight, font_size.to_bits());
                    groups.entry(key).or_default().push(vello::Glyph {
                        id: u32::from(glyph.glyph_id),
                        x: run.x + glyph.x + glyph.font_size * glyph.x_offset,
                        y: run.y + run.baseline + glyph.y - glyph.font_size * glyph.y_offset,
                    });
                    glyph_count += 1;
                }
            }
        }

        for ((font_id, weight, size_bits), glyphs) in groups {
            let font_data = self.font_data((font_id, weight))?;
            scene
                .draw_glyphs(&font_data)
                .font_size(f32::from_bits(size_bits))
                .brush(color)
                .hint(true)
                .draw(Fill::NonZero, glyphs.into_iter());
        }
        Ok(glyph_count)
    }

    fn draw_text_from_lines(&mut self, scene: &mut vello::Scene, op: &Op) -> Result<usize> {
        let attrs = Attrs::new().family(Family::SansSerif);
        let color = text_color(op);
        let font_size = font_size(op);
        let mut glyph_count = 0usize;
        let mut groups: HashMap<
            (cosmic_text::fontdb::ID, cosmic_text::fontdb::Weight),
            Vec<vello::Glyph>,
        > = HashMap::new();

        for line in op.lines.as_deref().unwrap_or(&[]) {
            let height = if line.height > 0.0 {
                line.height
            } else {
                line_height(op, font_size)
            };
            let mut buffer = Buffer::new(&mut self.font_system, Metrics::new(font_size, height));
            buffer.set_size(
                &mut self.font_system,
                Some(line.width.max(text_line_width(&line.text, font_size))),
                Some(height),
            );
            buffer.set_text(
                &mut self.font_system,
                &line.text,
                &attrs,
                Shaping::Advanced,
                text_align(op),
            );
            buffer.shape_until_scroll(&mut self.font_system, false);
            for shaped in buffer.layout_runs() {
                for glyph in shaped.glyphs {
                    let key = (glyph.font_id, glyph.font_weight);
                    groups.entry(key).or_default().push(vello::Glyph {
                        id: u32::from(glyph.glyph_id),
                        x: line.x + glyph.x + glyph.font_size * glyph.x_offset,
                        y: line.y + line.baseline + glyph.y - glyph.font_size * glyph.y_offset,
                    });
                    glyph_count += 1;
                }
            }
        }

        for (key, glyphs) in groups {
            let font_data = self.font_data(key)?;
            scene
                .draw_glyphs(&font_data)
                .font_size(font_size)
                .brush(color)
                .hint(true)
                .draw(Fill::NonZero, glyphs.into_iter());
        }
        Ok(glyph_count)
    }

    fn font_data(
        &mut self,
        key: (cosmic_text::fontdb::ID, cosmic_text::fontdb::Weight),
    ) -> Result<Font> {
        if let Some(data) = self.font_cache.get(&key) {
            return Ok(data.clone());
        }
        let font = self
            .font_system
            .get_font(key.0, key.1)
            .ok_or_else(|| anyhow!("load shaped font"))?;
        let index = self
            .font_system
            .db()
            .face(key.0)
            .map(|face| face.index)
            .unwrap_or(0);
        let data = Font::new(font.data().to_vec().into(), index);
        self.font_cache.insert(key, data.clone());
        Ok(data)
    }

    async fn render_scene(&mut self, scene: &vello::Scene) -> Result<Vec<u8>> {
        let texture = self.device.create_texture(&wgpu::TextureDescriptor {
            label: Some("vugra-vello-native-target"),
            size: wgpu::Extent3d {
                width: self.width,
                height: self.height,
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
        self.renderer
            .render_to_texture(
                &self.device,
                &self.queue,
                scene,
                &view,
                &vello::RenderParams {
                    base_color: Color::from_rgb8(250, 250, 250),
                    width: self.width,
                    height: self.height,
                    antialiasing_method: vello::AaConfig::Area,
                },
            )
            .context("render Vello scene to texture")?;

        let bytes_per_pixel = 4u32;
        let unpadded_bytes_per_row = self.width * bytes_per_pixel;
        let align = wgpu::COPY_BYTES_PER_ROW_ALIGNMENT;
        let padded_bytes_per_row = unpadded_bytes_per_row.div_ceil(align) * align;
        let output_buffer_size = padded_bytes_per_row as u64 * self.height as u64;
        let output_buffer = self.device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("vugra-vello-native-readback"),
            size: output_buffer_size,
            usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
            mapped_at_creation: false,
        });
        let mut encoder = self
            .device
            .create_command_encoder(&wgpu::CommandEncoderDescriptor {
                label: Some("vugra-vello-native-copy"),
            });
        encoder.copy_texture_to_buffer(
            texture.as_image_copy(),
            wgpu::ImageCopyBuffer {
                buffer: &output_buffer,
                layout: wgpu::ImageDataLayout {
                    offset: 0,
                    bytes_per_row: Some(padded_bytes_per_row),
                    rows_per_image: Some(self.height),
                },
            },
            wgpu::Extent3d {
                width: self.width,
                height: self.height,
                depth_or_array_layers: 1,
            },
        );
        self.queue.submit(Some(encoder.finish()));

        let buffer_slice = output_buffer.slice(..);
        let (sender, receiver) = futures_intrusive::channel::shared::oneshot_channel();
        buffer_slice.map_async(wgpu::MapMode::Read, move |result| {
            sender.send(result).ok();
        });
        self.device.poll(wgpu::Maintain::Wait);
        receiver
            .receive()
            .await
            .context("readback callback dropped")?
            .context("map Vello readback buffer")?;

        let data = buffer_slice.get_mapped_range();
        let mut pixels = Vec::with_capacity((self.width * self.height * 4) as usize);
        for row in 0..self.height as usize {
            let start = row * padded_bytes_per_row as usize;
            let end = start + unpadded_bytes_per_row as usize;
            pixels.extend_from_slice(&data[start..end]);
        }
        drop(data);
        output_buffer.unmap();
        Ok(pixels)
    }

    fn set_error(&mut self, err: anyhow::Error) {
        self.status = cstring(format!(
            "{{\"backend\":\"vello-native\",\"render\":\"error\",\"error\":{:?}}}",
            err.to_string()
        ));
    }
}

fn draw_svg(scene: &mut vello::Scene, op: &Op) -> Result<usize> {
    if op.svg.trim().is_empty() || op.rect.width <= 0.0 || op.rect.height <= 0.0 {
        return Ok(0);
    }
    let doc = roxmltree::Document::parse(&op.svg).context("parse SVG XML")?;
    let root = doc.root_element();
    let (min_x, min_y, view_width, view_height) = svg_view_box(root).unwrap_or_else(|| {
        (
            0.0,
            0.0,
            attr_f64(root, "width").unwrap_or(op.rect.width as f64),
            attr_f64(root, "height").unwrap_or(op.rect.height as f64),
        )
    });
    let scale_x = if view_width > 0.0 {
        op.rect.width as f64 / view_width
    } else {
        1.0
    };
    let scale_y = if view_height > 0.0 {
        op.rect.height as f64 / view_height
    } else {
        1.0
    };
    let transform = vello::kurbo::Affine::translate((op.rect.x as f64, op.rect.y as f64))
        * vello::kurbo::Affine::scale_non_uniform(scale_x, scale_y)
        * vello::kurbo::Affine::translate((-min_x, -min_y));
    for node in root.descendants().filter(|node| node.is_element()) {
        if node == root {
            continue;
        }
        draw_svg_node(scene, node, transform);
    }
    Ok(1)
}

fn draw_svg_node(
    scene: &mut vello::Scene,
    node: roxmltree::Node<'_, '_>,
    transform: vello::kurbo::Affine,
) {
    let tag = node.tag_name().name();
    match tag {
        "path" => {
            if let Some(data) = node.attribute("d") {
                if let Some(path) = parse_svg_path(data) {
                    paint_svg_shape(scene, node, transform, &path);
                }
            }
        }
        "rect" => {
            let x = attr_f64(node, "x").unwrap_or(0.0);
            let y = attr_f64(node, "y").unwrap_or(0.0);
            let width = attr_f64(node, "width").unwrap_or(0.0);
            let height = attr_f64(node, "height").unwrap_or(0.0);
            if width > 0.0 && height > 0.0 {
                let rect = vello::kurbo::Rect::new(x, y, x + width, y + height);
                paint_svg_shape(scene, node, transform, &rect);
            }
        }
        "circle" => {
            let cx = attr_f64(node, "cx").unwrap_or(0.0);
            let cy = attr_f64(node, "cy").unwrap_or(0.0);
            let r = attr_f64(node, "r").unwrap_or(0.0);
            if r > 0.0 {
                let circle = vello::kurbo::Circle::new((cx, cy), r);
                paint_svg_shape(scene, node, transform, &circle);
            }
        }
        "ellipse" => {
            let cx = attr_f64(node, "cx").unwrap_or(0.0);
            let cy = attr_f64(node, "cy").unwrap_or(0.0);
            let rx = attr_f64(node, "rx").unwrap_or(0.0);
            let ry = attr_f64(node, "ry").unwrap_or(0.0);
            if rx > 0.0 && ry > 0.0 {
                let ellipse = vello::kurbo::Ellipse::new((cx, cy), (rx, ry), 0.0);
                paint_svg_shape(scene, node, transform, &ellipse);
            }
        }
        "line" => {
            let x1 = attr_f64(node, "x1").unwrap_or(0.0);
            let y1 = attr_f64(node, "y1").unwrap_or(0.0);
            let x2 = attr_f64(node, "x2").unwrap_or(0.0);
            let y2 = attr_f64(node, "y2").unwrap_or(0.0);
            let line = vello::kurbo::Line::new((x1, y1), (x2, y2));
            stroke_svg_shape(scene, node, transform, &line);
        }
        "polyline" | "polygon" => {
            if let Some(points) = node.attribute("points") {
                if let Some(path) = parse_points(points, tag == "polygon") {
                    paint_svg_shape(scene, node, transform, &path);
                }
            }
        }
        _ => {}
    }
}

fn paint_svg_shape<S: vello::kurbo::Shape>(
    scene: &mut vello::Scene,
    node: roxmltree::Node<'_, '_>,
    transform: vello::kurbo::Affine,
    shape: &S,
) {
    let path = transform * shape.to_path(0.1);
    if let Some(fill) = svg_fill(node) {
        scene.fill(
            Fill::NonZero,
            vello::kurbo::Affine::IDENTITY,
            fill,
            None,
            &path,
        );
    }
    stroke_svg_path(scene, node, &path);
}

fn stroke_svg_shape<S: vello::kurbo::Shape>(
    scene: &mut vello::Scene,
    node: roxmltree::Node<'_, '_>,
    transform: vello::kurbo::Affine,
    shape: &S,
) {
    let path = transform * shape.to_path(0.1);
    stroke_svg_path(scene, node, &path);
}

fn stroke_svg_path(
    scene: &mut vello::Scene,
    node: roxmltree::Node<'_, '_>,
    path: &vello::kurbo::BezPath,
) {
    if let Some(stroke) = svg_stroke(node) {
        scene.stroke(
            &vello::kurbo::Stroke::new(svg_stroke_width(node)),
            vello::kurbo::Affine::IDENTITY,
            stroke,
            None,
            path,
        );
    }
}

fn parse_svg_path(data: &str) -> Option<vello::kurbo::BezPath> {
    let mut path = vello::kurbo::BezPath::new();
    let mut current = (0.0, 0.0);
    let mut start = (0.0, 0.0);
    for segment in svgtypes::PathParser::from(data) {
        match segment.ok()? {
            svgtypes::PathSegment::MoveTo { abs, x, y } => {
                current = absolute_point(abs, current, x, y);
                start = current;
                path.move_to(current);
            }
            svgtypes::PathSegment::LineTo { abs, x, y } => {
                current = absolute_point(abs, current, x, y);
                path.line_to(current);
            }
            svgtypes::PathSegment::HorizontalLineTo { abs, x } => {
                current.0 = if abs { x } else { current.0 + x };
                path.line_to(current);
            }
            svgtypes::PathSegment::VerticalLineTo { abs, y } => {
                current.1 = if abs { y } else { current.1 + y };
                path.line_to(current);
            }
            svgtypes::PathSegment::CurveTo {
                abs,
                x1,
                y1,
                x2,
                y2,
                x,
                y,
            } => {
                let p1 = absolute_point(abs, current, x1, y1);
                let p2 = absolute_point(abs, current, x2, y2);
                current = absolute_point(abs, current, x, y);
                path.curve_to(p1, p2, current);
            }
            svgtypes::PathSegment::Quadratic { abs, x1, y1, x, y } => {
                let p1 = absolute_point(abs, current, x1, y1);
                current = absolute_point(abs, current, x, y);
                path.quad_to(p1, current);
            }
            svgtypes::PathSegment::ClosePath { .. } => {
                path.close_path();
                current = start;
            }
            _ => {}
        }
    }
    Some(path)
}

fn parse_points(data: &str, close: bool) -> Option<vello::kurbo::BezPath> {
    let values: Vec<f64> = data
        .split(|c: char| c.is_ascii_whitespace() || c == ',')
        .filter(|part| !part.is_empty())
        .filter_map(|part| part.parse::<f64>().ok())
        .collect();
    if values.len() < 4 || values.len() % 2 != 0 {
        return None;
    }
    let mut path = vello::kurbo::BezPath::new();
    path.move_to((values[0], values[1]));
    for pair in values[2..].chunks(2) {
        path.line_to((pair[0], pair[1]));
    }
    if close {
        path.close_path();
    }
    Some(path)
}

fn absolute_point(abs: bool, current: (f64, f64), x: f64, y: f64) -> (f64, f64) {
    if abs {
        (x, y)
    } else {
        (current.0 + x, current.1 + y)
    }
}

fn svg_view_box(node: roxmltree::Node<'_, '_>) -> Option<(f64, f64, f64, f64)> {
    let values: Vec<f64> = node
        .attribute("viewBox")?
        .split(|c: char| c.is_ascii_whitespace() || c == ',')
        .filter(|part| !part.is_empty())
        .filter_map(|part| part.parse::<f64>().ok())
        .collect();
    if values.len() == 4 {
        Some((values[0], values[1], values[2], values[3]))
    } else {
        None
    }
}

fn svg_fill(node: roxmltree::Node<'_, '_>) -> Option<Color> {
    match node.attribute("fill") {
        Some("none") => None,
        Some(value) => parse_color(value),
        None => Some(Color::from_rgb8(0, 0, 0)),
    }
}

fn svg_stroke(node: roxmltree::Node<'_, '_>) -> Option<Color> {
    node.attribute("stroke").and_then(parse_color)
}

fn svg_stroke_width(node: roxmltree::Node<'_, '_>) -> f64 {
    attr_f64(node, "stroke-width").unwrap_or(1.0).max(0.1)
}

fn attr_f64(node: roxmltree::Node<'_, '_>, name: &str) -> Option<f64> {
    node.attribute(name)
        .and_then(|value| value.trim_end_matches("px").parse::<f64>().ok())
}

async fn create_device() -> Result<(wgpu::Device, wgpu::Queue)> {
    let instance = wgpu::Instance::new(wgpu::InstanceDescriptor::default());
    let adapter = instance
        .request_adapter(&wgpu::RequestAdapterOptions::default())
        .await
        .context("request wgpu adapter")?;
    adapter
        .request_device(
            &wgpu::DeviceDescriptor {
                label: Some("vugra-vello-native-device"),
                required_features: wgpu::Features::empty(),
                required_limits: adapter.limits(),
                memory_hints: wgpu::MemoryHints::Performance,
            },
            None,
        )
        .await
        .context("request wgpu device")
}

fn draw_control_fill(scene: &mut vello::Scene, op: &Op) {
    if !should_draw_element(op) {
        return;
    }
    let rect = shape(op);
    let color = apply_opacity(
        parse_color(&op.style.background_color).unwrap_or_else(|| match op.role.as_str() {
            "button" => Color::from_rgb8(238, 246, 255),
            "textbox" => Color::from_rgb8(255, 255, 255),
            "checkbox" => {
                if prop(op, "checked").is_some_and(|value| value == "true") {
                    Color::from_rgb8(37, 99, 235)
                } else {
                    Color::from_rgb8(255, 255, 255)
                }
            }
            "listitem" => Color::from_rgb8(255, 255, 255),
            _ => Color::from_rgb8(248, 250, 252),
        }),
        op.style.opacity,
    );
    scene.fill(
        Fill::NonZero,
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
            "button" => Color::from_rgb8(37, 99, 235),
            "textbox" | "checkbox" => Color::from_rgb8(148, 163, 184),
            "listitem" => Color::from_rgb8(203, 213, 225),
            _ => Color::from_rgb8(226, 232, 240),
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
            Color::from_rgb8(255, 255, 255),
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
    scene.push_layer(
        BlendMode::default(),
        1.0,
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
    if op.style.border_width_set && op.style.border_width <= 0.0 {
        return 0.0;
    }
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

fn line_height(op: &Op, font_size: f32) -> f32 {
    if op.style.line_height > 0.0 {
        return op.style.line_height;
    }
    if font_size >= 24.0 {
        font_size + 10.0
    } else if font_size >= 18.0 {
        font_size + 8.0
    } else {
        24.0
    }
}

fn text_layout_width(op: &Op, font_size: f32) -> f32 {
    let measured = op
        .text
        .lines()
        .map(|line| text_line_width(line, font_size))
        .fold(0.0, f32::max);
    op.rect.width.max(measured.ceil() + 2.0)
}

fn text_run_width(run: &GlyphRun, font_size: f32) -> f32 {
    text_line_width(&run.text, font_size)
}

fn text_line_width(text: &str, font_size: f32) -> f32 {
    text.chars()
        .map(|ch| {
            if ch.is_ascii() {
                font_size * 0.64
            } else {
                font_size
            }
        })
        .sum::<f32>()
        .ceil()
        + 2.0
}

fn measure_text(
    font_system: &mut FontSystem,
    text: &str,
    font_size: f32,
    line_height: f32,
) -> VugraTextMetrics {
    let font_size = if font_size > 0.0 { font_size } else { 16.0 };
    let line_height = if line_height > 0.0 {
        line_height
    } else if font_size >= 24.0 {
        font_size + 10.0
    } else if font_size >= 18.0 {
        font_size + 8.0
    } else {
        24.0
    };
    let attrs = Attrs::new().family(Family::SansSerif);
    let mut width = 0.0f32;
    let mut lines = 0usize;
    for line in text.split('\n') {
        let mut buffer = Buffer::new(font_system, Metrics::new(font_size, line_height));
        buffer.set_size(
            font_system,
            Some(text_line_width(line, font_size).max(1.0)),
            Some(line_height),
        );
        buffer.set_text(font_system, line, &attrs, Shaping::Advanced, None);
        buffer.shape_until_scroll(font_system, false);
        for run in buffer.layout_runs() {
            let run_width = run
                .glyphs
                .iter()
                .map(|glyph| glyph.x + glyph.w)
                .fold(0.0f32, f32::max);
            width = width.max(run_width);
        }
        lines += 1;
    }
    if lines == 0 {
        lines = 1;
    }
    VugraTextMetrics {
        width,
        height: line_height * lines as f32,
        baseline: (line_height - font_size) * 0.5 + font_size * 0.82,
    }
}

fn text_align(op: &Op) -> Option<Align> {
    match op.style.text_align.as_str() {
        "center" => Some(Align::Center),
        "right" => Some(Align::Right),
        "justify" => Some(Align::Justified),
        _ => None,
    }
}

fn text_color(op: &Op) -> Color {
    let color = if let Some(color) = parse_color(&op.style.color) {
        color
    } else if op.role == "button" {
        Color::from_rgb8(37, 99, 235)
    } else {
        Color::from_rgb8(15, 23, 42)
    };
    apply_opacity(color, op.style.opacity)
}

fn parse_color(value: &str) -> Option<Color> {
    let value = value.trim();
    if value.len() != 7 || !value.starts_with('#') {
        return None;
    }
    let rgb = u32::from_str_radix(&value[1..], 16).ok()?;
    Some(Color::from_rgb8(
        ((rgb >> 16) & 0xff) as u8,
        ((rgb >> 8) & 0xff) as u8,
        (rgb & 0xff) as u8,
    ))
}

fn apply_opacity(color: Color, opacity: f32) -> Color {
    if opacity <= 0.0 || opacity >= 1.0 {
        return color;
    }
    color.multiply_alpha(opacity)
}

fn cstring(value: impl Into<String>) -> CString {
    let value = value.into().replace('\0', "\\0");
    CString::new(value).unwrap_or_else(|_| CString::new("vello-native status unavailable").unwrap())
}

fn renderer_mut<'a>(renderer: *mut VugraNativeRenderer) -> Option<&'a mut VugraNativeRenderer> {
    if renderer.is_null() {
        None
    } else {
        Some(unsafe { &mut *renderer })
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_create(
    width: u32,
    height: u32,
) -> *mut VugraNativeRenderer {
    match VugraNativeRenderer::new(width, height) {
        Ok(renderer) => Box::into_raw(Box::new(renderer)),
        Err(_) => ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_destroy(renderer: *mut VugraNativeRenderer) {
    if !renderer.is_null() {
        unsafe {
            drop(Box::from_raw(renderer));
        }
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_resize(
    renderer: *mut VugraNativeRenderer,
    width: u32,
    height: u32,
) {
    if let Some(renderer) = renderer_mut(renderer) {
        renderer.resize(width, height);
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_render(
    renderer: *mut VugraNativeRenderer,
    json: *const u8,
    len: usize,
) -> c_int {
    let Some(renderer) = renderer_mut(renderer) else {
        return -1;
    };
    if json.is_null() && len > 0 {
        renderer.status = cstring("null JSON pointer");
        return -1;
    }
    let input = unsafe { std::slice::from_raw_parts(json, len) };
    match renderer.render(input) {
        Ok(()) => 0,
        Err(err) => {
            renderer.set_error(err);
            -1
        }
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_pixels(renderer: *const VugraNativeRenderer) -> *const u8 {
    if renderer.is_null() {
        return ptr::null();
    }
    unsafe { (*renderer).pixels.as_ptr() }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_pixels_len(renderer: *const VugraNativeRenderer) -> usize {
    if renderer.is_null() {
        return 0;
    }
    unsafe { (*renderer).pixels.len() }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_status(
    renderer: *const VugraNativeRenderer,
) -> *const c_char {
    if renderer.is_null() {
        return c"null renderer".as_ptr();
    }
    unsafe { (*renderer).status.as_ptr() }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_version() -> *const c_char {
    c"vello-native/0.1".as_ptr()
}

#[no_mangle]
pub extern "C" fn vuego_native_measure_text(
    text: *const c_char,
    font_size: f32,
    line_height: f32,
    out: *mut VugraTextMetrics,
) -> c_int {
    if text.is_null() || out.is_null() {
        return -1;
    }
    let Ok(text) = (unsafe { CStr::from_ptr(text) }).to_str() else {
        return -1;
    };
    let mut font_system = FontSystem::new();
    let metrics = measure_text(&mut font_system, text, font_size, line_height);
    unsafe {
        *out = metrics;
    }
    0
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_string_free(value: *mut c_char) {
    if !value.is_null() {
        unsafe {
            drop(CString::from_raw(value));
        }
    }
}

#[no_mangle]
pub extern "C" fn vuego_native_renderer_echo_status(value: *const c_char) -> *mut c_char {
    if value.is_null() {
        return cstring("null").into_raw();
    }
    let text = unsafe { CStr::from_ptr(value) }
        .to_string_lossy()
        .into_owned();
    cstring(text).into_raw()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn render_text_with_cosmic_text() {
        let mut renderer = VugraNativeRenderer::new(160, 80).expect("renderer");
        let ops = r#"[
          {"Kind":"fill-rect","Rect":{"x":0,"y":0,"width":160,"height":80},"Role":"button","Tag":"button","Style":{}},
          {"Kind":"text","Rect":{"x":8,"y":8,"width":144,"height":48},"Text":"中文 Todo fi","Role":"text","Tag":"p","Style":{"fontSize":18,"lineHeight":26}}
        ]"#;
        renderer.render(ops.as_bytes()).expect("render");
        assert_eq!(renderer.pixels.len(), 160 * 80 * 4);
        assert!(renderer.status.to_string_lossy().contains("glyphOps"));
    }

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

    #[test]
    fn render_svg_ops() {
        let mut renderer = VugraNativeRenderer::new(64, 40).expect("renderer");
        let ops = r##"[
		  {
		    "Kind":"svg",
		    "Rect":{"x":8,"y":4,"width":24,"height":24},
		    "SVG":"<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\"><circle cx=\"12\" cy=\"12\" r=\"10\" fill=\"#2563eb\"/><path d=\"M6 12h12\" stroke=\"#ffffff\" stroke-width=\"3\" fill=\"none\"/></svg>",
		    "Role":"image",
		    "Tag":"svg",
		    "Style":{}
		  }
		]"##;
        renderer.render(ops.as_bytes()).expect("render svg");
        assert_eq!(renderer.pixels.len(), 64 * 40 * 4);
        assert!(renderer.status.to_string_lossy().contains("\"svgOps\":1"));
    }

    #[test]
    fn text_layout_width_expands_underestimated_ascii_rects() {
        let op = Op {
            kind: "text".to_string(),
            rect: Rect {
                x: 0.0,
                y: 0.0,
                width: 24.0,
                height: 24.0,
            },
            text: "Add".to_string(),
            svg: String::new(),
            lines: None,
            glyphs: None,
            role: "text".to_string(),
            tag: "button".to_string(),
            props: None,
            style: Style {
                font_size: 16.0,
                ..Style::default()
            },
        };
        assert!(text_layout_width(&op, 16.0) > 24.0);
    }
}
