package app

import (
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/runtime"
)

type Options struct {
	Scheduler    *reactivity.Scheduler
	Constraints  layout.Constraints
	Measurer     layout.Measurer
	Layout       runtime.LayoutEngine
	SystemTokens runtime.SystemTokens
}

func Mount(result *compiler.Result, state runtime.State, target renderer.Renderer, options Options) *runtime.App {
	scheduler := options.Scheduler
	if scheduler == nil {
		scheduler = reactivity.NewScheduler()
	}
	return runtime.MountWithOptions(result.IR, state, target, runtime.Options{
		Scheduler:    scheduler,
		Styles:       result.Style,
		StyleCSS:     result.StyleCSS,
		AssetBase:    result.SFC.Path,
		Constraints:  options.Constraints,
		Measurer:     options.Measurer,
		Layout:       options.Layout,
		SystemTokens: options.SystemTokens,
	})
}
