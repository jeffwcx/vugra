package runtime

// Package runtime mounts Vugra component IR, evaluates bindings, dispatches
// events, and schedules updates through the reactivity layer.

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/scene"
	"github.com/vugra/vugra/internal/style"
)

type LayoutEngine string

const (
	LayoutEngineGo  LayoutEngine = ""
	LayoutEngineCSS LayoutEngine = "css"
)

type Signal = ir.Signal

type SignalValue[T any] struct {
	signal *reactivity.Value[T]
}

func NewSignal[T any](initial T, scheduler *reactivity.Scheduler) *SignalValue[T] {
	return &SignalValue[T]{signal: reactivity.NewWithScheduler(initial, scheduler)}
}

func (s *SignalValue[T]) Get() T {
	return s.signal.Get()
}

func (s *SignalValue[T]) Set(value T) {
	s.signal.Set(value)
}

func (s *SignalValue[T]) Update(update func(T) T) {
	s.signal.Update(update)
}

func (s *SignalValue[T]) GetAny() any {
	return s.signal.Get()
}

func (s *SignalValue[T]) SetAny(value any) {
	if typed, ok := value.(T); ok {
		s.signal.Set(typed)
	}
}

func (s *SignalValue[T]) Subscribe(fn func()) func() {
	return s.signal.Subscribe(fn)
}

type State = ir.RuntimeState

type SystemTokens = style.SystemTokens

type Modifiers = ir.Modifiers

type Event = ir.Event

type App struct {
	component            *ir.Component
	styles               *style.Stylesheet
	state                State
	renderer             renderer.Renderer
	scheduler            *reactivity.Scheduler
	effect               *reactivity.Effect
	scene                scene.Scene
	previousScene        scene.Previous
	events               map[string]func()
	eventMethods         map[string]func(Event)
	eventRects           map[string]layout.Rect
	hitEvents            map[string][]string
	hitTree              hitTree
	focusable            []string
	modalFocusable       []string
	focused              string
	dragTarget           string
	inputBindings        map[string]string
	inputStates          map[string]*State
	inputParentBindings  map[string]map[string]string
	inputSelections      map[string]textSelection
	toggleBindings       map[string]string
	toggleStates         map[string]*State
	toggleParentBindings map[string]map[string]string
	scrollOffsets        map[string]float32
	scrollContainers     []scrollContainer
	viewportScroll       float32
	viewportScrollMax    float32
	documentSelection    textSelection
	hasDocumentSelection bool
	selectingDocument    bool
	documentTextRuns     []documentTextRun
	layoutConstraints    layout.Constraints
	measurer             layout.Measurer
	systemTokens         style.SystemTokens
	styleCSS             string
	assetBase            string
	layoutEngine         LayoutEngine
	lastFrame            []renderer.Command
	mounted              bool
	componentStates      map[*ir.ComponentInstance]*State
}

type scrollContainer struct {
	id        string
	rect      layout.Rect
	maxOffset float32
}

type textSelection struct {
	Start int
	End   int
}

type documentTextRun struct {
	id    string
	text  string
	lines []renderer.LineBox
	rect  renderer.Rect
	start int
	end   int
	style renderer.Style
}

// TextSelection describes a renderer-neutral text selection inside a Vugra text
// input. Offsets are rune offsets, not bytes or browser DOM positions.
type TextSelection struct {
	ID      string
	Binding string
	Start   int
	End     int
}

func (s TextSelection) Collapsed() bool {
	return s.Start == s.End
}

func (s TextSelection) Caret() int {
	return s.End
}

type hitTree struct {
	viewport layout.Rect
	roots    []hitNode
}

type hitNode struct {
	id            string
	rect          layout.Rect
	eventIDs      []string
	inputBinding  string
	toggleBinding string
	scrollMax     float32
	children      []hitNode
}

func buildHitTree(commands []renderer.Command, events map[string][]string, inputs map[string]string, toggles map[string]string, scrolls []scrollContainer, viewport layout.Rect) hitTree {
	scrollByID := map[string]float32{}
	for _, scroll := range scrolls {
		scrollByID[scroll.id] = scroll.maxOffset
	}
	tree := hitTree{viewport: viewport}
	stack := []*hitNode{}
	for _, command := range commands {
		switch command.Kind {
		case "element":
			node := hitNode{
				id:            command.ID,
				rect:          layout.Rect(command.Rect),
				eventIDs:      append([]string(nil), events[command.ID]...),
				inputBinding:  inputs[command.ID],
				toggleBinding: toggles[command.ID],
				scrollMax:     scrollByID[command.ID],
			}
			if len(stack) == 0 {
				tree.roots = append(tree.roots, node)
				stack = append(stack, &tree.roots[len(tree.roots)-1])
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
				stack = append(stack, &parent.children[len(parent.children)-1])
			}
		case "end":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return tree
}

func (t hitTree) hit(x, y float32) (hitNode, bool) {
	if t.viewport.Width > 0 && t.viewport.Height > 0 && !contains(t.viewport, x, y) {
		return hitNode{}, false
	}
	for i := len(t.roots) - 1; i >= 0; i-- {
		if node, ok := t.roots[i].hit(x, y); ok {
			return node, true
		}
	}
	return hitNode{}, false
}

func (n hitNode) hit(x, y float32) (hitNode, bool) {
	if !contains(n.rect, x, y) {
		return hitNode{}, false
	}
	for i := len(n.children) - 1; i >= 0; i-- {
		if child, ok := n.children[i].hit(x, y); ok {
			return child, true
		}
	}
	if len(n.eventIDs) > 0 || n.inputBinding != "" || n.toggleBinding != "" {
		return n, true
	}
	return hitNode{}, false
}

func (t hitTree) scrollHit(x, y float32) (hitNode, bool) {
	if t.viewport.Width > 0 && t.viewport.Height > 0 && !contains(t.viewport, x, y) {
		return hitNode{}, false
	}
	for i := len(t.roots) - 1; i >= 0; i-- {
		if node, ok := t.roots[i].scrollHit(x, y); ok {
			return node, true
		}
	}
	return hitNode{}, false
}

func (n hitNode) scrollHit(x, y float32) (hitNode, bool) {
	if !contains(n.rect, x, y) {
		return hitNode{}, false
	}
	for i := len(n.children) - 1; i >= 0; i-- {
		if child, ok := n.children[i].scrollHit(x, y); ok {
			return child, true
		}
	}
	if n.scrollMax > 0 {
		return n, true
	}
	return hitNode{}, false
}

func Mount(component *ir.Component, state State, r renderer.Renderer, scheduler *reactivity.Scheduler) *App {
	return MountWithOptions(component, state, r, Options{Scheduler: scheduler})
}

type Options struct {
	Scheduler    *reactivity.Scheduler
	Styles       *style.Stylesheet
	StyleCSS     string
	AssetBase    string
	Constraints  layout.Constraints
	Measurer     layout.Measurer
	Layout       LayoutEngine
	SystemTokens style.SystemTokens
}

type SystemTokenProvider interface {
	SystemTokens() style.SystemTokens
}

func MountWithOptions(component *ir.Component, state State, r renderer.Renderer, options Options) *App {
	scheduler := options.Scheduler
	if scheduler == nil {
		scheduler = reactivity.DefaultScheduler
	}
	app := &App{
		component:            component,
		styles:               options.Styles,
		state:                state,
		renderer:             r,
		scheduler:            scheduler,
		styleCSS:             options.StyleCSS,
		assetBase:            options.AssetBase,
		layoutEngine:         options.Layout,
		systemTokens:         mergeSystemTokens(options.SystemTokens, systemTokensFromRenderer(r)),
		events:               map[string]func(){},
		eventMethods:         map[string]func(Event){},
		eventRects:           map[string]layout.Rect{},
		hitEvents:            map[string][]string{},
		inputBindings:        map[string]string{},
		inputStates:          map[string]*State{},
		inputParentBindings:  map[string]map[string]string{},
		inputSelections:      map[string]textSelection{},
		toggleBindings:       map[string]string{},
		toggleStates:         map[string]*State{},
		toggleParentBindings: map[string]map[string]string{},
		scrollOffsets:        map[string]float32{},
		focusable:            nil,
		componentStates:      map[*ir.ComponentInstance]*State{},
	}
	app.layoutConstraints = options.Constraints
	app.measurer = options.Measurer
	app.effect = reactivity.NewEffectWithScheduler(app.render, scheduler)
	return app
}

func (a *App) Dispatch(id string) bool {
	handler, ok := a.events[id]
	if !ok {
		return false
	}
	handler()
	return true
}

func (a *App) DispatchPointer(x, y float32) bool {
	return a.DispatchPointerEvent(x, y, Modifiers{})
}

func (a *App) DispatchPointerEvent(x, y float32, modifiers Modifiers) bool {
	target, ok := a.hitTree.hit(x, y)
	if !ok {
		a.dragTarget = ""
		return a.beginDocumentSelection(x, y)
	}
	a.dragTarget = firstTargetEvent(target.eventIDs, "drag")
	if target.inputBinding != "" {
		a.clearDocumentSelection()
		a.focus(target.id)
		return true
	}
	if target.toggleBinding != "" {
		a.clearDocumentSelection()
		a.focus(target.id)
		a.toggle(target.id, target.toggleBinding)
		return true
	}
	if len(target.eventIDs) > 0 {
		a.clearDocumentSelection()
		eventID := firstTargetEvent(target.eventIDs, "click")
		a.focus(eventID)
		return a.DispatchEvent(eventID, Event{Type: "click", X: x, Y: y, Modifiers: modifiers})
	}
	return a.selectingDocument
}

func (a *App) DispatchHover(x, y float32, modifiers Modifiers) bool {
	return a.dispatchPointerNamedEvent(x, y, "hover", modifiers)
}

func (a *App) DispatchDrag(x, y, deltaX, deltaY float32, modifiers Modifiers) bool {
	if a.updateDocumentSelection(x, y) {
		return true
	}
	eventID := a.dragTarget
	if eventID == "" {
		target, ok := a.hitTree.hit(x, y)
		if !ok {
			return false
		}
		eventID = firstTargetEvent(target.eventIDs, "drag")
	}
	if eventID == "" {
		return false
	}
	return a.DispatchEvent(eventID, Event{Type: "drag", X: x, Y: y, DeltaX: deltaX, DeltaY: deltaY, Modifiers: modifiers})
}

func (a *App) DispatchContextMenu(x, y float32, modifiers Modifiers) bool {
	return a.dispatchPointerNamedEvent(x, y, "contextmenu", modifiers)
}

func (a *App) DispatchDoubleClick(x, y float32, modifiers Modifiers) bool {
	return a.dispatchPointerNamedEvent(x, y, "dblclick", modifiers)
}

func (a *App) dispatchPointerNamedEvent(x, y float32, event string, modifiers Modifiers) bool {
	target, ok := a.hitTree.hit(x, y)
	if !ok {
		return false
	}
	eventID := firstTargetEvent(target.eventIDs, event)
	if eventID == "" {
		return false
	}
	a.focus(eventID)
	return a.DispatchEvent(eventID, Event{Type: event, X: x, Y: y, Modifiers: modifiers})
}

func (a *App) DispatchEvent(id string, event Event) bool {
	if handler := a.eventMethods[id]; handler != nil {
		handler(event)
		return true
	}
	return a.Dispatch(id)
}

func (a *App) HitTest(x, y float32) (string, bool) {
	target, ok := a.hitTree.hit(x, y)
	if !ok {
		return "", false
	}
	if target.inputBinding != "" || target.toggleBinding != "" {
		return target.id, true
	}
	if len(target.eventIDs) > 0 {
		return target.eventIDs[0], true
	}
	return target.id, true
}

func (a *App) DispatchScroll(x, y, deltaY float32) bool {
	if deltaY == 0 {
		return false
	}
	target, ok := a.hitTree.scrollHit(x, y)
	if ok && target.scrollMax > 0 {
		current := a.scrollOffsets[target.id]
		next := clampScrollOffset(current+deltaY, target.scrollMax)
		if next != current {
			a.scrollOffsets[target.id] = next
			a.scheduler.Schedule(a.render)
			return true
		}
	}
	return a.dispatchViewportScroll(deltaY)
}

func (a *App) dispatchViewportScroll(deltaY float32) bool {
	if a.viewportScrollMax <= 0 {
		return false
	}
	current := a.viewportScroll
	next := clampScrollOffset(current+deltaY, a.viewportScrollMax)
	if next == current {
		return false
	}
	a.viewportScroll = next
	a.scheduler.Schedule(a.render)
	return true
}

func (a *App) FocusNext() bool {
	if len(a.focusable) == 0 {
		return false
	}
	focusable := a.activeFocusables()
	if a.focused == "" {
		a.focus(focusable[0])
		return true
	}
	for i, id := range focusable {
		if id == a.focused {
			a.focus(focusable[(i+1)%len(focusable)])
			return true
		}
	}
	a.focus(focusable[0])
	return true
}

func (a *App) FocusPrevious() bool {
	if len(a.focusable) == 0 {
		return false
	}
	focusable := a.activeFocusables()
	if a.focused == "" {
		a.focus(focusable[len(focusable)-1])
		return true
	}
	for i, id := range focusable {
		if id == a.focused {
			a.focus(focusable[(i+len(focusable)-1)%len(focusable)])
			return true
		}
	}
	a.focus(focusable[len(focusable)-1])
	return true
}

func (a *App) FocusID(id string) bool {
	if id == "" {
		return false
	}
	for _, focusable := range a.activeFocusables() {
		if focusable == id {
			a.focus(id)
			return true
		}
		if focusedID, _, ok := strings.Cut(focusable, ":"); ok && focusedID == id {
			a.focus(focusable)
			return true
		}
	}
	return false
}

func (a *App) activeFocusables() []string {
	if len(a.modalFocusable) > 0 {
		return a.modalFocusable
	}
	return a.focusable
}

func (a *App) DispatchKey(key string) bool {
	switch key {
	case "Shift+Tab":
		return a.FocusPrevious()
	case "Tab":
		return a.FocusNext()
	case "Backspace":
		return a.deleteTextBackward()
	case "Delete":
		if a.deleteTextForward() {
			return true
		}
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "ArrowLeft":
		if a.moveTextCaret(-1) {
			return true
		}
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "ArrowRight":
		if a.moveTextCaret(1) {
			return true
		}
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "Home":
		if a.moveTextCaretTo(0) {
			return true
		}
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "End":
		if a.moveTextCaretToEnd() {
			return true
		}
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "Mod+A":
		return a.selectFocusedText()
	case "ArrowUp", "ArrowDown", "Escape":
		return a.dispatchKeyToFocusedOrGlobal(key)
	case "Enter", " ":
		if a.focused != "" {
			if binding := a.toggleBindings[a.focused]; binding != "" {
				a.toggle(a.focused, binding)
				return true
			}
			if a.DispatchEvent(a.focused, Event{Type: "key", Key: key}) {
				return true
			}
			return a.dispatchFocusedElementKey(key)
		}
	}
	return false
}

func (a *App) dispatchKeyToFocusedOrGlobal(key string) bool {
	if a.focused != "" {
		if a.DispatchEvent(a.focused, Event{Type: "key", Key: key}) {
			return true
		}
		if a.dispatchFocusedElementKey(key) {
			return true
		}
	}
	return a.dispatchGlobalKey(key)
}

func (a *App) focus(id string) {
	changed := a.focused != id
	a.focused = id
	if binding := a.inputBindings[id]; binding != "" {
		a.ensureInputSelection(id, binding, changed)
	}
}

func (a *App) dispatchFocusedElementKey(key string) bool {
	if a.focused == "" {
		return false
	}
	for _, eventID := range a.hitEvents[a.focused] {
		if a.DispatchEvent(eventID, Event{Type: "key", Key: key}) {
			return true
		}
	}
	return false
}

func (a *App) dispatchGlobalKey(key string) bool {
	if handler := a.state.EventMethods[key]; handler != nil {
		handler(Event{Type: "key", Key: key})
		return true
	}
	if handler := a.state.Methods[key]; handler != nil {
		handler()
		return true
	}
	return false
}

func (a *App) deleteTextBackward() bool {
	id, binding, signal, current, ok := a.focusedTextInput()
	if !ok {
		return false
	}
	runes := []rune(current)
	selection := a.clampedInputSelection(id, binding)
	if selection.Start != selection.End {
		next := string(runes[:selection.Start]) + string(runes[selection.End:])
		a.setInputString(id, binding, signal, next, selection.Start)
		return true
	}
	if selection.Start == 0 {
		return false
	}
	next := string(runes[:selection.Start-1]) + string(runes[selection.End:])
	a.setInputString(id, binding, signal, next, selection.Start-1)
	return true
}

func (a *App) deleteTextForward() bool {
	id, binding, signal, current, ok := a.focusedTextInput()
	if !ok {
		return false
	}
	runes := []rune(current)
	selection := a.clampedInputSelection(id, binding)
	if selection.Start != selection.End {
		next := string(runes[:selection.Start]) + string(runes[selection.End:])
		a.setInputString(id, binding, signal, next, selection.Start)
		return true
	}
	if selection.Start >= len(runes) {
		return false
	}
	next := string(runes[:selection.Start]) + string(runes[selection.Start+1:])
	a.setInputString(id, binding, signal, next, selection.Start)
	return true
}

func (a *App) moveTextCaret(delta int) bool {
	if a.focused == "" {
		return false
	}
	binding := a.inputBindings[a.focused]
	if binding == "" {
		return false
	}
	selection := a.clampedInputSelection(a.focused, binding)
	if selection.Start != selection.End {
		if delta < 0 {
			selection.End = selection.Start
		} else {
			selection.Start = selection.End
		}
		a.inputSelections[a.focused] = selection
		return true
	}
	length := len([]rune(a.inputValue(a.focused, binding)))
	next := clampInt(selection.Start+delta, 0, length)
	if next == selection.Start {
		return false
	}
	a.inputSelections[a.focused] = textSelection{Start: next, End: next}
	return true
}

func (a *App) moveTextCaretToEnd() bool {
	if a.focused == "" {
		return false
	}
	binding := a.inputBindings[a.focused]
	if binding == "" {
		return false
	}
	return a.moveTextCaretTo(len([]rune(a.inputValue(a.focused, binding))))
}

func (a *App) moveTextCaretTo(offset int) bool {
	if a.focused == "" {
		return false
	}
	binding := a.inputBindings[a.focused]
	if binding == "" {
		return false
	}
	length := len([]rune(a.inputValue(a.focused, binding)))
	next := clampInt(offset, 0, length)
	selection := a.clampedInputSelection(a.focused, binding)
	if selection.Start == next && selection.End == next {
		return false
	}
	a.inputSelections[a.focused] = textSelection{Start: next, End: next}
	return true
}

func (a *App) selectFocusedText() bool {
	if a.focused == "" {
		return false
	}
	binding := a.inputBindings[a.focused]
	if binding == "" {
		if a.DispatchEvent(a.focused, Event{Type: "key", Key: "Mod+A"}) {
			return true
		}
		if a.dispatchFocusedElementKey("Mod+A") {
			return true
		}
		return a.dispatchGlobalKey("Mod+A")
	}
	length := len([]rune(a.inputValue(a.focused, binding)))
	a.inputSelections[a.focused] = textSelection{Start: 0, End: length}
	return true
}

func (a *App) DispatchText(text string) bool {
	id, binding, signal, _, ok := a.focusedTextInput()
	if !ok {
		return false
	}
	signal.SetAny(text)
	a.syncParentBinding(id, binding, text)
	caret := len([]rune(text))
	a.inputSelections[id] = textSelection{Start: caret, End: caret}
	return true
}

func (a *App) DispatchTextInput(text string) bool {
	id, binding, signal, current, ok := a.focusedTextInput()
	if !ok {
		return false
	}
	runes := []rune(current)
	selection := a.clampedInputSelection(id, binding)
	inserted := len([]rune(text))
	next := string(runes[:selection.Start]) + text + string(runes[selection.End:])
	a.setInputString(id, binding, signal, next, selection.Start+inserted)
	return true
}

func (a *App) TextSelection() (TextSelection, bool) {
	if a.focused == "" {
		return TextSelection{}, false
	}
	return a.TextSelectionFor(a.focused)
}

func (a *App) TextSelectionFor(id string) (TextSelection, bool) {
	binding := a.inputBindings[id]
	if binding == "" {
		return TextSelection{}, false
	}
	state := a.stateForInput(id)
	signal := state.Signals[binding]
	if signal == nil {
		return TextSelection{}, false
	}
	if _, ok := signal.GetAny().(string); !ok {
		return TextSelection{}, false
	}
	selection := a.clampedInputSelection(id, binding)
	return TextSelection{ID: id, Binding: binding, Start: selection.Start, End: selection.End}, true
}

func (a *App) SetTextSelection(selection TextSelection) bool {
	id := selection.ID
	if id == "" {
		id = a.focused
	}
	if id == "" {
		return false
	}
	binding := a.inputBindings[id]
	if binding == "" {
		return false
	}
	if selection.Binding != "" && selection.Binding != binding {
		return false
	}
	state := a.stateForInput(id)
	signal := state.Signals[binding]
	if signal == nil {
		return false
	}
	current, ok := signal.GetAny().(string)
	if !ok {
		return false
	}
	length := len([]rune(current))
	clamped := clampSelection(textSelection{Start: selection.Start, End: selection.End}, length)
	a.focus(id)
	a.inputSelections[id] = clamped
	return true
}

func (a *App) CollapseTextSelection(offset int) bool {
	if a.focused == "" {
		return false
	}
	return a.CollapseTextSelectionFor(a.focused, offset)
}

func (a *App) CollapseTextSelectionFor(id string, offset int) bool {
	binding := a.inputBindings[id]
	if binding == "" {
		return false
	}
	state := a.stateForInput(id)
	signal := state.Signals[binding]
	if signal == nil {
		return false
	}
	current, ok := signal.GetAny().(string)
	if !ok {
		return false
	}
	caret := clampInt(offset, 0, len([]rune(current)))
	a.focus(id)
	a.inputSelections[id] = textSelection{Start: caret, End: caret}
	return true
}

func (a *App) SelectedText() (string, bool) {
	selection, ok := a.TextSelection()
	if !ok {
		return "", false
	}
	return a.selectedText(selection)
}

func (a *App) SelectedTextFor(id string) (string, bool) {
	selection, ok := a.TextSelectionFor(id)
	if !ok {
		return "", false
	}
	return a.selectedText(selection)
}

func (a *App) DocumentTextSelection() (TextSelection, bool) {
	if !a.hasDocumentSelection {
		return TextSelection{}, false
	}
	selection := clampSelection(a.documentSelection, a.documentTextLength())
	if selection.Start == selection.End {
		return TextSelection{}, false
	}
	a.documentSelection = selection
	return TextSelection{ID: "document", Start: selection.Start, End: selection.End}, true
}

func (a *App) SelectedDocumentText() (string, bool) {
	selection, ok := a.DocumentTextSelection()
	if !ok {
		return "", false
	}
	var b strings.Builder
	for _, run := range a.documentTextRuns {
		if run.end <= selection.Start || run.start >= selection.End {
			continue
		}
		start := maxInt(0, selection.Start-run.start)
		end := minInt(len([]rune(run.text)), selection.End-run.start)
		if start >= end {
			continue
		}
		b.WriteString(string([]rune(run.text)[start:end]))
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

func (a *App) ClearDocumentTextSelection() bool {
	if !a.hasDocumentSelection && !a.selectingDocument {
		return false
	}
	a.hasDocumentSelection = false
	a.selectingDocument = false
	a.documentSelection = textSelection{}
	a.scheduler.Schedule(a.render)
	return true
}

func (a *App) selectedText(selection TextSelection) (string, bool) {
	current := a.inputValue(selection.ID, selection.Binding)
	runes := []rune(current)
	if selection.Start == selection.End {
		return "", true
	}
	return string(runes[selection.Start:selection.End]), true
}

func (a *App) focusedTextInput() (string, string, Signal, string, bool) {
	if a.focused == "" {
		return "", "", nil, "", false
	}
	binding := a.inputBindings[a.focused]
	if binding == "" {
		return "", "", nil, "", false
	}
	signal := a.stateForInput(a.focused).Signals[binding]
	if signal == nil {
		return "", "", nil, "", false
	}
	current, ok := signal.GetAny().(string)
	if !ok {
		return "", "", nil, "", false
	}
	a.ensureInputSelection(a.focused, binding, false)
	return a.focused, binding, signal, current, true
}

func (a *App) inputValue(id, binding string) string {
	signal := a.stateForInput(id).Signals[binding]
	if signal == nil {
		return ""
	}
	value, _ := signal.GetAny().(string)
	return value
}

func (a *App) ensureInputSelection(id, binding string, reset bool) {
	length := len([]rune(a.inputValue(id, binding)))
	if reset {
		a.inputSelections[id] = textSelection{Start: length, End: length}
		return
	}
	selection, ok := a.inputSelections[id]
	if !ok {
		a.inputSelections[id] = textSelection{Start: length, End: length}
		return
	}
	a.inputSelections[id] = clampSelection(selection, length)
}

func (a *App) clampedInputSelection(id, binding string) textSelection {
	a.ensureInputSelection(id, binding, false)
	selection := a.inputSelections[id]
	length := len([]rune(a.inputValue(id, binding)))
	selection = clampSelection(selection, length)
	a.inputSelections[id] = selection
	return selection
}

func (a *App) setInputString(id, binding string, signal Signal, next string, caret int) {
	signal.SetAny(next)
	a.syncParentBinding(id, binding, next)
	caret = clampInt(caret, 0, len([]rune(next)))
	a.inputSelections[id] = textSelection{Start: caret, End: caret}
}

func (a *App) toggle(id, binding string) {
	state := a.state
	if id != "" {
		state = a.stateForToggle(id)
	}
	signal := state.Signals[binding]
	if signal == nil {
		return
	}
	current, _ := signal.GetAny().(bool)
	next := !current
	signal.SetAny(next)
	a.syncToggleParentBinding(id, binding, next)
}

func (a *App) syncParentBinding(id, binding string, value any) {
	parentBinding := a.inputParentBindings[id][binding]
	if parentBinding == "" {
		return
	}
	if signal := a.state.Signals[parentBinding]; signal != nil {
		signal.SetAny(value)
	}
}

func (a *App) syncToggleParentBinding(id, binding string, value any) {
	parentBinding := a.toggleParentBindings[id][binding]
	if parentBinding == "" {
		return
	}
	if signal := a.state.Signals[parentBinding]; signal != nil {
		signal.SetAny(value)
	}
}

func (a *App) stateForInput(id string) State {
	if state := a.inputStates[id]; state != nil {
		return *state
	}
	return a.state
}

func (a *App) stateForToggle(id string) State {
	if state := a.toggleStates[id]; state != nil {
		return *state
	}
	return a.state
}

func (a *App) FocusedEvent() string {
	return a.focused
}

func (a *App) FocusedID() string {
	focused, _, _ := strings.Cut(a.focused, ":")
	return focused
}

func (a *App) LastFrame() []renderer.Command {
	frame := make([]renderer.Command, len(a.lastFrame))
	copy(frame, a.lastFrame)
	return frame
}

func (a *App) Flush() {
	a.scheduler.Flush()
}

func (a *App) Unmount() {
	if !a.mounted {
		return
	}
	a.callComponentLifecycle("beforeUnmount")
	a.callLifecycle("beforeUnmount")
	if a.effect != nil {
		a.effect.Stop()
	}
	a.mounted = false
	a.callLifecycle("unmounted")
	a.callComponentLifecycle("unmounted")
}

func (a *App) Resize(width, height float32) {
	if width <= 0 {
		width = a.layoutConstraints.Width
	}
	if height <= 0 {
		height = a.layoutConstraints.Height
	}
	if a.layoutConstraints.Width == width && a.layoutConstraints.Height == height {
		return
	}
	a.layoutConstraints.Width = width
	a.layoutConstraints.Height = height
	if a.viewportScroll > 0 && a.viewportScroll > a.viewportScrollMax {
		a.viewportScroll = a.viewportScrollMax
	}
	a.scheduler.Schedule(a.render)
}

func (a *App) SetSystemTokens(tokens style.SystemTokens) {
	next := mergeSystemTokens(tokens, nil)
	if systemTokensEqual(a.systemTokens, next) {
		return
	}
	a.systemTokens = next
	a.scheduler.Schedule(a.render)
}

func (a *App) render() {
	initial := !a.mounted
	if initial {
		a.callLifecycle("beforeMount")
	} else {
		a.callLifecycle("beforeUpdate")
	}
	a.events = map[string]func(){}
	a.eventMethods = map[string]func(Event){}
	a.eventRects = map[string]layout.Rect{}
	a.hitEvents = map[string][]string{}
	a.hitTree = hitTree{}
	a.focusable = nil
	a.modalFocusable = nil
	a.inputBindings = map[string]string{}
	a.inputStates = map[string]*State{}
	a.inputParentBindings = map[string]map[string]string{}
	a.toggleBindings = map[string]string{}
	a.toggleStates = map[string]*State{}
	a.toggleParentBindings = map[string]map[string]string{}
	a.scrollContainers = nil
	a.documentTextRuns = nil
	a.viewportScrollMax = 0
	layoutInput := layout.Input{
		Nodes:         a.component.Nodes,
		CSS:           a.styleCSS,
		Styles:        a.styles,
		Constraints:   a.layoutConstraints,
		Measurer:      a.measurer,
		ResolveText:   a.readBinding,
		ResolveProp:   a.readBinding,
		ResolveTruthy: a.truthy,
		ResolveList:   a.readList,
		ResolveAsset:  layout.FileAssetResolver(a.assetBase),
		Component:     a.layoutInputForComponent,
		State:         &a.state,
		SystemTokens:  a.systemTokens,
	}
	boxes := a.computeLayout(layoutInput)
	commands := a.renderBoxes(boxes)
	a.lastFrame = make([]renderer.Command, len(commands))
	copy(a.lastFrame, commands)
	a.scene, a.previousScene = scene.Build(commands, a.scrollOffsets, &a.previousScene)
	a.renderer.Render(commands)
	if initial {
		a.mounted = true
		a.callLifecycle("mounted")
		a.callComponentLifecycle("mounted")
	} else {
		a.callLifecycle("updated")
		a.callComponentLifecycle("updated")
	}
}

func (a *App) callLifecycle(hook string) {
	if a.component == nil {
		return
	}
	for _, lifecycle := range a.component.Lifecycle {
		if lifecycle.Hook != hook {
			continue
		}
		if handler := a.state.Methods[lifecycle.Method]; handler != nil {
			handler()
		}
	}
}

func (a *App) callComponentLifecycle(hook string) {
	if a.component == nil {
		return
	}
	a.callComponentLifecycleNodes(a.component.Nodes, hook)
}

func (a *App) callComponentLifecycleNodes(nodes []ir.Node, hook string) {
	for _, node := range nodes {
		a.callComponentLifecycleNode(node, hook)
	}
}

func (a *App) callComponentLifecycleNode(node ir.Node, hook string) {
	switch n := node.(type) {
	case *ir.ComponentInstance:
		a.callComponentInstanceLifecycle(n, hook)
		a.callComponentLifecycleNodes(n.Nodes, hook)
	case *ir.DynamicComponent:
		for _, candidate := range n.Cases {
			a.callComponentLifecycleNodes(candidate.Nodes, hook)
		}
	case *ir.Element:
		a.callComponentLifecycleNodes(n.Children, hook)
	case *ir.Conditional:
		a.callComponentLifecycleNode(n.Child, hook)
	case *ir.Repeater:
		a.callComponentLifecycleNode(n.Child, hook)
	}
}

func (a *App) callComponentInstanceLifecycle(instance *ir.ComponentInstance, hook string) {
	for _, lifecycle := range instance.Lifecycle {
		if lifecycle.Hook != hook {
			continue
		}
		for _, listener := range instance.Events {
			if listener.Event != hook {
				continue
			}
			if handler := a.state.Methods[listener.Method]; handler != nil {
				handler()
			}
		}
	}
}

func (a *App) Scene() scene.Scene {
	return a.scene
}

func (a *App) computeLayout(input layout.Input) []layout.Box {
	if a.layoutEngine != LayoutEngineCSS || input.CSS == "" {
		return layout.Compute(input)
	}
	boxes, err := layout.ComputeCSS(context.Background(), input)
	if err != nil {
		log.Printf("vugra css layout fallback: %v", err)
		return layout.Compute(input)
	}
	return boxes
}

func (a *App) layoutInputForComponent(instance *ir.ComponentInstance) layout.Input {
	state := a.stateForComponent(instance)
	return layout.Input{
		Nodes:          instance.Nodes,
		CSS:            a.styleCSS,
		Styles:         a.styles,
		Constraints:    a.layoutConstraints,
		Measurer:       a.measurer,
		ResolveText:    readBindingFrom(*state),
		ResolveProp:    readBindingFrom(*state),
		ResolveTruthy:  truthyFrom(*state),
		ResolveList:    readListFrom(*state),
		ResolveAsset:   layout.FileAssetResolver(a.assetBase),
		Component:      a.layoutInputForComponent,
		State:          state,
		Emitters:       a.emittersForComponent(instance),
		ParentBindings: parentBindingsForProps(instance.Props),
		SystemTokens:   a.systemTokens,
	}
}

func systemTokensFromRenderer(r renderer.Renderer) style.SystemTokens {
	provider, ok := r.(SystemTokenProvider)
	if !ok {
		return nil
	}
	return provider.SystemTokens()
}

func mergeSystemTokens(first, second style.SystemTokens) style.SystemTokens {
	if len(first) == 0 && len(second) == 0 {
		return nil
	}
	out := style.SystemTokens{}
	for key, value := range second {
		out[key] = value
	}
	for key, value := range first {
		out[key] = value
	}
	return out
}

func systemTokensEqual(first, second style.SystemTokens) bool {
	if len(first) != len(second) {
		return false
	}
	for key, firstValue := range first {
		if second[key] != firstValue {
			return false
		}
	}
	return true
}

func (a *App) stateForComponent(instance *ir.ComponentInstance) *State {
	if instance == nil || instance.Component == nil || instance.Component.NewState == nil {
		return &a.state
	}
	state, ok := a.componentStates[instance]
	if !ok {
		created := normalizedState(instance.Component.NewState())
		state = &created
		a.componentStates[instance] = state
	}
	a.applyComponentProps(instance, *state)
	return state
}

func (a *App) emittersForComponent(instance *ir.ComponentInstance) map[string]string {
	if instance == nil || instance.Component == nil || len(instance.Component.Emits) == 0 || len(instance.Events) == 0 {
		return nil
	}
	listeners := map[string]string{}
	for _, event := range instance.Events {
		listeners[event.Event] = event.Method
	}
	out := map[string]string{}
	for _, emit := range instance.Component.Emits {
		if listener := listeners[emit.Event]; listener != "" {
			out[emit.Method] = listener
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizedState(state State) State {
	if state.Signals == nil {
		state.Signals = map[string]Signal{}
	}
	if state.Methods == nil {
		state.Methods = map[string]func(){}
	}
	if state.EventMethods == nil {
		state.EventMethods = map[string]func(Event){}
	}
	return state
}

func (a *App) applyComponentProps(instance *ir.ComponentInstance, state State) {
	for _, prop := range instance.Props {
		signal := state.Signals[prop.Name]
		if signal == nil {
			continue
		}
		if prop.Bound {
			parentSignal := a.state.Signals[prop.Binding]
			if parentSignal == nil {
				continue
			}
			signal.SetAny(parentSignal.GetAny())
			continue
		}
		signal.SetAny(prop.Value)
	}
}

func parentBindingsForProps(props []ir.Prop) map[string]string {
	out := map[string]string{}
	for _, prop := range props {
		if prop.Bound {
			out[prop.Name] = prop.Binding
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *App) renderBoxes(boxes []layout.Box) []renderer.Command {
	var commands []renderer.Command
	viewport := a.viewportRect()
	a.viewportScrollMax = maxViewportScrollOffset(boxes, viewport)
	if a.viewportScroll > a.viewportScrollMax {
		a.viewportScroll = a.viewportScrollMax
	}
	for _, box := range boxes {
		commands = append(commands, a.renderBox(box, 0, -a.viewportScroll, &viewport, false, true)...)
	}
	a.hitTree = buildHitTree(commands, a.hitEvents, a.inputBindings, a.toggleBindings, a.scrollContainers, viewport)
	return commands
}

func (a *App) renderBox(box layout.Box, dx, dy float32, clip *layout.Rect, modalScope bool, selectableText bool) []renderer.Command {
	switch box.Kind {
	case "element":
		rect := translateRect(box.Rect, dx, dy)
		eventRect := rect
		if clip != nil {
			eventRect = intersectRects(eventRect, *clip)
		}
		props := map[string]string{}
		for name, value := range box.Props {
			props[name] = value
		}
		disabled := isDisabledProps(props, "disabled")
		inModalScope := modalScope || props["focus-scope"] == "modal"
		childSelectableText := selectableText && !isControlBox(box)
		state := a.stateForBox(box)
		if !disabled {
			for _, event := range box.Events {
				method := state.Methods[event.Method]
				eventMethod := state.EventMethods[event.Method]
				if method == nil && eventMethod == nil {
					method, eventMethod = a.findComponentHandler(event.Method)
				}
				if parentMethod := box.Emitters[event.Method]; parentMethod != "" {
					method = a.wrapEmitMethod(method, parentMethod)
					eventMethod = a.wrapEmitEventMethod(eventMethod, parentMethod)
				}
				if method != nil || eventMethod != nil {
					eventID := box.ID + ":" + event.Event
					if method != nil {
						a.events[eventID] = method
					}
					if eventMethod != nil {
						a.eventMethods[eventID] = eventMethod
					}
					a.eventRects[eventID] = eventRect
					a.hitEvents[box.ID] = append(a.hitEvents[box.ID], eventID)
					a.focusable = append(a.focusable, eventID)
					if inModalScope {
						a.modalFocusable = append(a.modalFocusable, eventID)
					}
					props["on:"+event.Event] = eventID
				}
			}
		}
		commands := []renderer.Command{{
			Kind:     "element",
			ID:       box.ID,
			Tag:      box.Tag,
			Role:     box.Role,
			Rect:     renderer.Rect(rect),
			Props:    props,
			Bindings: box.Bindings,
			Style: renderer.Style{
				Display:         box.Style.Display,
				FlexDirection:   box.Style.FlexDirection,
				FlexWrap:        box.Style.FlexWrap,
				AlignItems:      box.Style.AlignItems,
				Justify:         box.Style.Justify,
				Margin:          box.Style.Margin,
				PaddingLeft:     box.Style.PaddingLeft,
				FlexGrow:        box.Style.FlexGrow,
				FlexBasis:       box.Style.FlexBasis,
				FontSize:        box.Style.FontSize,
				LineHeight:      box.Style.LineHeight,
				TextAlign:       box.Style.TextAlign,
				BackgroundColor: box.Style.BackgroundColor,
				Opacity:         box.Style.Opacity,
				BorderWidth:     box.Style.BorderWidth,
				BorderWidthSet:  box.Style.BorderWidthSet,
				BorderColor:     box.Style.BorderColor,
				BorderRadius:    box.Style.BorderRadius,
				Color:           box.Style.Color,
				Overflow:        box.Style.Overflow,
			},
		}}
		if !disabled && box.Role == "textbox" && box.Bindings["value"] != "" {
			a.inputBindings[box.ID] = box.Bindings["value"]
			a.inputStates[box.ID] = &state
			a.inputParentBindings[box.ID] = box.ParentBindings
			a.eventRects[box.ID] = eventRect
			a.focusable = append(a.focusable, box.ID)
			if inModalScope {
				a.modalFocusable = append(a.modalFocusable, box.ID)
			}
			if _, ok := props["autofocus"]; ok {
				a.focus(box.ID)
			}
		}
		if !disabled && box.Role == "checkbox" && box.Bindings["checked"] != "" {
			a.toggleBindings[box.ID] = box.Bindings["checked"]
			a.toggleStates[box.ID] = &state
			a.toggleParentBindings[box.ID] = box.ParentBindings
			a.eventRects[box.ID] = eventRect
			a.focusable = append(a.focusable, box.ID)
			if inModalScope {
				a.modalFocusable = append(a.modalFocusable, box.ID)
			}
		}
		if box.Role == "textbox" && props["value"] != "" {
			commands = append(commands, renderer.Command{
				Kind:   "text",
				ID:     box.ID + ":value",
				Tag:    box.Tag,
				Text:   props["value"],
				Lines:  textLines(props["value"], rect.X+8, rect.Y+7, maxFloat(0, rect.Width-16), box.Style),
				Glyphs: textGlyphs(props["value"], rect.X+8, rect.Y+7, box.Style),
				Role:   box.Role,
				Rect:   renderer.Rect{X: rect.X + 8, Y: rect.Y + 7, Width: maxFloat(0, rect.Width-16), Height: maxFloat(0, rect.Height-14)},
				Props:  props,
				Style: renderer.Style{
					FontSize:   box.Style.FontSize,
					LineHeight: box.Style.LineHeight,
					TextAlign:  box.Style.TextAlign,
					Opacity:    box.Style.Opacity,
					Color:      box.Style.Color,
				},
			})
		}
		childDY := dy
		childClip := clip
		if clipsOverflow(box.Style.Overflow) {
			maxOffset := maxScrollOffset(box)
			if current := a.scrollOffsets[box.ID]; current > maxOffset {
				a.scrollOffsets[box.ID] = maxOffset
			}
			if scrollsOverflow(box.Style.Overflow) {
				a.scrollContainers = append(a.scrollContainers, scrollContainer{
					id:        box.ID,
					rect:      rect,
					maxOffset: maxOffset,
				})
			}
			childDY -= a.scrollOffsets[box.ID]
			nextClip := rect
			if clip != nil {
				nextClip = intersectRects(nextClip, *clip)
			}
			childClip = &nextClip
		}
		for _, child := range box.Children {
			commands = append(commands, a.renderBox(child, dx, childDY, childClip, inModalScope, childSelectableText)...)
		}
		commands = append(commands, renderer.Command{
			Kind: "end",
			ID:   box.ID,
			Tag:  box.Tag,
			Rect: renderer.Rect(rect),
		})
		return commands
	case "svg":
		rect := translateRect(box.Rect, dx, dy)
		props := map[string]string{}
		for name, value := range box.Props {
			props[name] = value
		}
		return []renderer.Command{{
			Kind:  "svg",
			ID:    box.ID,
			Tag:   box.Tag,
			SVG:   box.SVG,
			Role:  box.Role,
			Rect:  renderer.Rect(rect),
			Props: props,
			Style: renderer.Style{
				Display:         box.Style.Display,
				FlexDirection:   box.Style.FlexDirection,
				FlexWrap:        box.Style.FlexWrap,
				AlignItems:      box.Style.AlignItems,
				Justify:         box.Style.Justify,
				Margin:          box.Style.Margin,
				PaddingLeft:     box.Style.PaddingLeft,
				FlexGrow:        box.Style.FlexGrow,
				FlexBasis:       box.Style.FlexBasis,
				FontSize:        box.Style.FontSize,
				LineHeight:      box.Style.LineHeight,
				TextAlign:       box.Style.TextAlign,
				BackgroundColor: box.Style.BackgroundColor,
				Opacity:         box.Style.Opacity,
				BorderWidth:     box.Style.BorderWidth,
				BorderWidthSet:  box.Style.BorderWidthSet,
				BorderColor:     box.Style.BorderColor,
				BorderRadius:    box.Style.BorderRadius,
				Color:           box.Style.Color,
				Overflow:        box.Style.Overflow,
			},
		}}
	case "text":
		if box.Text == "" {
			return nil
		}
		rect := translateRect(box.Rect, dx, dy)
		lines := layoutLinesToRenderer(box.Lines, dx, dy)
		glyphs := layoutGlyphsToRenderer(box.Glyphs, dx, dy)
		if selectableText {
			a.recordDocumentTextRun(box.ID, box.Text, lines, renderer.Rect(rect), renderer.Style{
				FontSize:   box.Style.FontSize,
				LineHeight: box.Style.LineHeight,
				TextAlign:  box.Style.TextAlign,
				Opacity:    box.Style.Opacity,
				Color:      box.Style.Color,
			})
		}
		commands := a.documentSelectionCommandsForLastRun()
		commands = append(commands, renderer.Command{
			Kind:   "text",
			ID:     box.ID,
			Tag:    box.Tag,
			Text:   box.Text,
			Lines:  lines,
			Glyphs: glyphs,
			Role:   box.Role,
			Rect:   renderer.Rect(rect),
			Style: renderer.Style{
				FontSize:   box.Style.FontSize,
				LineHeight: box.Style.LineHeight,
				TextAlign:  box.Style.TextAlign,
				Opacity:    box.Style.Opacity,
				Color:      box.Style.Color,
			},
		})
		return commands
	default:
		return nil
	}
}

func (a *App) stateForBox(box layout.Box) State {
	if box.State != nil {
		return *box.State
	}
	return a.state
}

func isControlBox(box layout.Box) bool {
	return box.Role == "button" ||
		box.Role == "textbox" ||
		box.Role == "checkbox" ||
		box.Tag == "button" ||
		box.Tag == "input"
}

func (a *App) findComponentHandler(methodName string) (func(), func(Event)) {
	for _, state := range a.componentStates {
		if state == nil {
			continue
		}
		if handler := state.Methods[methodName]; handler != nil {
			return handler, nil
		}
		if handler := state.EventMethods[methodName]; handler != nil {
			return nil, handler
		}
	}
	return nil, nil
}

func (a *App) wrapEmitMethod(child func(), parentMethod string) func() {
	parent := a.state.Methods[parentMethod]
	parentEvent := a.state.EventMethods[parentMethod]
	if child == nil && parent == nil && parentEvent == nil {
		return nil
	}
	return func() {
		if child != nil {
			child()
		}
		if parentEvent != nil {
			parentEvent(Event{Type: "emit"})
			return
		}
		if parent != nil {
			parent()
		}
	}
}

func (a *App) wrapEmitEventMethod(child func(Event), parentMethod string) func(Event) {
	parentEvent := a.state.EventMethods[parentMethod]
	if child == nil && parentEvent == nil {
		return nil
	}
	return func(event Event) {
		if child != nil {
			child(event)
		}
		if parentEvent != nil {
			parentEvent(event)
		}
	}
}

func firstTargetEvent(ids []string, event string) string {
	suffix := ":" + event
	for _, id := range ids {
		if strings.HasSuffix(id, suffix) {
			return id
		}
	}
	if event == "click" && len(ids) > 0 {
		return ids[0]
	}
	return ""
}

func (a *App) beginDocumentSelection(x, y float32) bool {
	offset, ok := a.documentTextOffsetAt(x, y)
	if !ok {
		a.selectingDocument = false
		return false
	}
	a.focused = ""
	a.selectingDocument = true
	a.hasDocumentSelection = true
	a.documentSelection = textSelection{Start: offset, End: offset}
	a.scheduler.Schedule(a.render)
	return true
}

func (a *App) updateDocumentSelection(x, y float32) bool {
	if !a.selectingDocument {
		return false
	}
	offset, ok := a.documentTextOffsetAt(x, y)
	if !ok {
		offset = a.nearestDocumentTextOffset(x, y)
	}
	a.documentSelection.End = offset
	a.hasDocumentSelection = true
	a.scheduler.Schedule(a.render)
	return true
}

func (a *App) clearDocumentSelection() {
	if !a.hasDocumentSelection && !a.selectingDocument {
		return
	}
	a.hasDocumentSelection = false
	a.selectingDocument = false
	a.documentSelection = textSelection{}
	a.scheduler.Schedule(a.render)
}

func (a *App) recordDocumentTextRun(id, text string, lines []renderer.LineBox, rect renderer.Rect, style renderer.Style) {
	if strings.TrimSpace(text) == "" {
		return
	}
	start := a.documentTextLength()
	end := start + len([]rune(text))
	a.documentTextRuns = append(a.documentTextRuns, documentTextRun{
		id:    id,
		text:  text,
		lines: lines,
		rect:  rect,
		start: start,
		end:   end,
		style: style,
	})
}

func (a *App) documentTextLength() int {
	if len(a.documentTextRuns) == 0 {
		return 0
	}
	return a.documentTextRuns[len(a.documentTextRuns)-1].end
}

func (a *App) documentTextOffsetAt(x, y float32) (int, bool) {
	bestOffset := 0
	bestDistance := float32(1 << 30)
	found := false
	for _, run := range a.documentTextRuns {
		for _, lineRange := range runLineRanges(run) {
			if y < lineRange.rect.Y || y > lineRange.rect.Y+lineRange.rect.Height {
				continue
			}
			nearestX := clampFloat32(x, lineRange.rect.X, lineRange.rect.X+lineRange.rect.Width)
			centerY := lineRange.rect.Y + lineRange.rect.Height/2
			distance := y - centerY
			if distance < 0 {
				distance = -distance
			}
			if !found || distance < bestDistance {
				found = true
				bestDistance = distance
				bestOffset = lineRange.start + lineOffsetAt(lineRange, nearestX)
			}
		}
	}
	return bestOffset, found
}

func (a *App) nearestDocumentTextOffset(x, y float32) int {
	if len(a.documentTextRuns) == 0 {
		return 0
	}
	bestOffset := 0
	bestDistance := float32(1 << 30)
	for _, run := range a.documentTextRuns {
		for _, lineRange := range runLineRanges(run) {
			nearestX := clampFloat32(x, lineRange.rect.X, lineRange.rect.X+lineRange.rect.Width)
			nearestY := clampFloat32(y, lineRange.rect.Y, lineRange.rect.Y+lineRange.rect.Height)
			dx := x - nearestX
			dy := y - nearestY
			distance := dx*dx + dy*dy
			if distance < bestDistance {
				bestDistance = distance
				bestOffset = lineRange.start + lineOffsetAt(lineRange, nearestX)
			}
		}
	}
	return bestOffset
}

type documentLineRange struct {
	text  string
	start int
	end   int
	rect  renderer.Rect
}

func runLineRanges(run documentTextRun) []documentLineRange {
	if len(run.lines) == 0 {
		height := run.style.LineHeight
		if height <= 0 {
			height = renderer.DefaultLineHeight("", "text", run.style)
		}
		return []documentLineRange{{
			text:  run.text,
			start: run.start,
			end:   run.end,
			rect:  renderer.Rect{X: run.rect.X, Y: run.rect.Y, Width: run.rect.Width, Height: height},
		}}
	}
	out := make([]documentLineRange, 0, len(run.lines))
	offset := run.start
	for _, line := range run.lines {
		length := len([]rune(line.Text))
		width := line.Width
		if width <= 0 {
			width = estimateTextWidth(line.Text, run.style)
		}
		height := line.Height
		if height <= 0 {
			height = renderer.DefaultLineHeight("", "text", run.style)
		}
		out = append(out, documentLineRange{
			text:  line.Text,
			start: offset,
			end:   offset + length,
			rect:  renderer.Rect{X: line.X, Y: line.Y, Width: width, Height: height},
		})
		offset += length
	}
	return out
}

func lineOffsetAt(line documentLineRange, x float32) int {
	runes := len([]rune(line.text))
	if runes <= 0 {
		return 0
	}
	charWidth := line.rect.Width / float32(runes)
	if charWidth <= 0 {
		charWidth = 8
	}
	offset := int((x - line.rect.X) / charWidth)
	if x > line.rect.X+line.rect.Width {
		offset = runes
	}
	return clampInt(offset, 0, runes)
}

func (a *App) documentSelectionCommands() []renderer.Command {
	selection, ok := a.DocumentTextSelection()
	if !ok {
		return nil
	}
	var commands []renderer.Command
	for _, run := range a.documentTextRuns {
		commands = append(commands, selectionCommandsForRun(run, selection)...)
	}
	return commands
}

func (a *App) documentSelectionCommandsForLastRun() []renderer.Command {
	if len(a.documentTextRuns) == 0 {
		return nil
	}
	selection, ok := a.DocumentTextSelection()
	if !ok {
		return nil
	}
	return selectionCommandsForRun(a.documentTextRuns[len(a.documentTextRuns)-1], selection)
}

func selectionCommandsForRun(run documentTextRun, selection TextSelection) []renderer.Command {
	if run.end <= selection.Start || run.start >= selection.End {
		return nil
	}
	var commands []renderer.Command
	for _, lineRange := range runLineRanges(run) {
		start := maxInt(selection.Start, lineRange.start)
		end := minInt(selection.End, lineRange.end)
		if start >= end {
			continue
		}
		x1 := lineXAt(lineRange, start-lineRange.start)
		x2 := lineXAt(lineRange, end-lineRange.start)
		commands = append(commands, renderer.Command{
			Kind: "selection",
			ID:   run.id + ":selection",
			Rect: renderer.Rect{X: x1, Y: lineRange.rect.Y, Width: maxFloat(1, x2-x1), Height: lineRange.rect.Height},
			Style: renderer.Style{
				BackgroundColor: "#bfdbfe",
				Opacity:         0.72,
			},
		})
	}
	return commands
}

func lineXAt(line documentLineRange, offset int) float32 {
	runes := len([]rune(line.text))
	if runes <= 0 {
		return line.rect.X
	}
	charWidth := line.rect.Width / float32(runes)
	if charWidth <= 0 {
		charWidth = 8
	}
	offset = clampInt(offset, 0, runes)
	return line.rect.X + float32(offset)*charWidth
}

func estimateTextWidth(text string, style renderer.Style) float32 {
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = renderer.DefaultFontSize("", "text", style)
	}
	return float32(len([]rune(text))) * maxFloat(1, fontSize/2)
}

func layoutLinesToRenderer(lines []layout.LineBox, dx, dy float32) []renderer.LineBox {
	out := make([]renderer.LineBox, 0, len(lines))
	for _, line := range lines {
		next := renderer.LineBox(line)
		next.X += dx
		next.Y += dy
		out = append(out, next)
	}
	return out
}

func layoutGlyphsToRenderer(glyphs []layout.GlyphRun, dx, dy float32) []renderer.GlyphRun {
	out := make([]renderer.GlyphRun, 0, len(glyphs))
	for _, glyph := range glyphs {
		next := renderer.GlyphRun(glyph)
		next.X += dx
		next.Y += dy
		out = append(out, next)
	}
	return out
}

func textLines(text string, x, y, width float32, style layout.Style) []renderer.LineBox {
	lineHeight := style.LineHeight
	if lineHeight <= 0 {
		lineHeight = 20
	}
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 16
	}
	baseline := (lineHeight-fontSize)*0.5 + fontSize*0.82
	parts := strings.Split(text, "\n")
	out := make([]renderer.LineBox, 0, len(parts))
	for i, part := range parts {
		out = append(out, renderer.LineBox{
			Text:     part,
			X:        x,
			Y:        y + float32(i)*lineHeight,
			Width:    width,
			Height:   lineHeight,
			Baseline: baseline,
		})
	}
	return out
}

func textGlyphs(text string, x, y float32, style layout.Style) []renderer.GlyphRun {
	lineHeight := style.LineHeight
	if lineHeight <= 0 {
		lineHeight = 20
	}
	fontSize := style.FontSize
	if fontSize <= 0 {
		fontSize = 16
	}
	charWidth := maxFloat(1, fontSize/2)
	baseline := (lineHeight-fontSize)*0.5 + fontSize*0.82
	parts := strings.Split(text, "\n")
	out := make([]renderer.GlyphRun, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, renderer.GlyphRun{
			Text:     part,
			Size:     fontSize,
			X:        x,
			Y:        y + float32(i)*lineHeight,
			Advance:  float32(len([]rune(part))) * charWidth,
			Baseline: baseline,
		})
	}
	return out
}

func clipsOverflow(overflow string) bool {
	return overflow == "hidden" || overflow == "scroll"
}

func scrollsOverflow(overflow string) bool {
	return overflow == "scroll"
}

func (a *App) viewportRect() layout.Rect {
	width := a.layoutConstraints.Width
	if width <= 0 {
		width = 800
	}
	height := a.layoutConstraints.Height
	if height <= 0 {
		return layout.Rect{}
	}
	return layout.Rect{Width: width, Height: height}
}

func maxViewportScrollOffset(boxes []layout.Box, viewport layout.Rect) float32 {
	if viewport.Height <= 0 {
		return 0
	}
	bottom := float32(0)
	for _, box := range boxes {
		if childBottom := viewportSubtreeBottom(box); childBottom > bottom {
			bottom = childBottom
		}
	}
	return maxFloat(0, bottom-viewport.Height)
}

func maxScrollOffset(box layout.Box) float32 {
	bottom := box.Rect.Y + box.Rect.Height
	for _, child := range box.Children {
		if childBottom := subtreeBottom(child); childBottom > bottom {
			bottom = childBottom
		}
	}
	return maxFloat(0, bottom-(box.Rect.Y+box.Rect.Height))
}

func clampScrollOffset(value, maxOffset float32) float32 {
	if value < 0 {
		return 0
	}
	if value > maxOffset {
		return maxOffset
	}
	return value
}

func clampSelection(selection textSelection, length int) textSelection {
	selection.Start = clampInt(selection.Start, 0, length)
	selection.End = clampInt(selection.End, 0, length)
	if selection.Start > selection.End {
		selection.Start, selection.End = selection.End, selection.Start
	}
	return selection
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func subtreeBottom(box layout.Box) float32 {
	bottom := box.Rect.Y + box.Rect.Height
	for _, child := range box.Children {
		if childBottom := subtreeBottom(child); childBottom > bottom {
			bottom = childBottom
		}
	}
	return bottom
}

func viewportSubtreeBottom(box layout.Box) float32 {
	bottom := box.Rect.Y + box.Rect.Height
	if clipsOverflow(box.Style.Overflow) {
		return bottom
	}
	for _, child := range box.Children {
		if childBottom := viewportSubtreeBottom(child); childBottom > bottom {
			bottom = childBottom
		}
	}
	return bottom
}

func translateRect(rect layout.Rect, dx, dy float32) layout.Rect {
	rect.X += dx
	rect.Y += dy
	return rect
}

func intersectRects(a, b layout.Rect) layout.Rect {
	x1 := maxFloat(a.X, b.X)
	y1 := maxFloat(a.Y, b.Y)
	x2 := minFloat(a.X+a.Width, b.X+b.Width)
	y2 := minFloat(a.Y+a.Height, b.Y+b.Height)
	if x2 <= x1 || y2 <= y1 {
		return layout.Rect{X: x1, Y: y1}
	}
	return layout.Rect{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clampFloat32(value, minValue, maxValue float32) float32 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func contains(rect layout.Rect, x, y float32) bool {
	return x >= rect.X &&
		y >= rect.Y &&
		x < rect.X+rect.Width &&
		y < rect.Y+rect.Height
}

func (a *App) readBinding(name string) string {
	return readBindingFrom(a.state)(name)
}

func readBindingFrom(state State) func(string) string {
	return func(name string) string {
		signal := state.Signals[name]
		if signal == nil {
			return ""
		}
		return fmt.Sprint(signal.GetAny())
	}
}

func (a *App) readList(name string) []string {
	return readListFrom(a.state)(name)
}

func readListFrom(state State) func(string) []string {
	return func(name string) []string {
		signal := state.Signals[name]
		if signal == nil {
			return nil
		}
		switch values := signal.GetAny().(type) {
		case []string:
			return values
		case []int:
			out := make([]string, 0, len(values))
			for _, value := range values {
				out = append(out, fmt.Sprint(value))
			}
			return out
		default:
			return nil
		}
	}
}

func (a *App) truthy(name string) bool {
	return truthyFrom(a.state)(name)
}

func truthyFrom(state State) func(string) bool {
	return func(name string) bool {
		signal := state.Signals[name]
		if signal == nil {
			return false
		}
		switch value := signal.GetAny().(type) {
		case bool:
			return value
		case string:
			return value != ""
		case int:
			return value != 0
		default:
			return value != nil
		}
	}
}

func isDisabledProps(props map[string]string, name string) bool {
	value, ok := props[name]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0":
		return false
	default:
		return true
	}
}
