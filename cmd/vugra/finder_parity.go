package main

import (
	"bytes"
	"fmt"
	"image/color"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/pkg/system"
)

var rustFinderParityCommand = func() *exec.Cmd {
	return rustFinderCommand("parity-summary")
}

type finderParitySummary struct {
	Phases map[string]*finderParityPhase
}

type finderParityPhase struct {
	Metrics map[string]int
	Rects   map[string]renderer.Rect
	Colors  map[string]string
	Fills   map[string]string
	Borders map[string]string
	Radii   map[string]float32
	Texts   []string
}

func runFinderParitySmoke(args []string) error {
	if len(args) != 0 {
		return usage()
	}
	goSummary, err := goFinderParitySummary()
	if err != nil {
		return err
	}
	rustSummary, rawRust, err := rustFinderParitySummary()
	if err != nil {
		if rawRust != "" {
			return fmt.Errorf("%w\n%s", err, rawRust)
		}
		return err
	}
	if err := compareFinderParity(goSummary, rustSummary); err != nil {
		return err
	}
	goSearch := goSummary.Phases["search"]
	rustSearch := rustSummary.Phases["search"]
	fmt.Printf("finder-parity go commands=%d drawn=%d\n", goSearch.Metrics["commands"], goSearch.Metrics["drawn"])
	fmt.Printf("finder-parity rust commands=%d drawn=%d\n", rustSearch.Metrics["commands"], rustSearch.Metrics["drawn"])
	fmt.Println("finder-parity-smoke ok")
	return nil
}

func goFinderParitySummary() (finderParitySummary, error) {
	fixture := goFinderMockFixture()
	componentPath, err := goFinderLiteComponentPath(".")
	if err != nil {
		return finderParitySummary{}, err
	}
	result, err := compiler.CompileFile(componentPath)
	if err != nil {
		return finderParitySummary{}, fmt.Errorf("compile FinderLite: %w", err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		return finderParitySummary{}, fmt.Errorf("compile FinderLite produced %d diagnostic(s)", len(diagnostics))
	}

	scheduler := reactivity.NewScheduler()
	target := &renderer.TestRenderer{}
	state := newFinderDemoStateWithFileSystem(scheduler, fixture.files, fixture.sidebar).state()
	mounted := app.Mount(result, state, target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    layout.FixedMeasurer{CharWidth: 8, LineHeight: 20},
		Layout:      layoutEngineFromEnv(),
	})

	summary := finderParitySummary{Phases: map[string]*finderParityPhase{}}
	summary.Phases["initial"] = summarizeGoFinderPhase(target.LastFrame())
	firstRow := goFinderElementWithText(target.LastFrame(), "Design")
	if firstRow.ID == "" {
		return finderParitySummary{}, fmt.Errorf("finder-parity go phase missing Design row")
	}
	if !mounted.DispatchPointerEvent(firstRow.Rect.X+2, firstRow.Rect.Y+2, runtime.Modifiers{}) {
		return finderParitySummary{}, fmt.Errorf("finder-parity go failed to dispatch Design click")
	}
	mounted.Flush()
	summary.Phases["selection"] = summarizeGoFinderPhase(target.LastFrame())

	search := goFinderElementByClass(target.LastFrame(), "search")
	if search.ID == "" {
		return finderParitySummary{}, fmt.Errorf("finder-parity go phase missing search textbox")
	}
	if !mounted.DispatchPointer(search.Rect.X+2, search.Rect.Y+2) {
		return finderParitySummary{}, fmt.Errorf("finder-parity go failed to focus search")
	}
	if !mounted.DispatchTextInput("road") {
		return finderParitySummary{}, fmt.Errorf("finder-parity go failed to input search text")
	}
	mounted.Flush()
	summary.Phases["search"] = summarizeGoFinderPhase(target.LastFrame())
	return summary, nil
}

type goFinderMock struct {
	files   system.FileSystem
	sidebar finderSidebar
}

func goFinderMockFixture() goFinderMock {
	entries := []system.Entry{
		{Name: "Documents", Path: "Documents", Kind: "folder"},
		{Name: "Downloads", Path: "Downloads", Kind: "folder"},
		{Name: "Pictures", Path: "Pictures", Kind: "folder"},
		{Name: "Project A", Path: "Project A", Kind: "folder"},
		{Name: "Project B", Path: "Project B", Kind: "folder"},
		{Name: "Design", Path: "Documents/Design", Kind: "folder"},
		{Name: "Roadmap.md", Path: "Documents/Roadmap.md", Kind: "file", Size: 12400},
		{Name: "Budget 2026.xlsx", Path: "Documents/Budget 2026.xlsx", Kind: "file", Size: 842000},
		{Name: "Meeting Notes.txt", Path: "Documents/Meeting Notes.txt", Kind: "file", Size: 17000},
		{Name: "Client Brief.pdf", Path: "Documents/Client Brief.pdf", Kind: "file", Size: 224000},
		{Name: "Contract Draft.docx", Path: "Documents/Contract Draft.docx", Kind: "file", Size: 96000},
		{Name: "Launch Plan.pages", Path: "Documents/Launch Plan.pages", Kind: "file", Size: 410000},
		{Name: "Research Summary.md", Path: "Documents/Research Summary.md", Kind: "file", Size: 38000},
		{Name: "Book Outline.txt", Path: "Documents/Book Outline.txt", Kind: "file", Size: 21000},
		{Name: "Ideas.txt", Path: "Documents/Ideas.txt", Kind: "file", Size: 7000},
		{Name: "Agenda.md", Path: "Documents/Agenda.md", Kind: "file", Size: 8000},
		{Name: "Notes Archive.txt", Path: "Documents/Notes Archive.txt", Kind: "file", Size: 53000},
		{Name: "Components.sketch", Path: "Documents/Design/Components.sketch", Kind: "file", Size: 1900000},
		{Name: "Assets", Path: "Documents/Design/Assets", Kind: "folder"},
		{Name: "Prototype.mov", Path: "Documents/Design/Prototype.mov", Kind: "file", Size: 4800000},
		{Name: "Installer.dmg", Path: "Downloads/Installer.dmg", Kind: "file", Size: 3400000},
		{Name: "Receipts", Path: "Downloads/Receipts", Kind: "folder"},
		{Name: "Archive.zip", Path: "Downloads/Archive.zip", Kind: "file", Size: 721000},
		{Name: "Vacation.jpg", Path: "Pictures/Vacation.jpg", Kind: "file", Size: 2100000},
		{Name: "Screenshots", Path: "Pictures/Screenshots", Kind: "folder"},
		{Name: "Profile.png", Path: "Pictures/Profile.png", Kind: "file", Size: 98000},
	}
	return goFinderMock{
		files: system.NewMockFileSystem(entries),
		sidebar: finderSidebar{
			documents: "Documents",
			downloads: "Downloads",
			pictures:  "Pictures",
			projectA:  "Project A",
			projectB:  "Project B",
		},
	}
}

func summarizeGoFinderPhase(frame []renderer.Command) *finderParityPhase {
	phase := &finderParityPhase{
		Metrics: map[string]int{"commands": len(frame)},
		Rects:   map[string]renderer.Rect{},
		Colors:  map[string]string{},
		Fills:   map[string]string{},
		Borders: map[string]string{},
		Radii:   map[string]float32{},
	}
	software := renderer.NewSoftware(800, 600)
	software.Render(frame)
	phase.Metrics["pixels"] = software.Width * software.Height
	phase.Metrics["drawn"] = goFinderDrawnPixelCountAgainst(software, color.RGBA{R: 250, G: 250, B: 250, A: 255})
	for _, key := range []string{
		"toolbar", "path", "search", "sidebar", "splitter", "file-pane", "file-header", "file-list", "statusbar",
	} {
		if command := goFinderElementByClass(frame, key); command.ID != "" {
			phase.Rects[key] = command.Rect
			phase.Fills[key] = goFinderEffectiveFillColor(command)
			if border := goFinderEffectiveBorderColor(command); border != "" {
				phase.Borders[key] = border
			}
			if radius := command.Style.BorderRadius; radius > 0 {
				phase.Radii[key] = radius
			}
		}
	}
	for _, command := range frame {
		if command.Kind != "text" || command.Text == "" {
			continue
		}
		phase.Texts = append(phase.Texts, command.Text)
		switch {
		case command.Text == "Documents":
			if _, exists := phase.Rects["path:text"]; !exists {
				phase.Rects["path:text"] = command.Rect
			}
			if color := normalizedColor(command.Style.Color); color != "" {
				if _, exists := phase.Colors["path:text"]; !exists {
					phase.Colors["path:text"] = color
				}
			}
		case command.Text == "Name":
			phase.Rects["header-name:text"] = command.Rect
			if color := normalizedColor(command.Style.Color); color != "" {
				phase.Colors["header-name:text"] = color
			}
		case command.Text == "Modified":
			phase.Rects["header-kind:text"] = command.Rect
			if color := normalizedColor(command.Style.Color); color != "" {
				phase.Colors["header-kind:text"] = color
			}
		case command.Text == "Size":
			phase.Rects["header-size:text"] = command.Rect
			if color := normalizedColor(command.Style.Color); color != "" {
				phase.Colors["header-size:text"] = color
			}
		case command.Text == "Search" || command.Text == "road":
			phase.Rects["search:text"] = command.Rect
			phase.Colors["search:text"] = goFinderEffectiveTextColor(command)
		case strings.Contains(command.Text, "items · Current path"):
			phase.Rects["status-text:text"] = command.Rect
			if color := normalizedColor(command.Style.Color); color != "" {
				phase.Colors["status-text:text"] = color
			}
		case strings.Contains(command.Text, "items selected"):
			phase.Rects["selected-summary:text"] = command.Rect
			if color := normalizedColor(command.Style.Color); color != "" {
				phase.Colors["selected-summary:text"] = color
			}
		}
	}
	if command := goFinderFirstRowElement(frame); command.ID != "" {
		phase.Rects["row1"] = command.Rect
		phase.Fills["row1"] = goFinderEffectiveFillColor(command)
		if border := goFinderEffectiveBorderColor(command); border != "" {
			phase.Borders["row1"] = border
		}
		if radius := command.Style.BorderRadius; radius > 0 {
			phase.Radii["row1"] = radius
		}
		if text := goFinderRowNameText(frame, command.Rect); text.ID != "" {
			phase.Rects["row1-name:text"] = text.Rect
			if color := normalizedColor(text.Style.Color); color != "" {
				phase.Colors["row1-name:text"] = color
			}
		}
		if text := goFinderRowColumnText(frame, command.Rect, "modified", "--"); text.ID != "" {
			phase.Rects["row1-modified:text"] = text.Rect
			if color := normalizedColor(text.Style.Color); color != "" {
				phase.Colors["row1-modified:text"] = color
			}
		}
		if text := goFinderRowColumnText(frame, command.Rect, "size", "12 KB"); text.ID != "" {
			phase.Rects["row1-size:text"] = text.Rect
			if color := normalizedColor(text.Style.Color); color != "" {
				phase.Colors["row1-size:text"] = color
			}
		}
	}
	if command := goFinderRowElementByName(frame, "Roadmap.md"); command.ID != "" {
		phase.Rects["row-roadmap"] = command.Rect
		if text := goFinderRowNameText(frame, command.Rect); text.ID != "" {
			phase.Rects["row-roadmap-name:text"] = text.Rect
			if color := normalizedColor(text.Style.Color); color != "" {
				phase.Colors["row-roadmap-name:text"] = color
			}
		}
		if text := goFinderRowColumnText(frame, command.Rect, "size", "12 KB"); text.ID != "" {
			phase.Rects["row-roadmap-size:text"] = text.Rect
			if color := normalizedColor(text.Style.Color); color != "" {
				phase.Colors["row-roadmap-size:text"] = color
			}
		}
	}
	if text := goFinderTextInsideElementWithClass(frame, "tree-item-selected"); text.ID != "" {
		if color := normalizedColor(text.Style.Color); color != "" {
			phase.Colors["sidebar-active-label:text"] = color
		}
		if element := goFinderNearestElementWithClass(frame, "tree-item-selected", text.Rect); element.ID != "" {
			phase.Fills["sidebar-active-item"] = goFinderEffectiveFillColor(element)
		}
	}
	return phase
}

func goFinderEffectiveFillColor(command renderer.Command) string {
	if color := normalizedColor(command.Style.BackgroundColor); color != "" {
		return color
	}
	switch command.Role {
	case "button":
		return "#e5f1ff"
	case "textbox", "checkbox", "listitem":
		return "#ffffff"
	default:
		return "#f8fafc"
	}
}

func goFinderEffectiveBorderColor(command renderer.Command) string {
	if command.Style.BorderWidthSet && command.Style.BorderWidth <= 0 {
		return ""
	}
	if command.Style.BorderWidth <= 0 && !goFinderIsControlElement(command) && command.Style.BorderColor == "" {
		return ""
	}
	if color := normalizedColor(command.Style.BorderColor); color != "" {
		return color
	}
	switch command.Role {
	case "button":
		return "#2563eb"
	case "textbox", "checkbox":
		return "#94a3b8"
	case "listitem":
		return "#cbd5e1"
	default:
		return "#e2e8f0"
	}
}

func goFinderIsControlElement(command renderer.Command) bool {
	return command.Role == "button" ||
		command.Role == "textbox" ||
		command.Role == "checkbox" ||
		command.Tag == "button" ||
		command.Tag == "input"
}

func goFinderEffectiveTextColor(command renderer.Command) string {
	if color := normalizedColor(command.Style.Color); color != "" {
		return color
	}
	if command.Role == "button" {
		return "#2563eb"
	}
	return "#0f172a"
}

func goFinderFirstRowElement(frame []renderer.Command) renderer.Command {
	for _, command := range frame {
		if command.Kind != "element" {
			continue
		}
		class := command.Props["class"]
		if class == "file-row" || strings.HasPrefix(class, "file-row-") {
			return command
		}
	}
	return renderer.Command{}
}

func goFinderRowNameText(frame []renderer.Command, row renderer.Rect) renderer.Command {
	for _, command := range frame {
		if command.Kind != "text" || command.Text == "" {
			continue
		}
		if !rectOriginInside(command.Rect, row) {
			continue
		}
		if command.Text == "Design" || command.Text == "Roadmap.md" {
			return command
		}
	}
	return renderer.Command{}
}

func goFinderRowElementByName(frame []renderer.Command, name string) renderer.Command {
	for _, command := range frame {
		if command.Kind != "text" || command.Text != name {
			continue
		}
		if element := goFinderRowElementContainingRect(frame, command.Rect); element.ID != "" {
			return element
		}
	}
	return renderer.Command{}
}

func goFinderRowElementContainingRect(frame []renderer.Command, textRect renderer.Rect) renderer.Command {
	for _, command := range frame {
		if command.Kind != "element" {
			continue
		}
		class := command.Props["class"]
		if class != "file-row" && !strings.HasPrefix(class, "file-row-") {
			continue
		}
		if rectOriginInside(textRect, command.Rect) {
			return command
		}
	}
	return renderer.Command{}
}

func goFinderRowColumnText(frame []renderer.Command, row renderer.Rect, column string, text string) renderer.Command {
	columnRect := goFinderRowColumnRect(row, column)
	for _, command := range frame {
		if command.Kind != "text" || command.Text != text {
			continue
		}
		if rectOriginInside(command.Rect, columnRect) {
			return command
		}
	}
	return renderer.Command{}
}

func goFinderRowColumnRect(row renderer.Rect, column string) renderer.Rect {
	const (
		rowPadding = float32(6)
		columnGap  = float32(10)
		dateWidth  = float32(150)
		sizeWidth  = float32(90)
	)
	sizeX := row.X + row.Width - rowPadding - sizeWidth
	dateX := sizeX - columnGap - dateWidth
	switch column {
	case "modified":
		return renderer.Rect{X: dateX, Y: row.Y, Width: dateWidth, Height: row.Height}
	case "size":
		return renderer.Rect{X: sizeX, Y: row.Y, Width: sizeWidth, Height: row.Height}
	default:
		return row
	}
}

func goFinderTextInsideElementWithClass(frame []renderer.Command, class string) renderer.Command {
	for i, command := range frame {
		if command.Kind != "text" || command.Text == "" {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if frame[j].Kind != "element" {
				continue
			}
			if frame[j].Props["class"] != class {
				continue
			}
			if rectOriginInside(command.Rect, frame[j].Rect) {
				return command
			}
		}
	}
	return renderer.Command{}
}

func goFinderNearestElementWithClass(frame []renderer.Command, class string, textRect renderer.Rect) renderer.Command {
	for _, command := range frame {
		if command.Kind != "element" || command.Props["class"] != class {
			continue
		}
		if rectOriginInside(textRect, command.Rect) {
			return command
		}
	}
	return renderer.Command{}
}

func rectOriginInside(inner, outer renderer.Rect) bool {
	return inner.X >= outer.X &&
		inner.X <= outer.X+outer.Width &&
		inner.Y >= outer.Y &&
		inner.Y <= outer.Y+outer.Height
}

func rustFinderParitySummary() (finderParitySummary, string, error) {
	cmd := rustFinderParityCommand()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return finderParitySummary{}, stdout.String() + stderr.String(), fmt.Errorf("run rust finder parity-summary: %w", err)
	}
	summary, err := parseFinderParitySummary(stdout.String())
	if err != nil {
		return finderParitySummary{}, stdout.String(), err
	}
	return summary, stdout.String(), nil
}

func parseFinderParitySummary(raw string) (finderParitySummary, error) {
	summary := finderParitySummary{Phases: map[string]*finderParityPhase{}}
	var current *finderParityPhase
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		switch fields[0] {
		case "phase":
			if len(fields) != 2 {
				return summary, fmt.Errorf("invalid parity phase line %q", line)
			}
			current = &finderParityPhase{Metrics: map[string]int{}, Rects: map[string]renderer.Rect{}, Colors: map[string]string{}, Fills: map[string]string{}, Borders: map[string]string{}, Radii: map[string]float32{}}
			summary.Phases[fields[1]] = current
		case "metric":
			if current == nil || len(fields) != 3 {
				return summary, fmt.Errorf("invalid parity metric line %q", line)
			}
			value, err := strconv.Atoi(fields[2])
			if err != nil {
				return summary, fmt.Errorf("parse parity metric %q: %w", line, err)
			}
			current.Metrics[fields[1]] = value
		case "rect":
			if current == nil || len(fields) != 6 {
				return summary, fmt.Errorf("invalid parity rect line %q", line)
			}
			rect, err := parseParityRect(fields[2:])
			if err != nil {
				return summary, fmt.Errorf("parse parity rect %q: %w", line, err)
			}
			current.Rects[fields[1]] = rect
		case "text":
			if current == nil || len(fields) < 2 {
				return summary, fmt.Errorf("invalid parity text line %q", line)
			}
			current.Texts = append(current.Texts, strings.Join(fields[1:], "\t"))
		case "color":
			if current == nil || len(fields) != 3 {
				return summary, fmt.Errorf("invalid parity color line %q", line)
			}
			current.Colors[fields[1]] = normalizedColor(fields[2])
		case "fill":
			if current == nil || len(fields) != 3 {
				return summary, fmt.Errorf("invalid parity fill line %q", line)
			}
			current.Fills[fields[1]] = normalizedColor(fields[2])
		case "border":
			if current == nil || len(fields) != 3 {
				return summary, fmt.Errorf("invalid parity border line %q", line)
			}
			current.Borders[fields[1]] = normalizedColor(fields[2])
		case "radius":
			if current == nil || len(fields) != 3 {
				return summary, fmt.Errorf("invalid parity radius line %q", line)
			}
			value, err := strconv.ParseFloat(fields[2], 32)
			if err != nil {
				return summary, fmt.Errorf("parse parity radius %q: %w", line, err)
			}
			current.Radii[fields[1]] = float32(value)
		default:
		}
	}
	for _, phase := range []string{"initial", "selection", "search"} {
		if summary.Phases[phase] == nil {
			return summary, fmt.Errorf("missing rust parity phase %s", phase)
		}
	}
	return summary, nil
}

func parseParityRect(fields []string) (renderer.Rect, error) {
	values := make([]float32, 4)
	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 32)
		if err != nil {
			return renderer.Rect{}, err
		}
		values[i] = float32(value)
	}
	return renderer.Rect{X: values[0], Y: values[1], Width: values[2], Height: values[3]}, nil
}

func compareFinderParity(goSummary, rustSummary finderParitySummary) error {
	var mismatches []string
	for _, phase := range []string{"initial", "selection", "search"} {
		goPhase := goSummary.Phases[phase]
		rustPhase := rustSummary.Phases[phase]
		if goPhase == nil || rustPhase == nil {
			mismatches = append(mismatches, fmt.Sprintf("missing phase %s", phase))
			continue
		}
		for _, text := range requiredFinderParityTexts(phase) {
			if !stringSliceContains(goPhase.Texts, text) {
				mismatches = append(mismatches, fmt.Sprintf("go %s missing text %q", phase, text))
			}
			if !stringSliceContains(rustPhase.Texts, text) {
				mismatches = append(mismatches, fmt.Sprintf("rust %s missing text %q", phase, text))
			}
		}
		for _, rectKey := range requiredFinderParityRectKeys(phase) {
			goRect, goOK := goPhase.Rects[rectKey]
			rustRect, rustOK := rustPhase.Rects[rectKey]
			if rectKey == "search:text" && !goOK && !rustOK {
				continue
			}
			if !goOK || !rustOK {
				mismatches = append(mismatches, fmt.Sprintf("%s missing rect %s go=%t rust=%t", phase, rectKey, goOK, rustOK))
				continue
			}
			if !finderParityRectsClose(rectKey, goRect, rustRect) {
				mismatches = append(mismatches, fmt.Sprintf("%s rect %s differs: go=%+v rust=%+v", phase, rectKey, goRect, rustRect))
			}
		}
		for _, colorKey := range requiredFinderParityColorKeys(phase) {
			goColor, goOK := goPhase.Colors[colorKey]
			rustColor, rustOK := rustPhase.Colors[colorKey]
			if !goOK || !rustOK {
				mismatches = append(mismatches, fmt.Sprintf("%s missing color %s go=%t rust=%t", phase, colorKey, goOK, rustOK))
				continue
			}
			if goColor != rustColor {
				mismatches = append(mismatches, fmt.Sprintf("%s color %s differs: go=%s rust=%s", phase, colorKey, goColor, rustColor))
			}
		}
		for _, fillKey := range requiredFinderParityFillKeys(phase) {
			goFill, goOK := goPhase.Fills[fillKey]
			rustFill, rustOK := rustPhase.Fills[fillKey]
			if !goOK || !rustOK {
				mismatches = append(mismatches, fmt.Sprintf("%s missing fill %s go=%t rust=%t", phase, fillKey, goOK, rustOK))
				continue
			}
			if goFill != rustFill {
				mismatches = append(mismatches, fmt.Sprintf("%s fill %s differs: go=%s rust=%s", phase, fillKey, goFill, rustFill))
			}
		}
		for _, borderKey := range requiredFinderParityBorderKeys(phase) {
			goBorder, goOK := goPhase.Borders[borderKey]
			rustBorder, rustOK := rustPhase.Borders[borderKey]
			if !goOK || !rustOK {
				mismatches = append(mismatches, fmt.Sprintf("%s missing border %s go=%t rust=%t", phase, borderKey, goOK, rustOK))
				continue
			}
			if goBorder != rustBorder {
				mismatches = append(mismatches, fmt.Sprintf("%s border %s differs: go=%s rust=%s", phase, borderKey, goBorder, rustBorder))
			}
		}
		for _, radiusKey := range requiredFinderParityRadiusKeys(phase) {
			goRadius, goOK := goPhase.Radii[radiusKey]
			rustRadius, rustOK := rustPhase.Radii[radiusKey]
			if !goOK || !rustOK {
				mismatches = append(mismatches, fmt.Sprintf("%s missing radius %s go=%t rust=%t", phase, radiusKey, goOK, rustOK))
				continue
			}
			if abs32(goRadius-rustRadius) > 0.1 {
				mismatches = append(mismatches, fmt.Sprintf("%s radius %s differs: go=%.1f rust=%.1f", phase, radiusKey, goRadius, rustRadius))
			}
		}
		if goPhase.Metrics["drawn"] == 0 || rustPhase.Metrics["drawn"] == 0 {
			mismatches = append(mismatches, fmt.Sprintf("%s produced blank frame: go=%d rust=%d", phase, goPhase.Metrics["drawn"], rustPhase.Metrics["drawn"]))
		}
	}
	if stringSliceContains(goSummary.Phases["search"].Texts, "Budget 2026.xlsx") {
		mismatches = append(mismatches, "go search still contains Budget 2026.xlsx")
	}
	if stringSliceContains(rustSummary.Phases["search"].Texts, "Budget 2026.xlsx") {
		mismatches = append(mismatches, "rust search still contains Budget 2026.xlsx")
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("finder-parity mismatches:\n%s", strings.Join(mismatches, "\n"))
	}
	return nil
}

func requiredFinderParityRadiusKeys(phase string) []string {
	switch phase {
	case "initial", "selection", "search":
		return []string{"path", "search", "row1"}
	default:
		return nil
	}
}

func requiredFinderParityRectKeys(phase string) []string {
	common := []string{"toolbar", "path", "path:text", "search", "search:text", "sidebar", "splitter", "file-pane", "file-header", "file-list", "header-name:text", "header-kind:text", "header-size:text", "statusbar", "status-text:text", "selected-summary:text", "row1", "row1-name:text"}
	switch phase {
	case "initial", "selection":
		return append(common, "row-roadmap", "row-roadmap-name:text", "row-roadmap-size:text")
	case "search":
		return append(common, "row-roadmap-size:text")
	default:
		return common
	}
}

func requiredFinderParityBorderKeys(phase string) []string {
	switch phase {
	case "initial", "selection":
		return []string{"toolbar", "path", "search", "sidebar", "file-header", "statusbar"}
	case "search":
		return []string{"toolbar", "path", "search", "sidebar", "file-header", "statusbar", "row1"}
	default:
		return nil
	}
}

func requiredFinderParityFillKeys(phase string) []string {
	switch phase {
	case "initial", "selection", "search":
		return []string{"toolbar", "path", "search", "sidebar", "sidebar-active-item", "splitter", "file-pane", "file-header", "file-list", "statusbar", "row1"}
	default:
		return nil
	}
}

func requiredFinderParityColorKeys(phase string) []string {
	switch phase {
	case "initial", "selection":
		return []string{"path:text", "header-name:text", "header-kind:text", "header-size:text", "sidebar-active-label:text", "status-text:text", "selected-summary:text", "row1-name:text", "row-roadmap-name:text", "row-roadmap-size:text"}
	case "search":
		return []string{"path:text", "search:text", "header-name:text", "header-kind:text", "header-size:text", "sidebar-active-label:text", "status-text:text", "selected-summary:text", "row1-name:text", "row-roadmap-size:text"}
	default:
		return nil
	}
}

func normalizedColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "#") && len(value) == 4 {
		return fmt.Sprintf("#%c%c%c%c%c%c", value[1], value[1], value[2], value[2], value[3], value[3])
	}
	return value
}

func requiredFinderParityTexts(phase string) []string {
	switch phase {
	case "initial":
		return []string{"Documents", "Design", "Roadmap.md", "12 items · Current path: Documents"}
	case "selection":
		return []string{"Documents", "Design", "Roadmap.md", "1 items selected"}
	case "search":
		return []string{"Documents", "Roadmap.md", "road", "1 items · Current path: Documents", "1 items selected"}
	default:
		return nil
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want || strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func rectsClose(a, b renderer.Rect, tolerance float32) bool {
	return abs32(a.X-b.X) <= tolerance &&
		abs32(a.Y-b.Y) <= tolerance &&
		abs32(a.Width-b.Width) <= tolerance &&
		abs32(a.Height-b.Height) <= tolerance
}

func finderParityRectsClose(key string, a, b renderer.Rect) bool {
	if strings.HasSuffix(key, ":text") {
		return abs32(a.X-b.X) <= 1.0 &&
			abs32(a.Y-b.Y) <= 1.0 &&
			abs32(a.Height-b.Height) <= 2.0
	}
	return rectsClose(a, b, 1.0)
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func goFinderDrawnPixelCountAgainst(r *renderer.SoftwareRenderer, background color.RGBA) int {
	if r.Image == nil {
		return 0
	}
	count := 0
	for y := 0; y < r.Image.Bounds().Dy(); y++ {
		for x := 0; x < r.Image.Bounds().Dx(); x++ {
			if r.Image.RGBAAt(x, y) != background {
				count++
			}
		}
	}
	return count
}
