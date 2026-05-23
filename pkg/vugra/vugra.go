package vugra

import (
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/pkg/system"
)

type App = runtime.App
type State = runtime.State
type Signal = runtime.Signal
type Modifiers = runtime.Modifiers
type Event = runtime.Event
type Options = runtime.Options
type LayoutEngine = runtime.LayoutEngine
type TextSelection = runtime.TextSelection
type SystemTokens = runtime.SystemTokens
type SystemGUI = system.GUI
type WindowChrome = system.WindowChrome
type WindowControls = system.WindowControls
type WindowTitlebarMode = system.WindowTitlebarMode
type SystemRect = system.Rect

const (
	LayoutEngineGo  = runtime.LayoutEngineGo
	LayoutEngineCSS = runtime.LayoutEngineCSS

	WindowTitlebarDefault = system.WindowTitlebarDefault
	WindowTitlebarHidden  = system.WindowTitlebarHidden

	WindowControlsXToken      = system.WindowControlsXToken
	WindowControlsYToken      = system.WindowControlsYToken
	WindowControlsWidthToken  = system.WindowControlsWidthToken
	WindowControlsHeightToken = system.WindowControlsHeightToken
	WindowControlsLeftToken   = system.WindowControlsLeftToken
	WindowControlsTopToken    = system.WindowControlsTopToken
)

type Component = ir.Component
type Node = ir.Node
type Element = ir.Element
type Text = ir.Text
type Interpolation = ir.Interpolation
type Conditional = ir.Conditional
type Repeater = ir.Repeater
type ComponentInstance = ir.ComponentInstance
type DynamicComponent = ir.DynamicComponent
type DynamicComponentCase = ir.DynamicComponentCase
type Prop = ir.Prop
type PropDef = ir.PropDef
type ProvideDef = ir.ProvideDef
type InjectDef = ir.InjectDef
type Slot = ir.Slot
type EmitDef = ir.Emit
type Lifecycle = ir.Lifecycle
type EventHandler = ir.EventHandler

type Stylesheet = style.Stylesheet
type Rule = style.Rule
type Declaration = style.Declaration

type Constraints = layout.Constraints
type Measurer = layout.Measurer
type FixedMeasurer = layout.FixedMeasurer

type Command = renderer.Command
type Rect = renderer.Rect
type Renderer = renderer.Renderer
type SoftwareRenderer = renderer.SoftwareRenderer

func Mount(component *Component, state State, target Renderer, options Options) *App {
	return runtime.MountWithOptions(component, state, target, options)
}

func NewSoftware(width, height int) *SoftwareRenderer {
	return renderer.NewSoftware(width, height)
}

func Emit(event string) {}
