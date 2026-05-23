//go:build js && wasm

package wasmhost

import (
	"strconv"
	"syscall/js"

	"github.com/vugra/vugra/internal/renderer"
)

type CanvasRenderer struct {
	canvas js.Value
	ctx    js.Value
	width  int
	height int
	scale  float64
}

type CanvasMeasurer struct {
	ctx js.Value
}

func NewCanvasRenderer(canvas js.Value, width, height int) *CanvasRenderer {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	ctx := canvas.Call("getContext", "2d")
	renderer := &CanvasRenderer{
		canvas: canvas,
		ctx:    ctx,
	}
	renderer.Resize(width, height)
	return renderer
}

func (r *CanvasRenderer) Resize(width, height int) {
	if width <= 0 {
		width = r.width
	}
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = r.height
	}
	if height <= 0 {
		height = 600
	}
	scale := PixelRatio()
	r.canvas.Set("width", int(float64(width)*scale))
	r.canvas.Set("height", int(float64(height)*scale))
	r.width = width
	r.height = height
	r.scale = scale
	if scale != 1 {
		r.ctx.Call("setTransform", scale, 0, 0, scale, 0, 0)
	}
}

func NewCanvasMeasurer(canvas js.Value) *CanvasMeasurer {
	ctx := canvas.Call("getContext", "2d")
	return &CanvasMeasurer{ctx: ctx}
}

func (m *CanvasMeasurer) MeasureText(text string) (float32, float32) {
	return m.MeasureStyledText(text, 16, 24)
}

func (m *CanvasMeasurer) MeasureStyledText(text string, fontSize, lineHeight float32) (float32, float32) {
	if fontSize <= 0 {
		fontSize = 16
	}
	if lineHeight <= 0 {
		lineHeight = 24
	}
	m.ctx.Set("font", canvasFont(fontSize))
	metrics := m.ctx.Call("measureText", text)
	width := float32(metrics.Get("width").Float())
	height := lineHeight
	actualHeight := float32(metrics.Get("actualBoundingBoxAscent").Float() + metrics.Get("actualBoundingBoxDescent").Float())
	if actualHeight > height {
		height = actualHeight
	}
	return width, height
}

func (r *CanvasRenderer) Render(commands []renderer.Command) {
	r.ctx.Call("setTransform", r.scale, 0, 0, r.scale, 0, 0)
	r.ctx.Set("fillStyle", "#fafafa")
	r.ctx.Call("fillRect", 0, 0, r.width, r.height)
	elementClipStack := []bool{}
	for _, command := range commands {
		switch command.Kind {
		case "element":
			r.drawElement(command)
			clipped := clipsOverflow(command.Style.Overflow)
			elementClipStack = append(elementClipStack, clipped)
			if clipped {
				r.pushClip(command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
			}
		case "text":
			r.drawText(command)
		case "selection":
			r.drawSelection(command)
		case "svg":
			r.drawSVG(command)
		case "end":
			if len(elementClipStack) == 0 {
				continue
			}
			clipped := elementClipStack[len(elementClipStack)-1]
			elementClipStack = elementClipStack[:len(elementClipStack)-1]
			if clipped {
				r.ctx.Call("restore")
			}
		}
	}
}

func (r *CanvasRenderer) drawSelection(command renderer.Command) {
	r.setAlpha(command.Style.Opacity)
	defer r.resetAlpha(command.Style.Opacity)
	fill := command.Style.BackgroundColor
	if fill == "" {
		fill = "#bfdbfe"
	}
	r.ctx.Set("fillStyle", fill)
	r.ctx.Call("fillRect", command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
}

func clipsOverflow(overflow string) bool {
	return overflow == "hidden" || overflow == "scroll"
}

func (r *CanvasRenderer) pushClip(x, y, width, height float32) {
	r.ctx.Call("save")
	r.ctx.Call("beginPath")
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	r.ctx.Call("rect", x, y, width, height)
	r.ctx.Call("clip")
}

func (r *CanvasRenderer) drawSVGFallback(command renderer.Command) {
	r.setAlpha(command.Style.Opacity)
	defer r.resetAlpha(command.Style.Opacity)
	r.ctx.Set("fillStyle", "#eff6ff")
	r.ctx.Call("fillRect", command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
	r.ctx.Set("strokeStyle", "#60a5fa")
	r.ctx.Call("strokeRect", command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
}

func (r *CanvasRenderer) drawSVG(command renderer.Command) {
	doc, err := parseSVG(command.SVG)
	if err != nil || len(doc.Shapes) == 0 || !doc.HasViewBox || doc.ViewBox.Width <= 0 || doc.ViewBox.Height <= 0 {
		r.drawSVGFallback(command)
		return
	}
	r.setAlpha(command.Style.Opacity)
	defer r.resetAlpha(command.Style.Opacity)
	r.ctx.Call("save")
	defer r.ctx.Call("restore")
	r.ctx.Call("translate", command.Rect.X, command.Rect.Y)
	r.ctx.Call("scale", command.Rect.Width/doc.ViewBox.Width, command.Rect.Height/doc.ViewBox.Height)
	if doc.ViewBox.X != 0 || doc.ViewBox.Y != 0 {
		r.ctx.Call("translate", -doc.ViewBox.X, -doc.ViewBox.Y)
	}
	for _, shape := range doc.Shapes {
		r.drawSVGShape(shape)
	}
}

func (r *CanvasRenderer) drawSVGShape(shape svgShape) {
	switch shape.Kind {
	case "rect":
		if shape.Width <= 0 || shape.Height <= 0 {
			return
		}
		if shape.RX > 0 {
			r.roundedRectPath(shape.X, shape.Y, shape.Width, shape.Height, shape.RX)
		} else {
			r.ctx.Call("beginPath")
			r.ctx.Call("rect", shape.X, shape.Y, shape.Width, shape.Height)
		}
	case "circle":
		if shape.R <= 0 {
			return
		}
		r.ctx.Call("beginPath")
		r.ctx.Call("arc", shape.CX, shape.CY, shape.R, 0, 6.283185307179586)
	case "path":
		if len(shape.Path) == 0 {
			return
		}
		r.ctx.Call("beginPath")
		for _, command := range shape.Path {
			switch command.Op {
			case 'M':
				r.ctx.Call("moveTo", command.X, command.Y)
			case 'L':
				r.ctx.Call("lineTo", command.X, command.Y)
			case 'C':
				r.ctx.Call("bezierCurveTo", command.X1, command.Y1, command.X2, command.Y2, command.X, command.Y)
			case 'Q':
				r.ctx.Call("quadraticCurveTo", command.X1, command.Y1, command.X, command.Y)
			case 'Z':
				r.ctx.Call("closePath")
			}
		}
	default:
		return
	}
	if shape.Fill != "" && shape.Fill != "none" {
		r.ctx.Set("fillStyle", shape.Fill)
		r.ctx.Call("fill")
	}
	if shape.Stroke != "" && shape.Stroke != "none" {
		r.ctx.Set("strokeStyle", shape.Stroke)
		r.ctx.Set("lineWidth", shape.StrokeWidth)
		r.ctx.Call("stroke")
	}
}

func (r *CanvasRenderer) drawElement(command renderer.Command) {
	if !shouldDrawElement(command) {
		return
	}
	r.setAlpha(command.Style.Opacity)
	defer r.resetAlpha(command.Style.Opacity)
	fill := command.Style.BackgroundColor
	stroke := command.Style.BorderColor
	if fill == "" {
		fill = "#f5f7fa"
	}
	if stroke == "" {
		stroke = "#2d3748"
	}
	if command.Tag == "button" && command.Style.BackgroundColor == "" {
		fill = "#e5f1ff"
	}
	if command.Tag == "button" && command.Style.BorderColor == "" {
		stroke = "#2563eb"
	}
	r.ctx.Set("fillStyle", fill)
	if borderWidth := elementBorderWidth(command); borderWidth > 0 {
		r.drawRect(command, stroke, borderWidth)
		r.drawCheckboxCheck(command)
		return
	}
	r.drawRect(command, "", 0)
	r.drawCheckboxCheck(command)
}

func shouldDrawElement(command renderer.Command) bool {
	return command.Role == "button" ||
		command.Role == "textbox" ||
		command.Role == "checkbox" ||
		command.Tag == "button" ||
		command.Tag == "input" ||
		command.Style.BackgroundColor != "" ||
		command.Style.BorderColor != "" ||
		command.Style.BorderWidth > 0 ||
		command.Style.BorderRadius > 0
}

func (r *CanvasRenderer) drawText(command renderer.Command) {
	r.setAlpha(command.Style.Opacity)
	defer r.resetAlpha(command.Style.Opacity)
	fill := command.Style.Color
	if fill == "" {
		fill = "#0f172a"
	}
	r.ctx.Set("fillStyle", fill)
	fontSize := command.Style.FontSize
	if fontSize <= 0 {
		fontSize = 16
	}
	r.ctx.Set("font", canvasFont(fontSize))
	lines := command.Lines
	if len(lines) == 0 {
		lines = []renderer.LineBox{{Text: command.Text, X: command.Rect.X, Y: command.Rect.Y, Baseline: 14}}
	}
	for _, line := range lines {
		baseline := line.Baseline
		if baseline <= 0 {
			baseline = fontSize * 0.82
		}
		r.ctx.Call("fillText", line.Text, line.X, line.Y+baseline)
	}
}

func canvasFont(fontSize float32) string {
	return strconv.FormatFloat(float64(fontSize), 'f', -1, 32) + "px ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
}

func (r *CanvasRenderer) setAlpha(opacity float32) {
	if opacity <= 0 || opacity >= 1 {
		return
	}
	r.ctx.Set("globalAlpha", opacity)
}

func (r *CanvasRenderer) resetAlpha(opacity float32) {
	if opacity <= 0 || opacity >= 1 {
		return
	}
	r.ctx.Set("globalAlpha", 1)
}

func elementBorderWidth(command renderer.Command) float32 {
	if command.Style.BorderWidthSet && command.Style.BorderWidth <= 0 {
		return 0
	}
	if command.Style.BorderWidth > 0 {
		return command.Style.BorderWidth
	}
	if command.Style.BorderColor != "" || isControlElement(command) {
		return 1
	}
	return 0
}

func isControlElement(command renderer.Command) bool {
	return command.Role == "button" ||
		command.Role == "textbox" ||
		command.Role == "checkbox" ||
		command.Tag == "button" ||
		command.Tag == "input"
}

func (r *CanvasRenderer) drawRect(command renderer.Command, stroke string, borderWidth float32) {
	if command.Style.BorderRadius <= 0 {
		r.ctx.Call("fillRect", command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
		if borderWidth > 0 {
			r.ctx.Set("strokeStyle", stroke)
			r.ctx.Set("lineWidth", borderWidth)
			r.ctx.Call("strokeRect", command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height)
		}
		return
	}
	r.roundedRectPath(command.Rect.X, command.Rect.Y, command.Rect.Width, command.Rect.Height, command.Style.BorderRadius)
	r.ctx.Call("fill")
	if borderWidth > 0 {
		r.ctx.Set("strokeStyle", stroke)
		r.ctx.Set("lineWidth", borderWidth)
		r.ctx.Call("stroke")
	}
}

func (r *CanvasRenderer) drawCheckboxCheck(command renderer.Command) {
	if command.Role != "checkbox" || command.Props["checked"] != "true" {
		return
	}
	x := command.Rect.X
	y := command.Rect.Y
	r.ctx.Set("strokeStyle", "#ffffff")
	r.ctx.Set("lineWidth", 2)
	r.ctx.Call("beginPath")
	r.ctx.Call("moveTo", x+4, y+command.Rect.Height*0.52)
	r.ctx.Call("lineTo", x+command.Rect.Width*0.42, y+command.Rect.Height-5)
	r.ctx.Call("lineTo", x+command.Rect.Width-4, y+5)
	r.ctx.Call("stroke")
}

func (r *CanvasRenderer) roundedRectPath(x, y, width, height, radius float32) {
	maxRadius := width / 2
	if height/2 < maxRadius {
		maxRadius = height / 2
	}
	if radius > maxRadius {
		radius = maxRadius
	}
	if radius < 0 {
		radius = 0
	}
	right := x + width
	bottom := y + height
	r.ctx.Call("beginPath")
	r.ctx.Call("moveTo", x+radius, y)
	r.ctx.Call("lineTo", right-radius, y)
	r.ctx.Call("arcTo", right, y, right, y+radius, radius)
	r.ctx.Call("lineTo", right, bottom-radius)
	r.ctx.Call("arcTo", right, bottom, right-radius, bottom, radius)
	r.ctx.Call("lineTo", x+radius, bottom)
	r.ctx.Call("arcTo", x, bottom, x, bottom-radius, radius)
	r.ctx.Call("lineTo", x, y+radius)
	r.ctx.Call("arcTo", x, y, x+radius, y, radius)
	r.ctx.Call("closePath")
}

func InstallPointerEvents(canvas js.Value, dispatch func(x, y int)) js.Func {
	return InstallPointerEventDetails(canvas, func(event string, x, y, deltaX, deltaY int, shift, ctrl, meta, alt bool) {
		if event == "click" {
			dispatch(x, y)
		}
	})
}

func InstallPointerEventDetails(canvas js.Value, dispatch func(event string, x, y, deltaX, deltaY int, shift, ctrl, meta, alt bool)) js.Func {
	var lastX int
	var lastY int
	pressed := false
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		rect := canvas.Call("getBoundingClientRect")
		x := int(event.Get("clientX").Float() - rect.Get("left").Float())
		y := int(event.Get("clientY").Float() - rect.Get("top").Float())
		eventType := event.Get("type").String()
		deltaX := x - lastX
		deltaY := y - lastY
		lastX = x
		lastY = y
		switch eventType {
		case "mousemove":
			eventType = "hover"
		case "mousedown":
			pressed = true
			eventType = "click"
		case "mouseup":
			pressed = false
			return nil
		case "dblclick", "contextmenu":
		default:
			return nil
		}
		if eventType == "click" || eventType == "drag" {
			event.Call("preventDefault")
		}
		dispatch(eventType, x, y, deltaX, deltaY, event.Get("shiftKey").Bool(), event.Get("ctrlKey").Bool(), event.Get("metaKey").Bool(), event.Get("altKey").Bool())
		if eventType == "contextmenu" {
			event.Call("preventDefault")
		}
		return nil
	})
	drag := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		rect := canvas.Call("getBoundingClientRect")
		x := int(event.Get("clientX").Float() - rect.Get("left").Float())
		y := int(event.Get("clientY").Float() - rect.Get("top").Float())
		event.Call("preventDefault")
		if pressed {
			dispatch("drag", x, y, int(event.Get("movementX").Float()), int(event.Get("movementY").Float()), event.Get("shiftKey").Bool(), event.Get("ctrlKey").Bool(), event.Get("metaKey").Bool(), event.Get("altKey").Bool())
		}
		return nil
	})
	canvas.Call("addEventListener", "mousedown", handler)
	canvas.Call("addEventListener", "mousemove", handler)
	canvas.Call("addEventListener", "mousemove", drag)
	canvas.Call("addEventListener", "mouseup", handler)
	canvas.Call("addEventListener", "dblclick", handler)
	canvas.Call("addEventListener", "contextmenu", handler)
	return handler
}

func InstallScrollEvents(canvas js.Value, dispatch func(x, y, deltaY int)) js.Func {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		rect := canvas.Call("getBoundingClientRect")
		x := int(event.Get("clientX").Float() - rect.Get("left").Float())
		y := int(event.Get("clientY").Float() - rect.Get("top").Float())
		dispatch(x, y, int(event.Get("deltaY").Float()))
		event.Call("preventDefault")
		return nil
	})
	canvas.Call("addEventListener", "wheel", handler, map[string]any{"passive": false})
	return handler
}

func InstallKeyboardEvents(dispatch func(key string)) js.Func {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		key := event.Get("key").String()
		if jsBool(event.Get("shiftKey")) && key == "Tab" {
			key = "Shift+Tab"
		}
		if key == "Tab" || key == "Shift+Tab" || key == "Home" || key == "End" {
			event.Call("preventDefault")
		}
		if (event.Get("ctrlKey").Bool() || event.Get("metaKey").Bool()) && (key == "a" || key == "A") {
			key = "Mod+A"
		}
		dispatch(key)
		return nil
	})
	js.Global().Call("addEventListener", "keydown", handler)
	return handler
}

func InstallTextEvents(dispatch func(text string)) js.Func {
	composing := false
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		if composing || jsBool(event.Get("isComposing")) {
			return nil
		}
		if jsBool(event.Get("ctrlKey")) || jsBool(event.Get("metaKey")) || jsBool(event.Get("altKey")) {
			return nil
		}
		key := event.Get("key").String()
		if len([]rune(key)) != 1 {
			return nil
		}
		dispatch(key)
		event.Call("preventDefault")
		return nil
	})
	compositionStart := js.FuncOf(func(this js.Value, args []js.Value) any {
		composing = true
		return nil
	})
	compositionEnd := js.FuncOf(func(this js.Value, args []js.Value) any {
		composing = false
		if len(args) == 0 {
			return nil
		}
		text := args[0].Get("data").String()
		if text == "" {
			return nil
		}
		dispatch(text)
		args[0].Call("preventDefault")
		return nil
	})
	paste := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		clipboard := args[0].Get("clipboardData")
		if !clipboard.Truthy() {
			return nil
		}
		text := clipboard.Call("getData", "text/plain").String()
		if text == "" {
			return nil
		}
		dispatch(text)
		args[0].Call("preventDefault")
		return nil
	})
	js.Global().Call("addEventListener", "keydown", handler)
	js.Global().Call("addEventListener", "compositionstart", compositionStart)
	js.Global().Call("addEventListener", "compositionend", compositionEnd)
	js.Global().Call("addEventListener", "paste", paste)
	return handler
}

func ExportFrameCommands(name string, commands func() []renderer.Command) {
	js.Global().Set(name, js.FuncOf(func(this js.Value, args []js.Value) any {
		var out []any
		for _, command := range commands() {
			out = append(out, map[string]any{
				"kind":   command.Kind,
				"id":     command.ID,
				"tag":    command.Tag,
				"text":   command.Text,
				"lines":  exportLines(command.Lines),
				"glyphs": exportGlyphs(command.Glyphs),
				"x":      command.Rect.X,
				"y":      command.Rect.Y,
				"width":  command.Rect.Width,
				"height": command.Rect.Height,
				"props":  command.Props,
			})
		}
		return out
	}))
}

func exportLines(lines []renderer.LineBox) []any {
	out := make([]any, 0, len(lines))
	for _, line := range lines {
		out = append(out, map[string]any{
			"text":     line.Text,
			"x":        line.X,
			"y":        line.Y,
			"width":    line.Width,
			"height":   line.Height,
			"baseline": line.Baseline,
		})
	}
	return out
}

func exportGlyphs(glyphs []renderer.GlyphRun) []any {
	out := make([]any, 0, len(glyphs))
	for _, glyph := range glyphs {
		out = append(out, map[string]any{
			"text":     glyph.Text,
			"font":     glyph.Font,
			"size":     glyph.Size,
			"x":        glyph.X,
			"y":        glyph.Y,
			"advance":  glyph.Advance,
			"baseline": glyph.Baseline,
		})
	}
	return out
}

func SetStatus(id string, text string) {
	element := js.Global().Get("document").Call("getElementById", id)
	if element.Truthy() {
		element.Set("textContent", text)
	}
}

func PixelRatio() float64 {
	ratio := js.Global().Get("devicePixelRatio")
	if !ratio.Truthy() {
		return 1
	}
	value := ratio.Float()
	if value <= 0 {
		return 1
	}
	return value
}

func jsBool(value js.Value) bool {
	if value.Type() != js.TypeBoolean {
		return false
	}
	return value.Bool()
}
