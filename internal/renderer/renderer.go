package renderer

// Package renderer defines backend-neutral render commands and renderer
// interfaces for test, native, and wasm backends.

type Command struct {
	Kind     string            `json:"kind"`
	ID       string            `json:"id,omitempty"`
	Tag      string            `json:"tag,omitempty"`
	Text     string            `json:"text,omitempty"`
	SVG      string            `json:"svg,omitempty"`
	Lines    []LineBox         `json:"lines,omitempty"`
	Glyphs   []GlyphRun        `json:"glyphs,omitempty"`
	Role     string            `json:"role,omitempty"`
	Event    string            `json:"event,omitempty"`
	Rect     Rect              `json:"rect"`
	Props    map[string]string `json:"props,omitempty"`
	Bindings map[string]string `json:"bindings,omitempty"`
	Style    Style             `json:"style,omitempty"`
}

type Rect struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
}

type LineBox struct {
	Text     string  `json:"text"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Width    float32 `json:"width"`
	Height   float32 `json:"height"`
	Baseline float32 `json:"baseline"`
}

type GlyphRun struct {
	Text     string  `json:"text"`
	Font     string  `json:"font,omitempty"`
	Size     float32 `json:"size"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Advance  float32 `json:"advance"`
	Baseline float32 `json:"baseline"`
}

type Renderer interface {
	Render([]Command)
}

type Style struct {
	Display         string  `json:"display,omitempty"`
	FlexDirection   string  `json:"flexDirection,omitempty"`
	FlexWrap        string  `json:"flexWrap,omitempty"`
	AlignItems      string  `json:"alignItems,omitempty"`
	Justify         string  `json:"justify,omitempty"`
	Margin          float32 `json:"margin,omitempty"`
	PaddingLeft     float32 `json:"paddingLeft,omitempty"`
	FlexGrow        float32 `json:"flexGrow,omitempty"`
	FlexBasis       float32 `json:"flexBasis,omitempty"`
	FontSize        float32 `json:"fontSize,omitempty"`
	LineHeight      float32 `json:"lineHeight,omitempty"`
	TextAlign       string  `json:"textAlign,omitempty"`
	BackgroundColor string  `json:"backgroundColor,omitempty"`
	Opacity         float32 `json:"opacity,omitempty"`
	BorderWidth     float32 `json:"borderWidth,omitempty"`
	BorderWidthSet  bool    `json:"borderWidthSet,omitempty"`
	BorderColor     string  `json:"borderColor,omitempty"`
	BorderRadius    float32 `json:"borderRadius,omitempty"`
	Color           string  `json:"color,omitempty"`
	Overflow        string  `json:"overflow,omitempty"`
}

func DefaultFontSize(tag string, role string, style Style) float32 {
	if style.FontSize > 0 {
		return style.FontSize
	}
	switch tag {
	case "h1":
		return 28
	case "h2":
		return 24
	case "h3":
		return 20
	case "h4", "h5", "h6":
		return 18
	}
	return 16
}

func DefaultLineHeight(tag string, role string, style Style) float32 {
	if style.LineHeight > 0 {
		return style.LineHeight
	}
	size := DefaultFontSize(tag, role, style)
	if size >= 24 {
		return size + 10
	}
	if size >= 18 {
		return size + 8
	}
	return 24
}

type TestRenderer struct {
	Frames [][]Command
}

func (r *TestRenderer) Render(commands []Command) {
	frame := make([]Command, len(commands))
	copy(frame, commands)
	r.Frames = append(r.Frames, frame)
}

func (r *TestRenderer) LastFrame() []Command {
	if len(r.Frames) == 0 {
		return nil
	}
	return r.Frames[len(r.Frames)-1]
}
