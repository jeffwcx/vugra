package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/rustcodegen"
)

func runRustSFCSmoke(args []string) error {
	if len(args) > 1 {
		return usage()
	}
	path, err := rustSFCExamplePath(".")
	if err != nil {
		return err
	}
	if len(args) == 1 {
		path = args[0]
	}
	result, err := compileRustSFC(path)
	if err != nil {
		return err
	}
	generated := rustcodegen.GenerateStateAdapter(result, "RustSFCAdapter")
	if len(generated.Diagnostics) > 0 {
		return fmt.Errorf("rust-sfc-smoke codegen failed with %d diagnostic(s): %s", len(generated.Diagnostics), generated.Diagnostics[0].Code)
	}
	if len(generated.Signals) == 0 {
		return fmt.Errorf("rust-sfc-smoke codegen failed: no generated signal bindings")
	}
	if err := compileGeneratedRustSFCAdapter(generated.Source); err != nil {
		return err
	}
	finderPath, err := rustFinderSFCExamplePath(".")
	if err != nil {
		return err
	}
	finder, err := compileRustSFC(finderPath)
	if err != nil {
		return err
	}
	finderGenerated := rustcodegen.GenerateStateAdapterWithOptions(finder, rustcodegen.Options{
		AdapterName: "FinderLiteSFCAdapter",
		Contract:    rustcodegen.ContractFinderLite,
	})
	if len(finderGenerated.Diagnostics) > 0 {
		return fmt.Errorf("rust-sfc-smoke finder codegen failed with %d diagnostic(s): %s", len(finderGenerated.Diagnostics), finderGenerated.Diagnostics[0].Code)
	}
	if err := compileGeneratedFinderLiteSFCAdapter(finderGenerated.Source); err != nil {
		return err
	}
	fmt.Printf(
		"rust-sfc path=%s fields=%d methods=%d ir_nodes=%d adapter_bytes=%d\n",
		path,
		len(result.Rust.State.Fields),
		len(result.Rust.Methods),
		len(result.IR.Nodes),
		len(generated.Source),
	)
	fmt.Printf("rust-sfc-codegen signals=%d methods=%d\n", len(generated.Signals), len(generated.Methods))
	fmt.Printf("rust-finder-sfc path=%s signals=%d methods=%d adapter_bytes=%d\n", finderPath, len(finderGenerated.Signals), len(finderGenerated.Methods), len(finderGenerated.Source))
	if len(finderGenerated.Signals) != 98 || len(finderGenerated.Methods) != 79 {
		return fmt.Errorf("rust-sfc-smoke finder codegen contract mismatch: signals=%d methods=%d", len(finderGenerated.Signals), len(finderGenerated.Methods))
	}
	fmt.Println("rust-sfc-smoke ok")
	return nil
}

func runRustFinderSFC(args []string) error {
	variant := "native"
	if len(args) > 1 {
		return usage()
	}
	if len(args) == 1 {
		variant = args[0]
	}
	if !isRustFinderSFCVariant(variant) {
		return fmt.Errorf("rust-finder-sfc variant must be native, native-software, native-vello, native-wgpu, native-window-smoke, native-software-window-smoke, native-vello-window-smoke, or native-wgpu-window-smoke")
	}
	finderPath, err := rustFinderSFCExamplePath(".")
	if err != nil {
		return err
	}
	finder, err := compileRustSFC(finderPath)
	if err != nil {
		return err
	}
	generated := rustcodegen.GenerateStateAdapterWithOptions(finder, rustcodegen.Options{
		AdapterName: "FinderLiteSFCAdapter",
		Contract:    rustcodegen.ContractFinderLite,
	})
	if len(generated.Diagnostics) > 0 {
		return fmt.Errorf("rust-finder-sfc codegen failed with %d diagnostic(s): %s", len(generated.Diagnostics), generated.Diagnostics[0].Code)
	}
	return runGeneratedFinderLiteSFCApp(generated.Source, variant)
}

func isRustFinderSFCVariant(variant string) bool {
	switch variant {
	case "native", "native-software", "native-vello", "native-wgpu", "native-window-smoke", "native-software-window-smoke", "native-vello-window-smoke", "native-wgpu-window-smoke":
		return true
	default:
		return false
	}
}

func compileRustSFC(path string) (*compiler.Result, error) {
	result, err := compiler.CompileFile(path)
	if err != nil {
		return nil, fmt.Errorf("compile rust SFC %s: %w", path, err)
	}
	if result.SFC == nil || result.SFC.Script == nil || result.SFC.Script.Lang != "rust" {
		return nil, fmt.Errorf("rust-sfc-smoke failed: %s is not a <script lang=\"rust\"> component", path)
	}
	if result.Rust.State == nil {
		return nil, fmt.Errorf("rust-sfc-smoke failed: missing Rust State metadata")
	}
	if result.IR == nil {
		return nil, fmt.Errorf("rust-sfc-smoke failed: missing Vugra IR")
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) > 0 {
		return nil, fmt.Errorf("rust-sfc-smoke failed with %d diagnostic(s): %s", len(diagnostics), diagnostics[0].Code)
	}
	return result, nil
}

func compileGeneratedRustSFCAdapter(source string) error {
	root, err := repoRoot(".")
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "vugra-rust-sfc-codegen-*")
	if err != nil {
		return fmt.Errorf("create rust SFC codegen smoke dir: %w", err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(rustSFCSmokeCargoToml(root)), 0o644); err != nil {
		return fmt.Errorf("write rust SFC smoke Cargo.toml: %w", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		return fmt.Errorf("create rust SFC smoke src dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "lib.rs"), []byte(rustSFCSmokeLib(source)), 0o644); err != nil {
		return fmt.Errorf("write rust SFC smoke lib.rs: %w", err)
	}
	cmd := exec.Command("cargo", "test", "--quiet")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile generated rust SFC adapter: %w\n%s", err, string(output))
	}
	return nil
}

func compileGeneratedFinderLiteSFCAdapter(source string) error {
	root, err := repoRoot(".")
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "vugra-rust-finder-sfc-codegen-*")
	if err != nil {
		return fmt.Errorf("create rust Finder SFC codegen smoke dir: %w", err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(rustSFCSmokeCargoToml(root)), 0o644); err != nil {
		return fmt.Errorf("write rust Finder SFC smoke Cargo.toml: %w", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		return fmt.Errorf("create rust Finder SFC smoke src dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "lib.rs"), []byte(rustFinderSFCSmokeLib(source)), 0o644); err != nil {
		return fmt.Errorf("write rust Finder SFC smoke lib.rs: %w", err)
	}
	cmd := exec.Command("cargo", "test", "--quiet")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile generated rust Finder SFC adapter: %w\n%s", err, string(output))
	}
	return nil
}

func runGeneratedFinderLiteSFCApp(source, variant string) error {
	root, err := repoRoot(".")
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "vugra-rust-finder-sfc-app-*")
	if err != nil {
		return fmt.Errorf("create rust Finder SFC app dir: %w", err)
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(rustSFCSmokeCargoToml(root)), 0o644); err != nil {
		return fmt.Errorf("write rust Finder SFC app Cargo.toml: %w", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		return fmt.Errorf("create rust Finder SFC app src dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(rustFinderSFCAppMain(source)), 0o644); err != nil {
		return fmt.Errorf("write rust Finder SFC app main.rs: %w", err)
	}
	cmd := exec.Command("cargo", "run", "--quiet", "--", variant)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run generated rust Finder SFC app %s: %w", variant, err)
	}
	return nil
}

func rustSFCSmokeCargoToml(root string) string {
	return fmt.Sprintf(`[package]
name = "vugra-rust-sfc-smoke"
version = "0.1.0"
edition = "2021"

[dependencies]
vugra-core = { path = %q }
vugra-host-native = { path = %q }
vugra-ir = { path = %q }
vugra-layout = { path = %q }
vugra-render = { path = %q }
`, filepath.Join(root, "crates", "vugra-core"), filepath.Join(root, "crates", "vugra-host-native"), filepath.Join(root, "crates", "vugra-ir"), filepath.Join(root, "crates", "vugra-layout"), filepath.Join(root, "crates", "vugra-render"))
}

func rustSFCSmokeLib(source string) string {
	return source + `

#[derive(Default)]
struct State {
    value: i32,
}

#[allow(non_snake_case)]
impl RustSFCBindings for State {
    fn count(&self) -> vugra_core::Value {
        self.value.to_string().into()
    }

    fn set_count(&mut self, value: vugra_core::Value) {
        self.value = value.as_text().parse().unwrap_or_default();
    }

    fn Inc(&mut self) {
        self.value += 1;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_core::ComponentState;

    #[test]
    fn generated_adapter_runs() {
        let mut adapter = RustSFCAdapter::new(State::default());
        assert_eq!(adapter.get_signal(vugra_core::SignalId(1)).as_text(), "0");
        adapter.call_method(vugra_core::MethodId(1));
        assert_eq!(adapter.get_signal(vugra_core::SignalId(1)).as_text(), "1");
        adapter.set_signal(vugra_core::SignalId(1), "41".to_string().into());
        adapter.call_method(vugra_core::MethodId(1));
        assert_eq!(adapter.get_signal(vugra_core::SignalId(1)).as_text(), "42");

        let mut app = vugra_core::App::new(generated_component_contract(), adapter);
        app.dispatch(vugra_core::MethodId(1));
        let adapter = app.state();
        assert_eq!(adapter.get_signal(vugra_core::SignalId(1)).as_text(), "43");

        let native = vugra_host_native::render_native_frame(
            &app,
            vugra_layout::Constraints {
                width: 320.0,
                height: 240.0,
            },
            vugra_host_native::NativeRenderBackend::Software,
            320,
            240,
        );
        assert!(native.commands.iter().any(|command| {
            matches!(
                command,
                vugra_render::RenderCommand::Text { text, .. } if text == "count: 43"
            )
        }));
        assert!(native.pixels.iter().any(|pixel| *pixel != 0x00f7f7f8));
    }
}
`
}

func rustFinderSFCSmokeLib(source string) string {
	return source + rustFinderSFCStateBindings() + `
#[cfg(test)]
mod tests {
    use super::*;
    use vugra_core::ComponentState;

    #[test]
    fn generated_finder_adapter_renders_native_frame() {
        let adapter = FinderLiteSFCAdapter::new(FinderState::default());
        let mut app = vugra_core::App::new(generated_component_contract(), adapter);
        assert_eq!(app.render_frame().rows[0].name, "Design");
        app.dispatch(vugra_core::MethodId(3));
        assert!(matches!(app.state().get_signal(vugra_core::SignalId(31)), vugra_core::Value::Bool(true)));
        app.dispatch_event(
            vugra_core::MethodId(10),
            vugra_core::Event {
                kind: "text".to_string(),
                text: "road".to_string(),
                ..vugra_core::Event::default()
            },
        );
        assert_eq!(app.render_frame().search_query, "road");
        app.dispatch(vugra_core::MethodId(81));
        assert_eq!(app.render_frame().splitter.as_ref().expect("splitter").splitter_class, "splitter-hover");
        app.dispatch_event(
            vugra_core::MethodId(82),
            vugra_core::Event {
                kind: "drag".to_string(),
                delta_x: 80.0,
                ..vugra_core::Event::default()
            },
        );
        assert_eq!(app.render_frame().splitter.as_ref().expect("splitter").sidebar_class, "sidebar-200");

        let native = vugra_host_native::render_native_frame(
            &app,
            vugra_layout::Constraints {
                width: 800.0,
                height: 600.0,
            },
            vugra_host_native::NativeRenderBackend::Software,
            800,
            600,
        );
        assert!(native.commands.iter().any(|command| {
            matches!(
                command,
                vugra_render::RenderCommand::Element {
                    id,
                    role,
                    selected: true,
                    ..
                } if id == "row2" && role == "row"
            )
        }));
        assert!(native.commands.iter().any(|command| {
            matches!(
                command,
                vugra_render::RenderCommand::Text {
                    id,
                    text,
                    selected: true,
                    ..
                } if id == "row2-name:text" && text == "Roadmap.md"
            )
        }));
        assert!(native.commands.iter().any(|command| {
            matches!(
                command,
                vugra_render::RenderCommand::Text {
                    id,
                    text,
                    selected: true,
                    ..
                } if id == "row2-size:text" && text == "12 KB"
            )
        }));
        assert!(native.pixels.iter().any(|pixel| *pixel != 0x00f7f7f8));

        assert!(vugra_host_native::dispatch_native_text(&mut app, "x".to_string()));
        assert_eq!(app.render_frame().search_query, "roadx");
        let native = vugra_host_native::render_native_frame(
            &app,
            vugra_layout::Constraints {
                width: 800.0,
                height: 600.0,
            },
            vugra_host_native::NativeRenderBackend::Software,
            800,
            600,
        );
        assert!(vugra_host_native::dispatch_native_drag(&mut app, &native, 202.0, 80.0, -80.0, 0.0));
        assert_eq!(app.render_frame().splitter.as_ref().expect("splitter").sidebar_class, "sidebar");
    }
}
`
}

func rustFinderSFCAppMain(source string) string {
	return source + rustFinderSFCStateBindings() + `

fn main() {
    let variant = std::env::args().nth(1).unwrap_or_else(|| "native".to_string());
    let adapter = FinderLiteSFCAdapter::new(FinderState::default());
    let mut app = vugra_core::App::new(generated_component_contract(), adapter);
    match variant.as_str() {
        "native" | "native-software" | "native-vello" | "native-wgpu" => {
            let backend = backend_for_variant(&variant).unwrap();
            let config = vugra_host_native::NativeWindowConfig {
                title: format!("Vugra Rust Finder Lite SFC ({})", backend.as_str()),
                width: 800,
                height: 600,
                backend,
            };
            if let Err(err) = vugra_host_native::run_app_window(&mut app, config) {
                eprintln!("run generated Finder SFC native window: {err}");
                std::process::exit(1);
            }
        }
        "native-window-smoke" | "native-software-window-smoke" | "native-vello-window-smoke" | "native-wgpu-window-smoke" => {
            let backend = backend_for_variant(&variant).unwrap();
            app.dispatch(vugra_core::MethodId(3));
            let frame = vugra_host_native::render_native_frame(
                &app,
                vugra_layout::Constraints {
                    width: 640.0,
                    height: 420.0,
                },
                backend,
                640,
                420,
            );
            if !vugra_host_native::dispatch_native_text(&mut app, "road".to_string()) {
                eprintln!("generated Finder SFC {variant} failed: native text was not dispatched");
                std::process::exit(1);
            }
            if app.render_frame().search_query != "road" {
                eprintln!("generated Finder SFC {variant} failed: native text did not update search query");
                std::process::exit(1);
            }
            if !vugra_host_native::dispatch_native_drag(&mut app, &frame, 242.0, 80.0, 80.0, 0.0) {
                eprintln!("generated Finder SFC {variant} failed: native drag was not dispatched");
                std::process::exit(1);
            }
            if app.render_frame().splitter.as_ref().map(|splitter| splitter.sidebar_class.as_str()) != Some("sidebar-200") {
                eprintln!("generated Finder SFC {variant} failed: native drag did not resize sidebar");
                std::process::exit(1);
            }
            let config = vugra_host_native::NativeWindowConfig {
                title: format!("Vugra Rust Finder Lite SFC smoke ({})", backend.as_str()),
                width: 640,
                height: 420,
                backend,
            };
            let smoke = match vugra_host_native::run_app_window_for_frames(&mut app, config, 2) {
                Ok(smoke) => smoke,
                Err(err) => {
                    eprintln!("run generated Finder SFC native smoke: {err}");
                    std::process::exit(1);
                }
            };
            if smoke.frames_presented < 1 || smoke.commands == 0 || smoke.drawn_pixels == 0 {
                eprintln!("generated Finder SFC {variant} failed: {smoke:?}");
                std::process::exit(1);
            }
            println!(
                "rust-finder-sfc-window backend={} frames={} commands={} pixels={} drawn={}",
                backend.as_str(),
                smoke.frames_presented,
                smoke.commands,
                smoke.pixels,
                smoke.drawn_pixels
            );
            println!("rust-finder-sfc {variant} ok");
        }
        _ => {
            eprintln!("usage: generated-finder-sfc [native|native-software|native-vello|native-wgpu|native-window-smoke|native-software-window-smoke|native-vello-window-smoke|native-wgpu-window-smoke]");
            std::process::exit(2);
        }
    }
}

fn backend_for_variant(variant: &str) -> Option<vugra_host_native::NativeRenderBackend> {
    match variant {
        "native" | "native-window-smoke" => Some(vugra_host_native::NativeRenderBackend::Vello),
        "native-software" | "native-software-window-smoke" => Some(vugra_host_native::NativeRenderBackend::Software),
        "native-vello" | "native-vello-window-smoke" => Some(vugra_host_native::NativeRenderBackend::Vello),
        "native-wgpu" | "native-wgpu-window-smoke" => Some(vugra_host_native::NativeRenderBackend::Wgpu),
        _ => None,
    }
}
`
}

func rustFinderSFCStateBindings() string {
	return `

#[derive(Default)]
struct FinderState {
    selected: usize,
    sidebar_mode: usize,
    splitter_hovered: bool,
    search_query: String,
    item_menu_open: bool,
    blank_menu_open: bool,
    rename_text: String,
    preview_open: bool,
    preview_title: String,
    preview_body: String,
}

struct FinderRow {
    name: &'static str,
    kind: &'static str,
    modified: &'static str,
    size: &'static str,
}

impl FinderState {
    fn rows(&self) -> [FinderRow; 12] {
        [
            FinderRow { name: "Design", kind: "folder", modified: "--", size: "--" },
            FinderRow { name: "Roadmap.md", kind: "file", modified: "--", size: "12 KB" },
            FinderRow { name: "Budget 2026.xlsx", kind: "file", modified: "--", size: "842 KB" },
            FinderRow { name: "Meeting Notes.txt", kind: "file", modified: "--", size: "17 KB" },
            FinderRow { name: "Client Brief.pdf", kind: "file", modified: "--", size: "224 KB" },
            FinderRow { name: "Contract Draft.docx", kind: "file", modified: "--", size: "96 KB" },
            FinderRow { name: "Launch Plan.pages", kind: "file", modified: "--", size: "410 KB" },
            FinderRow { name: "Research Summary.md", kind: "file", modified: "--", size: "38 KB" },
            FinderRow { name: "Book Outline.txt", kind: "file", modified: "--", size: "21 KB" },
            FinderRow { name: "Ideas.txt", kind: "file", modified: "--", size: "7 KB" },
            FinderRow { name: "Agenda.md", kind: "file", modified: "--", size: "8 KB" },
            FinderRow { name: "Notes Archive.txt", kind: "file", modified: "--", size: "53 KB" },
        ]
    }

    fn row_name(&self, index: usize) -> vugra_core::Value { self.rows()[index].name.into() }
    fn row_kind(&self, index: usize) -> vugra_core::Value { self.rows()[index].kind.into() }
    fn row_modified(&self, index: usize) -> vugra_core::Value { self.rows()[index].modified.into() }
    fn row_size(&self, index: usize) -> vugra_core::Value { self.rows()[index].size.into() }
    fn row_class(&self, index: usize) -> vugra_core::Value {
        if self.selected == index { "file-row-selected".into() } else { "file-row".into() }
    }
    fn row_selected(&self, index: usize) -> vugra_core::Value { (self.selected == index).into() }
    fn sidebar_class(&self) -> &'static str {
        ["sidebar", "sidebar-200", "sidebar-280", "sidebar-320"][self.sidebar_mode]
    }
    fn splitter_class(&self) -> &'static str {
        if self.splitter_hovered { "splitter-hover" } else { "splitter" }
    }
}

#[allow(non_snake_case)]
impl RustSFCBindings for FinderState {
    fn path(&self) -> vugra_core::Value { "Documents".into() }
    fn set_path(&mut self, _: vugra_core::Value) {}
    fn status(&self) -> vugra_core::Value { "12 items · Current path: Documents".into() }
    fn set_status(&mut self, _: vugra_core::Value) {}
    fn selectedSummary(&self) -> vugra_core::Value { "1 items selected".into() }
    fn set_selectedSummary(&mut self, _: vugra_core::Value) {}
    fn row1Name(&self) -> vugra_core::Value { self.row_name(0) }
    fn set_row1Name(&mut self, _: vugra_core::Value) {}
    fn row1Kind(&self) -> vugra_core::Value { self.row_kind(0) }
    fn set_row1Kind(&mut self, _: vugra_core::Value) {}
    fn row1Modified(&self) -> vugra_core::Value { self.row_modified(0) }
    fn set_row1Modified(&mut self, _: vugra_core::Value) {}
    fn row1Size(&self) -> vugra_core::Value { self.row_size(0) }
    fn set_row1Size(&mut self, _: vugra_core::Value) {}
    fn row1Class(&self) -> vugra_core::Value { self.row_class(0) }
    fn set_row1Class(&mut self, _: vugra_core::Value) {}
    fn row1Selected(&self) -> vugra_core::Value { self.row_selected(0) }
    fn set_row1Selected(&mut self, _: vugra_core::Value) {}
    fn row2Name(&self) -> vugra_core::Value { self.row_name(1) }
    fn set_row2Name(&mut self, _: vugra_core::Value) {}
    fn row2Kind(&self) -> vugra_core::Value { self.row_kind(1) }
    fn set_row2Kind(&mut self, _: vugra_core::Value) {}
    fn row2Modified(&self) -> vugra_core::Value { self.row_modified(1) }
    fn set_row2Modified(&mut self, _: vugra_core::Value) {}
    fn row2Size(&self) -> vugra_core::Value { self.row_size(1) }
    fn set_row2Size(&mut self, _: vugra_core::Value) {}
    fn row2Class(&self) -> vugra_core::Value { self.row_class(1) }
    fn set_row2Class(&mut self, _: vugra_core::Value) {}
    fn row2Selected(&self) -> vugra_core::Value { self.row_selected(1) }
    fn set_row2Selected(&mut self, _: vugra_core::Value) {}
    fn row3Name(&self) -> vugra_core::Value { self.row_name(2) }
    fn set_row3Name(&mut self, _: vugra_core::Value) {}
    fn row3Kind(&self) -> vugra_core::Value { self.row_kind(2) }
    fn set_row3Kind(&mut self, _: vugra_core::Value) {}
    fn row3Modified(&self) -> vugra_core::Value { self.row_modified(2) }
    fn set_row3Modified(&mut self, _: vugra_core::Value) {}
    fn row3Size(&self) -> vugra_core::Value { self.row_size(2) }
    fn set_row3Size(&mut self, _: vugra_core::Value) {}
    fn row3Class(&self) -> vugra_core::Value { self.row_class(2) }
    fn set_row3Class(&mut self, _: vugra_core::Value) {}
    fn row3Selected(&self) -> vugra_core::Value { self.row_selected(2) }
    fn set_row3Selected(&mut self, _: vugra_core::Value) {}
    fn row4Name(&self) -> vugra_core::Value { self.row_name(3) }
    fn set_row4Name(&mut self, _: vugra_core::Value) {}
    fn row4Kind(&self) -> vugra_core::Value { self.row_kind(3) }
    fn set_row4Kind(&mut self, _: vugra_core::Value) {}
    fn row4Modified(&self) -> vugra_core::Value { self.row_modified(3) }
    fn set_row4Modified(&mut self, _: vugra_core::Value) {}
    fn row4Size(&self) -> vugra_core::Value { self.row_size(3) }
    fn set_row4Size(&mut self, _: vugra_core::Value) {}
    fn row4Class(&self) -> vugra_core::Value { self.row_class(3) }
    fn set_row4Class(&mut self, _: vugra_core::Value) {}
    fn row4Selected(&self) -> vugra_core::Value { self.row_selected(3) }
    fn set_row4Selected(&mut self, _: vugra_core::Value) {}
    fn row5Name(&self) -> vugra_core::Value { self.row_name(4) }
    fn set_row5Name(&mut self, _: vugra_core::Value) {}
    fn row5Kind(&self) -> vugra_core::Value { self.row_kind(4) }
    fn set_row5Kind(&mut self, _: vugra_core::Value) {}
    fn row5Modified(&self) -> vugra_core::Value { self.row_modified(4) }
    fn set_row5Modified(&mut self, _: vugra_core::Value) {}
    fn row5Size(&self) -> vugra_core::Value { self.row_size(4) }
    fn set_row5Size(&mut self, _: vugra_core::Value) {}
    fn row5Class(&self) -> vugra_core::Value { self.row_class(4) }
    fn set_row5Class(&mut self, _: vugra_core::Value) {}
    fn row5Selected(&self) -> vugra_core::Value { self.row_selected(4) }
    fn set_row5Selected(&mut self, _: vugra_core::Value) {}
    fn row6Name(&self) -> vugra_core::Value { self.row_name(5) }
    fn set_row6Name(&mut self, _: vugra_core::Value) {}
    fn row6Kind(&self) -> vugra_core::Value { self.row_kind(5) }
    fn set_row6Kind(&mut self, _: vugra_core::Value) {}
    fn row6Modified(&self) -> vugra_core::Value { self.row_modified(5) }
    fn set_row6Modified(&mut self, _: vugra_core::Value) {}
    fn row6Size(&self) -> vugra_core::Value { self.row_size(5) }
    fn set_row6Size(&mut self, _: vugra_core::Value) {}
    fn row6Class(&self) -> vugra_core::Value { self.row_class(5) }
    fn set_row6Class(&mut self, _: vugra_core::Value) {}
    fn row6Selected(&self) -> vugra_core::Value { self.row_selected(5) }
    fn set_row6Selected(&mut self, _: vugra_core::Value) {}
    fn row7Name(&self) -> vugra_core::Value { self.row_name(6) }
    fn set_row7Name(&mut self, _: vugra_core::Value) {}
    fn row7Kind(&self) -> vugra_core::Value { self.row_kind(6) }
    fn set_row7Kind(&mut self, _: vugra_core::Value) {}
    fn row7Modified(&self) -> vugra_core::Value { self.row_modified(6) }
    fn set_row7Modified(&mut self, _: vugra_core::Value) {}
    fn row7Size(&self) -> vugra_core::Value { self.row_size(6) }
    fn set_row7Size(&mut self, _: vugra_core::Value) {}
    fn row7Class(&self) -> vugra_core::Value { self.row_class(6) }
    fn set_row7Class(&mut self, _: vugra_core::Value) {}
    fn row7Selected(&self) -> vugra_core::Value { self.row_selected(6) }
    fn set_row7Selected(&mut self, _: vugra_core::Value) {}
    fn row8Name(&self) -> vugra_core::Value { self.row_name(7) }
    fn set_row8Name(&mut self, _: vugra_core::Value) {}
    fn row8Kind(&self) -> vugra_core::Value { self.row_kind(7) }
    fn set_row8Kind(&mut self, _: vugra_core::Value) {}
    fn row8Modified(&self) -> vugra_core::Value { self.row_modified(7) }
    fn set_row8Modified(&mut self, _: vugra_core::Value) {}
    fn row8Size(&self) -> vugra_core::Value { self.row_size(7) }
    fn set_row8Size(&mut self, _: vugra_core::Value) {}
    fn row8Class(&self) -> vugra_core::Value { self.row_class(7) }
    fn set_row8Class(&mut self, _: vugra_core::Value) {}
    fn row8Selected(&self) -> vugra_core::Value { self.row_selected(7) }
    fn set_row8Selected(&mut self, _: vugra_core::Value) {}
    fn row9Name(&self) -> vugra_core::Value { self.row_name(8) }
    fn set_row9Name(&mut self, _: vugra_core::Value) {}
    fn row9Kind(&self) -> vugra_core::Value { self.row_kind(8) }
    fn set_row9Kind(&mut self, _: vugra_core::Value) {}
    fn row9Modified(&self) -> vugra_core::Value { self.row_modified(8) }
    fn set_row9Modified(&mut self, _: vugra_core::Value) {}
    fn row9Size(&self) -> vugra_core::Value { self.row_size(8) }
    fn set_row9Size(&mut self, _: vugra_core::Value) {}
    fn row9Class(&self) -> vugra_core::Value { self.row_class(8) }
    fn set_row9Class(&mut self, _: vugra_core::Value) {}
    fn row9Selected(&self) -> vugra_core::Value { self.row_selected(8) }
    fn set_row9Selected(&mut self, _: vugra_core::Value) {}
    fn row10Name(&self) -> vugra_core::Value { self.row_name(9) }
    fn set_row10Name(&mut self, _: vugra_core::Value) {}
    fn row10Kind(&self) -> vugra_core::Value { self.row_kind(9) }
    fn set_row10Kind(&mut self, _: vugra_core::Value) {}
    fn row10Modified(&self) -> vugra_core::Value { self.row_modified(9) }
    fn set_row10Modified(&mut self, _: vugra_core::Value) {}
    fn row10Size(&self) -> vugra_core::Value { self.row_size(9) }
    fn set_row10Size(&mut self, _: vugra_core::Value) {}
    fn row10Class(&self) -> vugra_core::Value { self.row_class(9) }
    fn set_row10Class(&mut self, _: vugra_core::Value) {}
    fn row10Selected(&self) -> vugra_core::Value { self.row_selected(9) }
    fn set_row10Selected(&mut self, _: vugra_core::Value) {}
    fn row11Name(&self) -> vugra_core::Value { self.row_name(10) }
    fn set_row11Name(&mut self, _: vugra_core::Value) {}
    fn row11Kind(&self) -> vugra_core::Value { self.row_kind(10) }
    fn set_row11Kind(&mut self, _: vugra_core::Value) {}
    fn row11Modified(&self) -> vugra_core::Value { self.row_modified(10) }
    fn set_row11Modified(&mut self, _: vugra_core::Value) {}
    fn row11Size(&self) -> vugra_core::Value { self.row_size(10) }
    fn set_row11Size(&mut self, _: vugra_core::Value) {}
    fn row11Class(&self) -> vugra_core::Value { self.row_class(10) }
    fn set_row11Class(&mut self, _: vugra_core::Value) {}
    fn row11Selected(&self) -> vugra_core::Value { self.row_selected(10) }
    fn set_row11Selected(&mut self, _: vugra_core::Value) {}
    fn row12Name(&self) -> vugra_core::Value { self.row_name(11) }
    fn set_row12Name(&mut self, _: vugra_core::Value) {}
    fn row12Kind(&self) -> vugra_core::Value { self.row_kind(11) }
    fn set_row12Kind(&mut self, _: vugra_core::Value) {}
    fn row12Modified(&self) -> vugra_core::Value { self.row_modified(11) }
    fn set_row12Modified(&mut self, _: vugra_core::Value) {}
    fn row12Size(&self) -> vugra_core::Value { self.row_size(11) }
    fn set_row12Size(&mut self, _: vugra_core::Value) {}
    fn row12Class(&self) -> vugra_core::Value { self.row_class(11) }
    fn set_row12Class(&mut self, _: vugra_core::Value) {}
    fn row12Selected(&self) -> vugra_core::Value { self.row_selected(11) }
    fn set_row12Selected(&mut self, _: vugra_core::Value) {}
    fn documentsLabel(&self) -> vugra_core::Value { "Documents".into() }
    fn set_documentsLabel(&mut self, _: vugra_core::Value) {}
    fn downloadsLabel(&self) -> vugra_core::Value { "Downloads".into() }
    fn set_downloadsLabel(&mut self, _: vugra_core::Value) {}
    fn picturesLabel(&self) -> vugra_core::Value { "Pictures".into() }
    fn set_picturesLabel(&mut self, _: vugra_core::Value) {}
    fn documentsActive(&self) -> vugra_core::Value { true.into() }
    fn set_documentsActive(&mut self, _: vugra_core::Value) {}
    fn downloadsActive(&self) -> vugra_core::Value { false.into() }
    fn set_downloadsActive(&mut self, _: vugra_core::Value) {}
    fn picturesActive(&self) -> vugra_core::Value { false.into() }
    fn set_picturesActive(&mut self, _: vugra_core::Value) {}
    fn searchQuery(&self) -> vugra_core::Value { self.search_query.clone().into() }
    fn set_searchQuery(&mut self, value: vugra_core::Value) { self.search_query = value.as_text(); }
    fn favoritesLabel(&self) -> vugra_core::Value { "Favorites".into() }
    fn set_favoritesLabel(&mut self, _: vugra_core::Value) {}
    fn workspaceLabel(&self) -> vugra_core::Value { "Workspace".into() }
    fn set_workspaceLabel(&mut self, _: vugra_core::Value) {}
    fn favoritesOpen(&self) -> vugra_core::Value { true.into() }
    fn set_favoritesOpen(&mut self, _: vugra_core::Value) {}
    fn workspaceOpen(&self) -> vugra_core::Value { true.into() }
    fn set_workspaceOpen(&mut self, _: vugra_core::Value) {}
    fn projectALabel(&self) -> vugra_core::Value { "Current Project".into() }
    fn set_projectALabel(&mut self, _: vugra_core::Value) {}
    fn projectBLabel(&self) -> vugra_core::Value { "Parent Folder".into() }
    fn set_projectBLabel(&mut self, _: vugra_core::Value) {}
    fn projectAActive(&self) -> vugra_core::Value { false.into() }
    fn set_projectAActive(&mut self, _: vugra_core::Value) {}
    fn projectBActive(&self) -> vugra_core::Value { false.into() }
    fn set_projectBActive(&mut self, _: vugra_core::Value) {}
    fn itemMenuOpen(&self) -> vugra_core::Value { self.item_menu_open.into() }
    fn set_itemMenuOpen(&mut self, value: vugra_core::Value) { self.item_menu_open = matches!(value, vugra_core::Value::Bool(true)); }
    fn blankMenuOpen(&self) -> vugra_core::Value { self.blank_menu_open.into() }
    fn set_blankMenuOpen(&mut self, value: vugra_core::Value) { self.blank_menu_open = matches!(value, vugra_core::Value::Bool(true)); }
    fn renameText(&self) -> vugra_core::Value { self.rename_text.clone().into() }
    fn set_renameText(&mut self, value: vugra_core::Value) { self.rename_text = value.as_text(); }
    fn previewOpen(&self) -> vugra_core::Value { self.preview_open.into() }
    fn set_previewOpen(&mut self, value: vugra_core::Value) { self.preview_open = matches!(value, vugra_core::Value::Bool(true)); }
    fn previewTitle(&self) -> vugra_core::Value { self.preview_title.clone().into() }
    fn set_previewTitle(&mut self, value: vugra_core::Value) { self.preview_title = value.as_text(); }
    fn previewBody(&self) -> vugra_core::Value { self.preview_body.clone().into() }
    fn set_previewBody(&mut self, value: vugra_core::Value) { self.preview_body = value.as_text(); }
    fn sidebarClass(&self) -> vugra_core::Value { self.sidebar_class().into() }
    fn set_sidebarClass(&mut self, _: vugra_core::Value) {}
    fn splitterClass(&self) -> vugra_core::Value { self.splitter_class().into() }
    fn set_splitterClass(&mut self, _: vugra_core::Value) {}

    fn Back(&mut self) {}
    fn SelectRow1(&mut self) { self.selected = 0; }
    fn SelectRow2(&mut self) { self.selected = 1; }
    fn SelectRow3(&mut self) { self.selected = 2; }
    fn SelectRow4(&mut self) { self.selected = 3; }
    fn SelectRow5(&mut self) { self.selected = 4; }
    fn SelectRow6(&mut self) { self.selected = 5; }
    fn SelectRow7(&mut self) { self.selected = 6; }
    fn SelectRow8(&mut self) { self.selected = 7; }
    fn SelectRow9(&mut self) { self.selected = 8; }
    fn SelectRow10(&mut self) { self.selected = 9; }
    fn SelectRow11(&mut self) { self.selected = 10; }
    fn SelectRow12(&mut self) { self.selected = 11; }
    fn OpenDocuments(&mut self) {}
    fn OpenDownloads(&mut self) {}
    fn OpenPictures(&mut self) {}
    fn SelectPrevious(&mut self) { self.selected = self.selected.saturating_sub(1); }
    fn SelectNext(&mut self) { self.selected = (self.selected + 1).min(11); }
    fn SearchInput(&mut self, event: vugra_core::Event) { self.search_query.push_str(&event.text); }
    fn SearchBackspace(&mut self) { self.search_query.pop(); }
    fn SearchClear(&mut self) { self.search_query.clear(); }
    fn OpenSelected(&mut self) {}
    fn OpenParent(&mut self) {}
    fn ToggleFavorites(&mut self) {}
    fn ToggleWorkspace(&mut self) {}
    fn OpenProjectA(&mut self) {}
    fn OpenProjectB(&mut self) {}
    fn DismissOverlay(&mut self) { self.item_menu_open = false; self.blank_menu_open = false; }
    fn Forward(&mut self) {}
    fn BeginRename(&mut self) { self.rename_text = self.rows()[self.selected].name.to_string(); self.DismissOverlay(); }
    fn CancelRename(&mut self) { self.rename_text.clear(); }
    fn CommitRename(&mut self) { self.rename_text.clear(); }
    fn DeleteSelected(&mut self) {}
    fn DuplicateSelected(&mut self) {}
    fn NewFolder(&mut self) {}
    fn ShowBlankMenu(&mut self) { self.item_menu_open = false; self.blank_menu_open = true; }
    fn ClosePreview(&mut self) { self.preview_open = false; }
    fn ClearSelection(&mut self) { self.selected = 0; self.item_menu_open = false; self.blank_menu_open = false; self.rename_text.clear(); }
    fn Paste(&mut self) { self.DismissOverlay(); }
    fn Refresh(&mut self) { self.DismissOverlay(); }
    fn SelectAll(&mut self) {}
    fn HoverSplitter(&mut self) { self.splitter_hovered = true; }
    fn ResizeSidebar(&mut self, event: vugra_core::Event) {
        if event.delta_x < -8.0 && self.sidebar_mode > 1 {
            self.sidebar_mode -= 1;
        } else if event.delta_x < -8.0 && self.sidebar_mode == 1 {
            self.sidebar_mode = 0;
        } else if event.delta_x > 8.0 && self.sidebar_mode < 3 {
            self.sidebar_mode += 1;
        }
        self.splitter_hovered = false;
    }
    fn ShowRow1Menu(&mut self) { self.selected = 0; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow2Menu(&mut self) { self.selected = 1; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow3Menu(&mut self) { self.selected = 2; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow4Menu(&mut self) { self.selected = 3; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow5Menu(&mut self) { self.selected = 4; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow6Menu(&mut self) { self.selected = 5; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow7Menu(&mut self) { self.selected = 6; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow8Menu(&mut self) { self.selected = 7; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow9Menu(&mut self) { self.selected = 8; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow10Menu(&mut self) { self.selected = 9; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow11Menu(&mut self) { self.selected = 10; self.item_menu_open = true; self.blank_menu_open = false; }
    fn ShowRow12Menu(&mut self) { self.selected = 11; self.item_menu_open = true; self.blank_menu_open = false; }
    fn HoverRow1(&mut self) {}
    fn HoverRow2(&mut self) {}
    fn HoverRow3(&mut self) {}
    fn HoverRow4(&mut self) {}
    fn HoverRow5(&mut self) {}
    fn HoverRow6(&mut self) {}
    fn HoverRow7(&mut self) {}
    fn HoverRow8(&mut self) {}
    fn HoverRow9(&mut self) {}
    fn HoverRow10(&mut self) {}
    fn HoverRow11(&mut self) {}
    fn HoverRow12(&mut self) {}
    fn OpenRow1(&mut self) { self.OpenSelected(); }
    fn OpenRow2(&mut self) { self.selected = 1; self.OpenSelected(); }
    fn OpenRow3(&mut self) { self.selected = 2; self.OpenSelected(); }
    fn OpenRow4(&mut self) { self.selected = 3; self.OpenSelected(); }
    fn OpenRow5(&mut self) { self.selected = 4; self.OpenSelected(); }
    fn OpenRow6(&mut self) { self.selected = 5; self.OpenSelected(); }
    fn OpenRow7(&mut self) { self.selected = 6; self.OpenSelected(); }
    fn OpenRow8(&mut self) { self.selected = 7; self.OpenSelected(); }
    fn OpenRow9(&mut self) { self.selected = 8; self.OpenSelected(); }
    fn OpenRow10(&mut self) { self.selected = 9; self.OpenSelected(); }
    fn OpenRow11(&mut self) { self.selected = 10; self.OpenSelected(); }
    fn OpenRow12(&mut self) { self.selected = 11; self.OpenSelected(); }
}
`
}

func repoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if modulePath, readErr := os.ReadFile(filepath.Join(dir, "go.mod")); readErr == nil && strings.Contains(string(modulePath), "module github.com/vugra/vugra") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	_, file, _, ok := runtime.Caller(0)
	if ok {
		return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
	}
	return "", fmt.Errorf("locate repository root")
}

func rustSFCExamplePath(start string) (string, error) {
	dir, err := repoRoot(start)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(dir, "examples", "rust-counter", "Counter.vue")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("locate examples/rust-counter/Counter.vue")
}

func rustFinderSFCExamplePath(start string) (string, error) {
	dir, err := repoRoot(start)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(dir, "examples", "finder-rust-sfc", "FinderLite.vue")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("locate examples/finder-rust-sfc/FinderLite.vue")
}
