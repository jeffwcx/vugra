use anyhow::{anyhow, Context, Result};
use lightningcss::properties::Property;
use lightningcss::rules::CssRule;
use lightningcss::stylesheet::{ParserOptions, PrinterOptions, StyleSheet};
use lightningcss::traits::ToCss;
use parley::{Alignment, AlignmentOptions, FontContext, LayoutContext, StyleProperty};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};
use taffy::prelude::*;

#[derive(Debug, Deserialize)]
pub struct Input {
    css: String,
    root: NodeInput,
    viewport: Viewport,
}

#[derive(Debug, Deserialize)]
pub struct Viewport {
    width: f32,
    #[serde(default)]
    height: Option<f32>,
}

#[derive(Debug, Deserialize)]
pub struct NodeInput {
    id: String,
    tag: String,
    #[serde(default)]
    class: String,
    #[serde(default)]
    text: String,
    #[serde(default)]
    children: Vec<NodeInput>,
}

#[derive(Debug, Default, Clone)]
struct CssStyle {
    display: String,
    flex_direction: String,
    flex_wrap: String,
    align_items: String,
    justify_content: String,
    gap: f32,
    padding: f32,
    margin: f32,
    width: Option<CssLength>,
    height: Option<CssLength>,
    min_height: Option<CssLength>,
    font_size: f32,
    line_height: f32,
    color: String,
    background_color: String,
    border_width: f32,
    border_color: String,
    border_radius: f32,
    overflow: String,
}

#[derive(Debug, Clone, Copy)]
enum CssLength {
    Px(f32),
    Percent(f32),
}

#[derive(Debug, Serialize)]
pub struct Output {
    boxes: Vec<BoxOutput>,
}

#[derive(Debug, Serialize)]
pub struct BoxOutput {
    id: String,
    tag: String,
    text: String,
    x: f32,
    y: f32,
    width: f32,
    height: f32,
    style: StyleOutput,
}

#[derive(Debug, Serialize)]
pub struct StyleOutput {
    display: String,
    font_size: f32,
    line_height: f32,
    color: String,
    background_color: String,
    border_width: f32,
    border_color: String,
    border_radius: f32,
    overflow: String,
}

#[derive(Debug)]
struct BuiltNode {
    id: String,
    tag: String,
    text: String,
    style: CssStyle,
    node: NodeId,
    children: Vec<BuiltNode>,
}

struct TextMeasure {
    text: String,
    font_size: f32,
    line_height: f32,
}

pub fn compute(input: Input) -> Result<Output> {
    let rules = cached_css_rules(&input.css)?;
    let mut taffy: TaffyTree<TextMeasure> = TaffyTree::new();
    let root = build_node(&mut taffy, &rules, &input.root)?;
    let mut text = TextMeasurer::new();
    taffy
        .compute_layout_with_measure(
            root.node,
            Size {
                width: AvailableSpace::Definite(input.viewport.width),
                height: input
                    .viewport
                    .height
                    .map(AvailableSpace::Definite)
                    .unwrap_or(AvailableSpace::MaxContent),
            },
            |known, available, _node_id, context, _style| {
                measure_text(context, known, available, &mut text)
            },
        )
        .context("compute Taffy layout")?;

    let mut boxes = Vec::new();
    collect_boxes(&taffy, &root, 0.0, 0.0, &mut boxes)?;
    Ok(Output { boxes })
}

fn cached_css_rules(css: &str) -> Result<Vec<(String, CssStyle)>> {
    static CACHE: OnceLock<Mutex<HashMap<String, Vec<(String, CssStyle)>>>> = OnceLock::new();
    let cache = CACHE.get_or_init(|| Mutex::new(HashMap::new()));
    {
        let guard = cache
            .lock()
            .map_err(|_| anyhow!("CSS rule cache poisoned"))?;
        if let Some(rules) = guard.get(css) {
            return Ok(rules.clone());
        }
    }
    let rules = parse_css(css)?;
    let mut guard = cache
        .lock()
        .map_err(|_| anyhow!("CSS rule cache poisoned"))?;
    let entry = guard
        .entry(css.to_string())
        .or_insert_with(|| rules.clone());
    Ok(entry.clone())
}

fn parse_css(css: &str) -> Result<Vec<(String, CssStyle)>> {
    let sheet = StyleSheet::parse(css, ParserOptions::default())
        .map_err(|err| anyhow!("parse CSS with Lightning CSS: {err:?}"))?;
    let mut rules = Vec::new();
    for rule in &sheet.rules.0 {
        let CssRule::Style(rule) = rule else {
            continue;
        };
        let selector = rule
            .selectors
            .to_css_string(PrinterOptions::default())
            .map_err(|err| anyhow!("serialize Lightning CSS selector: {err:?}"))?;
        let Some(class_name) = selector.strip_prefix('.') else {
            continue;
        };
        if class_name.contains(' ') || class_name.contains(',') || class_name.is_empty() {
            continue;
        }
        let mut style = CssStyle::default();
        for decl in rule.declarations.iter() {
            let name = decl.0.property_id().name().to_string();
            let value = property_value_to_css(decl.0)?;
            apply_decl(&mut style, name.trim(), value.trim());
        }
        rules.push((class_name.to_string(), style));
    }
    Ok(rules)
}

fn property_value_to_css(property: &Property<'_>) -> Result<String> {
    let full = property
        .to_css_string(false, PrinterOptions::default())
        .map_err(|err| anyhow!("serialize Lightning CSS property: {err:?}"))?;
    let Some((_, value)) = full.split_once(':') else {
        return Ok(full);
    };
    Ok(value.trim().to_string())
}

fn apply_decl(style: &mut CssStyle, name: &str, value: &str) {
    match name {
        "display" => style.display = value.to_string(),
        "flex-direction" => style.flex_direction = value.to_string(),
        "flex-wrap" => style.flex_wrap = value.to_string(),
        "align-items" => style.align_items = value.to_string(),
        "justify-content" => style.justify_content = value.to_string(),
        "gap" => style.gap = px(value),
        "padding" => style.padding = px(value),
        "margin" => style.margin = px(value),
        "width" => style.width = length(value),
        "height" => style.height = length(value),
        "min-height" => style.min_height = length(value),
        "font-size" => style.font_size = px(value),
        "line-height" => style.line_height = px(value),
        "color" => style.color = value.to_string(),
        "background-color" => style.background_color = value.to_string(),
        "border-width" => style.border_width = px(value),
        "border-color" => style.border_color = value.to_string(),
        "border-radius" => style.border_radius = px(value),
        "overflow" => style.overflow = value.to_string(),
        "border" => apply_border(style, value),
        _ => {}
    }
}

fn apply_border(style: &mut CssStyle, value: &str) {
    for part in value.split_whitespace() {
        if part.ends_with("px") {
            style.border_width = px(part);
        } else if part.starts_with('#') {
            style.border_color = part.to_string();
        }
    }
}

fn px(value: &str) -> f32 {
    value.trim_end_matches("px").parse().unwrap_or(0.0)
}

fn length(value: &str) -> Option<CssLength> {
    let trimmed = value.trim();
    if let Some(raw) = trimmed.strip_suffix('%') {
        return raw
            .parse::<f32>()
            .ok()
            .map(|percent| CssLength::Percent(percent / 100.0));
    }
    if let Some(raw) = trimmed.strip_suffix("px") {
        return raw.parse::<f32>().ok().map(CssLength::Px);
    }
    trimmed.parse::<f32>().ok().map(CssLength::Px)
}

fn build_node(
    taffy: &mut TaffyTree<TextMeasure>,
    rules: &[(String, CssStyle)],
    input: &NodeInput,
) -> Result<BuiltNode> {
    let style = compute_style(rules, &input.class, &input.tag);
    build_node_with_parent_style(taffy, rules, input, &style)
}

fn build_node_with_parent_style(
    taffy: &mut TaffyTree<TextMeasure>,
    rules: &[(String, CssStyle)],
    input: &NodeInput,
    parent_style: &CssStyle,
) -> Result<BuiltNode> {
    let mut style = compute_style(rules, &input.class, &input.tag);
    inherit_text_style(&mut style, parent_style);
    let mut children = Vec::new();
    let mut child_ids = Vec::new();
    for child in &input.children {
        let built = build_node_with_parent_style(taffy, rules, child, &style)?;
        child_ids.push(built.node);
        children.push(built);
    }
    let taffy_style = to_taffy_style(&style);
    let node = if child_ids.is_empty() && !input.text.is_empty() {
        let measure = TextMeasure {
            text: input.text.clone(),
            font_size: effective_font_size(&style, &input.tag),
            line_height: effective_line_height(&style, &input.tag),
        };
        taffy
            .new_leaf_with_context(taffy_style, measure)
            .context("create Taffy text leaf")?
    } else {
        taffy
            .new_with_children(taffy_style, &child_ids)
            .context("create Taffy node")?
    };
    Ok(BuiltNode {
        id: input.id.clone(),
        tag: input.tag.clone(),
        text: input.text.clone(),
        style,
        node,
        children,
    })
}

fn inherit_text_style(style: &mut CssStyle, parent: &CssStyle) {
    if style.font_size == 0.0 {
        style.font_size = parent.font_size;
    }
    if style.line_height == 0.0 {
        style.line_height = parent.line_height;
    }
    if style.color.is_empty() {
        style.color = parent.color.clone();
    }
}

fn compute_style(rules: &[(String, CssStyle)], class_list: &str, tag: &str) -> CssStyle {
    let mut out = CssStyle {
        display: "block".to_string(),
        flex_direction: "row".to_string(),
        font_size: default_font_size(tag),
        line_height: default_line_height(tag),
        ..CssStyle::default()
    };
    let classes: HashMap<&str, ()> = class_list
        .split_whitespace()
        .map(|name| (name, ()))
        .collect();
    for (class_name, rule) in rules {
        if classes.contains_key(class_name.as_str()) {
            merge_style(&mut out, rule);
        }
    }
    out
}

fn merge_style(out: &mut CssStyle, rule: &CssStyle) {
    if !rule.display.is_empty() {
        out.display = rule.display.clone();
    }
    if !rule.flex_direction.is_empty() {
        out.flex_direction = rule.flex_direction.clone();
    }
    if !rule.flex_wrap.is_empty() {
        out.flex_wrap = rule.flex_wrap.clone();
    }
    if !rule.align_items.is_empty() {
        out.align_items = rule.align_items.clone();
    }
    if !rule.justify_content.is_empty() {
        out.justify_content = rule.justify_content.clone();
    }
    if rule.gap > 0.0 {
        out.gap = rule.gap;
    }
    if rule.padding > 0.0 {
        out.padding = rule.padding;
    }
    if rule.margin > 0.0 {
        out.margin = rule.margin;
    }
    if rule.width.is_some() {
        out.width = rule.width;
    }
    if rule.height.is_some() {
        out.height = rule.height;
    }
    if rule.min_height.is_some() {
        out.min_height = rule.min_height;
    }
    if rule.font_size > 0.0 {
        out.font_size = rule.font_size;
    }
    if rule.line_height > 0.0 {
        out.line_height = rule.line_height;
    }
    if !rule.color.is_empty() {
        out.color = rule.color.clone();
    }
    if !rule.background_color.is_empty() {
        out.background_color = rule.background_color.clone();
    }
    if rule.border_width > 0.0 {
        out.border_width = rule.border_width;
    }
    if !rule.border_color.is_empty() {
        out.border_color = rule.border_color.clone();
    }
    if rule.border_radius > 0.0 {
        out.border_radius = rule.border_radius;
    }
    if !rule.overflow.is_empty() {
        out.overflow = rule.overflow.clone();
    }
}

fn to_taffy_style(style: &CssStyle) -> Style {
    Style {
        display: match style.display.as_str() {
            "flex" => Display::Flex,
            "grid" => Display::Grid,
            "none" => Display::None,
            _ => Display::Block,
        },
        flex_direction: match style.flex_direction.as_str() {
            "column" => FlexDirection::Column,
            _ => FlexDirection::Row,
        },
        flex_wrap: match style.flex_wrap.as_str() {
            "wrap" => FlexWrap::Wrap,
            _ => FlexWrap::NoWrap,
        },
        align_items: match style.align_items.as_str() {
            "center" => Some(AlignItems::Center),
            "flex-end" => Some(AlignItems::FlexEnd),
            "stretch" => Some(AlignItems::Stretch),
            _ => None,
        },
        justify_content: match style.justify_content.as_str() {
            "center" => Some(JustifyContent::Center),
            "space-between" => Some(JustifyContent::SpaceBetween),
            "flex-end" => Some(JustifyContent::FlexEnd),
            _ => None,
        },
        gap: Size::length(style.gap),
        padding: Rect::length(style.padding),
        margin: Rect::length(style.margin),
        size: Size {
            width: style
                .width
                .map(to_taffy_dimension)
                .unwrap_or(Dimension::AUTO),
            height: style
                .height
                .map(to_taffy_dimension)
                .unwrap_or(Dimension::AUTO),
        },
        min_size: Size {
            width: Dimension::AUTO,
            height: style
                .min_height
                .map(to_taffy_dimension)
                .unwrap_or(Dimension::AUTO),
        },
        ..Default::default()
    }
}

fn to_taffy_dimension(length: CssLength) -> Dimension {
    match length {
        CssLength::Px(value) => Dimension::from_length(value),
        CssLength::Percent(value) => Dimension::from_percent(value),
    }
}

fn collect_boxes(
    taffy: &TaffyTree<TextMeasure>,
    built: &BuiltNode,
    parent_x: f32,
    parent_y: f32,
    out: &mut Vec<BoxOutput>,
) -> Result<()> {
    let layout = taffy.layout(built.node).context("read Taffy layout")?;
    let x = parent_x + layout.location.x;
    let y = parent_y + layout.location.y;
    out.push(BoxOutput {
        id: built.id.clone(),
        tag: built.tag.clone(),
        text: built.text.clone(),
        x,
        y,
        width: layout.size.width,
        height: layout.size.height,
        style: StyleOutput {
            display: built.style.display.clone(),
            font_size: built.style.font_size,
            line_height: built.style.line_height,
            color: built.style.color.clone(),
            background_color: built.style.background_color.clone(),
            border_width: built.style.border_width,
            border_color: built.style.border_color.clone(),
            border_radius: built.style.border_radius,
            overflow: built.style.overflow.clone(),
        },
    });
    for child in &built.children {
        collect_boxes(taffy, child, x, y, out)?;
    }
    Ok(())
}

struct TextMeasurer {
    font: FontContext,
    layout: LayoutContext,
}

impl TextMeasurer {
    fn new() -> Self {
        Self {
            font: FontContext::new(),
            layout: LayoutContext::new(),
        }
    }
}

fn measure_text(
    context: Option<&mut TextMeasure>,
    known: Size<Option<f32>>,
    available: Size<AvailableSpace>,
    text: &mut TextMeasurer,
) -> Size<f32> {
    let Some(context) = context else {
        return Size::ZERO;
    };
    let measured = parley_measure(text, context, available.width);
    Size {
        width: known.width.unwrap_or(measured.width),
        height: known.height.unwrap_or(measured.height),
    }
}

fn parley_measure(
    text: &mut TextMeasurer,
    context: &TextMeasure,
    available: AvailableSpace,
) -> Size<f32> {
    let max_width = match available {
        AvailableSpace::Definite(width) if width > 0.0 => Some(width),
        _ => None,
    };
    let mut builder = text
        .layout
        .ranged_builder(&mut text.font, &context.text, 1.0, true);
    builder.push_default(StyleProperty::FontSize(context.font_size));
    let mut layout = builder.build(&context.text);
    layout.break_all_lines(max_width);
    layout.align(Alignment::Start, AlignmentOptions::default());
    Size {
        width: layout.width(),
        height: layout
            .height()
            .max(context.line_height.max(context.font_size)),
    }
}

fn effective_font_size(style: &CssStyle, tag: &str) -> f32 {
    if style.font_size > 0.0 {
        style.font_size
    } else {
        default_font_size(tag)
    }
}

fn effective_line_height(style: &CssStyle, tag: &str) -> f32 {
    if style.line_height > 0.0 {
        style.line_height
    } else {
        default_line_height(tag)
    }
}

fn default_font_size(tag: &str) -> f32 {
    match tag {
        "h1" => 28.0,
        "h2" => 24.0,
        "h3" => 20.0,
        "h4" | "h5" | "h6" => 18.0,
        _ => 16.0,
    }
}

fn default_line_height(tag: &str) -> f32 {
    let size = default_font_size(tag);
    if size >= 24.0 {
        size + 10.0
    } else if size >= 18.0 {
        size + 8.0
    } else {
        24.0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_css_with_lightning_css() {
        let rules =
            parse_css(".card { display: flex; gap: 8px; border: 1px solid #eee; }").unwrap();
        assert_eq!(rules[0].0, "card");
        assert_eq!(rules[0].1.display, "flex");
        assert_eq!(rules[0].1.gap, 8.0);
        assert_eq!(rules[0].1.border_width, 1.0);
    }

    #[test]
    fn computes_flex_layout_with_taffy_and_parley_text() {
        let input = Input {
            css: ".row { display: flex; gap: 12px; width: 200px; } .button { height: 32px; }"
                .to_string(),
            viewport: Viewport {
                width: 300.0,
                height: None,
            },
            root: NodeInput {
                id: "root".to_string(),
                tag: "div".to_string(),
                class: "row".to_string(),
                text: String::new(),
                children: vec![
                    NodeInput {
                        id: "a".to_string(),
                        tag: "button".to_string(),
                        class: "button".to_string(),
                        text: "Add".to_string(),
                        children: vec![],
                    },
                    NodeInput {
                        id: "b".to_string(),
                        tag: "button".to_string(),
                        class: "button".to_string(),
                        text: "Done".to_string(),
                        children: vec![],
                    },
                ],
            },
        };
        let out = compute(input).unwrap();
        assert_eq!(out.boxes.len(), 3);
        assert!(out.boxes[1].width > 24.0);
        assert!(out.boxes[2].x > out.boxes[1].x);
    }

    #[test]
    fn percent_width_uses_parent_width() {
        let input = Input {
            css: ".viewport { width: 800px; } .app { width: 100%; padding: 24px; } .panel { width: 50%; height: 20px; }".to_string(),
            viewport: Viewport {
                width: 800.0,
                height: None,
            },
            root: NodeInput {
                id: "root".to_string(),
                tag: "div".to_string(),
                class: "viewport".to_string(),
                text: String::new(),
                children: vec![NodeInput {
                    id: "app".to_string(),
                    tag: "div".to_string(),
                    class: "app".to_string(),
                    text: String::new(),
                    children: vec![NodeInput {
                        id: "panel".to_string(),
                        tag: "div".to_string(),
                        class: "panel".to_string(),
                        text: String::new(),
                        children: vec![],
                    }],
                }],
            },
        };
        let out = compute(input).unwrap();
        assert_eq!(out.boxes[1].id, "app");
        assert_eq!(out.boxes[1].width, 800.0);
        assert_eq!(out.boxes[2].id, "panel");
        assert_eq!(out.boxes[2].width, 376.0);
    }

    #[test]
    fn percent_min_height_uses_parent_height() {
        let input = Input {
            css: ".viewport { width: 320px; height: 180px; } .app { min-height: 100%; }"
                .to_string(),
            viewport: Viewport {
                width: 320.0,
                height: Some(180.0),
            },
            root: NodeInput {
                id: "root".to_string(),
                tag: "div".to_string(),
                class: "viewport".to_string(),
                text: String::new(),
                children: vec![NodeInput {
                    id: "app".to_string(),
                    tag: "div".to_string(),
                    class: "app".to_string(),
                    text: "A".to_string(),
                    children: vec![],
                }],
            },
        };
        let out = compute(input).unwrap();
        assert_eq!(out.boxes[1].id, "app");
        assert_eq!(out.boxes[1].height, 180.0);
    }

    #[test]
    fn reuses_cached_lightning_css_rules() {
        let input = || Input {
            css: ".row { display: flex; gap: 12px; width: 200px; } .button { height: 32px; }"
                .to_string(),
            viewport: Viewport {
                width: 300.0,
                height: None,
            },
            root: NodeInput {
                id: "root".to_string(),
                tag: "div".to_string(),
                class: "row".to_string(),
                text: String::new(),
                children: vec![NodeInput {
                    id: "a".to_string(),
                    tag: "button".to_string(),
                    class: "button".to_string(),
                    text: "Add".to_string(),
                    children: vec![],
                }],
            },
        };
        let first = compute(input()).unwrap();
        let second = compute(input()).unwrap();
        assert_eq!(first.boxes[0].width, second.boxes[0].width);
        assert_eq!(first.boxes[1].height, 32.0);
    }
}
