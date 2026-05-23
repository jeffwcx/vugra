package renderer

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type SoftwareRenderer struct {
	Width      int
	Height     int
	Background color.Color
	Image      *image.RGBA
}

func NewSoftware(width, height int) *SoftwareRenderer {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return &SoftwareRenderer{
		Width:      width,
		Height:     height,
		Background: color.RGBA{R: 250, G: 250, B: 250, A: 255},
	}
}

func (r *SoftwareRenderer) Render(commands []Command) {
	img := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: r.Background}, image.Point{}, draw.Src)
	clipStack := []image.Rectangle{img.Bounds()}
	elementClipStack := []bool{}
	for _, command := range commands {
		currentClip := clipStack[len(clipStack)-1]
		switch command.Kind {
		case "element":
			r.drawElement(img, command, currentClip)
			clipped := clipsOverflow(command.Style.Overflow)
			elementClipStack = append(elementClipStack, clipped)
			if clipped {
				clipStack = append(clipStack, commandRect(command).Intersect(currentClip))
			}
		case "text":
			r.drawText(img, command, currentClip)
		case "selection":
			r.drawSelection(img, command, currentClip)
		case "svg":
			r.drawSVGFallback(img, command, currentClip)
		case "end":
			if len(elementClipStack) == 0 {
				continue
			}
			clipped := elementClipStack[len(elementClipStack)-1]
			elementClipStack = elementClipStack[:len(elementClipStack)-1]
			if clipped && len(clipStack) > 1 {
				clipStack = clipStack[:len(clipStack)-1]
			}
		}
	}
	r.Image = img
}

func (r *SoftwareRenderer) drawSelection(img *image.RGBA, command Command, clip image.Rectangle) {
	rect := commandRect(command).Intersect(img.Bounds()).Intersect(clip)
	if rect.Empty() {
		return
	}
	fill := color.RGBA{R: 191, G: 219, B: 254, A: 184}
	if c, ok := parseHexColor(command.Style.BackgroundColor); ok {
		fill = c
	}
	fill = applyOpacity(fill, command.Style.Opacity)
	draw.Draw(img, rect, &image.Uniform{C: fill}, image.Point{}, draw.Over)
}

func (r *SoftwareRenderer) drawSVGFallback(img *image.RGBA, command Command, clip image.Rectangle) {
	rect := commandRect(command).Intersect(img.Bounds()).Intersect(clip)
	if rect.Empty() {
		return
	}
	fill := color.RGBA{R: 239, G: 246, B: 255, A: 255}
	border := color.RGBA{R: 96, G: 165, B: 250, A: 255}
	drawRoundedRect(img, rect, roundFloat(command.Style.BorderRadius), fill)
	drawRoundedBorder(img, rect, maxInt(1, roundFloat(command.Style.BorderRadius)), 1, border)
}

func clipsOverflow(overflow string) bool {
	return overflow == "hidden" || overflow == "scroll"
}

func (r *SoftwareRenderer) SavePNG(path string) error {
	if r.Image == nil {
		r.Image = image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, r.Image)
}

func (r *SoftwareRenderer) drawElement(img *image.RGBA, command Command, clip image.Rectangle) {
	rect := commandRect(command).Intersect(img.Bounds()).Intersect(clip)
	if rect.Empty() {
		return
	}
	if !shouldDrawElement(command) {
		return
	}
	fill := color.RGBA{R: 248, G: 250, B: 252, A: 255}
	border := color.RGBA{R: 226, G: 232, B: 240, A: 255}
	switch command.Role {
	case "button":
		fill = color.RGBA{R: 229, G: 241, B: 255, A: 255}
		border = color.RGBA{R: 37, G: 99, B: 235, A: 255}
	case "textbox":
		fill = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		border = color.RGBA{R: 148, G: 163, B: 184, A: 255}
	case "checkbox":
		fill = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		border = color.RGBA{R: 148, G: 163, B: 184, A: 255}
		if command.Props["checked"] == "true" {
			fill = color.RGBA{R: 37, G: 99, B: 235, A: 255}
		}
	case "listitem":
		fill = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		border = color.RGBA{R: 203, G: 213, B: 225, A: 255}
	}
	if c, ok := parseHexColor(command.Style.BackgroundColor); ok {
		fill = c
	}
	if c, ok := parseHexColor(command.Style.BorderColor); ok {
		border = c
	}
	fill = applyOpacity(fill, command.Style.Opacity)
	border = applyOpacity(border, command.Style.Opacity)
	borderWidth := elementBorderWidth(command)
	radius := roundFloat(command.Style.BorderRadius)
	drawRoundedRect(img, rect, radius, fill)
	if borderWidth > 0 {
		drawRoundedBorder(img, rect, radius, borderWidth, border)
	}
	if command.Role == "checkbox" && command.Props["checked"] == "true" {
		drawCheck(img, rect, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
}

func shouldDrawElement(command Command) bool {
	if isControlElement(command) {
		return true
	}
	return command.Style.BackgroundColor != "" ||
		command.Style.BorderColor != "" ||
		command.Style.BorderWidth > 0 ||
		command.Style.BorderRadius > 0
}

func isControlElement(command Command) bool {
	return command.Role == "button" ||
		command.Role == "textbox" ||
		command.Role == "checkbox" ||
		command.Tag == "button" ||
		command.Tag == "input"
}

func elementBorderWidth(command Command) int {
	if command.Style.BorderWidthSet && command.Style.BorderWidth <= 0 {
		return 0
	}
	if command.Style.BorderWidth > 0 {
		return maxInt(1, roundFloat(command.Style.BorderWidth))
	}
	if !isControlElement(command) {
		if command.Style.BorderColor == "" {
			return 0
		}
		return 1
	}
	return 1
}

func (r *SoftwareRenderer) drawText(img *image.RGBA, command Command, clip image.Rectangle) {
	clip = clip.Intersect(img.Bounds())
	if clip.Empty() {
		return
	}
	lineStep := command.Style.LineHeight
	if lineStep <= 0 {
		lineStep = 20
	}
	textColor := color.RGBA{R: 15, G: 23, B: 42, A: 255}
	if command.Role == "button" {
		textColor = color.RGBA{R: 37, G: 99, B: 235, A: 255}
	}
	if c, ok := parseHexColor(command.Style.Color); ok {
		textColor = c
	}
	textColor = applyOpacity(textColor, command.Style.Opacity)
	face := basicfont.Face7x13
	lineY := roundFloat(command.Rect.Y)
	lines := command.Lines
	if len(lines) == 0 {
		for _, text := range strings.Split(command.Text, "\n") {
			lines = append(lines, LineBox{Text: text, X: command.Rect.X, Y: float32(lineY), Width: command.Rect.Width, Height: lineStep, Baseline: 13})
			lineY += roundFloat(lineStep)
		}
	}
	for _, line := range lines {
		y := roundFloat(line.Y)
		x := roundFloat(command.Rect.X)
		if line.Baseline == 0 {
			line.Baseline = 13
		}
		if line.Height > 0 {
			lineStep = line.Height
		}
		lineText := line.Text
		if lineText != "" {
			if line.X != 0 {
				x = roundFloat(line.X)
			}
			lineRect := image.Rect(x, y, x+roundFloat(line.Width), y+roundFloat(lineStep))
			if lineRect.Intersect(clip).Empty() {
				continue
			}
			drawer := &font.Drawer{
				Dst:  clippedImage{img: img, clip: clip},
				Src:  image.NewUniform(textColor),
				Face: face,
				Dot:  fixed.P(x, y+roundFloat(line.Baseline)),
			}
			drawer.DrawString(lineText)
		}
	}
}

type clippedImage struct {
	img  *image.RGBA
	clip image.Rectangle
}

func (i clippedImage) ColorModel() color.Model {
	return i.img.ColorModel()
}

func (i clippedImage) Bounds() image.Rectangle {
	return i.img.Bounds()
}

func (i clippedImage) At(x, y int) color.Color {
	return i.img.At(x, y)
}

func (i clippedImage) Set(x, y int, c color.Color) {
	if image.Pt(x, y).In(i.clip) {
		i.img.Set(x, y, c)
	}
}

func commandRect(command Command) image.Rectangle {
	minX := floorFloat(command.Rect.X)
	minY := floorFloat(command.Rect.Y)
	maxX := ceilFloat(command.Rect.X + command.Rect.Width)
	maxY := ceilFloat(command.Rect.Y + command.Rect.Height)
	return image.Rect(minX, minY, maxX, maxY)
}

func drawBorder(img *image.RGBA, rect image.Rectangle, c color.Color) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, c)
		img.Set(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, c)
		img.Set(rect.Max.X-1, y, c)
	}
}

func drawRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, c color.Color) {
	if radius <= 0 {
		draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
		return
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if pointInRoundedRect(x, y, rect, radius) {
				img.Set(x, y, c)
			}
		}
	}
}

func drawRoundedBorder(img *image.RGBA, rect image.Rectangle, radius, width int, c color.Color) {
	if width <= 0 {
		return
	}
	if radius <= 0 {
		for i := 0; i < width; i++ {
			drawBorder(img, rect.Inset(i), c)
		}
		return
	}
	inner := rect.Inset(width)
	innerRadius := radius - width
	if innerRadius < 0 {
		innerRadius = 0
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if !pointInRoundedRect(x, y, rect, radius) {
				continue
			}
			if inner.Empty() || !pointInRoundedRect(x, y, inner, innerRadius) {
				img.Set(x, y, c)
			}
		}
	}
}

func applyOpacity(c color.RGBA, opacity float32) color.RGBA {
	if opacity <= 0 || opacity >= 1 {
		return c
	}
	c.A = uint8(float32(c.A) * opacity)
	return c
}

func pointInRoundedRect(x, y int, rect image.Rectangle, radius int) bool {
	if !image.Pt(x, y).In(rect) {
		return false
	}
	maxRadius := minInt(rect.Dx(), rect.Dy()) / 2
	if radius > maxRadius {
		radius = maxRadius
	}
	if radius <= 0 {
		return true
	}
	left := rect.Min.X + radius
	right := rect.Max.X - radius - 1
	top := rect.Min.Y + radius
	bottom := rect.Max.Y - radius - 1
	cx := clampInt(x, left, right)
	cy := clampInt(y, top, bottom)
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= radius*radius
}

func drawCheck(img *image.RGBA, rect image.Rectangle, c color.Color) {
	points := []image.Point{
		{X: rect.Min.X + 4, Y: rect.Min.Y + rect.Dy()/2},
		{X: rect.Min.X + 8, Y: rect.Max.Y - 5},
		{X: rect.Max.X - 4, Y: rect.Min.Y + 4},
	}
	drawLine(img, points[0], points[1], c)
	drawLine(img, points[1], points[2], c)
}

func drawLine(img *image.RGBA, from, to image.Point, c color.Color) {
	dx := absInt(to.X - from.X)
	dy := -absInt(to.Y - from.Y)
	sx := -1
	if from.X < to.X {
		sx = 1
	}
	sy := -1
	if from.Y < to.Y {
		sy = 1
	}
	err := dx + dy
	for {
		if from.In(img.Bounds()) {
			img.Set(from.X, from.Y, c)
		}
		if from == to {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			from.X += sx
		}
		if e2 <= dx {
			err += dx
			from.Y += sy
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func roundFloat(value float32) int {
	return int(math.Round(float64(value)))
}

func floorFloat(value float32) int {
	return int(math.Floor(float64(value)))
}

func ceilFloat(value float32) int {
	return int(math.Ceil(float64(value)))
}

func parseHexColor(value string) (color.RGBA, bool) {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return color.RGBA{}, false
	}
	n, err := strconv.ParseUint(value[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 255}, true
}
