package wasmhost

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type svgDocument struct {
	ViewBox    svgViewBox
	HasViewBox bool
	Shapes     []svgShape
}

type svgViewBox struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type svgShape struct {
	Kind        string
	X           float32
	Y           float32
	Width       float32
	Height      float32
	RX          float32
	CX          float32
	CY          float32
	R           float32
	Fill        string
	Stroke      string
	StrokeWidth float32
	Path        []svgPathCommand
}

type svgPathCommand struct {
	Op     byte
	X1, Y1 float32
	X2, Y2 float32
	X, Y   float32
}

func parseSVG(source string) (svgDocument, error) {
	decoder := xml.NewDecoder(strings.NewReader(source))
	var doc svgDocument
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return svgDocument{}, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "svg":
			doc.ViewBox, doc.HasViewBox = parseSVGViewBox(start.Attr)
		case "rect":
			doc.Shapes = append(doc.Shapes, parseSVGRect(start.Attr))
		case "circle":
			doc.Shapes = append(doc.Shapes, parseSVGCircle(start.Attr))
		case "path":
			shape, ok := parseSVGPathShape(start.Attr)
			if ok {
				doc.Shapes = append(doc.Shapes, shape)
			}
		}
	}
	if !doc.HasViewBox {
		for _, shape := range doc.Shapes {
			doc.ViewBox = expandSVGViewBox(doc.ViewBox, shape)
		}
		doc.HasViewBox = doc.ViewBox.Width > 0 && doc.ViewBox.Height > 0
	}
	return doc, nil
}

func parseSVGViewBox(attrs []xml.Attr) (svgViewBox, bool) {
	viewBox := attrValue(attrs, "viewBox")
	if viewBox != "" {
		values := parseNumberList(viewBox)
		if len(values) == 4 && values[2] > 0 && values[3] > 0 {
			return svgViewBox{X: values[0], Y: values[1], Width: values[2], Height: values[3]}, true
		}
	}
	width := parseSVGFloat(attrValue(attrs, "width"))
	height := parseSVGFloat(attrValue(attrs, "height"))
	if width > 0 && height > 0 {
		return svgViewBox{Width: width, Height: height}, true
	}
	return svgViewBox{}, false
}

func parseSVGRect(attrs []xml.Attr) svgShape {
	style := parseSVGStyle(attrs)
	rx := parseSVGFloat(firstNonEmptyString(attrValue(attrs, "rx"), attrValue(attrs, "ry")))
	return svgShape{
		Kind:        "rect",
		X:           parseSVGFloat(attrValue(attrs, "x")),
		Y:           parseSVGFloat(attrValue(attrs, "y")),
		Width:       parseSVGFloat(attrValue(attrs, "width")),
		Height:      parseSVGFloat(attrValue(attrs, "height")),
		RX:          rx,
		Fill:        style.fill,
		Stroke:      style.stroke,
		StrokeWidth: style.strokeWidth,
	}
}

func parseSVGCircle(attrs []xml.Attr) svgShape {
	style := parseSVGStyle(attrs)
	return svgShape{
		Kind:        "circle",
		CX:          parseSVGFloat(attrValue(attrs, "cx")),
		CY:          parseSVGFloat(attrValue(attrs, "cy")),
		R:           parseSVGFloat(attrValue(attrs, "r")),
		Fill:        style.fill,
		Stroke:      style.stroke,
		StrokeWidth: style.strokeWidth,
	}
}

func parseSVGPathShape(attrs []xml.Attr) (svgShape, bool) {
	path := parseSVGPath(attrValue(attrs, "d"))
	if len(path) == 0 {
		return svgShape{}, false
	}
	style := parseSVGStyle(attrs)
	return svgShape{
		Kind:        "path",
		Fill:        style.fill,
		Stroke:      style.stroke,
		StrokeWidth: style.strokeWidth,
		Path:        path,
	}, true
}

type svgStyle struct {
	fill        string
	stroke      string
	strokeWidth float32
}

func parseSVGStyle(attrs []xml.Attr) svgStyle {
	style := svgStyle{fill: "#000000", strokeWidth: 1}
	if inline := attrValue(attrs, "style"); inline != "" {
		for _, part := range strings.Split(inline, ";") {
			name, value, ok := strings.Cut(part, ":")
			if !ok {
				continue
			}
			setSVGStyle(&style, strings.TrimSpace(name), strings.TrimSpace(value))
		}
	}
	for _, attr := range attrs {
		setSVGStyle(&style, attr.Name.Local, strings.TrimSpace(attr.Value))
	}
	return style
}

func setSVGStyle(style *svgStyle, name string, value string) {
	if value == "" {
		return
	}
	switch name {
	case "fill":
		style.fill = value
	case "stroke":
		style.stroke = value
	case "stroke-width":
		if width := parseSVGFloat(value); width > 0 {
			style.strokeWidth = width
		}
	}
}

func expandSVGViewBox(viewBox svgViewBox, shape svgShape) svgViewBox {
	minX, minY, maxX, maxY := shapeBounds(shape)
	if maxX <= minX || maxY <= minY {
		return viewBox
	}
	if viewBox.Width <= 0 || viewBox.Height <= 0 {
		return svgViewBox{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
	}
	currentMaxX := viewBox.X + viewBox.Width
	currentMaxY := viewBox.Y + viewBox.Height
	if minX < viewBox.X {
		viewBox.X = minX
	}
	if minY < viewBox.Y {
		viewBox.Y = minY
	}
	if maxX > currentMaxX {
		currentMaxX = maxX
	}
	if maxY > currentMaxY {
		currentMaxY = maxY
	}
	viewBox.Width = currentMaxX - viewBox.X
	viewBox.Height = currentMaxY - viewBox.Y
	return viewBox
}

func shapeBounds(shape svgShape) (float32, float32, float32, float32) {
	switch shape.Kind {
	case "rect":
		return shape.X, shape.Y, shape.X + shape.Width, shape.Y + shape.Height
	case "circle":
		return shape.CX - shape.R, shape.CY - shape.R, shape.CX + shape.R, shape.CY + shape.R
	case "path":
		var minX, minY, maxX, maxY float32
		initialized := false
		add := func(x, y float32) {
			if !initialized {
				minX, maxX = x, x
				minY, maxY = y, y
				initialized = true
				return
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
		for _, command := range shape.Path {
			add(command.X, command.Y)
			add(command.X1, command.Y1)
			add(command.X2, command.Y2)
		}
		return minX, minY, maxX, maxY
	default:
		return 0, 0, 0, 0
	}
}

func parseSVGPath(source string) []svgPathCommand {
	tokens := tokenizeSVGPath(source)
	var out []svgPathCommand
	var command byte
	var currentX, currentY float32
	var startX, startY float32
	for index := 0; index < len(tokens); {
		if isSVGPathCommand(tokens[index]) {
			command = tokens[index][0]
			index++
		}
		if command == 0 {
			return out
		}
		switch command {
		case 'M', 'm':
			first := true
			for hasSVGPathNumbers(tokens, index, 2) {
				x := parseSVGFloat(tokens[index])
				y := parseSVGFloat(tokens[index+1])
				index += 2
				if command == 'm' {
					x += currentX
					y += currentY
				}
				op := byte('M')
				if !first {
					op = 'L'
				}
				out = append(out, svgPathCommand{Op: op, X: x, Y: y})
				currentX, currentY = x, y
				if first {
					startX, startY = x, y
				}
				first = false
			}
			if command == 'M' {
				command = 'L'
			} else {
				command = 'l'
			}
		case 'L', 'l':
			for hasSVGPathNumbers(tokens, index, 2) {
				x := parseSVGFloat(tokens[index])
				y := parseSVGFloat(tokens[index+1])
				index += 2
				if command == 'l' {
					x += currentX
					y += currentY
				}
				out = append(out, svgPathCommand{Op: 'L', X: x, Y: y})
				currentX, currentY = x, y
			}
		case 'H', 'h':
			for hasSVGPathNumbers(tokens, index, 1) {
				x := parseSVGFloat(tokens[index])
				index++
				if command == 'h' {
					x += currentX
				}
				out = append(out, svgPathCommand{Op: 'L', X: x, Y: currentY})
				currentX = x
			}
		case 'V', 'v':
			for hasSVGPathNumbers(tokens, index, 1) {
				y := parseSVGFloat(tokens[index])
				index++
				if command == 'v' {
					y += currentY
				}
				out = append(out, svgPathCommand{Op: 'L', X: currentX, Y: y})
				currentY = y
			}
		case 'C', 'c':
			for hasSVGPathNumbers(tokens, index, 6) {
				x1 := parseSVGFloat(tokens[index])
				y1 := parseSVGFloat(tokens[index+1])
				x2 := parseSVGFloat(tokens[index+2])
				y2 := parseSVGFloat(tokens[index+3])
				x := parseSVGFloat(tokens[index+4])
				y := parseSVGFloat(tokens[index+5])
				index += 6
				if command == 'c' {
					x1 += currentX
					y1 += currentY
					x2 += currentX
					y2 += currentY
					x += currentX
					y += currentY
				}
				out = append(out, svgPathCommand{Op: 'C', X1: x1, Y1: y1, X2: x2, Y2: y2, X: x, Y: y})
				currentX, currentY = x, y
			}
		case 'Q', 'q':
			for hasSVGPathNumbers(tokens, index, 4) {
				x1 := parseSVGFloat(tokens[index])
				y1 := parseSVGFloat(tokens[index+1])
				x := parseSVGFloat(tokens[index+2])
				y := parseSVGFloat(tokens[index+3])
				index += 4
				if command == 'q' {
					x1 += currentX
					y1 += currentY
					x += currentX
					y += currentY
				}
				out = append(out, svgPathCommand{Op: 'Q', X1: x1, Y1: y1, X: x, Y: y})
				currentX, currentY = x, y
			}
		case 'Z', 'z':
			out = append(out, svgPathCommand{Op: 'Z', X: startX, Y: startY})
			currentX, currentY = startX, startY
			command = 0
		default:
			return out
		}
	}
	return out
}

func tokenizeSVGPath(source string) []string {
	var tokens []string
	for index := 0; index < len(source); {
		r := rune(source[index])
		if unicode.IsSpace(r) || source[index] == ',' {
			index++
			continue
		}
		if isSVGPathCommandByte(source[index]) {
			tokens = append(tokens, source[index:index+1])
			index++
			continue
		}
		start := index
		if source[index] == '+' || source[index] == '-' {
			index++
		}
		digitSeen := false
		for index < len(source) && source[index] >= '0' && source[index] <= '9' {
			index++
			digitSeen = true
		}
		if index < len(source) && source[index] == '.' {
			index++
			for index < len(source) && source[index] >= '0' && source[index] <= '9' {
				index++
				digitSeen = true
			}
		}
		if index < len(source) && (source[index] == 'e' || source[index] == 'E') {
			exponent := index
			index++
			if index < len(source) && (source[index] == '+' || source[index] == '-') {
				index++
			}
			exponentDigit := false
			for index < len(source) && source[index] >= '0' && source[index] <= '9' {
				index++
				exponentDigit = true
			}
			if !exponentDigit {
				index = exponent
			}
		}
		if !digitSeen {
			index++
			continue
		}
		tokens = append(tokens, source[start:index])
	}
	return tokens
}

func hasSVGPathNumbers(tokens []string, index int, count int) bool {
	if index+count > len(tokens) {
		return false
	}
	for offset := 0; offset < count; offset++ {
		if isSVGPathCommand(tokens[index+offset]) {
			return false
		}
	}
	return true
}

func isSVGPathCommand(token string) bool {
	return len(token) == 1 && isSVGPathCommandByte(token[0])
}

func isSVGPathCommandByte(value byte) bool {
	return strings.ContainsRune("MmLlHhVvCcQqZz", rune(value))
}

func parseNumberList(source string) []float32 {
	source = strings.ReplaceAll(source, ",", " ")
	fields := strings.Fields(source)
	values := make([]float32, 0, len(fields))
	for _, field := range fields {
		values = append(values, parseSVGFloat(field))
	}
	return values
}

func parseSVGFloat(source string) float32 {
	source = strings.TrimSpace(source)
	source = strings.TrimSuffix(source, "px")
	if source == "" {
		return 0
	}
	value, err := strconv.ParseFloat(source, 32)
	if err != nil {
		return 0
	}
	return float32(value)
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
