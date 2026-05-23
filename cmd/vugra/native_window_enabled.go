//go:build darwin && cgo && vuego_native_window

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vugra/vugra/internal/app"
	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/layout"
	"github.com/vugra/vugra/internal/nativewindow"
	"github.com/vugra/vugra/internal/project"
	"github.com/vugra/vugra/internal/reactivity"
	"github.com/vugra/vugra/internal/runtime"
	"github.com/vugra/vugra/internal/vellonative"
)

func runNativeWindow(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	entryPath := args[0]
	printNativeProgress("native-window", "compiling "+entryPath)
	result, err := compiler.CompileFile(args[0])
	if err != nil {
		return fmt.Errorf("compile %s: %w", args[0], err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("native-window failed with %d diagnostic(s)", len(diagnostics))
	}
	scheduler := reactivity.NewScheduler()
	if isFinderLiteExample(args[0]) && firstEnv("VUGRA_NATIVE_TITLEBAR", "VUEGO_NATIVE_TITLEBAR") == "" {
		os.Setenv("VUGRA_NATIVE_TITLEBAR", "hidden")
	}
	printNativeProgress("native-window", fmt.Sprintf("creating native window renderer=%s size=%dx%d", nativeRendererModeName(), 800, 600))
	target, err := nativewindow.New("Vugra", 800, 600)
	if err != nil {
		return err
	}
	target.DeferRenderUntilOpen()
	printNativeProgress("native-window", "mounting app")
	measurer := nativeMeasurerOrFixed()
	if closer, ok := measurer.(interface{ Close() }); ok {
		defer closer.Close()
	}
	mounted := app.Mount(result, demoStateFor(args[0], scheduler), target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: 800, Height: 600},
		Measurer:    measurer,
		Layout:      layoutEngineFromEnv(),
	})
	printNativeLaunch("native-window", "Vugra", entryPath, 800, 600, nativeRendererModeName(), layoutEngineName(layoutEngineFromEnv()))
	printNativeProgress("native-window", "entering native event loop")
	return target.Run(mounted)
}

func runProject(args []string) error {
	if len(args) > 1 {
		return usage()
	}
	configPath := ""
	if len(args) == 1 {
		configPath = args[0]
	}
	printNativeProgress("run", "loading project "+projectPathForLog(configPath))
	cfg, err := project.Load(configPath)
	if err != nil {
		return err
	}
	restoreEnv := applyProjectRuntimeEnv(cfg)
	defer restoreEnv()

	entryPath := cfg.EntryPath()
	printNativeProgress("run", "compiling "+entryPath)
	result, err := compiler.CompileFile(entryPath)
	if err != nil {
		return fmt.Errorf("compile %s: %w", entryPath, err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		encoded, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			return fmt.Errorf("encode diagnostics: %w", err)
		}
		fmt.Println(string(encoded))
		return fmt.Errorf("run failed with %d diagnostic(s)", len(diagnostics))
	}

	scheduler := reactivity.NewScheduler()
	printNativeProgress("run", fmt.Sprintf("creating native window renderer=%s size=%dx%d", nativeRendererModeName(), cfg.App.Width, cfg.App.Height))
	target, err := nativewindow.New(cfg.App.Title, cfg.App.Width, cfg.App.Height)
	if err != nil {
		return err
	}
	target.DeferRenderUntilOpen()
	if err := project.ApplySystemActions(target, cfg.LaunchActions()); err != nil {
		return err
	}
	printNativeProgress("run", "mounting app")
	measurer := nativeMeasurerOrFixed()
	if closer, ok := measurer.(interface{ Close() }); ok {
		defer closer.Close()
	}
	mounted := app.Mount(result, demoStateFor(entryPath, scheduler), target, app.Options{
		Scheduler:   scheduler,
		Constraints: layout.Constraints{Width: float32(cfg.App.Width), Height: float32(cfg.App.Height)},
		Measurer:    measurer,
		Layout:      layoutEngineFromConfig(cfg.Runtime.Layout),
	})
	printNativeLaunch("run", cfg.App.Title, entryPath, cfg.App.Width, cfg.App.Height, nativeRendererModeName(), cfg.Runtime.Layout)
	printNativeProgress("run", "entering native event loop")
	return target.Run(mounted)
}

func nativeMeasurerOrFixed() layout.Measurer {
	if nativeRendererModeName() == "vello-native" {
		return vellonative.NewLazyMeasurer()
	}
	return layout.FixedMeasurer{CharWidth: 8, LineHeight: 20}
}

func printNativeProgress(command, message string) {
	fmt.Fprintf(os.Stderr, "vugra %s: %s\n", command, message)
}

func printNativeLaunch(command, title, entryPath string, width, height int, renderer, layoutName string) {
	if layoutName == "" {
		layoutName = "go"
	}
	fmt.Fprintf(os.Stderr, "vugra %s: opening %q (%dx%d)\n", command, title, width, height)
	fmt.Fprintf(os.Stderr, "  entry: %s\n", entryPath)
	fmt.Fprintf(os.Stderr, "  renderer: %s\n", renderer)
	fmt.Fprintf(os.Stderr, "  layout: %s\n", layoutName)
	fmt.Fprintln(os.Stderr, "  close the native window to stop this command")
}

func nativeRendererModeName() string {
	if mode := firstEnv("VUGRA_NATIVE_RENDERER", "VUEGO_NATIVE_RENDERER"); mode != "" {
		return mode
	}
	return "vello-native"
}

func projectPathForLog(path string) string {
	if path == "" {
		return project.DefaultConfigFile
	}
	return path
}

func layoutEngineName(engine runtime.LayoutEngine) string {
	if engine == runtime.LayoutEngineCSS {
		return "css"
	}
	return "go"
}
