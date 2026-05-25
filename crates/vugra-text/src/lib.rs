//! Text measurement boundary for the Rust kernel path.

use font8x8::UnicodeFonts;
use std::fmt;
use std::path::{Path, PathBuf};
use std::sync::{Arc, OnceLock};

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct TextMetrics {
    pub width: f32,
    pub height: f32,
    pub baseline: f32,
}

pub trait TextMeasurer {
    fn measure(&self, text: &str, font_size: f32) -> TextMetrics;
}

pub trait GlyphMetricsProvider {
    fn glyph_metrics(&self, ch: char, font_size: f32) -> Option<GlyphMetrics>;
    fn line_metrics(&self, font_size: f32) -> Option<LineMetrics>;
    fn shape_text(&self, text: &str, font_size: f32) -> Option<Vec<ShapedGlyph>> {
        let _ = (text, font_size);
        None
    }
    fn shaped_glyph_pixels(&self, glyph: ShapedGlyph, font_size: f32) -> Option<Vec<GlyphPixel>> {
        let _ = (glyph, font_size);
        None
    }
    fn glyph_pixels(
        &self,
        ch: char,
        x: f32,
        y: f32,
        font_size: f32,
        baseline: f32,
    ) -> Option<Vec<GlyphPixel>> {
        let _ = (ch, x, y, font_size, baseline);
        None
    }
    fn glyph_source(&self, ch: char, font_size: f32) -> Option<GlyphSource> {
        self.glyph_metrics(ch, font_size)
            .map(|_| GlyphSource::LoadedFont)
    }
    fn glyph_font_key(&self, ch: char, font_size: f32) -> Option<FontKey> {
        let _ = (ch, font_size);
        None
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct FontKey(pub usize);

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GlyphMetrics {
    pub advance: f32,
    pub width: f32,
    pub height: f32,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct LineMetrics {
    pub height: f32,
    pub baseline: f32,
}

#[derive(Clone)]
pub struct LoadedFontTextMeasurer {
    data: Vec<u8>,
    face_index: u32,
    family: String,
    raster_font: Arc<OnceLock<Option<fontdue::Font>>>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FontLoadError {
    Io(String),
    Parse,
}

impl LoadedFontTextMeasurer {
    pub fn from_file(path: impl AsRef<Path>) -> Result<Self, FontLoadError> {
        Self::from_file_with_index(path, 0)
    }

    pub fn from_file_with_index(
        path: impl AsRef<Path>,
        face_index: u32,
    ) -> Result<Self, FontLoadError> {
        let path = path.as_ref();
        let data = std::fs::read(path)
            .map_err(|err| FontLoadError::Io(format!("{}: {err}", path.display())))?;
        Self::from_bytes(data, face_index)
    }

    pub fn from_bytes(data: Vec<u8>, face_index: u32) -> Result<Self, FontLoadError> {
        let face = ttf_parser::Face::parse(&data, face_index).map_err(|_| FontLoadError::Parse)?;
        let family = face
            .names()
            .into_iter()
            .find(|name| name.name_id == ttf_parser::name_id::FULL_NAME)
            .and_then(|name| name.to_string())
            .unwrap_or_else(|| "loaded-font".to_string());
        Ok(Self {
            data,
            face_index,
            family,
            raster_font: Arc::new(OnceLock::new()),
        })
    }

    pub fn family(&self) -> &str {
        &self.family
    }

    pub fn supports(&self, ch: char) -> bool {
        self.face().and_then(|face| face.glyph_index(ch)).is_some()
    }

    fn face(&self) -> Option<ttf_parser::Face<'_>> {
        ttf_parser::Face::parse(&self.data, self.face_index).ok()
    }

    fn raster_font(&self) -> Option<&fontdue::Font> {
        self.raster_font
            .get_or_init(|| {
                let settings = fontdue::FontSettings {
                    collection_index: self.face_index,
                    ..fontdue::FontSettings::default()
                };
                fontdue::Font::from_bytes(self.data.clone(), settings).ok()
            })
            .as_ref()
    }

    fn rasterize_shaped_glyph(
        &self,
        glyph: ShapedGlyph,
        font_size: f32,
    ) -> Option<Vec<GlyphPixel>> {
        let font = self.raster_font()?;
        let glyph_index = u16::try_from(glyph.glyph_id).ok()?;
        let (metrics, bitmap) = font.rasterize_indexed(glyph_index, font_size);
        let origin_x = glyph.x + metrics.xmin as f32;
        let origin_y = glyph.y + glyph.baseline - metrics.ymin as f32 - metrics.height as f32;
        Some(glyph_pixels_from_bitmap(
            origin_x,
            origin_y,
            metrics.width,
            metrics.height,
            &bitmap,
        ))
    }

    fn shape_loaded_text(
        &self,
        text: &str,
        font_size: f32,
        font_key: FontKey,
    ) -> Option<Vec<ShapedGlyph>> {
        let face = rustybuzz::Face::from_slice(&self.data, self.face_index)?;
        let units = face.units_per_em() as f32;
        if units <= 0.0 {
            return None;
        }
        let scale = font_size / units;
        let mut buffer = rustybuzz::UnicodeBuffer::new();
        for (byte_index, ch) in text.char_indices() {
            buffer.add(ch, byte_index as u32);
        }
        let shaped = rustybuzz::shape(&face, &[], buffer);
        let infos = shaped.glyph_infos();
        let positions = shaped.glyph_positions();
        if infos.len() != positions.len() {
            return None;
        }
        Some(
            infos
                .iter()
                .zip(positions.iter())
                .map(|(info, position)| ShapedGlyph {
                    glyph_id: info.glyph_id,
                    cluster: info.cluster,
                    x: 0.0,
                    y: 0.0,
                    baseline: 0.0,
                    x_advance: position.x_advance as f32 * scale,
                    y_advance: position.y_advance as f32 * scale,
                    x_offset: position.x_offset as f32 * scale,
                    y_offset: position.y_offset as f32 * scale,
                    line_index: 0,
                    font_key: Some(font_key),
                })
                .collect(),
        )
    }
}

impl fmt::Debug for LoadedFontTextMeasurer {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("LoadedFontTextMeasurer")
            .field("face_index", &self.face_index)
            .field("family", &self.family)
            .field("data_len", &self.data.len())
            .finish()
    }
}

impl PartialEq for LoadedFontTextMeasurer {
    fn eq(&self, other: &Self) -> bool {
        self.face_index == other.face_index
            && self.family == other.family
            && self.data == other.data
    }
}

impl GlyphMetricsProvider for LoadedFontTextMeasurer {
    fn glyph_metrics(&self, ch: char, font_size: f32) -> Option<GlyphMetrics> {
        let face = self.face()?;
        let units = face.units_per_em() as f32;
        if units <= 0.0 {
            return None;
        }
        let scale = font_size / units;
        let glyph = face.glyph_index(ch)?;
        let advance = face
            .glyph_hor_advance(glyph)
            .map(|advance| advance as f32 * scale)
            .unwrap_or(font_size * 0.6);
        let width = face
            .glyph_bounding_box(glyph)
            .map(|bbox| (bbox.x_max - bbox.x_min).max(0) as f32 * scale)
            .unwrap_or(advance);
        let height = face
            .glyph_bounding_box(glyph)
            .map(|bbox| (bbox.y_max - bbox.y_min).max(0) as f32 * scale)
            .unwrap_or(font_size);
        Some(GlyphMetrics {
            advance: advance.max(1.0),
            width: width.max(1.0),
            height: height.max(1.0),
        })
    }

    fn line_metrics(&self, font_size: f32) -> Option<LineMetrics> {
        let face = self.face()?;
        let units = face.units_per_em() as f32;
        if units <= 0.0 {
            return None;
        }
        let scale = font_size / units;
        let ascender = face.ascender() as f32 * scale;
        let descender = face.descender().abs() as f32 * scale;
        let line_gap = face.line_gap().max(0) as f32 * scale;
        Some(LineMetrics {
            height: (ascender + descender + line_gap).max(font_size),
            baseline: ascender.max(0.0),
        })
    }

    fn shape_text(&self, text: &str, font_size: f32) -> Option<Vec<ShapedGlyph>> {
        if text.is_empty() {
            return Some(Vec::new());
        }
        if !text
            .chars()
            .filter(|ch| !ch.is_control())
            .all(|ch| self.supports(ch))
        {
            return None;
        }
        self.shape_loaded_text(text, font_size, FontKey(0))
    }

    fn shaped_glyph_pixels(&self, glyph: ShapedGlyph, font_size: f32) -> Option<Vec<GlyphPixel>> {
        if !matches!(glyph.font_key, Some(FontKey(0))) {
            return None;
        }
        self.rasterize_shaped_glyph(glyph, font_size)
    }

    fn glyph_pixels(
        &self,
        ch: char,
        x: f32,
        y: f32,
        font_size: f32,
        baseline: f32,
    ) -> Option<Vec<GlyphPixel>> {
        let font = self.raster_font()?;
        let (metrics, bitmap) = font.rasterize(ch, font_size);
        if bitmap.is_empty() || metrics.width == 0 || metrics.height == 0 {
            return Some(Vec::new());
        }
        let origin_x = x + metrics.xmin as f32;
        let origin_y = y + baseline - metrics.ymin as f32 - metrics.height as f32;
        Some(glyph_pixels_from_bitmap(
            origin_x,
            origin_y,
            metrics.width,
            metrics.height,
            &bitmap,
        ))
    }

    fn glyph_source(&self, ch: char, font_size: f32) -> Option<GlyphSource> {
        self.glyph_metrics(ch, font_size)
            .map(|_| GlyphSource::LoadedFont)
    }

    fn glyph_font_key(&self, ch: char, font_size: f32) -> Option<FontKey> {
        self.glyph_metrics(ch, font_size).map(|_| FontKey(0))
    }
}

impl TextMeasurer for LoadedFontTextMeasurer {
    fn measure(&self, text: &str, font_size: f32) -> TextMetrics {
        let run = layout_text_run_with_provider(text, 0.0, 0.0, font_size, None, self);
        run.metrics
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct SystemFontTextMeasurer {
    search_paths: Vec<PathBuf>,
}

#[derive(Clone, Debug)]
pub struct SystemFontFallbackProvider {
    fonts: Vec<LoadedFontTextMeasurer>,
}

impl SystemFontTextMeasurer {
    pub fn new() -> Self {
        Self {
            search_paths: default_system_font_paths(),
        }
    }

    pub fn with_search_paths(search_paths: Vec<PathBuf>) -> Self {
        Self { search_paths }
    }

    pub fn load(&self) -> Option<LoadedFontTextMeasurer> {
        self.search_paths
            .iter()
            .find_map(|path| LoadedFontTextMeasurer::from_file(path).ok())
    }

    pub fn load_for_text(&self, text: &str) -> Option<LoadedFontTextMeasurer> {
        self.search_paths
            .iter()
            .filter_map(|path| LoadedFontTextMeasurer::from_file(path).ok())
            .find(|font| {
                text.chars()
                    .filter(|ch| !ch.is_control())
                    .all(|ch| font.supports(ch))
            })
            .or_else(|| self.load())
    }

    pub fn fallback_provider(&self) -> SystemFontFallbackProvider {
        let fonts = self
            .search_paths
            .iter()
            .filter_map(|path| LoadedFontTextMeasurer::from_file(path).ok())
            .collect();
        SystemFontFallbackProvider { fonts }
    }
}

impl TextMeasurer for SystemFontTextMeasurer {
    fn measure(&self, text: &str, font_size: f32) -> TextMetrics {
        self.load_for_text(text)
            .map(|font| font.measure(text, font_size))
            .unwrap_or_else(|| FixedTextMeasurer::default().measure(text, font_size))
    }
}

impl SystemFontFallbackProvider {
    pub fn is_empty(&self) -> bool {
        self.fonts.is_empty()
    }

    pub fn font_count(&self) -> usize {
        self.fonts.len()
    }

    fn font_for(&self, ch: char) -> Option<(usize, &LoadedFontTextMeasurer)> {
        self.fonts
            .iter()
            .enumerate()
            .find(|(_, font)| font.supports(ch))
    }
}

impl GlyphMetricsProvider for SystemFontFallbackProvider {
    fn glyph_metrics(&self, ch: char, font_size: f32) -> Option<GlyphMetrics> {
        self.font_for(ch)
            .and_then(|(_, font)| font.glyph_metrics(ch, font_size))
    }

    fn line_metrics(&self, font_size: f32) -> Option<LineMetrics> {
        self.fonts
            .first()
            .and_then(|font| font.line_metrics(font_size))
    }

    fn shape_text(&self, text: &str, font_size: f32) -> Option<Vec<ShapedGlyph>> {
        if text.is_empty() {
            return Some(Vec::new());
        }
        let mut shaped = Vec::new();
        let mut segment_start = 0usize;
        let mut segment_text = String::new();
        let mut segment_font: Option<(usize, &LoadedFontTextMeasurer)> = None;

        for (byte_index, ch) in text.char_indices() {
            if ch.is_control() {
                if !segment_text.is_empty() {
                    append_shaped_segment(
                        &mut shaped,
                        segment_start,
                        &segment_text,
                        font_size,
                        segment_font?,
                    )?;
                    segment_text.clear();
                    segment_font = None;
                }
                continue;
            }

            let font = self.font_for(ch)?;
            if segment_font
                .map(|(index, _)| index != font.0)
                .unwrap_or(false)
            {
                append_shaped_segment(
                    &mut shaped,
                    segment_start,
                    &segment_text,
                    font_size,
                    segment_font?,
                )?;
                segment_text.clear();
                segment_font = None;
            }

            if segment_text.is_empty() {
                segment_start = byte_index;
                segment_font = Some(font);
            }
            segment_text.push(ch);
        }

        if !segment_text.is_empty() {
            append_shaped_segment(
                &mut shaped,
                segment_start,
                &segment_text,
                font_size,
                segment_font?,
            )?;
        }

        Some(shaped)
    }

    fn shaped_glyph_pixels(&self, glyph: ShapedGlyph, font_size: f32) -> Option<Vec<GlyphPixel>> {
        let FontKey(font_index) = glyph.font_key?;
        self.fonts
            .get(font_index)
            .and_then(|font| font.rasterize_shaped_glyph(glyph, font_size))
    }

    fn glyph_pixels(
        &self,
        ch: char,
        x: f32,
        y: f32,
        font_size: f32,
        baseline: f32,
    ) -> Option<Vec<GlyphPixel>> {
        self.font_for(ch)
            .and_then(|(_, font)| font.glyph_pixels(ch, x, y, font_size, baseline))
    }

    fn glyph_source(&self, ch: char, font_size: f32) -> Option<GlyphSource> {
        self.glyph_metrics(ch, font_size)
            .map(|_| GlyphSource::LoadedFont)
    }

    fn glyph_font_key(&self, ch: char, font_size: f32) -> Option<FontKey> {
        self.glyph_metrics(ch, font_size)?;
        self.font_for(ch).map(|(index, _)| FontKey(index))
    }
}

fn append_shaped_segment(
    shaped: &mut Vec<ShapedGlyph>,
    segment_start: usize,
    segment_text: &str,
    font_size: f32,
    font: (usize, &LoadedFontTextMeasurer),
) -> Option<()> {
    let (font_index, font) = font;
    let mut segment = font.shape_loaded_text(segment_text, font_size, FontKey(font_index))?;
    for glyph in &mut segment {
        glyph.cluster += segment_start as u32;
    }
    shaped.append(&mut segment);
    Some(())
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FixedTextMeasurer {
    pub char_width: f32,
    pub line_height: f32,
}

impl Default for FixedTextMeasurer {
    fn default() -> Self {
        Self {
            char_width: 8.0,
            line_height: 20.0,
        }
    }
}

impl TextMeasurer for FixedTextMeasurer {
    fn measure(&self, text: &str, font_size: f32) -> TextMetrics {
        TextMetrics {
            width: text.chars().count() as f32 * self.char_width,
            height: self.line_height.max(font_size),
            baseline: self.line_height * 0.8,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GlyphPixel {
    pub x: f32,
    pub y: f32,
    pub width: f32,
    pub height: f32,
    pub alpha: f32,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BitmapGlyphRun {
    pub pixels: Vec<GlyphPixel>,
    pub glyphs: Vec<GlyphRunGlyph>,
    pub positioned_glyphs: Vec<PositionedGlyph>,
    pub lines: Vec<TextLineBox>,
    pub advance: f32,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GlyphRunGlyph {
    pub ch: char,
    pub source: GlyphSource,
}

#[derive(Clone, Debug, PartialEq)]
pub struct PositionedGlyph {
    pub ch: char,
    pub byte_index: usize,
    pub source: GlyphSource,
    pub font_key: Option<FontKey>,
    pub x: f32,
    pub y: f32,
    pub width: f32,
    pub height: f32,
    pub advance: f32,
    pub baseline: f32,
    pub line_index: usize,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ShapedGlyph {
    pub glyph_id: u32,
    pub cluster: u32,
    pub x: f32,
    pub y: f32,
    pub baseline: f32,
    pub x_advance: f32,
    pub y_advance: f32,
    pub x_offset: f32,
    pub y_offset: f32,
    pub line_index: usize,
    pub font_key: Option<FontKey>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GlyphSource {
    LoadedFont,
    Font8x8,
    FallbackBox,
}

#[derive(Clone, Debug, PartialEq)]
pub struct TextLineBox {
    pub text: String,
    pub x: f32,
    pub y: f32,
    pub width: f32,
    pub height: f32,
    pub baseline: f32,
}

#[derive(Clone, Debug, PartialEq)]
pub struct TextRun {
    pub text: String,
    pub x: f32,
    pub y: f32,
    pub font_size: f32,
    pub font_weight: u16,
    pub metrics: TextMetrics,
    pub glyphs: Vec<GlyphRunGlyph>,
    pub positioned_glyphs: Vec<PositionedGlyph>,
    pub shaped_glyphs: Vec<ShapedGlyph>,
    pub lines: Vec<TextLineBox>,
    pub pixels: Vec<GlyphPixel>,
}

impl TextRun {
    pub fn advance(&self) -> f32 {
        self.metrics.width
    }

    fn with_pixels(mut self, pixels: Vec<GlyphPixel>) -> Self {
        self.pixels = pixels;
        self
    }
}

pub fn layout_text_run(text: &str, x: f32, y: f32, font_size: f32) -> TextRun {
    layout_text_run_wrapped(text, x, y, font_size, None)
}

pub fn layout_text_run_wrapped(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    max_width: Option<f32>,
) -> TextRun {
    layout_text_run_wrapped_weighted(text, x, y, font_size, 400, max_width)
}

pub fn layout_text_run_wrapped_weighted(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    font_weight: u16,
    max_width: Option<f32>,
) -> TextRun {
    let glyphs = bitmap_glyph_run_wrapped(text, x, y, font_size, max_width);
    let pixels = embolden_pixels_if_needed(&glyphs.pixels, font_weight);
    text_run_from_glyphs(text, x, y, font_size, font_weight, glyphs, Vec::new()).with_pixels(pixels)
}

pub fn layout_text_run_with_provider(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    max_width: Option<f32>,
    provider: &impl GlyphMetricsProvider,
) -> TextRun {
    layout_text_run_with_provider_weighted(text, x, y, font_size, 400, max_width, provider)
}

pub fn layout_text_run_with_provider_weighted(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    font_weight: u16,
    max_width: Option<f32>,
    provider: &impl GlyphMetricsProvider,
) -> TextRun {
    let glyphs = glyph_run_wrapped(text, x, y, font_size, max_width, provider);
    let shaped_glyphs = position_shaped_glyphs(
        provider.shape_text(text, font_size).unwrap_or_default(),
        &glyphs.positioned_glyphs,
    );
    let pixels =
        shaped_pixels(provider, &shaped_glyphs, font_size).unwrap_or_else(|| glyphs.pixels.clone());
    let pixels = embolden_pixels_if_needed(&pixels, font_weight);
    text_run_from_glyphs(text, x, y, font_size, font_weight, glyphs, shaped_glyphs)
        .with_pixels(pixels)
}

fn text_run_from_glyphs(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    font_weight: u16,
    glyphs: BitmapGlyphRun,
    shaped_glyphs: Vec<ShapedGlyph>,
) -> TextRun {
    TextRun {
        text: text.to_string(),
        x,
        y,
        font_size,
        font_weight,
        metrics: TextMetrics {
            width: glyphs.advance,
            height: glyphs
                .lines
                .iter()
                .map(|line| line.y + line.height - y)
                .fold(0.0, f32::max),
            baseline: glyphs.lines.first().map_or(0.0, |line| line.baseline),
        },
        glyphs: glyphs.glyphs,
        positioned_glyphs: glyphs.positioned_glyphs,
        shaped_glyphs,
        lines: glyphs.lines,
        pixels: glyphs.pixels,
    }
}

fn embolden_pixels_if_needed(pixels: &[GlyphPixel], font_weight: u16) -> Vec<GlyphPixel> {
    if font_weight < 600 || pixels.is_empty() {
        return pixels.to_vec();
    }
    let mut bold = Vec::with_capacity(pixels.len() * 2);
    bold.extend_from_slice(pixels);
    bold.extend(pixels.iter().map(|pixel| GlyphPixel {
        x: pixel.x + 1.0,
        y: pixel.y,
        width: pixel.width,
        height: pixel.height,
        alpha: pixel.alpha,
    }));
    bold
}

fn position_shaped_glyphs(
    mut shaped_glyphs: Vec<ShapedGlyph>,
    positioned_glyphs: &[PositionedGlyph],
) -> Vec<ShapedGlyph> {
    if shaped_glyphs.is_empty() || positioned_glyphs.is_empty() {
        return shaped_glyphs;
    }

    for glyph in &mut shaped_glyphs {
        if let Some(positioned) = positioned_glyphs
            .iter()
            .find(|positioned| positioned.byte_index == glyph.cluster as usize)
        {
            glyph.x = positioned.x + glyph.x_offset;
            glyph.y = positioned.y + glyph.y_offset;
            glyph.baseline = positioned.baseline;
            glyph.line_index = positioned.line_index;
        }
    }

    shaped_glyphs
}

fn shaped_pixels(
    provider: &impl GlyphMetricsProvider,
    shaped_glyphs: &[ShapedGlyph],
    font_size: f32,
) -> Option<Vec<GlyphPixel>> {
    if shaped_glyphs.is_empty() {
        return None;
    }
    let mut pixels = Vec::new();
    for glyph in shaped_glyphs {
        let mut glyph_pixels = provider.shaped_glyph_pixels(*glyph, font_size)?;
        pixels.append(&mut glyph_pixels);
    }
    Some(pixels)
}

pub fn bitmap_glyph_run(text: &str, x: f32, y: f32) -> BitmapGlyphRun {
    bitmap_glyph_run_wrapped(text, x, y, 13.0, None)
}

pub fn bitmap_glyph_run_wrapped(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    max_width: Option<f32>,
) -> BitmapGlyphRun {
    glyph_run_wrapped(
        text,
        x,
        y,
        font_size,
        max_width,
        &BitmapGlyphMetricsProvider,
    )
}

fn glyph_run_wrapped(
    text: &str,
    x: f32,
    y: f32,
    font_size: f32,
    max_width: Option<f32>,
    provider: &impl GlyphMetricsProvider,
) -> BitmapGlyphRun {
    let mut pixels = Vec::new();
    let mut glyphs = Vec::new();
    let mut positioned_glyphs = Vec::new();
    let mut lines = Vec::new();
    let mut cursor_x = x;
    let mut cursor_y = y;
    let line_metrics = provider
        .line_metrics(font_size)
        .unwrap_or_else(|| LineMetrics {
            height: line_height_for(font_size),
            baseline: baseline_for(font_size),
        });
    let line_height = line_metrics.height;
    let baseline = line_metrics.baseline;
    let mut line_text = String::new();
    let mut line_start_x = x;
    let mut line_start_y = y;
    let mut line_width: f32 = 0.0;
    let mut max_line_width: f32 = 0.0;
    let mut line_index = 0usize;

    for (byte_index, ch) in text.char_indices() {
        if ch == '\n' {
            push_text_line(
                &mut lines,
                &mut line_text,
                line_start_x,
                line_start_y,
                line_width,
                line_height,
                baseline,
            );
            max_line_width = max_line_width.max(line_width);
            cursor_x = x;
            cursor_y += line_height;
            line_start_x = x;
            line_start_y = cursor_y;
            line_width = 0.0;
            line_index += 1;
            continue;
        }

        let fallback_source = glyph_source_for(ch);
        let provided_metrics = provider.glyph_metrics(ch, font_size);
        let source = provider
            .glyph_source(ch, font_size)
            .filter(|_| provided_metrics.is_some())
            .unwrap_or(fallback_source);
        let font_key = provider.glyph_font_key(ch, font_size);
        let metrics = provided_metrics
            .unwrap_or_else(|| bitmap_glyph_metrics(ch, fallback_source, font_size));
        let advance = metrics.advance;
        let glyph_width = metrics.width;
        let glyph_height = metrics.height;

        if should_wrap_line(max_width, line_width, advance) {
            push_text_line(
                &mut lines,
                &mut line_text,
                line_start_x,
                line_start_y,
                line_width,
                line_height,
                baseline,
            );
            max_line_width = max_line_width.max(line_width);
            cursor_x = x;
            cursor_y += line_height;
            line_start_x = x;
            line_start_y = cursor_y;
            line_width = 0.0;
            line_index += 1;
        }

        if source == GlyphSource::LoadedFont {
            if let Some(mut glyph_pixels) =
                provider.glyph_pixels(ch, cursor_x, cursor_y, font_size, baseline)
            {
                pixels.append(&mut glyph_pixels);
            }
        } else {
            push_bitmap_glyph_pixels(&mut pixels, cursor_x, cursor_y, ch, source);
        }
        glyphs.push(GlyphRunGlyph { ch, source });
        positioned_glyphs.push(PositionedGlyph {
            ch,
            byte_index,
            source,
            font_key,
            x: cursor_x,
            y: cursor_y,
            width: glyph_width,
            height: glyph_height,
            advance,
            baseline,
            line_index,
        });
        line_text.push(ch);
        cursor_x += advance;
        line_width += advance;
    }
    push_text_line(
        &mut lines,
        &mut line_text,
        line_start_x,
        line_start_y,
        line_width,
        line_height,
        baseline,
    );
    max_line_width = max_line_width.max(line_width);

    BitmapGlyphRun {
        pixels,
        glyphs,
        positioned_glyphs,
        lines,
        advance: max_line_width,
    }
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
struct BitmapGlyphMetricsProvider;

impl GlyphMetricsProvider for BitmapGlyphMetricsProvider {
    fn glyph_metrics(&self, ch: char, font_size: f32) -> Option<GlyphMetrics> {
        let source = glyph_source_for(ch);
        Some(bitmap_glyph_metrics(ch, source, font_size))
    }

    fn line_metrics(&self, font_size: f32) -> Option<LineMetrics> {
        Some(LineMetrics {
            height: line_height_for(font_size),
            baseline: baseline_for(font_size),
        })
    }

    fn glyph_source(&self, ch: char, _font_size: f32) -> Option<GlyphSource> {
        Some(glyph_source_for(ch))
    }
}

fn should_wrap_line(max_width: Option<f32>, line_width: f32, next_advance: f32) -> bool {
    matches!(max_width, Some(width) if width > 0.0 && line_width > 0.0 && line_width + next_advance > width)
}

fn line_height_for(font_size: f32) -> f32 {
    font_size.max(8.0)
}

fn baseline_for(font_size: f32) -> f32 {
    7.0_f32.min(line_height_for(font_size))
}

fn push_text_line(
    lines: &mut Vec<TextLineBox>,
    line_text: &mut String,
    x: f32,
    y: f32,
    width: f32,
    height: f32,
    baseline: f32,
) {
    lines.push(TextLineBox {
        text: std::mem::take(line_text),
        x,
        y,
        width,
        height,
        baseline,
    });
}

fn glyph_source_for(ch: char) -> GlyphSource {
    if font8x8::BASIC_FONTS.get(ch).is_some()
        || font8x8::BASIC_FONTS.get(ch.to_ascii_uppercase()).is_some()
    {
        GlyphSource::Font8x8
    } else {
        GlyphSource::FallbackBox
    }
}

fn glyph_advance_for(ch: char, source: GlyphSource, font_size: f32) -> f32 {
    let base = match source {
        GlyphSource::LoadedFont => 9.0,
        GlyphSource::Font8x8 if ch == ' ' => 4.0,
        GlyphSource::Font8x8 => 9.0,
        GlyphSource::FallbackBox => 10.0,
    };
    (base * font_scale(font_size)).max(1.0)
}

fn bitmap_glyph_metrics(ch: char, source: GlyphSource, font_size: f32) -> GlyphMetrics {
    GlyphMetrics {
        advance: glyph_advance_for(ch, source, font_size),
        width: glyph_width_for(source, font_size),
        height: glyph_height_for(font_size),
    }
}

fn glyph_width_for(source: GlyphSource, font_size: f32) -> f32 {
    let base = match source {
        GlyphSource::LoadedFont => 8.0,
        GlyphSource::Font8x8 => 8.0,
        GlyphSource::FallbackBox => 9.0,
    };
    (base * font_scale(font_size)).max(1.0)
}

fn glyph_height_for(font_size: f32) -> f32 {
    (8.0 * font_scale(font_size))
        .max(1.0)
        .min(line_height_for(font_size))
}

fn font_scale(font_size: f32) -> f32 {
    (font_size / 13.0).max(0.5)
}

fn push_bitmap_glyph_pixels(
    pixels: &mut Vec<GlyphPixel>,
    x: f32,
    y: f32,
    ch: char,
    source: GlyphSource,
) {
    if source == GlyphSource::LoadedFont {
        return;
    }
    let Some(glyph) = font8x8::BASIC_FONTS
        .get(ch)
        .or_else(|| font8x8::BASIC_FONTS.get(ch.to_ascii_uppercase()))
    else {
        push_fallback_box_pixels(pixels, x, y);
        return;
    };
    for (row, bits) in glyph.iter().enumerate() {
        for col in 0..8 {
            if (bits >> col) & 1 == 1 {
                pixels.push(GlyphPixel {
                    x: x + col as f32,
                    y: y + row as f32,
                    width: 1.0,
                    height: 1.0,
                    alpha: 1.0,
                });
            }
        }
    }
    if source == GlyphSource::FallbackBox {
        push_fallback_box_pixels(pixels, x, y);
    }
}

fn glyph_pixels_from_bitmap(
    origin_x: f32,
    origin_y: f32,
    width: usize,
    height: usize,
    bitmap: &[u8],
) -> Vec<GlyphPixel> {
    if bitmap.is_empty() || width == 0 || height == 0 {
        return Vec::new();
    }
    let mut pixels = Vec::new();
    for row in 0..height {
        for col in 0..width {
            let alpha = bitmap[row * width + col];
            if alpha == 0 {
                continue;
            }
            pixels.push(GlyphPixel {
                x: origin_x + col as f32,
                y: origin_y + row as f32,
                width: 1.0,
                height: 1.0,
                alpha: (alpha as f32 / 255.0).clamp(1.0 / 255.0, 1.0),
            });
        }
    }
    pixels
}

fn push_fallback_box_pixels(pixels: &mut Vec<GlyphPixel>, x: f32, y: f32) {
    for row in 0..8 {
        for col in 0..8 {
            let border = row == 0 || row == 7 || col == 0 || col == 7;
            let diagonal = row == col || row + col == 7;
            if border || diagonal {
                pixels.push(GlyphPixel {
                    x: x + col as f32,
                    y: y + row as f32,
                    width: 1.0,
                    height: 1.0,
                    alpha: 1.0,
                });
            }
        }
    }
}

fn default_system_font_paths() -> Vec<PathBuf> {
    [
        "/System/Library/Fonts/SFNS.ttf",
        "/System/Library/Fonts/SFNSMono.ttf",
        "/System/Library/Fonts/HelveticaNeue.ttc",
        "/System/Library/Fonts/Hiragino Sans GB.ttc",
        "/System/Library/Fonts/STHeiti Medium.ttc",
        "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
    ]
    .into_iter()
    .map(PathBuf::from)
    .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixed_text_measurer_counts_chars_not_bytes() {
        let metrics = FixedTextMeasurer::default().measure("文ab", 13.0);
        assert_eq!(metrics.width, 24.0);
        assert_eq!(metrics.height, 20.0);
    }

    #[test]
    fn loaded_system_font_measures_real_glyph_advances_when_available() {
        let Some(font) = SystemFontTextMeasurer::new().load_for_text("Finder 文") else {
            eprintln!("no system font available for loaded font measurement test");
            return;
        };
        let run = layout_text_run_with_provider("Finder 文", 10.0, 12.0, 16.0, None, &font);

        assert!(run.metrics.width > 40.0, "{run:?}");
        assert!(run.metrics.height >= 16.0, "{run:?}");
        assert!(run.metrics.baseline > 8.0, "{run:?}");
        assert!(
            run.pixels.len() > 20,
            "loaded font geometry should include raster pixels: {run:?}"
        );
        assert!(run
            .positioned_glyphs
            .iter()
            .any(|glyph| glyph.source == GlyphSource::LoadedFont));
        assert_eq!(
            run.positioned_glyphs.last().map(|glyph| glyph.ch),
            Some('文')
        );
        assert!(run.pixels.iter().any(|pixel| pixel.alpha < 1.0));
        assert!(run.pixels.iter().all(|pixel| pixel.width == 1.0));
    }

    #[test]
    fn loaded_system_font_attaches_shaped_glyph_metadata() {
        let Some(font) = SystemFontTextMeasurer::new().load_for_text("office") else {
            eprintln!("no system font available for shaping metadata test");
            return;
        };
        let run = layout_text_run_with_provider("office", 0.0, 0.0, 16.0, None, &font);

        assert!(!run.shaped_glyphs.is_empty(), "{run:?}");
        assert!(run
            .shaped_glyphs
            .iter()
            .all(|glyph| glyph.font_key == Some(FontKey(0))));
        assert!(run.shaped_glyphs.iter().any(|glyph| glyph.x_advance > 0.0));
        assert!(run.shaped_glyphs.iter().all(|glyph| {
            let cluster = glyph.cluster as usize;
            cluster < run.text.len() && run.text.is_char_boundary(cluster)
        }));
        assert!(run
            .shaped_glyphs
            .iter()
            .any(|glyph| glyph.x >= 0.0 && glyph.baseline > 0.0));
        let shaped_pixels = shaped_pixels(&font, &run.shaped_glyphs, run.font_size)
            .expect("loaded shaped glyphs should rasterize");
        assert_eq!(run.pixels, shaped_pixels);
    }

    #[test]
    fn loaded_font_provider_falls_back_per_missing_glyph() {
        let Some(font) = SystemFontTextMeasurer::with_search_paths(vec![PathBuf::from(
            "/System/Library/Fonts/SFNSMono.ttf",
        )])
        .load() else {
            eprintln!("SFNSMono unavailable for fallback test");
            return;
        };
        let run = layout_text_run_with_provider("\u{10ffff}", 0.0, 0.0, 13.0, None, &font);

        assert_eq!(run.positioned_glyphs.len(), 1);
        assert_eq!(run.positioned_glyphs[0].source, GlyphSource::FallbackBox);
        assert!(!run.pixels.is_empty());
    }

    #[test]
    fn system_font_fallback_provider_selects_loaded_font_per_glyph() {
        let provider = SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for fallback provider test");
            return;
        }
        let run = layout_text_run_with_provider("Finder 文", 10.0, 12.0, 16.0, None, &provider);

        assert!(run.pixels.len() > 20, "{run:?}");
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.source == GlyphSource::LoadedFont));
        assert!(run
            .positioned_glyphs
            .iter()
            .all(|glyph| glyph.font_key.is_some()));
        let unique_fonts = run
            .positioned_glyphs
            .iter()
            .filter_map(|glyph| glyph.font_key)
            .collect::<std::collections::BTreeSet<_>>();
        assert!(
            !unique_fonts.is_empty(),
            "expected at least one loaded font key: {run:?}"
        );
    }

    #[test]
    fn system_font_fallback_provider_shapes_segments_with_original_clusters() {
        let provider = SystemFontTextMeasurer::new().fallback_provider();
        if provider.is_empty() {
            eprintln!("no system fonts available for fallback provider shaping test");
            return;
        }
        let run = layout_text_run_with_provider("A文", 0.0, 0.0, 16.0, None, &provider);
        if run.shaped_glyphs.is_empty() {
            eprintln!("system fonts could not shape fallback provider test text");
            return;
        }

        assert!(run.shaped_glyphs.iter().all(|glyph| {
            let cluster = glyph.cluster as usize;
            cluster < run.text.len() && run.text.is_char_boundary(cluster)
        }));
        assert!(run
            .shaped_glyphs
            .iter()
            .all(|glyph| glyph.font_key.is_some()));
        assert!(run.shaped_glyphs.iter().any(|glyph| glyph.cluster == 0));
        assert!(run.shaped_glyphs.iter().any(|glyph| glyph.cluster == 1));
    }

    #[test]
    fn bitmap_glyph_run_emits_pixels_for_ascii_text() {
        let run = bitmap_glyph_run("Fi", 10.0, 12.0);
        assert!(run.pixels.len() > 8);
        assert_eq!(
            run.glyphs,
            vec![
                GlyphRunGlyph {
                    ch: 'F',
                    source: GlyphSource::Font8x8
                },
                GlyphRunGlyph {
                    ch: 'i',
                    source: GlyphSource::Font8x8
                }
            ]
        );
        assert_eq!(run.advance, 18.0);
        assert_eq!(
            run.lines,
            vec![TextLineBox {
                text: "Fi".to_string(),
                x: 10.0,
                y: 12.0,
                width: 18.0,
                height: 13.0,
                baseline: 7.0,
            }]
        );
        assert!(run.pixels.iter().all(|pixel| pixel.x >= 10.0));
        assert!(run.pixels.iter().all(|pixel| pixel.y >= 12.0));
        assert_eq!(
            run.positioned_glyphs,
            vec![
                PositionedGlyph {
                    ch: 'F',
                    byte_index: 0,
                    source: GlyphSource::Font8x8,
                    font_key: None,
                    x: 10.0,
                    y: 12.0,
                    width: 8.0,
                    height: 8.0,
                    advance: 9.0,
                    baseline: 7.0,
                    line_index: 0,
                },
                PositionedGlyph {
                    ch: 'i',
                    byte_index: 1,
                    source: GlyphSource::Font8x8,
                    font_key: None,
                    x: 19.0,
                    y: 12.0,
                    width: 8.0,
                    height: 8.0,
                    advance: 9.0,
                    baseline: 7.0,
                    line_index: 0,
                },
            ]
        );
    }

    #[test]
    fn bitmap_glyph_run_emits_visible_fallback_for_cjk_without_question_mark_substitution() {
        let run = bitmap_glyph_run("文a", 10.0, 12.0);
        assert_eq!(run.advance, 19.0);
        assert_eq!(
            run.glyphs,
            vec![
                GlyphRunGlyph {
                    ch: '文',
                    source: GlyphSource::FallbackBox
                },
                GlyphRunGlyph {
                    ch: 'a',
                    source: GlyphSource::Font8x8
                }
            ]
        );
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.x >= 10.0 && pixel.x < 18.0));
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.x >= 19.0 && pixel.x < 27.0));
        assert_eq!(run.positioned_glyphs[0].advance, 10.0);
        assert_eq!(run.positioned_glyphs[0].width, 9.0);
        assert_eq!(run.positioned_glyphs[1].x, 20.0);
    }

    #[test]
    fn text_run_keeps_measurement_and_paint_data_together() {
        let run = layout_text_run("Fi", 10.0, 12.0, 13.0);
        assert_eq!(run.text, "Fi");
        assert_eq!(run.font_weight, 400);
        assert_eq!(run.metrics.width, run.advance());
        assert_eq!(run.metrics.width, 18.0);
        assert_eq!(run.metrics.height, 13.0);
        assert_eq!(run.glyphs.len(), 2);
        assert_eq!(run.positioned_glyphs.len(), 2);
        assert_eq!(run.lines.len(), 1);
        assert!(run.pixels.len() > 8);
    }

    #[test]
    fn weighted_text_run_emboldens_pixels_without_changing_measurement() {
        let regular = layout_text_run_wrapped_weighted("Header", 10.0, 12.0, 13.0, 400, None);
        let bold = layout_text_run_wrapped_weighted("Header", 10.0, 12.0, 13.0, 600, None);
        assert_eq!(regular.font_weight, 400);
        assert_eq!(bold.font_weight, 600);
        assert_eq!(regular.metrics, bold.metrics);
        assert_eq!(regular.positioned_glyphs, bold.positioned_glyphs);
        assert!(bold.pixels.len() > regular.pixels.len());
        assert!(bold.pixels.iter().any(|pixel| {
            regular
                .pixels
                .iter()
                .any(|base| pixel.x == base.x + 1.0 && pixel.y == base.y)
        }));
    }

    #[test]
    fn text_run_wraps_into_line_boxes_and_moves_glyph_pixels() {
        let run = layout_text_run_wrapped("abcd", 10.0, 12.0, 13.0, Some(18.0));
        assert_eq!(
            run.lines,
            vec![
                TextLineBox {
                    text: "ab".to_string(),
                    x: 10.0,
                    y: 12.0,
                    width: 18.0,
                    height: 13.0,
                    baseline: 7.0,
                },
                TextLineBox {
                    text: "cd".to_string(),
                    x: 10.0,
                    y: 25.0,
                    width: 18.0,
                    height: 13.0,
                    baseline: 7.0,
                },
            ]
        );
        assert_eq!(run.metrics.width, 18.0);
        assert_eq!(run.metrics.height, 26.0);
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.y >= 12.0 && pixel.y < 20.0));
        assert!(run
            .pixels
            .iter()
            .any(|pixel| pixel.y >= 25.0 && pixel.y < 33.0));
        assert_eq!(run.positioned_glyphs[0].line_index, 0);
        assert_eq!(run.positioned_glyphs[1].line_index, 0);
        assert_eq!(run.positioned_glyphs[2].line_index, 1);
        assert_eq!(run.positioned_glyphs[2].x, 10.0);
        assert_eq!(run.positioned_glyphs[2].y, 25.0);
    }

    #[test]
    fn text_run_uses_glyph_advance_for_wrapping_and_scaled_geometry() {
        let run = layout_text_run_wrapped("a 文", 4.0, 6.0, 26.0, Some(26.0));
        assert_eq!(run.lines.len(), 2);
        assert_eq!(run.lines[0].text, "a ");
        assert_eq!(run.lines[0].width, 26.0);
        assert_eq!(run.lines[1].text, "文");
        assert_eq!(run.lines[1].width, 20.0);
        assert_eq!(run.positioned_glyphs[0].advance, 18.0);
        assert_eq!(run.positioned_glyphs[1].advance, 8.0);
        assert_eq!(run.positioned_glyphs[2].advance, 20.0);
        assert_eq!(run.positioned_glyphs[2].line_index, 1);
        assert_eq!(run.positioned_glyphs[2].x, 4.0);
        assert_eq!(run.positioned_glyphs[2].y, 32.0);
    }
}
