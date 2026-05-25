use std::collections::{HashMap, HashSet};
use std::env;
use std::path::Path;
use std::time::SystemTime;

use chrono::{DateTime, Datelike, Local};
use vugra_abi::AbiState;
use vugra_core::{finder_lite_contract, App, ComponentState, Event, Value};
#[cfg(test)]
use vugra_host_native::{
    dispatch_native_context_menu, dispatch_native_double_click, dispatch_native_drag,
    dispatch_native_hover, dispatch_native_pointer_with_modifiers, render_native_frame,
};
use vugra_host_native::{
    dispatch_native_key, dispatch_native_pointer, dispatch_native_text, render_test_frame,
    NativeFrame, NativeKey, NativeRenderBackend, NativeWindowConfig,
};
use vugra_ir::{MethodId, SignalId};
use vugra_layout::Constraints;
use vugra_render::RenderCommand;
use vugra_rs::App as RustApp;
use vugra_system::{
    clean_path_str, join_clean, sibling_path, split_parent_name, Entry as SystemEntry, FileSystem,
    FsError, OsFileSystem,
};

#[allow(dead_code)]
mod generated {
    include!("../../../crates/vugra-rs-codegen/tests/generated_finder_lite_adapter.rs");
}

fn main() {
    let variant = env::args().nth(1).unwrap_or_else(|| "direct".to_string());
    match variant.as_str() {
        "direct" => print_direct(),
        "abi" => print_abi(),
        "generated-adapter-smoke" => {
            if let Err(err) = run_generated_adapter_smoke() {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "parity-summary" => {
            if let Err(err) = run_parity_summary() {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "native" | "direct-native" | "native-vello" => {
            run_direct_native(NativeRenderBackend::Vello)
        }
        "native-software" => run_direct_native(NativeRenderBackend::Software),
        "native-wgpu" => run_direct_native(NativeRenderBackend::Wgpu),
        "abi-native" | "abi-native-vello" => run_abi_native(NativeRenderBackend::Vello),
        "abi-native-software" => run_abi_native(NativeRenderBackend::Software),
        "abi-native-wgpu" => run_abi_native(NativeRenderBackend::Wgpu),
        "native-smoke" => {
            if let Err(err) = run_native_smoke() {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "native-window-smoke" => {
            if let Err(err) =
                run_native_window_smoke("native-window-smoke", NativeRenderBackend::Vello)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "native-vello-window-smoke" => {
            if let Err(err) =
                run_native_window_smoke("native-vello-window-smoke", NativeRenderBackend::Vello)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "native-software-window-smoke" => {
            if let Err(err) = run_native_window_smoke(
                "native-software-window-smoke",
                NativeRenderBackend::Software,
            ) {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "native-wgpu-window-smoke" => {
            if let Err(err) =
                run_native_window_smoke("native-wgpu-window-smoke", NativeRenderBackend::Wgpu)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "abi-window-smoke" => {
            if let Err(err) = run_abi_window_smoke("abi-window-smoke", NativeRenderBackend::Vello) {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "abi-vello-window-smoke" => {
            if let Err(err) =
                run_abi_window_smoke("abi-vello-window-smoke", NativeRenderBackend::Vello)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "abi-software-window-smoke" => {
            if let Err(err) =
                run_abi_window_smoke("abi-software-window-smoke", NativeRenderBackend::Software)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "abi-wgpu-window-smoke" => {
            if let Err(err) =
                run_abi_window_smoke("abi-wgpu-window-smoke", NativeRenderBackend::Wgpu)
            {
                eprintln!("{err}");
                std::process::exit(1);
            }
        }
        "vello-device-smoke" => {
            if let Err(err) = run_vello_device_smoke() {
                eprintln!("{err:#}");
                std::process::exit(1);
            }
        }
        "wgpu-device-smoke" => {
            if let Err(err) = run_wgpu_device_smoke() {
                eprintln!("{err:#}");
                std::process::exit(1);
            }
        }
        _ => {
            eprintln!(
                "usage: finder-rust [direct|abi|generated-adapter-smoke|parity-summary|native|native-software|native-vello|native-wgpu|abi-native|abi-native-software|abi-native-vello|abi-native-wgpu|native-smoke|native-window-smoke|native-software-window-smoke|native-vello-window-smoke|native-wgpu-window-smoke|abi-window-smoke|abi-software-window-smoke|abi-vello-window-smoke|abi-wgpu-window-smoke|vello-device-smoke|wgpu-device-smoke]"
            );
            std::process::exit(2);
        }
    }
}

fn print_direct() {
    print!("{}", render_direct_text());
}

fn render_direct_text() -> String {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::new());
    app.dispatch(MethodId(2));
    render_rust_api_for_test_host(&app)
}

fn run_direct_native(backend: NativeRenderBackend) {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::with_os_file_system());
    let config = NativeWindowConfig {
        backend,
        ..NativeWindowConfig::default()
    };
    if let Err(err) = app.run_native(config) {
        eprintln!("run native Finder Lite window: {err}");
        std::process::exit(1);
    }
}

fn run_abi_native(backend: NativeRenderBackend) {
    let component = finder_lite_contract();
    let mut app = App::new(component, finder_lite_abi());
    let config = NativeWindowConfig {
        backend,
        title: format!("Vugra Rust Finder Lite ABI ({})", backend.as_str()),
        ..NativeWindowConfig::default()
    };
    if let Err(err) = vugra_host_native::run_app_window(&mut app, config) {
        eprintln!("run ABI native Finder Lite window: {err}");
        std::process::exit(1);
    }
}

fn run_native_smoke() -> Result<(), String> {
    let constraints = Constraints {
        width: 800.0,
        height: 600.0,
    };
    let width = 800;
    let height = 600;

    for backend in [
        NativeRenderBackend::Software,
        NativeRenderBackend::Vello,
        NativeRenderBackend::Wgpu,
    ] {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));

        let direct_frame = vugra_host_native::render_native_frame(
            direct.inner(),
            constraints,
            backend,
            width,
            height,
        );
        let abi_frame =
            vugra_host_native::render_native_frame(&abi, constraints, backend, width, height);

        validate_native_smoke_frame("direct", backend, &direct_frame, width, height)?;
        validate_native_smoke_frame("abi", backend, &abi_frame, width, height)?;
        if direct_frame != abi_frame {
            return Err(format!(
                "native-smoke failed: direct and ABI frames differ for {}",
                backend.as_str()
            ));
        }
        validate_native_interactions(backend, constraints, width, height)?;
        println!(
            "direct {} commands={} pixels={} drawn={}",
            backend.as_str(),
            direct_frame.commands.len(),
            direct_frame.pixels.len(),
            drawn_pixel_count(&direct_frame)
        );
        println!(
            "abi {} commands={} pixels={} drawn={}",
            backend.as_str(),
            abi_frame.commands.len(),
            abi_frame.pixels.len(),
            drawn_pixel_count(&abi_frame)
        );
    }

    println!("native-smoke ok");
    Ok(())
}

fn run_generated_adapter_smoke() -> Result<(), String> {
    let component = finder_lite_contract();
    let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
    let mut generated = App::new(
        component,
        generated::FinderLiteAdapter::new(FinderLiteDirect::new()),
    );

    direct.dispatch(MethodId(2));
    generated.dispatch(MethodId(2));
    assert_generated_matches_direct(&direct, &generated, "select row 1")?;

    direct.dispatch(MethodId(9));
    generated.dispatch(MethodId(9));
    assert_generated_matches_direct(&direct, &generated, "select next")?;

    let event = Event {
        kind: "text".to_string(),
        text: "road".to_string(),
        ..Event::default()
    };
    direct.dispatch_event(MethodId(10), event.clone());
    generated.dispatch_event(MethodId(10), event);
    assert_generated_matches_direct(&direct, &generated, "search input")?;

    for _ in 0..4 {
        direct.dispatch(MethodId(11));
        generated.dispatch(MethodId(11));
    }
    assert_generated_matches_direct(&direct, &generated, "search backspace")?;

    direct.dispatch(MethodId(13));
    generated.dispatch(MethodId(13));
    assert_generated_matches_direct(&direct, &generated, "open selected")?;

    let frame = generated.render_frame();
    println!(
        "generated-adapter path={} row={} status={:?}",
        frame.path,
        frame
            .rows
            .first()
            .map(|row| row.name.as_str())
            .unwrap_or(""),
        frame.status
    );
    println!("generated-adapter-smoke ok");
    Ok(())
}

fn run_parity_summary() -> Result<(), String> {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::new());
    print_parity_phase("initial", &app);
    app.dispatch(MethodId(2));
    print_parity_phase("selection", &app);
    app.dispatch_event(
        MethodId(10),
        Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        },
    );
    print_parity_phase("search", &app);
    println!("rust parity-summary ok");
    Ok(())
}

fn print_parity_phase(label: &str, app: &RustApp<FinderLiteDirect>) {
    let constraints = Constraints {
        width: 800.0,
        height: 600.0,
    };
    let frame = vugra_host_native::render_native_frame(
        app.inner(),
        constraints,
        NativeRenderBackend::Software,
        800,
        600,
    );
    println!("phase\t{label}");
    println!("metric\tcommands\t{}", frame.commands.len());
    println!("metric\tpixels\t{}", frame.pixels.len());
    println!("metric\tdrawn\t{}", drawn_pixel_count(&frame));
    for key in [
        "toolbar",
        "path",
        "path:text",
        "search",
        "search:text",
        "sidebar",
        "splitter",
        "file-pane",
        "file-header",
        "file-list",
        "header-name",
        "header-name:text",
        "header-kind",
        "header-kind:text",
        "header-size",
        "header-size:text",
        "statusbar",
        "status-text",
        "status-text:text",
        "selected-summary",
        "selected-summary:text",
        "row1",
        "row1-name",
        "row1-name:text",
        "row1-modified",
        "row1-modified:text",
        "row1-size",
        "row1-size:text",
        "row2",
        "row2-name",
        "row2-name:text",
        "row2-modified",
        "row2-modified:text",
        "row2-size",
        "row2-size:text",
    ] {
        if let Some(rect) = rust_command_rect_by_id(&frame.commands, key) {
            println!(
                "rect\t{key}\t{:.1}\t{:.1}\t{:.1}\t{:.1}",
                rect.x, rect.y, rect.width, rect.height
            );
        }
    }
    for command in &frame.commands {
        if let RenderCommand::Text { text, .. } = command {
            if !text.is_empty() {
                println!("text\t{text}");
            }
        }
    }
    let ops = vugra_render_vello::lower_commands(&frame.commands);
    for (key, id) in [
        ("toolbar", "toolbar"),
        ("path", "path"),
        ("search", "search"),
        ("sidebar", "sidebar"),
        ("sidebar-active-item", "sidebar-documents"),
        ("splitter", "splitter"),
        ("file-pane", "file-pane"),
        ("file-header", "file-header"),
        ("file-list", "file-list"),
        ("statusbar", "statusbar"),
        ("row1", "row1"),
    ] {
        if let Some(color) = rust_vello_fill_color_by_id(&ops, id) {
            println!("fill\t{key}\t{color}");
        }
    }
    if let Some(row_index) = rust_row_index_by_name(&frame.commands, "Roadmap.md") {
        let row_key = format!("row{row_index}");
        let name_key = format!("{row_key}-name:text");
        let size_key = format!("{row_key}-size:text");
        if let Some(rect) = rust_command_rect_by_id(&frame.commands, &row_key) {
            println!(
                "rect\trow-roadmap\t{:.1}\t{:.1}\t{:.1}\t{:.1}",
                rect.x, rect.y, rect.width, rect.height
            );
        }
        if let Some(rect) = rust_vello_text_rect_by_id(&ops, &name_key)
            .or_else(|| rust_command_rect_by_id(&frame.commands, &name_key))
        {
            println!(
                "rect\trow-roadmap-name:text\t{:.1}\t{:.1}\t{:.1}\t{:.1}",
                rect.x, rect.y, rect.width, rect.height
            );
        }
        if let Some(rect) = rust_vello_text_rect_by_id(&ops, &size_key)
            .or_else(|| rust_command_rect_by_id(&frame.commands, &size_key))
        {
            println!(
                "rect\trow-roadmap-size:text\t{:.1}\t{:.1}\t{:.1}\t{:.1}",
                rect.x, rect.y, rect.width, rect.height
            );
        }
    }
    for key in [
        "toolbar",
        "path",
        "search",
        "sidebar",
        "file-header",
        "statusbar",
        "row1",
    ] {
        if let Some(rect) = rust_command_rect_by_id(&frame.commands, key) {
            if let Some(color) = rust_sample_border_color(&frame, rect) {
                println!("border\t{key}\t{color}");
            }
        }
    }
    for key in ["path", "search", "row1"] {
        if let Some(radius) = rust_command_corner_radius_by_id(&frame.commands, key) {
            println!("radius\t{key}\t{radius:.1}");
        }
    }
    for (key, id) in [
        ("path:text", "path:text"),
        ("search:text", "search:text"),
        ("header-name:text", "header-name:text"),
        ("header-kind:text", "header-kind:text"),
        ("header-size:text", "header-size:text"),
        ("sidebar-active-label:text", "sidebar-documents-label:text"),
        ("status-text:text", "status-text:text"),
        ("selected-summary:text", "selected-summary:text"),
        ("row1-name:text", "row1-name:text"),
        ("row1-modified:text", "row1-modified:text"),
        ("row1-size:text", "row1-size:text"),
        ("row2-name:text", "row2-name:text"),
        ("row2-modified:text", "row2-modified:text"),
        ("row2-size:text", "row2-size:text"),
    ] {
        if let Some(color) = rust_vello_text_color_by_id(&ops, id) {
            println!("color\t{key}\t{color}");
        }
    }
    if let Some(row_index) = rust_row_index_by_name(&frame.commands, "Roadmap.md") {
        for (key, id) in [
            ("row-roadmap-name:text", format!("row{row_index}-name:text")),
            ("row-roadmap-size:text", format!("row{row_index}-size:text")),
        ] {
            if let Some(color) = rust_vello_text_color_by_id(&ops, &id) {
                println!("color\t{key}\t{color}");
            }
        }
    }
}

fn rust_row_index_by_name(commands: &[RenderCommand], name: &str) -> Option<usize> {
    commands.iter().find_map(|command| match command {
        RenderCommand::Text { id, text, .. } if text == name => id
            .strip_prefix("row")
            .and_then(|rest| rest.strip_suffix("-name:text"))
            .and_then(|index| index.parse::<usize>().ok()),
        _ => None,
    })
}

fn rust_command_rect_by_id(commands: &[RenderCommand], id: &str) -> Option<vugra_layout::Rect> {
    commands.iter().find_map(|command| match command {
        RenderCommand::Element {
            id: command_id,
            rect,
            ..
        } if command_id == id => Some(*rect),
        RenderCommand::Text {
            id: command_id,
            rect,
            ..
        } if command_id == id => Some(*rect),
        _ => None,
    })
}

fn rust_vello_text_rect_by_id(
    ops: &[vugra_render_vello::VelloOp],
    id: &str,
) -> Option<vugra_layout::Rect> {
    ops.iter().find_map(|op| match op {
        vugra_render_vello::VelloOp::Text {
            id: command_id,
            rect,
            run,
            ..
        } if command_id == id => Some(vugra_layout::Rect {
            x: run.x,
            y: run.y,
            width: run.advance(),
            height: rect.height,
        }),
        _ => None,
    })
}

fn rust_command_corner_radius_by_id(commands: &[RenderCommand], id: &str) -> Option<f32> {
    commands.iter().find_map(|command| match command {
        RenderCommand::Element {
            id: command_id,
            role,
            visual_state,
            ..
        } if command_id == id => vugra_host_native::native_role_corner_radius(role, *visual_state),
        _ => None,
    })
}

fn rust_sample_border_color(frame: &NativeFrame, rect: vugra_layout::Rect) -> Option<String> {
    let x = rect.x.round() as isize;
    let y = rect.y.round() as isize;
    if x < 0 || y < 0 {
        return None;
    }
    let x = x as usize;
    let y = y as usize;
    let width = 800usize;
    let height = 600usize;
    if x >= width || y >= height || frame.pixels.len() != width * height {
        return None;
    }
    let color = frame.pixels[y * width + x];
    Some(format!(
        "#{:02x}{:02x}{:02x}",
        (color >> 16) & 0xff,
        (color >> 8) & 0xff,
        color & 0xff
    ))
}

fn rust_vello_fill_color_by_id(ops: &[vugra_render_vello::VelloOp], id: &str) -> Option<String> {
    ops.iter().find_map(|op| match op {
        vugra_render_vello::VelloOp::Fill {
            id: command_id,
            color,
            ..
        } if command_id == id => Some(format!("#{:02x}{:02x}{:02x}", color.0, color.1, color.2)),
        _ => None,
    })
}

fn rust_vello_text_color_by_id(ops: &[vugra_render_vello::VelloOp], id: &str) -> Option<String> {
    ops.iter().find_map(|op| match op {
        vugra_render_vello::VelloOp::Text {
            id: command_id,
            color,
            ..
        } if command_id == id => Some(format!("#{:02x}{:02x}{:02x}", color.0, color.1, color.2)),
        _ => None,
    })
}

fn assert_generated_matches_direct(
    direct: &RustApp<FinderLiteDirect>,
    generated: &App<generated::FinderLiteAdapter<FinderLiteDirect>>,
    phase: &str,
) -> Result<(), String> {
    let direct_frame = direct.inner().render_frame();
    let generated_frame = generated.render_frame();
    if direct_frame != generated_frame {
        return Err(format!(
            "generated-adapter-smoke failed after {phase}: direct={direct_frame:?} generated={generated_frame:?}"
        ));
    }
    Ok(())
}

fn run_native_window_smoke(
    variant: &'static str,
    backend: NativeRenderBackend,
) -> Result<(), String> {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::new());
    app.dispatch(MethodId(2));
    let config = NativeWindowConfig {
        title: format!("Vugra Rust Finder Lite smoke ({})", backend.as_str()),
        width: 640,
        height: 420,
        backend,
    };
    let smoke = app.run_native_for_frames(config, 2).map_err(|err| {
        format!(
            "native-{}-window-smoke failed to open/present native window: {err}",
            backend.as_str()
        )
    })?;
    if smoke.frames_presented < 1 {
        return Err(format!("{variant} failed: no frames were presented"));
    }
    if smoke.commands == 0 {
        return Err(format!("{variant} failed: no commands were rendered"));
    }
    if smoke.pixels != 640 * 420 {
        return Err(format!(
            "{variant} failed: pixels={} expected={}",
            smoke.pixels,
            640 * 420
        ));
    }
    if smoke.drawn_pixels == 0 {
        return Err(format!(
            "{variant} failed: only background pixels were presented"
        ));
    }
    println!(
        "native-window backend={} frames={} commands={} pixels={} drawn={}",
        backend.as_str(),
        smoke.frames_presented,
        smoke.commands,
        smoke.pixels,
        smoke.drawn_pixels
    );
    println!("{variant} ok");
    Ok(())
}

fn run_abi_window_smoke(variant: &'static str, backend: NativeRenderBackend) -> Result<(), String> {
    let component = vugra_abi::vugra_component_finder_lite();
    let state = vugra_abi::create_finder_lite_state();
    let app = vugra_abi::vugra_app_create(component, state);
    if app == 0 {
        return Err("abi-window-smoke failed: app handle was zero".to_string());
    }
    let backend_code = match backend {
        NativeRenderBackend::Software => vugra_abi::VUGRA_NATIVE_BACKEND_SOFTWARE,
        NativeRenderBackend::Vello => vugra_abi::VUGRA_NATIVE_BACKEND_VELLO,
        NativeRenderBackend::Wgpu => vugra_abi::VUGRA_NATIVE_BACKEND_WGPU,
    };
    let smoke = vugra_abi::vugra_app_run_native_window_for_frames(app, backend_code, 640, 420, 2);
    vugra_abi::vugra_app_destroy(app);
    vugra_abi::vugra_state_destroy(state);
    vugra_abi::vugra_component_destroy(component);

    if !smoke.ok {
        return Err(format!(
            "{variant} failed to open/present native window through ABI"
        ));
    }
    if smoke.frames_presented < 1 {
        return Err(format!("{variant} failed: no frames were presented"));
    }
    if smoke.commands_len == 0 {
        return Err(format!("{variant} failed: no commands were rendered"));
    }
    if smoke.pixels_len != 640 * 420 {
        return Err(format!(
            "{variant} failed: pixels={} expected={}",
            smoke.pixels_len,
            640 * 420
        ));
    }
    if smoke.drawn_pixels == 0 {
        return Err(format!(
            "{variant} failed: only background pixels were presented"
        ));
    }
    println!(
        "abi-window backend={} frames={} commands={} pixels={} drawn={}",
        backend.as_str(),
        smoke.frames_presented,
        smoke.commands_len,
        smoke.pixels_len,
        smoke.drawn_pixels
    );
    println!("{variant} ok");
    Ok(())
}

fn run_vello_device_smoke() -> anyhow::Result<()> {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::new());
    app.dispatch(MethodId(2));
    let constraints = Constraints {
        width: 800.0,
        height: 600.0,
    };
    let frame = app.inner().render_frame();
    let layout = vugra_layout::layout_frame(&frame, constraints);
    let scene = vugra_scene::build_scene(&layout);
    let mut renderer = vugra_render_vello::VelloRenderer::default();
    vugra_render::Renderer::render(&mut renderer, &scene);
    let image = vugra_render_vello::render_ops_offscreen(renderer.ops(), 800, 600)
        .map_err(anyhow::Error::msg)?;
    if image.pixels.len() != 800 * 600 * 4 {
        anyhow::bail!(
            "vello-device-smoke failed: pixels={} expected={}",
            image.pixels.len(),
            800 * 600 * 4
        );
    }
    if image.checksum == 0 {
        anyhow::bail!("vello-device-smoke failed: zero checksum");
    }
    println!(
        "vello-device commands={} ops={} pixels={} checksum={}",
        renderer.commands().len(),
        renderer.ops().len(),
        image.pixels.len(),
        image.checksum
    );
    println!("vello-device-smoke ok");
    Ok(())
}

fn run_wgpu_device_smoke() -> anyhow::Result<()> {
    let component = finder_lite_contract();
    let mut app = RustApp::mount(component, FinderLiteDirect::new());
    app.dispatch(MethodId(2));
    let constraints = Constraints {
        width: 800.0,
        height: 600.0,
    };
    let frame = app.inner().render_frame();
    let layout = vugra_layout::layout_frame(&frame, constraints);
    let scene = vugra_scene::build_scene(&layout);
    let mut renderer = vugra_render_wgpu::WgpuRenderer::default();
    vugra_render::Renderer::render(&mut renderer, &scene);
    if !renderer.pass().iter().any(|op| {
        matches!(
            op,
            vugra_render_wgpu::WgpuPassOp::Text { text, .. } if text == "Documents"
        )
    }) {
        anyhow::bail!(
            "wgpu-device-smoke failed: Documents path text was not lowered into the pass"
        );
    }
    let image = vugra_render_wgpu::render_pass_offscreen(renderer.pass(), 800, 600)
        .map_err(anyhow::Error::msg)?;
    if image.pixels.len() != 800 * 600 * 4 {
        anyhow::bail!(
            "wgpu-device-smoke failed: pixels={} expected={}",
            image.pixels.len(),
            800 * 600 * 4
        );
    }
    if image.checksum == 0 {
        anyhow::bail!("wgpu-device-smoke failed: zero checksum");
    }
    let text_pixels = image
        .pixels
        .chunks_exact(4)
        .filter(|pixel| pixel[0] == 75 && pixel[1] == 85 && pixel[2] == 99 && pixel[3] == 255)
        .count();
    if text_pixels < 20 {
        anyhow::bail!(
            "wgpu-device-smoke failed: expected dark text pixels, got {}",
            text_pixels
        );
    }
    println!(
        "wgpu-device commands={} pass={} pixels={} text_pixels={} checksum={}",
        renderer.commands().len(),
        renderer.pass().len(),
        image.pixels.len(),
        text_pixels,
        image.checksum
    );
    println!("wgpu-device-smoke ok");
    Ok(())
}

fn validate_native_interactions(
    backend: NativeRenderBackend,
    constraints: Constraints,
    width: usize,
    height: usize,
) -> Result<(), String> {
    let component = finder_lite_contract();
    let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
    let mut abi = App::new(component, finder_lite_abi());
    direct.dispatch(MethodId(2));
    abi.dispatch(MethodId(2));

    let direct_frame =
        vugra_host_native::render_native_frame(direct.inner(), constraints, backend, width, height);
    let abi_frame =
        vugra_host_native::render_native_frame(&abi, constraints, backend, width, height);
    if !dispatch_native_pointer(direct.inner_mut(), &direct_frame, 260.0, 134.0) {
        return Err(format!(
            "native-smoke failed: direct {} row pointer did not dispatch",
            backend.as_str()
        ));
    }
    if !dispatch_native_pointer(&mut abi, &abi_frame, 260.0, 134.0) {
        return Err(format!(
            "native-smoke failed: abi {} row pointer did not dispatch",
            backend.as_str()
        ));
    }
    assert_matching_frames(
        backend,
        constraints,
        width,
        height,
        &direct,
        &abi,
        "row pointer",
    )?;

    if !dispatch_native_text(direct.inner_mut(), "road".to_string())
        || !dispatch_native_text(&mut abi, "road".to_string())
    {
        return Err(format!(
            "native-smoke failed: {} search text did not dispatch",
            backend.as_str()
        ));
    }
    assert_matching_frames(
        backend,
        constraints,
        width,
        height,
        &direct,
        &abi,
        "search text",
    )?;
    let searched = direct.inner().render_frame();
    if searched.search_query != "road"
        || searched.rows.len() != 1
        || searched.rows[0].name != "Roadmap.md"
    {
        return Err(format!(
            "native-smoke failed: direct {} search did not filter Roadmap.md",
            backend.as_str()
        ));
    }

    for _ in 0..4 {
        if !dispatch_native_key(direct.inner_mut(), NativeKey::Backspace)
            || !dispatch_native_key(&mut abi, NativeKey::Backspace)
        {
            return Err(format!(
                "native-smoke failed: {} Backspace did not dispatch",
                backend.as_str()
            ));
        }
    }
    assert_matching_frames(
        backend,
        constraints,
        width,
        height,
        &direct,
        &abi,
        "Backspace",
    )?;

    if !dispatch_native_key(direct.inner_mut(), NativeKey::Enter)
        || !dispatch_native_key(&mut abi, NativeKey::Enter)
    {
        return Err(format!(
            "native-smoke failed: {} Enter did not dispatch",
            backend.as_str()
        ));
    }
    assert_matching_frames(backend, constraints, width, height, &direct, &abi, "Enter")?;
    if !direct.inner().render_frame().overlays.preview_open {
        return Err(format!(
            "native-smoke failed: direct {} Enter did not open selected file preview",
            backend.as_str()
        ));
    }

    direct.dispatch(MethodId(40));
    abi.dispatch(MethodId(40));
    direct.dispatch(MethodId(2));
    abi.dispatch(MethodId(2));
    if !dispatch_native_key(direct.inner_mut(), NativeKey::Enter)
        || !dispatch_native_key(&mut abi, NativeKey::Enter)
    {
        return Err(format!(
            "native-smoke failed: {} folder Enter did not dispatch",
            backend.as_str()
        ));
    }
    assert_matching_frames(
        backend,
        constraints,
        width,
        height,
        &direct,
        &abi,
        "folder Enter",
    )?;
    if direct.inner().render_frame().path != "Documents/Design" {
        return Err(format!(
            "native-smoke failed: direct {} folder Enter did not open selected folder",
            backend.as_str()
        ));
    }

    dispatch_native_key(direct.inner_mut(), NativeKey::Backspace);
    dispatch_native_key(&mut abi, NativeKey::Backspace);
    dispatch_native_key(direct.inner_mut(), NativeKey::ArrowDown);
    dispatch_native_key(&mut abi, NativeKey::ArrowDown);
    direct.dispatch(MethodId(33));
    abi.dispatch(MethodId(33));
    if !dispatch_native_text(direct.inner_mut(), " v2".to_string())
        || !dispatch_native_text(&mut abi, " v2".to_string())
    {
        return Err(format!(
            "native-smoke failed: {} rename text did not dispatch",
            backend.as_str()
        ));
    }
    if !dispatch_native_key(direct.inner_mut(), NativeKey::Backspace)
        || !dispatch_native_key(&mut abi, NativeKey::Backspace)
    {
        return Err(format!(
            "native-smoke failed: {} rename Backspace did not dispatch",
            backend.as_str()
        ));
    }
    direct
        .inner_mut()
        .state_mut()
        .set_signal(SignalId(102), "Components v2.sketch".into());
    abi.state_mut()
        .set_signal(SignalId(102), "Components v2.sketch".into());
    if !dispatch_native_key(direct.inner_mut(), NativeKey::Enter)
        || !dispatch_native_key(&mut abi, NativeKey::Enter)
    {
        return Err(format!(
            "native-smoke failed: {} rename Enter did not dispatch",
            backend.as_str()
        ));
    }
    assert_matching_frames(
        backend,
        constraints,
        width,
        height,
        &direct,
        &abi,
        "rename Enter",
    )?;
    if !direct
        .inner()
        .render_frame()
        .rows
        .iter()
        .any(|row| row.name == "Components v2.sketch")
    {
        return Err(format!(
            "native-smoke failed: direct {} rename did not commit",
            backend.as_str()
        ));
    }

    Ok(())
}

fn assert_matching_frames(
    backend: NativeRenderBackend,
    constraints: Constraints,
    width: usize,
    height: usize,
    direct: &RustApp<FinderLiteDirect>,
    abi: &App<AbiState>,
    phase: &str,
) -> Result<(), String> {
    let direct_frame =
        vugra_host_native::render_native_frame(direct.inner(), constraints, backend, width, height);
    let abi_frame =
        vugra_host_native::render_native_frame(abi, constraints, backend, width, height);
    if direct_frame != abi_frame {
        return Err(format!(
            "native-smoke failed: direct and ABI frames differ after {phase} for {}",
            backend.as_str()
        ));
    }
    Ok(())
}

fn validate_native_smoke_frame(
    label: &str,
    backend: NativeRenderBackend,
    frame: &NativeFrame,
    width: usize,
    height: usize,
) -> Result<(), String> {
    if frame.commands.is_empty() {
        return Err(format!(
            "native-smoke failed: {label} {} produced no commands",
            backend.as_str()
        ));
    }
    if frame.pixels.len() != width * height {
        return Err(format!(
            "native-smoke failed: {label} {} produced {} pixels, expected {}",
            backend.as_str(),
            frame.pixels.len(),
            width * height
        ));
    }
    if drawn_pixel_count(frame) == 0 {
        return Err(format!(
            "native-smoke failed: {label} {} produced only background pixels",
            backend.as_str()
        ));
    }
    let has_method_bound_row = frame.commands.iter().any(|command| {
        matches!(
            command,
            RenderCommand::Element {
                role,
                method: Some(_),
                ..
            } if role == "row"
        )
    });
    if !has_method_bound_row {
        return Err(format!(
            "native-smoke failed: {label} {} has no method-bound row command",
            backend.as_str()
        ));
    }
    Ok(())
}

fn drawn_pixel_count(frame: &NativeFrame) -> usize {
    frame
        .pixels
        .iter()
        .filter(|pixel| **pixel != 0x00f7f7f8)
        .count()
}

fn print_abi() {
    let component = finder_lite_contract();
    let mut app = App::new(component, finder_lite_abi());
    app.dispatch(MethodId(2));
    print!("{}", render_for_test_host(&app));
}

fn render_for_test_host<S: ComponentState>(app: &App<S>) -> String {
    render_test_frame(
        app,
        Constraints {
            width: 800.0,
            height: 600.0,
        },
    )
    .to_string()
}

fn render_rust_api_for_test_host<S: ComponentState>(app: &RustApp<S>) -> String {
    app.render_test(Constraints {
        width: 800.0,
        height: 600.0,
    })
    .to_string()
}

struct FinderLiteDirect {
    files: Box<dyn FileSystem>,
    sidebar: FinderSidebar,
    path: String,
    history: Vec<String>,
    forward: Vec<String>,
    status: String,
    selected_summary: String,
    rows: Vec<FinderRow>,
    selected_path: Option<String>,
    selected_paths: HashSet<String>,
    anchor: Option<usize>,
    focus: Option<usize>,
    hover: Option<usize>,
    location: Option<FinderSidebarLocation>,
    search_query: String,
    favorites_open: bool,
    workspace_open: bool,
    sidebar_mode: usize,
    splitter_hovered: bool,
    item_menu_open: bool,
    blank_menu_open: bool,
    rename_text: String,
    rename_path: Option<String>,
    preview_open: bool,
    preview_title: String,
    preview_body: String,
}

#[derive(Clone)]
struct FinderRow {
    #[allow(dead_code)]
    id: String,
    name: String,
    kind: String,
    #[allow(dead_code)]
    size: u64,
    #[allow(dead_code)]
    modified: String,
    path: String,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum FinderSidebarLocation {
    Documents,
    Downloads,
    Pictures,
    ProjectA,
    ProjectB,
}

#[derive(Clone, Debug)]
struct FinderSidebar {
    documents: String,
    downloads: String,
    pictures: String,
    project_a: String,
    project_b: String,
}

impl FinderLiteDirect {
    fn new() -> Self {
        Self::with_file_system(Box::new(FixtureFileSystem::finder_lite()))
    }

    fn with_os_file_system() -> Self {
        Self::with_sidebar_and_file_system(Box::new(OsFileSystem), FinderSidebar::os_default())
    }

    #[cfg(test)]
    fn with_fixture(files: FixtureFileSystem, sidebar: FinderSidebar) -> Self {
        Self::with_sidebar_and_file_system(Box::new(files), sidebar)
    }

    fn with_file_system(files: Box<dyn FileSystem>) -> Self {
        Self::with_sidebar_and_file_system(files, FinderSidebar::default())
    }

    fn with_sidebar_and_file_system(files: Box<dyn FileSystem>, sidebar: FinderSidebar) -> Self {
        let path = first_existing_finder_path(files.as_ref(), &sidebar);
        let location = sidebar.location_for_path(&path);
        let mut finder = Self {
            files,
            sidebar,
            path,
            history: Vec::new(),
            forward: Vec::new(),
            status: String::new(),
            selected_summary: "0 items selected".to_string(),
            rows: Vec::new(),
            selected_path: None,
            selected_paths: HashSet::new(),
            anchor: None,
            focus: None,
            hover: None,
            location,
            search_query: String::new(),
            favorites_open: true,
            workspace_open: true,
            sidebar_mode: 0,
            splitter_hovered: false,
            item_menu_open: false,
            blank_menu_open: false,
            rename_text: String::new(),
            rename_path: None,
            preview_open: false,
            preview_title: String::new(),
            preview_body: String::new(),
        };
        finder.reload_current_directory();
        finder.sync_selection_summary();
        finder
    }
}

impl Default for FinderSidebar {
    fn default() -> Self {
        Self {
            documents: "Documents".to_string(),
            downloads: "Downloads".to_string(),
            pictures: "Pictures".to_string(),
            project_a: "Project A".to_string(),
            project_b: "Project B".to_string(),
        }
    }
}

impl FinderSidebar {
    fn os_default() -> Self {
        let home = env::var("HOME").unwrap_or_else(|_| ".".to_string());
        Self {
            documents: join_clean(&home, "Documents"),
            downloads: join_clean(&home, "Downloads"),
            pictures: join_clean(&home, "Pictures"),
            project_a: clean_path_str("."),
            project_b: clean_path_str(".."),
        }
    }

    fn location_for_path(&self, path: &str) -> Option<FinderSidebarLocation> {
        if path_within(path, &self.documents) {
            Some(FinderSidebarLocation::Documents)
        } else if path_within(path, &self.downloads) {
            Some(FinderSidebarLocation::Downloads)
        } else if path_within(path, &self.pictures) {
            Some(FinderSidebarLocation::Pictures)
        } else if path_within(path, &self.project_a) {
            Some(FinderSidebarLocation::ProjectA)
        } else if path_within(path, &self.project_b) {
            Some(FinderSidebarLocation::ProjectB)
        } else {
            None
        }
    }

    fn path_for(&self, location: FinderSidebarLocation) -> &str {
        match location {
            FinderSidebarLocation::Documents => &self.documents,
            FinderSidebarLocation::Downloads => &self.downloads,
            FinderSidebarLocation::Pictures => &self.pictures,
            FinderSidebarLocation::ProjectA => &self.project_a,
            FinderSidebarLocation::ProjectB => &self.project_b,
        }
    }
}

impl FinderLiteDirect {
    fn visible_rows(&self) -> Vec<FinderRow> {
        filter_rows(&self.rows, &self.search_query)
    }

    fn sync_selection_summary(&mut self) {
        self.selected_summary = format!("{} items selected", self.selected_paths.len());
    }

    fn select(&mut self, index: usize) {
        self.select_with_modifiers(index, vugra_core::Modifiers::default());
    }

    fn select_with_modifiers(&mut self, index: usize, modifiers: vugra_core::Modifiers) {
        let visible_rows = self.visible_rows();
        let index = index.min(visible_rows.len().min(12).saturating_sub(1));
        let Some(row) = visible_rows.get(index) else {
            return;
        };
        if modifiers.shift {
            if let Some(anchor) = self.anchor {
                let (start, end) = if anchor <= index {
                    (anchor, index)
                } else {
                    (index, anchor)
                };
                self.selected_paths.clear();
                for row in visible_rows.iter().take(end + 1).skip(start) {
                    self.selected_paths.insert(row.path.clone());
                }
            } else {
                self.selected_paths.clear();
                self.selected_paths.insert(row.path.clone());
                self.anchor = Some(index);
            }
        } else if modifiers.meta || modifiers.ctrl {
            if !self.selected_paths.remove(&row.path) {
                self.selected_paths.insert(row.path.clone());
            }
            self.anchor = Some(index);
        } else {
            self.selected_paths.clear();
            self.selected_paths.insert(row.path.clone());
            self.anchor = Some(index);
        }
        self.selected_path = self.selected_paths.iter().next().cloned();
        if self.selected_paths.contains(&row.path) {
            self.selected_path = Some(row.path.clone());
        }
        self.focus = Some(index);
        self.dismiss_overlay();
        self.sync_selection_summary();
    }

    fn select_previous(&mut self) {
        let next = self
            .focus
            .or_else(|| self.selected_visible_index())
            .map(|current| current.saturating_sub(1))
            .unwrap_or(0);
        self.select(next);
    }

    fn select_next(&mut self) {
        let next = self
            .focus
            .or_else(|| self.selected_visible_index())
            .map(|current| current + 1)
            .unwrap_or(0);
        self.select(next.min(self.visible_rows().len().saturating_sub(1)));
    }

    fn navigate_to_path(&mut self, path: String, remember: bool) {
        let path = clean_path_str(&path);
        match self.files.stat(&path) {
            Ok(entry) if entry.kind == "folder" => {}
            Ok(_) => {
                self.status = format!("{path} is not a folder");
                return;
            }
            Err(err) => {
                self.status = err.to_string();
                return;
            }
        }
        if remember && path != self.path {
            self.history.push(self.path.clone());
            self.forward.clear();
        }
        self.path = path;
        self.location = self.sidebar.location_for_path(&self.path);
        self.selected_path = None;
        self.selected_paths.clear();
        self.anchor = None;
        self.focus = None;
        self.hover = None;
        self.dismiss_overlay();
        self.reload_current_directory();
        self.sync_selection_summary();
    }

    fn open_location(&mut self, location: FinderSidebarLocation) {
        self.location = Some(location);
        self.navigate_to_path(self.sidebar.path_for(location).to_string(), true);
    }

    fn set_directory_path(&mut self, path: String) {
        self.navigate_to_path(path, true);
    }

    fn back(&mut self) {
        let Some(path) = self.history.pop() else {
            return;
        };
        self.forward.push(self.path.clone());
        self.navigate_to_path(path, false);
    }

    fn forward(&mut self) {
        let Some(path) = self.forward.pop() else {
            return;
        };
        self.history.push(self.path.clone());
        self.navigate_to_path(path, false);
    }

    fn reload_current_directory(&mut self) {
        match self.files.read_dir(&self.path) {
            Ok(entries) => {
                self.rows = entries
                    .into_iter()
                    .map(finder_row_from_system_entry)
                    .collect();
                self.status = format!("{} items · Current path: {}", self.rows.len(), self.path);
            }
            Err(err) => {
                self.rows.clear();
                self.status = err;
            }
        }
        self.location = self.sidebar.location_for_path(&self.path);
    }

    fn append_search_text(&mut self, text: &str) {
        self.search_query.push_str(text);
        self.sync_selection_summary();
    }

    fn search_backspace(&mut self) {
        self.search_query.pop();
        self.sync_selection_summary();
    }

    fn clear_search(&mut self) {
        self.search_query.clear();
        self.sync_selection_summary();
    }

    fn select_all(&mut self) {
        self.selected_paths.clear();
        for row in self.visible_rows() {
            self.selected_paths.insert(row.path);
        }
        self.selected_path = self.selected_rows().first().map(|row| row.path.clone());
        if self.selected_paths.is_empty() {
            self.anchor = None;
            self.focus = None;
        } else {
            self.anchor = Some(0);
            self.focus = Some(0);
        }
        self.sync_selection_summary();
    }

    fn selected_row(&self) -> Option<FinderRow> {
        self.rows
            .iter()
            .find(|row| self.selected_paths.contains(&row.path))
            .cloned()
    }

    fn selected_rows(&self) -> Vec<FinderRow> {
        self.rows
            .iter()
            .filter(|row| self.selected_paths.contains(&row.path))
            .cloned()
            .collect()
    }

    fn selected_visible_index(&self) -> Option<usize> {
        let selected_path = self.selected_path.as_ref()?;
        self.visible_rows()
            .iter()
            .position(|row| &row.path == selected_path)
    }

    fn dismiss_overlay(&mut self) {
        self.item_menu_open = false;
        self.blank_menu_open = false;
    }

    fn clear_selection(&mut self) {
        self.selected_path = None;
        self.selected_paths.clear();
        self.anchor = None;
        self.focus = None;
        self.rename_path = None;
        self.rename_text.clear();
        self.dismiss_overlay();
        self.sync_selection_summary();
    }

    fn show_blank_menu(&mut self) {
        self.item_menu_open = false;
        self.blank_menu_open = true;
        self.preview_open = false;
    }

    fn show_row_menu(&mut self, index: usize) {
        self.show_row_menu_with_modifiers(index, vugra_core::Modifiers::default());
    }

    fn show_row_menu_with_modifiers(&mut self, index: usize, modifiers: vugra_core::Modifiers) {
        if self.visible_rows().get(index).is_none() {
            return;
        }
        let row_path = self.visible_rows().get(index).map(|row| row.path.clone());
        if !row_path
            .as_ref()
            .is_some_and(|path| self.selected_paths.contains(path))
        {
            self.select_with_modifiers(index, modifiers);
        }
        self.blank_menu_open = false;
        self.item_menu_open = true;
        self.preview_open = false;
    }

    fn hover_row(&mut self, index: usize) {
        if self.visible_rows().get(index).is_some() {
            self.hover = Some(index);
        }
    }

    fn begin_rename(&mut self) {
        let Some(row) = self.selected_row() else {
            return;
        };
        self.rename_path = Some(row.path);
        self.rename_text = row.name;
        self.dismiss_overlay();
    }

    fn cancel_rename(&mut self) {
        self.rename_path = None;
        self.rename_text.clear();
    }

    fn commit_rename(&mut self) {
        let Some(source) = self.rename_path.clone() else {
            return;
        };
        let name = self.rename_text.trim().to_string();
        if name.is_empty() {
            return;
        }
        let target = sibling_path(&source, &name);
        match self.files.rename(&source, &target) {
            Ok(()) => {
                self.cancel_rename();
                self.reload_current_directory();
                self.select_path(&target, true);
            }
            Err(err) => self.selected_summary = err,
        }
    }

    fn delete_selected(&mut self) {
        let rows = self.selected_rows();
        if rows.is_empty() {
            return;
        }
        for row in rows {
            if let Err(err) = self.files.remove(&row.path) {
                self.selected_summary = err;
                return;
            }
        }
        self.dismiss_overlay();
        self.reload_current_directory();
        self.selected_path = None;
        self.selected_paths.clear();
        self.anchor = None;
        self.focus = None;
        self.sync_selection_summary();
    }

    fn duplicate_selected(&mut self) {
        let rows = self.selected_rows();
        if rows.is_empty() {
            return;
        }
        for row in rows {
            let target = match self.available_sibling_path(&row.path, &format!("{} copy", row.name))
            {
                Ok(target) => target,
                Err(err) => {
                    self.selected_summary = err;
                    return;
                }
            };
            if let Err(err) = self.files.duplicate(&row.path, &target) {
                self.selected_summary = err;
                return;
            }
        }
        self.dismiss_overlay();
        self.reload_current_directory();
        self.sync_selection_summary();
    }

    fn new_folder(&mut self) {
        let target = match self.available_child_path("Untitled Folder") {
            Ok(target) => target,
            Err(err) => {
                self.selected_summary = err;
                return;
            }
        };
        match self.files.mkdir(&target) {
            Ok(()) => {
                self.dismiss_overlay();
                self.reload_current_directory();
                self.select_path(&target, true);
            }
            Err(err) => self.selected_summary = err,
        }
    }

    fn paste(&mut self) {
        self.dismiss_overlay();
    }

    fn refresh(&mut self) {
        self.dismiss_overlay();
        self.reload_current_directory();
        self.sync_selection_summary();
    }

    fn sidebar_class(&self) -> &'static str {
        ["sidebar", "sidebar-200", "sidebar-280", "sidebar-320"][self.sidebar_mode]
    }

    fn splitter_class(&self) -> &'static str {
        if self.splitter_hovered {
            "splitter-hover"
        } else {
            "splitter"
        }
    }

    fn hover_splitter(&mut self) {
        self.splitter_hovered = true;
    }

    fn resize_sidebar(&mut self, delta_x: f32) {
        if delta_x < -8.0 && self.sidebar_mode > 1 {
            self.sidebar_mode -= 1;
        } else if delta_x < -8.0 && self.sidebar_mode == 1 {
            self.sidebar_mode = 0;
        } else if delta_x > 8.0 && self.sidebar_mode < 3 {
            self.sidebar_mode += 1;
        }
        self.splitter_hovered = false;
    }

    fn close_preview(&mut self) {
        self.preview_open = false;
    }

    fn show_preview(&mut self, row: &FinderRow) {
        self.preview_open = true;
        self.item_menu_open = false;
        self.blank_menu_open = false;
        self.preview_title = row.name.clone();
        self.preview_body = format!("System file · {} · {}", row.modified, format_size(row));
    }

    fn open_row(&mut self, index: usize) {
        let Some(row) = self.visible_rows().get(index).cloned() else {
            return;
        };
        self.focus = Some(index);
        self.selected_paths.clear();
        self.selected_paths.insert(row.path.clone());
        self.selected_path = Some(row.path.clone());
        self.anchor = Some(index);
        self.sync_selection_summary();
        if row.kind == "folder" {
            self.set_directory_path(row.path);
        } else {
            self.show_preview(&row);
        }
    }

    fn select_path(&mut self, path: &str, update_focus: bool) {
        let path = clean_path_str(path);
        self.selected_paths.clear();
        self.selected_paths.insert(path.clone());
        self.selected_path = Some(path.clone());
        if let Some(index) = self.visible_rows().iter().position(|row| row.path == path) {
            if update_focus {
                self.focus = Some(index);
            }
            self.anchor = Some(index);
        } else if update_focus && !self.visible_rows().is_empty() {
            self.focus = Some(0);
            self.anchor = Some(0);
        }
        self.sync_selection_summary();
    }

    fn available_child_path(&self, name: &str) -> Result<String, String> {
        let mut candidate = join_clean(&self.path, name);
        let mut copy = 2;
        loop {
            match self.files.stat(&candidate) {
                Err(err) if err.is_not_exist() => return Ok(candidate),
                Err(err) => return Err(err.to_string()),
                Ok(_) => {
                    candidate = join_clean(&self.path, &format!("{name} {copy}"));
                    copy += 1;
                }
            }
        }
    }

    fn available_sibling_path(&self, source: &str, name: &str) -> Result<String, String> {
        let mut candidate = sibling_path(source, name);
        let mut copy = 2;
        loop {
            match self.files.stat(&candidate) {
                Err(err) if err.is_not_exist() => return Ok(candidate),
                Err(err) => return Err(err.to_string()),
                Ok(_) => {
                    candidate = sibling_path(source, &format!("{name} {copy}"));
                    copy += 1;
                }
            }
        }
    }

    fn open_selected(&mut self) {
        let row = self
            .focus
            .and_then(|focus| self.visible_rows().get(focus).cloned())
            .or_else(|| {
                self.selected_visible_index()
                    .and_then(|index| self.visible_rows().get(index).cloned())
            })
            .or_else(|| self.selected_row());
        let Some(row) = row else {
            self.selected_summary = "0 items selected".to_string();
            return;
        };
        if row.kind == "folder" {
            self.set_directory_path(row.path);
        } else {
            self.show_preview(&row);
        }
    }

    fn open_parent(&mut self) {
        let parent = Path::new(&self.path)
            .parent()
            .and_then(Path::to_str)
            .filter(|parent| !parent.is_empty())
            .map(ToOwned::to_owned);
        if let Some(parent) = parent {
            self.set_directory_path(parent);
        }
    }
}

fn finder_row_from_system_entry(entry: SystemEntry) -> FinderRow {
    FinderRow {
        id: entry.path.clone(),
        name: entry.name,
        kind: if entry.kind.is_empty() {
            "file".to_string()
        } else {
            entry.kind
        },
        size: entry.size,
        modified: entry
            .modified_at
            .map(format_modified)
            .unwrap_or_else(|| "--".to_string()),
        path: entry.path,
    }
}

impl ComponentState for FinderLiteDirect {
    fn get_signal(&self, id: SignalId) -> Value {
        let visible_rows = self.visible_rows();
        let row_value = |index: usize, field: &str| -> Value {
            let Some(row) = visible_rows.get(index) else {
                return Value::None;
            };
            match field {
                "name" => row.name.clone().into(),
                "kind" => row.kind.clone().into(),
                "modified" => row.modified.clone().into(),
                "size" => format_size(row).into(),
                "class" => {
                    if self.rename_path.as_ref() == Some(&row.path) {
                        "file-row-editing".into()
                    } else if self.selected_paths.contains(&row.path) {
                        "file-row-selected".into()
                    } else if self.focus == Some(index) {
                        "file-row-focus".into()
                    } else if self.hover == Some(index) {
                        "file-row-hover".into()
                    } else {
                        "file-row".into()
                    }
                }
                "selected" => Value::Bool(self.selected_paths.contains(&row.path)),
                _ => Value::None,
            }
        };
        if let Some((index, field)) = row_signal(id) {
            return row_value(index, field);
        }
        match id.0 {
            1 => self.path.clone().into(),
            2 => {
                if self.search_query.is_empty() {
                    self.status.clone().into()
                } else {
                    format!("{} items · Current path: {}", visible_rows.len(), self.path).into()
                }
            }
            3 => self.selected_summary.clone().into(),
            13 => "Documents".into(),
            14 => "Downloads".into(),
            15 => "Pictures".into(),
            16 => Value::Bool(self.location == Some(FinderSidebarLocation::Documents)),
            17 => Value::Bool(self.location == Some(FinderSidebarLocation::Downloads)),
            18 => Value::Bool(self.location == Some(FinderSidebarLocation::Pictures)),
            19 => self.search_query.clone().into(),
            92 => "Favorites".into(),
            93 => "Workspace".into(),
            94 => self.favorites_open.into(),
            95 => self.workspace_open.into(),
            96 => "Current Project".into(),
            97 => "Parent Folder".into(),
            98 => Value::Bool(self.location == Some(FinderSidebarLocation::ProjectA)),
            99 => Value::Bool(self.location == Some(FinderSidebarLocation::ProjectB)),
            100 => self.item_menu_open.into(),
            101 => self.blank_menu_open.into(),
            102 => self.rename_text.clone().into(),
            103 => self.preview_open.into(),
            104 => self.preview_title.clone().into(),
            105 => self.preview_body.clone().into(),
            106 => self.sidebar_class().into(),
            107 => self.splitter_class().into(),
            _ => Value::None,
        }
    }

    fn set_signal(&mut self, id: SignalId, value: Value) {
        match id.0 {
            1 => self.path = value.as_text(),
            3 => self.selected_summary = value.as_text(),
            19 => {
                self.search_query = value.as_text();
                self.sync_selection_summary();
            }
            102 => self.rename_text = value.as_text(),
            _ => {}
        }
    }

    fn call_method(&mut self, id: MethodId) {
        match id.0 {
            1 => self.back(),
            2 => self.select(0),
            3 => self.select(1),
            4 => self.select(2),
            5 => self.open_location(FinderSidebarLocation::Documents),
            6 => self.open_location(FinderSidebarLocation::Downloads),
            7 => self.open_location(FinderSidebarLocation::Pictures),
            8 => self.select_previous(),
            9 => self.select_next(),
            11 => self.search_backspace(),
            12 => self.clear_search(),
            13 => self.open_selected(),
            14 => self.open_parent(),
            15 => self.favorites_open = !self.favorites_open,
            16 => self.workspace_open = !self.workspace_open,
            17 => self.open_location(FinderSidebarLocation::ProjectA),
            18 => self.open_location(FinderSidebarLocation::ProjectB),
            19 => self.dismiss_overlay(),
            20 => self.forward(),
            24..=32 => self.select((id.0 - 21) as usize),
            33 => self.begin_rename(),
            34 => self.cancel_rename(),
            35 => self.commit_rename(),
            36 => self.delete_selected(),
            37 => self.duplicate_selected(),
            38 => self.new_folder(),
            39 => self.show_blank_menu(),
            40 => self.close_preview(),
            41..=52 => self.show_row_menu((id.0 - 41) as usize),
            53..=64 => self.hover_row((id.0 - 53) as usize),
            65..=76 => self.open_row((id.0 - 65) as usize),
            77 => self.clear_selection(),
            78 => self.paste(),
            79 => self.refresh(),
            80 => self.select_all(),
            81 => self.hover_splitter(),
            82 => self.resize_sidebar(0.0),
            _ => {}
        }
    }

    fn call_event_method(&mut self, id: MethodId, event: Event) {
        match id.0 {
            10 => self.append_search_text(&event.text),
            2 => self.select_with_modifiers(0, event.modifiers),
            3 => self.select_with_modifiers(1, event.modifiers),
            4 => self.select_with_modifiers(2, event.modifiers),
            24..=32 => self.select_with_modifiers((id.0 - 21) as usize, event.modifiers),
            41..=52 => self.show_row_menu_with_modifiers((id.0 - 41) as usize, event.modifiers),
            65..=76 => self.open_row((id.0 - 65) as usize),
            77 => self.clear_selection(),
            82 => self.resize_sidebar(event.delta_x),
            _ => self.call_method(id),
        }
    }
}

impl generated::FinderLiteBindings for FinderLiteDirect {
    fn path(&self) -> Value {
        self.path.clone().into()
    }

    fn status(&self) -> Value {
        if self.search_query.is_empty() {
            self.status.clone().into()
        } else {
            format!(
                "{} items · Current path: {}",
                self.visible_rows().len(),
                self.path
            )
            .into()
        }
    }

    fn selected_summary(&self) -> Value {
        self.selected_summary.clone().into()
    }

    fn documents_label(&self) -> Value {
        "Documents".into()
    }

    fn downloads_label(&self) -> Value {
        "Downloads".into()
    }

    fn pictures_label(&self) -> Value {
        "Pictures".into()
    }

    fn documents_active(&self) -> Value {
        Value::Bool(self.location == Some(FinderSidebarLocation::Documents))
    }

    fn downloads_active(&self) -> Value {
        Value::Bool(self.location == Some(FinderSidebarLocation::Downloads))
    }

    fn pictures_active(&self) -> Value {
        Value::Bool(self.location == Some(FinderSidebarLocation::Pictures))
    }

    fn search_query(&self) -> Value {
        self.search_query.clone().into()
    }

    fn favorites_label(&self) -> Value {
        "Favorites".into()
    }

    fn workspace_label(&self) -> Value {
        "Workspace".into()
    }

    fn favorites_open(&self) -> Value {
        self.favorites_open.into()
    }

    fn workspace_open(&self) -> Value {
        self.workspace_open.into()
    }

    fn project_a_label(&self) -> Value {
        "Current Project".into()
    }

    fn project_b_label(&self) -> Value {
        "Parent Folder".into()
    }

    fn project_a_active(&self) -> Value {
        Value::Bool(self.location == Some(FinderSidebarLocation::ProjectA))
    }

    fn project_b_active(&self) -> Value {
        Value::Bool(self.location == Some(FinderSidebarLocation::ProjectB))
    }

    fn item_menu_open(&self) -> Value {
        self.item_menu_open.into()
    }

    fn blank_menu_open(&self) -> Value {
        self.blank_menu_open.into()
    }

    fn rename_text(&self) -> Value {
        self.rename_text.clone().into()
    }

    fn preview_open(&self) -> Value {
        self.preview_open.into()
    }

    fn preview_title(&self) -> Value {
        self.preview_title.clone().into()
    }

    fn preview_body(&self) -> Value {
        self.preview_body.clone().into()
    }

    fn sidebar_class(&self) -> Value {
        self.sidebar_class().into()
    }

    fn splitter_class(&self) -> Value {
        self.splitter_class().into()
    }

    fn row_value(&self, index: usize, field: &str) -> Value {
        self.generated_row_value(index, field)
    }

    fn back(&mut self) {
        self.back();
    }

    fn forward(&mut self) {
        self.forward();
    }

    fn select_row(&mut self, index: usize) {
        self.select(index);
    }

    fn open_documents(&mut self) {
        self.open_location(FinderSidebarLocation::Documents);
    }

    fn open_downloads(&mut self) {
        self.open_location(FinderSidebarLocation::Downloads);
    }

    fn open_pictures(&mut self) {
        self.open_location(FinderSidebarLocation::Pictures);
    }

    fn select_previous(&mut self) {
        self.select_previous();
    }

    fn select_next(&mut self) {
        self.select_next();
    }

    fn search_input(&mut self, event: Event) {
        self.append_search_text(&event.text);
    }

    fn search_backspace(&mut self) {
        self.search_backspace();
    }

    fn search_clear(&mut self) {
        self.clear_search();
    }

    fn open_selected(&mut self) {
        self.open_selected();
    }

    fn open_parent(&mut self) {
        self.open_parent();
    }

    fn toggle_favorites(&mut self) {
        self.favorites_open = !self.favorites_open;
    }

    fn toggle_workspace(&mut self) {
        self.workspace_open = !self.workspace_open;
    }

    fn open_project_a(&mut self) {
        self.open_location(FinderSidebarLocation::ProjectA);
    }

    fn open_project_b(&mut self) {
        self.open_location(FinderSidebarLocation::ProjectB);
    }

    fn dismiss_overlay(&mut self) {
        self.dismiss_overlay();
    }

    fn clear_selection(&mut self) {
        self.clear_selection();
    }

    fn begin_rename(&mut self) {
        self.begin_rename();
    }

    fn cancel_rename(&mut self) {
        self.cancel_rename();
    }

    fn commit_rename(&mut self) {
        self.commit_rename();
    }

    fn delete_selected(&mut self) {
        self.delete_selected();
    }

    fn duplicate_selected(&mut self) {
        self.duplicate_selected();
    }

    fn new_folder(&mut self) {
        self.new_folder();
    }

    fn paste(&mut self) {
        self.paste();
    }

    fn refresh(&mut self) {
        self.refresh();
    }

    fn select_all(&mut self) {
        self.select_all();
    }

    fn hover_splitter(&mut self) {
        self.hover_splitter();
    }

    fn resize_sidebar(&mut self, event: Event) {
        self.resize_sidebar(event.delta_x);
    }

    fn show_blank_menu(&mut self) {
        self.show_blank_menu();
    }

    fn close_preview(&mut self) {
        self.close_preview();
    }

    fn show_row1_menu(&mut self) {
        self.show_row_menu(0);
    }
    fn show_row2_menu(&mut self) {
        self.show_row_menu(1);
    }
    fn show_row3_menu(&mut self) {
        self.show_row_menu(2);
    }
    fn show_row4_menu(&mut self) {
        self.show_row_menu(3);
    }
    fn show_row5_menu(&mut self) {
        self.show_row_menu(4);
    }
    fn show_row6_menu(&mut self) {
        self.show_row_menu(5);
    }
    fn show_row7_menu(&mut self) {
        self.show_row_menu(6);
    }
    fn show_row8_menu(&mut self) {
        self.show_row_menu(7);
    }
    fn show_row9_menu(&mut self) {
        self.show_row_menu(8);
    }
    fn show_row10_menu(&mut self) {
        self.show_row_menu(9);
    }
    fn show_row11_menu(&mut self) {
        self.show_row_menu(10);
    }
    fn show_row12_menu(&mut self) {
        self.show_row_menu(11);
    }
    fn hover_row1(&mut self) {
        self.hover_row(0);
    }
    fn hover_row2(&mut self) {
        self.hover_row(1);
    }
    fn hover_row3(&mut self) {
        self.hover_row(2);
    }
    fn hover_row4(&mut self) {
        self.hover_row(3);
    }
    fn hover_row5(&mut self) {
        self.hover_row(4);
    }
    fn hover_row6(&mut self) {
        self.hover_row(5);
    }
    fn hover_row7(&mut self) {
        self.hover_row(6);
    }
    fn hover_row8(&mut self) {
        self.hover_row(7);
    }
    fn hover_row9(&mut self) {
        self.hover_row(8);
    }
    fn hover_row10(&mut self) {
        self.hover_row(9);
    }
    fn hover_row11(&mut self) {
        self.hover_row(10);
    }
    fn hover_row12(&mut self) {
        self.hover_row(11);
    }
    fn open_row1(&mut self) {
        self.open_row(0);
    }
    fn open_row2(&mut self) {
        self.open_row(1);
    }
    fn open_row3(&mut self) {
        self.open_row(2);
    }
    fn open_row4(&mut self) {
        self.open_row(3);
    }
    fn open_row5(&mut self) {
        self.open_row(4);
    }
    fn open_row6(&mut self) {
        self.open_row(5);
    }
    fn open_row7(&mut self) {
        self.open_row(6);
    }
    fn open_row8(&mut self) {
        self.open_row(7);
    }
    fn open_row9(&mut self) {
        self.open_row(8);
    }
    fn open_row10(&mut self) {
        self.open_row(9);
    }
    fn open_row11(&mut self) {
        self.open_row(10);
    }
    fn open_row12(&mut self) {
        self.open_row(11);
    }
}

impl FinderLiteDirect {
    fn generated_row_value(&self, index: usize, field: &str) -> Value {
        let visible_rows = self.visible_rows();
        let Some(row) = visible_rows.get(index) else {
            return Value::None;
        };
        match field {
            "name" => row.name.clone().into(),
            "kind" => row.kind.clone().into(),
            "modified" => row.modified.clone().into(),
            "size" => format_size(row).into(),
            "class" if self.rename_path.as_ref() == Some(&row.path) => "file-row-editing".into(),
            "class" if self.selected_paths.contains(&row.path) => "file-row-selected".into(),
            "class" if self.focus == Some(index) => "file-row-focus".into(),
            "class" if self.hover == Some(index) => "file-row-hover".into(),
            "class" => "file-row".into(),
            "selected" => Value::Bool(self.selected_paths.contains(&row.path)),
            _ => Value::None,
        }
    }
}

fn row_signal(id: SignalId) -> Option<(usize, &'static str)> {
    if id.0 < 20 {
        return None;
    }
    let offset = id.0 - 20;
    let index = (offset / 6) as usize;
    if index >= 12 {
        return None;
    }
    let field = match offset % 6 {
        0 => "name",
        1 => "kind",
        2 => "modified",
        3 => "size",
        4 => "class",
        5 => "selected",
        _ => return None,
    };
    Some((index, field))
}

fn format_size(row: &FinderRow) -> String {
    if row.kind == "folder" {
        return "--".to_string();
    }
    if row.size >= 1_000_000 {
        format!("{:.1} MB", row.size as f64 / 1_000_000.0)
    } else if row.size >= 1_000 {
        format!("{:.0} KB", row.size as f64 / 1_000.0)
    } else {
        format!("{} B", row.size)
    }
}

fn format_modified(value: SystemTime) -> String {
    format_modified_at(DateTime::<Local>::from(value), Local::now())
}

fn format_modified_at(value: DateTime<Local>, now: DateTime<Local>) -> String {
    if value.date_naive() == now.date_naive() {
        return format!("Today {}", value.format("%H:%M"));
    }
    if value.date_naive() == (now - chrono::Duration::days(1)).date_naive() {
        return "Yesterday".to_string();
    }
    if value.year() == now.year() {
        return value.format("%b %d").to_string();
    }
    value.format("%Y-%m-%d").to_string()
}

fn filter_rows(rows: &[FinderRow], query: &str) -> Vec<FinderRow> {
    let query = query.trim();
    if query.is_empty() {
        return rows.to_vec();
    }
    let query = query.to_lowercase();
    rows.iter()
        .filter(|row| row.name.to_lowercase().contains(&query))
        .cloned()
        .collect()
}

#[derive(Clone, Debug, Default)]
struct FixtureFileSystem {
    folders: HashMap<String, Vec<FixtureEntry>>,
}

#[derive(Clone, Debug)]
struct FixtureEntry {
    name: String,
    kind: String,
    size: u64,
}

impl FixtureFileSystem {
    fn finder_lite() -> Self {
        let mut fs = Self::default();
        fs.add_folder(
            "Documents",
            [
                fixture_folder("Design"),
                fixture_file("Roadmap.md", 12_400),
                fixture_file("Budget 2026.xlsx", 842_000),
                fixture_file("Meeting Notes.txt", 17_000),
                fixture_file("Client Brief.pdf", 224_000),
                fixture_file("Contract Draft.docx", 96_000),
                fixture_file("Launch Plan.pages", 410_000),
                fixture_file("Research Summary.md", 38_000),
                fixture_file("Book Outline.txt", 21_000),
                fixture_file("Ideas.txt", 7_000),
                fixture_file("Agenda.md", 8_000),
                fixture_file("Notes Archive.txt", 53_000),
            ],
        );
        fs.add_folder(
            "Downloads",
            [
                fixture_file("Installer.dmg", 3_400_000),
                fixture_folder("Receipts"),
                fixture_file("Archive.zip", 721_000),
            ],
        );
        fs.add_folder(
            "Pictures",
            [
                fixture_file("Vacation.jpg", 2_100_000),
                fixture_folder("Screenshots"),
                fixture_file("Profile.png", 98_000),
            ],
        );
        fs.add_folder(
            "Documents/Design",
            [
                fixture_file("Components.sketch", 1_900_000),
                fixture_folder("Assets"),
                fixture_file("Prototype.mov", 4_800_000),
            ],
        );
        fs.add_folder(
            "Downloads/Receipts",
            [
                fixture_file("May.pdf", 210_000),
                fixture_file("April.pdf", 198_000),
                fixture_folder("Archive"),
            ],
        );
        fs.add_folder(
            "Pictures/Screenshots",
            [
                fixture_file("Desktop.png", 1_100_000),
                fixture_file("Window.png", 860_000),
                fixture_folder("Exports"),
            ],
        );
        fs.add_folder(
            "Documents/Design/Assets",
            [
                fixture_file("Icon.png", 56_000),
                fixture_file("Toolbar.png", 74_000),
            ],
        );
        fs.add_folder(
            "Downloads/Receipts/Archive",
            [
                fixture_file("2025.pdf", 184_000),
                fixture_file("2024.pdf", 193_000),
            ],
        );
        fs.add_folder(
            "Pictures/Screenshots/Exports",
            [
                fixture_file("Header.png", 64_000),
                fixture_file("Sidebar.png", 67_000),
            ],
        );
        fs
    }

    fn add_folder<const N: usize>(&mut self, path: &str, entries: [FixtureEntry; N]) {
        self.folders
            .insert(clean_path_str(path), entries.into_iter().collect());
    }
}

impl FileSystem for FixtureFileSystem {
    fn read_dir(&self, path: &str) -> Result<Vec<SystemEntry>, String> {
        let path = clean_path_str(path);
        let entries = self
            .folders
            .get(&path)
            .ok_or_else(|| format!("{path}: folder not found"))?;
        let mut out: Vec<SystemEntry> = entries
            .iter()
            .map(|entry| {
                let child_path = join_clean(&path, &entry.name);
                SystemEntry {
                    name: entry.name.clone(),
                    path: child_path,
                    kind: entry.kind.clone(),
                    size: entry.size,
                    modified_at: None,
                }
            })
            .collect();
        out.sort_by(|a, b| match (a.kind.as_str(), b.kind.as_str()) {
            ("folder", "file") => std::cmp::Ordering::Less,
            ("file", "folder") => std::cmp::Ordering::Greater,
            _ => a.name.to_lowercase().cmp(&b.name.to_lowercase()),
        });
        Ok(out)
    }

    fn stat(&self, path: &str) -> Result<SystemEntry, FsError> {
        let path = clean_path_str(path);
        if self.folders.contains_key(&path) {
            return Ok(SystemEntry {
                name: path
                    .rsplit('/')
                    .next()
                    .filter(|name| !name.is_empty())
                    .unwrap_or(&path)
                    .to_string(),
                path,
                kind: "folder".to_string(),
                size: 0,
                modified_at: None,
            });
        }
        let Some((parent, name)) = split_parent_name(&path) else {
            return Err(FsError::NotExist(format!("{path}: not found")));
        };
        let Some(entry) = self
            .folders
            .get(&parent)
            .and_then(|entries| entries.iter().find(|entry| entry.name == name))
        else {
            return Err(FsError::NotExist(format!("{path}: not found")));
        };
        Ok(SystemEntry {
            name: entry.name.clone(),
            path,
            kind: entry.kind.clone(),
            size: entry.size,
            modified_at: None,
        })
    }

    fn mkdir(&mut self, path: &str) -> Result<(), String> {
        let path = clean_path_str(path);
        if self.folders.contains_key(&path) {
            return Err(format!("{path}: folder already exists"));
        }
        let Some((parent, name)) = split_parent_name(&path) else {
            return Err(format!("{path}: invalid folder path"));
        };
        self.folders
            .get_mut(&parent)
            .ok_or_else(|| format!("{parent}: folder not found"))?
            .push(fixture_folder(&name));
        self.folders.insert(path, Vec::new());
        Ok(())
    }

    fn rename(&mut self, old_path: &str, new_path: &str) -> Result<(), String> {
        let old_path = clean_path_str(old_path);
        let new_path = clean_path_str(new_path);
        if self.stat(&new_path).is_ok() {
            return Err(format!("{new_path}: already exists"));
        }
        let Some((parent, old_name)) = split_parent_name(&old_path) else {
            return Err(format!("{old_path}: cannot rename root"));
        };
        let Some((new_parent, new_name)) = split_parent_name(&new_path) else {
            return Err(format!("{new_path}: invalid target"));
        };
        if parent != new_parent {
            return Err("cross-folder rename is not supported by fixture fs".to_string());
        }
        let entries = self
            .folders
            .get_mut(&parent)
            .ok_or_else(|| format!("{parent}: folder not found"))?;
        let entry = entries
            .iter_mut()
            .find(|entry| entry.name == old_name)
            .ok_or_else(|| format!("{old_path}: entry not found"))?;
        entry.name = new_name;
        let nested_prefix = format!("{old_path}/");
        let nested: Vec<(String, Vec<FixtureEntry>)> = self
            .folders
            .iter()
            .filter_map(|(path, children)| {
                (path == &old_path || path.starts_with(&nested_prefix))
                    .then(|| (path.clone(), children.clone()))
            })
            .collect();
        for (path, _) in &nested {
            self.folders.remove(path);
        }
        for (path, children) in nested {
            let target_path = if path == old_path {
                new_path.clone()
            } else {
                format!("{new_path}{}", path.trim_start_matches(&old_path))
            };
            self.folders.insert(target_path, children);
        }
        Ok(())
    }

    fn remove(&mut self, path: &str) -> Result<(), String> {
        let path = clean_path_str(path);
        let Some((parent, name)) = split_parent_name(&path) else {
            return Err(format!("{path}: cannot remove root"));
        };
        let entries = self
            .folders
            .get_mut(&parent)
            .ok_or_else(|| format!("{parent}: folder not found"))?;
        let before = entries.len();
        entries.retain(|entry| entry.name != name);
        if entries.len() == before {
            return Err(format!("{path}: entry not found"));
        }
        let prefix = format!("{path}/");
        self.folders
            .retain(|folder, _| folder != &path && !folder.starts_with(&prefix));
        Ok(())
    }

    fn duplicate(&mut self, source: &str, target: &str) -> Result<(), String> {
        let source = clean_path_str(source);
        let target = clean_path_str(target);
        if self.stat(&target).is_ok() {
            return Err(format!("{target}: already exists"));
        }
        let Some((source_parent, source_name)) = split_parent_name(&source) else {
            return Err(format!("{source}: invalid source"));
        };
        let Some((target_parent, target_name)) = split_parent_name(&target) else {
            return Err(format!("{target}: invalid target"));
        };
        let source_entry = self
            .folders
            .get(&source_parent)
            .and_then(|entries| entries.iter().find(|entry| entry.name == source_name))
            .cloned()
            .ok_or_else(|| format!("{source}: entry not found"))?;
        let mut target_entry = source_entry;
        target_entry.name = target_name;
        self.folders
            .get_mut(&target_parent)
            .ok_or_else(|| format!("{target_parent}: folder not found"))?
            .push(target_entry);
        let nested_prefix = format!("{source}/");
        let nested: Vec<(String, Vec<FixtureEntry>)> = self
            .folders
            .iter()
            .filter_map(|(path, children)| {
                (path == &source || path.starts_with(&nested_prefix))
                    .then(|| (path.clone(), children.clone()))
            })
            .collect();
        for (path, children) in nested {
            let target_path = if path == source {
                target.clone()
            } else {
                format!("{target}{}", path.trim_start_matches(&source))
            };
            self.folders.insert(target_path, children);
        }
        Ok(())
    }
}

fn fixture_folder(name: &str) -> FixtureEntry {
    FixtureEntry {
        name: name.to_string(),
        kind: "folder".to_string(),
        size: 0,
    }
}

fn fixture_file(name: &str, size: u64) -> FixtureEntry {
    FixtureEntry {
        name: name.to_string(),
        kind: "file".to_string(),
        size,
    }
}

fn first_existing_finder_path(files: &dyn FileSystem, sidebar: &FinderSidebar) -> String {
    for path in [
        sidebar.documents.as_str(),
        sidebar.downloads.as_str(),
        sidebar.pictures.as_str(),
        sidebar.project_a.as_str(),
        sidebar.project_b.as_str(),
        ".",
    ] {
        if path.trim().is_empty() {
            continue;
        }
        if let Ok(entry) = files.stat(path) {
            if entry.kind == "folder" {
                return clean_path_str(path);
            }
        }
    }
    ".".to_string()
}

fn path_within(path: &str, base: &str) -> bool {
    let path = clean_path_str(path);
    let base = clean_path_str(base);
    path == base || path.strip_prefix(&(base + "/")).is_some()
}

fn rows_for_path(path: &str) -> (Vec<FinderRow>, Option<FinderSidebarLocation>) {
    rows_for_path_with_overrides(path, "")
}

fn rows_for_path_with_overrides(
    path: &str,
    directory_overrides: &str,
) -> (Vec<FinderRow>, Option<FinderSidebarLocation>) {
    let fs = FixtureFileSystem::finder_lite();
    let sidebar = FinderSidebar::default();
    if let Some(rows) = abi_directory_override_rows(directory_overrides, path) {
        return (rows, sidebar.location_for_path(path));
    }
    let rows = fs
        .read_dir(path)
        .unwrap_or_default()
        .into_iter()
        .map(finder_row_from_system_entry)
        .collect();
    (rows, sidebar.location_for_path(path))
}

fn abi_directory_override_rows(directory_overrides: &str, path: &str) -> Option<Vec<FinderRow>> {
    let prefix = format!("dir\t{path}\t");
    directory_overrides.lines().find_map(|line| {
        line.strip_prefix(&prefix)
            .map(|payload| abi_decode_directory_rows(path, payload))
    })
}

fn abi_decode_directory_rows(path: &str, payload: &str) -> Vec<FinderRow> {
    payload
        .split('|')
        .filter(|entry| !entry.is_empty())
        .filter_map(|entry| {
            let mut parts = entry.splitn(4, ',');
            let name = abi_unescape(parts.next()?)?;
            let kind = abi_unescape(parts.next()?)?;
            let modified = abi_unescape(parts.next()?)?;
            let size_text = abi_unescape(parts.next()?)?;
            Some(FinderRow {
                id: join_clean(path, &name),
                name: name.clone(),
                kind,
                size: 0,
                modified,
                path: join_clean(path, &name),
            })
            .map(|mut row| {
                if row.kind != "folder" {
                    row.size = parse_size_text(&size_text);
                }
                row
            })
        })
        .collect()
}

fn abi_encode_directory_rows(path: &str, rows: &[FinderRow]) -> String {
    let payload = rows
        .iter()
        .map(|row| {
            [
                abi_escape(&row.name),
                abi_escape(&row.kind),
                abi_escape(&row.modified),
                abi_escape(&format_size(row)),
            ]
            .join(",")
        })
        .collect::<Vec<_>>()
        .join("|");
    format!("dir\t{path}\t{payload}")
}

fn abi_escape(value: &str) -> String {
    value
        .replace('%', "%25")
        .replace('\t', "%09")
        .replace('\n', "%0A")
        .replace('|', "%7C")
        .replace(',', "%2C")
}

fn abi_unescape(value: &str) -> Option<String> {
    let mut out = String::new();
    let mut chars = value.chars();
    while let Some(ch) = chars.next() {
        if ch != '%' {
            out.push(ch);
            continue;
        }
        let hi = chars.next()?;
        let lo = chars.next()?;
        let hex = [hi, lo].iter().collect::<String>();
        let byte = u8::from_str_radix(&hex, 16).ok()?;
        out.push(byte as char);
    }
    Some(out)
}

fn parse_size_text(size: &str) -> u64 {
    let Some((number, unit)) = size.split_once(' ') else {
        return 0;
    };
    let value = number.parse::<f64>().unwrap_or(0.0);
    match unit {
        "MB" => (value * 1_000_000.0) as u64,
        "KB" => (value * 1_000.0) as u64,
        "B" => value as u64,
        _ => 0,
    }
}

fn set_abi_sidebar_active(
    values: &mut HashMap<SignalId, Value>,
    location: Option<FinderSidebarLocation>,
) {
    values.insert(
        SignalId(16),
        Value::Bool(location == Some(FinderSidebarLocation::Documents)),
    );
    values.insert(
        SignalId(17),
        Value::Bool(location == Some(FinderSidebarLocation::Downloads)),
    );
    values.insert(
        SignalId(18),
        Value::Bool(location == Some(FinderSidebarLocation::Pictures)),
    );
    values.insert(
        SignalId(98),
        Value::Bool(location == Some(FinderSidebarLocation::ProjectA)),
    );
    values.insert(
        SignalId(99),
        Value::Bool(location == Some(FinderSidebarLocation::ProjectB)),
    );
}

fn finder_lite_abi() -> AbiState {
    let mut state = AbiState::new()
        .with_signal(SignalId(1), "Documents")
        .with_signal(SignalId(2), "12 items · Current path: Documents")
        .with_signal(SignalId(3), "0 items selected")
        .with_signal(SignalId(13), "Documents")
        .with_signal(SignalId(14), "Downloads")
        .with_signal(SignalId(15), "Pictures")
        .with_signal(SignalId(16), true)
        .with_signal(SignalId(17), false)
        .with_signal(SignalId(18), false)
        .with_signal(SignalId(19), "")
        .with_signal(SignalId(92), "Favorites")
        .with_signal(SignalId(93), "Workspace")
        .with_signal(SignalId(94), true)
        .with_signal(SignalId(95), true)
        .with_signal(SignalId(96), "Current Project")
        .with_signal(SignalId(97), "Parent Folder")
        .with_signal(SignalId(98), false)
        .with_signal(SignalId(99), false)
        .with_signal(SignalId(100), false)
        .with_signal(SignalId(101), false)
        .with_signal(SignalId(102), "")
        .with_signal(SignalId(103), false)
        .with_signal(SignalId(104), "")
        .with_signal(SignalId(105), "")
        .with_signal(SignalId(106), "sidebar")
        .with_signal(SignalId(107), "splitter")
        .with_signal(ABI_HISTORY_SIGNAL, "")
        .with_signal(ABI_FORWARD_SIGNAL, "")
        .with_signal(ABI_DIRECTORY_OVERRIDES_SIGNAL, "")
        .with_event_method(
            MethodId(2),
            |values: &mut HashMap<SignalId, Value>, event| {
                select_abi_index_with_modifiers(values, 0, event.modifiers);
            },
        )
        .with_event_method(
            MethodId(3),
            |values: &mut HashMap<SignalId, Value>, event| {
                select_abi_index_with_modifiers(values, 1, event.modifiers);
            },
        )
        .with_event_method(
            MethodId(4),
            |values: &mut HashMap<SignalId, Value>, event| {
                select_abi_index_with_modifiers(values, 2, event.modifiers);
            },
        )
        .with_method(MethodId(2), |values: &mut HashMap<SignalId, Value>| {
            select_abi_index(values, 0);
        })
        .with_method(MethodId(3), |values: &mut HashMap<SignalId, Value>| {
            select_abi_index(values, 1);
        })
        .with_method(MethodId(4), |values: &mut HashMap<SignalId, Value>| {
            select_abi_index(values, 2);
        })
        .with_method(MethodId(1), |values: &mut HashMap<SignalId, Value>| {
            back_abi(values);
        })
        .with_method(MethodId(5), |values: &mut HashMap<SignalId, Value>| {
            navigate_abi(values, "Documents", true);
        })
        .with_method(MethodId(6), |values: &mut HashMap<SignalId, Value>| {
            navigate_abi(values, "Downloads", true);
        })
        .with_method(MethodId(7), |values: &mut HashMap<SignalId, Value>| {
            navigate_abi(values, "Pictures", true);
        })
        .with_method(MethodId(8), |values: &mut HashMap<SignalId, Value>| {
            select_abi_delta(values, -1);
        })
        .with_method(MethodId(9), |values: &mut HashMap<SignalId, Value>| {
            select_abi_delta(values, 1);
        })
        .with_method(MethodId(11), |values: &mut HashMap<SignalId, Value>| {
            let mut query = values
                .get(&SignalId(19))
                .map(Value::as_text)
                .unwrap_or_default();
            query.pop();
            values.insert(SignalId(19), query.into());
            project_abi_rows(values);
            project_abi_selection_from_sources(values);
        })
        .with_method(MethodId(12), |values: &mut HashMap<SignalId, Value>| {
            values.insert(SignalId(19), String::new().into());
            project_abi_rows(values);
            project_abi_selection_from_sources(values);
        })
        .with_method(MethodId(13), |values: &mut HashMap<SignalId, Value>| {
            open_abi_selected(values);
        })
        .with_method(MethodId(14), |values: &mut HashMap<SignalId, Value>| {
            open_abi_parent(values);
        })
        .with_method(MethodId(15), |values: &mut HashMap<SignalId, Value>| {
            let next = !matches!(values.get(&SignalId(94)), Some(Value::Bool(true)));
            values.insert(SignalId(94), Value::Bool(next));
        })
        .with_method(MethodId(16), |values: &mut HashMap<SignalId, Value>| {
            let next = !matches!(values.get(&SignalId(95)), Some(Value::Bool(true)));
            values.insert(SignalId(95), Value::Bool(next));
        })
        .with_method(MethodId(17), |values: &mut HashMap<SignalId, Value>| {
            navigate_abi(values, "Project A", true);
        })
        .with_method(MethodId(18), |values: &mut HashMap<SignalId, Value>| {
            navigate_abi(values, "Project B", true);
        })
        .with_method(MethodId(19), dismiss_abi_overlay)
        .with_method(MethodId(20), |values: &mut HashMap<SignalId, Value>| {
            forward_abi(values);
        })
        .with_method(MethodId(33), begin_abi_rename)
        .with_method(MethodId(34), cancel_abi_rename)
        .with_method(MethodId(35), commit_abi_rename)
        .with_method(MethodId(36), delete_abi_selected)
        .with_method(MethodId(37), duplicate_abi_selected)
        .with_method(MethodId(38), new_abi_folder)
        .with_method(MethodId(39), show_abi_blank_menu)
        .with_method(MethodId(40), close_abi_preview)
        .with_method(MethodId(77), clear_abi_selection)
        .with_method(MethodId(78), dismiss_abi_overlay)
        .with_method(MethodId(79), refresh_abi)
        .with_method(MethodId(80), select_abi_all)
        .with_method(MethodId(81), hover_abi_splitter)
        .with_event_method(MethodId(10), |values, event| {
            append_abi_search_text(values, &event.text);
        })
        .with_event_method(MethodId(77), |values, _event| {
            clear_abi_selection(values);
        })
        .with_event_method(MethodId(82), |values, event| {
            resize_abi_sidebar(values, event.delta_x);
        });
    state = state
        .with_event_method(MethodId(24), |values, event| {
            select_abi_index_with_modifiers(values, 3, event.modifiers)
        })
        .with_event_method(MethodId(25), |values, event| {
            select_abi_index_with_modifiers(values, 4, event.modifiers)
        })
        .with_event_method(MethodId(26), |values, event| {
            select_abi_index_with_modifiers(values, 5, event.modifiers)
        })
        .with_event_method(MethodId(27), |values, event| {
            select_abi_index_with_modifiers(values, 6, event.modifiers)
        })
        .with_event_method(MethodId(28), |values, event| {
            select_abi_index_with_modifiers(values, 7, event.modifiers)
        })
        .with_event_method(MethodId(29), |values, event| {
            select_abi_index_with_modifiers(values, 8, event.modifiers)
        })
        .with_event_method(MethodId(30), |values, event| {
            select_abi_index_with_modifiers(values, 9, event.modifiers)
        })
        .with_event_method(MethodId(31), |values, event| {
            select_abi_index_with_modifiers(values, 10, event.modifiers)
        })
        .with_event_method(MethodId(32), |values, event| {
            select_abi_index_with_modifiers(values, 11, event.modifiers)
        })
        .with_event_method(MethodId(41), |values, event| {
            show_abi_row_menu_with_modifiers(values, 0, event.modifiers)
        })
        .with_event_method(MethodId(42), |values, event| {
            show_abi_row_menu_with_modifiers(values, 1, event.modifiers)
        })
        .with_event_method(MethodId(43), |values, event| {
            show_abi_row_menu_with_modifiers(values, 2, event.modifiers)
        })
        .with_event_method(MethodId(44), |values, event| {
            show_abi_row_menu_with_modifiers(values, 3, event.modifiers)
        })
        .with_event_method(MethodId(45), |values, event| {
            show_abi_row_menu_with_modifiers(values, 4, event.modifiers)
        })
        .with_event_method(MethodId(46), |values, event| {
            show_abi_row_menu_with_modifiers(values, 5, event.modifiers)
        })
        .with_event_method(MethodId(47), |values, event| {
            show_abi_row_menu_with_modifiers(values, 6, event.modifiers)
        })
        .with_event_method(MethodId(48), |values, event| {
            show_abi_row_menu_with_modifiers(values, 7, event.modifiers)
        })
        .with_event_method(MethodId(49), |values, event| {
            show_abi_row_menu_with_modifiers(values, 8, event.modifiers)
        })
        .with_event_method(MethodId(50), |values, event| {
            show_abi_row_menu_with_modifiers(values, 9, event.modifiers)
        })
        .with_event_method(MethodId(51), |values, event| {
            show_abi_row_menu_with_modifiers(values, 10, event.modifiers)
        })
        .with_event_method(MethodId(52), |values, event| {
            show_abi_row_menu_with_modifiers(values, 11, event.modifiers)
        })
        .with_method(MethodId(24), |values| select_abi_index(values, 3))
        .with_method(MethodId(25), |values| select_abi_index(values, 4))
        .with_method(MethodId(26), |values| select_abi_index(values, 5))
        .with_method(MethodId(27), |values| select_abi_index(values, 6))
        .with_method(MethodId(28), |values| select_abi_index(values, 7))
        .with_method(MethodId(29), |values| select_abi_index(values, 8))
        .with_method(MethodId(30), |values| select_abi_index(values, 9))
        .with_method(MethodId(31), |values| select_abi_index(values, 10))
        .with_method(MethodId(32), |values| select_abi_index(values, 11))
        .with_method(MethodId(41), |values| show_abi_row_menu(values, 0))
        .with_method(MethodId(42), |values| show_abi_row_menu(values, 1))
        .with_method(MethodId(43), |values| show_abi_row_menu(values, 2))
        .with_method(MethodId(44), |values| show_abi_row_menu(values, 3))
        .with_method(MethodId(45), |values| show_abi_row_menu(values, 4))
        .with_method(MethodId(46), |values| show_abi_row_menu(values, 5))
        .with_method(MethodId(47), |values| show_abi_row_menu(values, 6))
        .with_method(MethodId(48), |values| show_abi_row_menu(values, 7))
        .with_method(MethodId(49), |values| show_abi_row_menu(values, 8))
        .with_method(MethodId(50), |values| show_abi_row_menu(values, 9))
        .with_method(MethodId(51), |values| show_abi_row_menu(values, 10))
        .with_method(MethodId(52), |values| show_abi_row_menu(values, 11))
        .with_method(MethodId(53), |values| hover_abi_row(values, 0))
        .with_method(MethodId(54), |values| hover_abi_row(values, 1))
        .with_method(MethodId(55), |values| hover_abi_row(values, 2))
        .with_method(MethodId(56), |values| hover_abi_row(values, 3))
        .with_method(MethodId(57), |values| hover_abi_row(values, 4))
        .with_method(MethodId(58), |values| hover_abi_row(values, 5))
        .with_method(MethodId(59), |values| hover_abi_row(values, 6))
        .with_method(MethodId(60), |values| hover_abi_row(values, 7))
        .with_method(MethodId(61), |values| hover_abi_row(values, 8))
        .with_method(MethodId(62), |values| hover_abi_row(values, 9))
        .with_method(MethodId(63), |values| hover_abi_row(values, 10))
        .with_method(MethodId(64), |values| hover_abi_row(values, 11))
        .with_method(MethodId(82), |values| resize_abi_sidebar(values, 0.0))
        .with_method(MethodId(65), |values| open_abi_row(values, 0))
        .with_method(MethodId(66), |values| open_abi_row(values, 1))
        .with_method(MethodId(67), |values| open_abi_row(values, 2))
        .with_method(MethodId(68), |values| open_abi_row(values, 3))
        .with_method(MethodId(69), |values| open_abi_row(values, 4))
        .with_method(MethodId(70), |values| open_abi_row(values, 5))
        .with_method(MethodId(71), |values| open_abi_row(values, 6))
        .with_method(MethodId(72), |values| open_abi_row(values, 7))
        .with_method(MethodId(73), |values| open_abi_row(values, 8))
        .with_method(MethodId(74), |values| open_abi_row(values, 9))
        .with_method(MethodId(75), |values| open_abi_row(values, 10))
        .with_method(MethodId(76), |values| open_abi_row(values, 11));
    let rows = rows_for_path("Documents").0;
    for index in 0..24 {
        let source = abi_source_base(index);
        let base = abi_row_base(index);
        if let Some(row) = rows.get(index) {
            state = state
                .with_signal(SignalId(source), row.name.clone())
                .with_signal(SignalId(source + 1), row.kind.clone())
                .with_signal(SignalId(source + 2), row.modified.clone())
                .with_signal(SignalId(source + 3), format_size(row))
                .with_signal(SignalId(base), row.name.clone())
                .with_signal(SignalId(base + 1), row.kind.clone())
                .with_signal(SignalId(base + 2), row.modified.clone())
                .with_signal(SignalId(base + 3), format_size(row))
                .with_signal(SignalId(base + 4), "file-row")
                .with_signal(SignalId(base + 5), false)
                .with_signal(
                    SignalId(ABI_VISIBLE_SOURCE_BASE + index as u32),
                    Value::Number(index as f64),
                );
        }
    }
    state
}

fn select_abi_delta(values: &mut HashMap<SignalId, Value>, delta: isize) {
    let Some(current) =
        abi_index_signal(values, ABI_FOCUS_SIGNAL).or_else(|| selected_abi_index(values))
    else {
        select_abi_index(values, 0);
        return;
    };
    let visible_len = abi_visible_len(values);
    let next = (current as isize + delta).clamp(0, visible_len.saturating_sub(1) as isize) as usize;
    select_abi_index(values, next);
}

fn select_abi_index(values: &mut HashMap<SignalId, Value>, selected: usize) {
    select_abi_index_with_modifiers(values, selected, vugra_core::Modifiers::default());
}

fn select_abi_all(values: &mut HashMap<SignalId, Value>) {
    clear_abi_source_selection(values);
    let selected_sources = abi_filtered_sources(values);
    for source in &selected_sources {
        set_abi_source_selected(values, *source, true);
    }
    if selected_sources.is_empty() {
        set_abi_index_signal(values, ABI_FOCUS_SIGNAL, None);
        set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, None);
    } else {
        set_abi_index_signal(values, ABI_FOCUS_SIGNAL, Some(0));
        set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(0));
    }
    project_abi_selection_from_sources(values);
}

fn select_abi_index_with_modifiers(
    values: &mut HashMap<SignalId, Value>,
    selected: usize,
    modifiers: vugra_core::Modifiers,
) {
    let visible_len = abi_visible_len(values);
    if visible_len == 0 {
        clear_abi_selection(values);
        return;
    }
    let selected = selected.min(visible_len.min(12).saturating_sub(1));
    if let Some(source) = abi_visible_source(values, selected) {
        if modifiers.shift {
            if let Some(anchor) = abi_index_signal(values, ABI_ANCHOR_SIGNAL) {
                let (start, end) = if anchor <= selected {
                    (anchor, selected)
                } else {
                    (selected, anchor)
                };
                clear_abi_source_selection(values);
                for index in start..=end.min(visible_len.min(12).saturating_sub(1)) {
                    if let Some(source) = abi_visible_source(values, index) {
                        set_abi_source_selected(values, source, true);
                    }
                }
            } else {
                clear_abi_source_selection(values);
                set_abi_source_selected(values, source, true);
                set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(selected));
            }
        } else if modifiers.meta || modifiers.ctrl {
            let selected_now = abi_source_selected(values, source);
            set_abi_source_selected(values, source, !selected_now);
            set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(selected));
        } else {
            clear_abi_source_selection(values);
            set_abi_source_selected(values, source, true);
            set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(selected));
        }
        if abi_source_selected(values, source) {
            set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, Some(source));
        } else {
            set_abi_index_signal(
                values,
                ABI_SELECTED_SOURCE_SIGNAL,
                first_selected_abi_source(values),
            );
        }
    }
    set_abi_index_signal(values, ABI_FOCUS_SIGNAL, Some(selected));
    dismiss_abi_overlay(values);
    project_abi_selection_from_sources(values);
}

fn set_abi_selected_index(
    values: &mut HashMap<SignalId, Value>,
    selected: usize,
    update_focus: bool,
) {
    let visible_len = abi_visible_len(values);
    if visible_len == 0 {
        clear_abi_selection(values);
        return;
    }
    let selected = selected.min(visible_len.min(12).saturating_sub(1));
    clear_abi_source_selection(values);
    if let Some(source) = abi_visible_source(values, selected) {
        set_abi_source_selected(values, source, true);
        set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, Some(source));
    }
    if update_focus {
        set_abi_index_signal(
            values,
            ABI_FOCUS_SIGNAL,
            (visible_len > 0).then_some(selected),
        );
    }
    set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(selected));
    project_abi_selection_from_sources(values);
}

fn clear_abi_selection(values: &mut HashMap<SignalId, Value>) {
    clear_abi_source_selection(values);
    set_abi_index_signal(values, ABI_FOCUS_SIGNAL, None);
    set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, None);
    set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, None);
    values.insert(SignalId(102), String::new().into());
    values.insert(SignalId(100), Value::Bool(false));
    values.insert(SignalId(101), Value::Bool(false));
    project_abi_selection_from_sources(values);
}

fn clear_abi_visible_selection(values: &mut HashMap<SignalId, Value>) {
    for index in 0..12 {
        let base = abi_row_base(index);
        if index < abi_visible_len(values).min(12) {
            values.insert(SignalId(base + 5), Value::Bool(false));
        } else {
            values.insert(SignalId(base + 5), Value::None);
        }
    }
    project_abi_visual_state(values);
}

fn hover_abi_splitter(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(107), "splitter-hover".into());
}

fn resize_abi_sidebar(values: &mut HashMap<SignalId, Value>, delta_x: f32) {
    let mut mode = abi_index_signal(values, ABI_SIDEBAR_MODE_SIGNAL).unwrap_or(0);
    if delta_x < -8.0 && mode > 1 {
        mode -= 1;
    } else if delta_x < -8.0 && mode == 1 {
        mode = 0;
    } else if delta_x > 8.0 && mode < 3 {
        mode += 1;
    }
    set_abi_index_signal(values, ABI_SIDEBAR_MODE_SIGNAL, Some(mode));
    let class = ["sidebar", "sidebar-200", "sidebar-280", "sidebar-320"][mode];
    values.insert(SignalId(106), class.into());
    values.insert(SignalId(107), "splitter".into());
}

fn project_abi_selection_from_sources(values: &mut HashMap<SignalId, Value>) {
    let selected_count = selected_abi_source_count(values);
    values.insert(
        SignalId(3),
        format!("{selected_count} items selected").into(),
    );
    set_abi_index_signal(
        values,
        ABI_SELECTED_SOURCE_SIGNAL,
        first_selected_abi_source(values),
    );
    for index in 0..12 {
        let base = abi_row_base(index);
        if index < abi_visible_len(values).min(12) {
            values.insert(
                SignalId(base + 5),
                Value::Bool(
                    abi_visible_source(values, index)
                        .is_some_and(|source| abi_source_selected(values, source)),
                ),
            );
        } else {
            values.insert(SignalId(base + 5), Value::None);
        }
    }
    project_abi_visual_state(values);
}

fn abi_filtered_sources(values: &HashMap<SignalId, Value>) -> Vec<usize> {
    let query = values
        .get(&SignalId(19))
        .map(Value::as_text)
        .unwrap_or_default()
        .trim()
        .to_lowercase();
    let mut sources = Vec::new();
    for index in 0..24 {
        let name = values
            .get(&SignalId(abi_source_base(index)))
            .map(Value::as_text)
            .unwrap_or_default();
        if name.is_empty() {
            continue;
        }
        if query.is_empty() || name.to_lowercase().contains(&query) {
            sources.push(index);
        }
    }
    sources
}

fn selected_abi_index(values: &HashMap<SignalId, Value>) -> Option<usize> {
    (0..12).find(|index| {
        matches!(
            values.get(&SignalId(abi_row_base(*index) + 5)),
            Some(Value::Bool(true))
        )
    })
}

fn selected_abi_sources(values: &HashMap<SignalId, Value>) -> Vec<usize> {
    (0..24)
        .filter(|source| abi_source_selected(values, *source))
        .collect()
}

fn first_selected_abi_source(values: &HashMap<SignalId, Value>) -> Option<usize> {
    selected_abi_sources(values).into_iter().next()
}

fn selected_abi_source_count(values: &HashMap<SignalId, Value>) -> usize {
    selected_abi_sources(values).len()
}

fn abi_visible_len(values: &HashMap<SignalId, Value>) -> usize {
    (0..12)
        .filter(|index| {
            values
                .get(&SignalId(abi_row_base(*index)))
                .map(Value::as_text)
                .is_some_and(|value| !value.is_empty())
        })
        .count()
}

fn project_abi_visual_state(values: &mut HashMap<SignalId, Value>) {
    let visible_len = abi_visible_len(values).min(12);
    let focus = abi_index_signal(values, ABI_FOCUS_SIGNAL);
    let hover = abi_index_signal(values, ABI_HOVER_SIGNAL);
    let editing = if values
        .get(&SignalId(102))
        .map(Value::as_text)
        .unwrap_or_default()
        .is_empty()
    {
        None
    } else {
        abi_index_signal(values, ABI_RENAME_SOURCE_SIGNAL)
            .or_else(|| abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL))
            .and_then(|source| abi_visible_index_for_source(values, source))
            .or_else(|| selected_abi_index(values))
    };
    for index in 0..12 {
        let base = abi_row_base(index);
        if index >= visible_len {
            values.insert(SignalId(base + 4), Value::None);
            continue;
        }
        let selected = abi_visible_source(values, index)
            .is_some_and(|source| abi_source_selected(values, source));
        let focused = focus == Some(index);
        let class = if editing == Some(index) {
            "file-row-editing"
        } else if selected {
            "file-row-selected"
        } else if focused {
            "file-row-focus"
        } else if hover == Some(index) {
            "file-row-hover"
        } else {
            "file-row"
        };
        values.insert(SignalId(base + 4), class.into());
    }
}

fn abi_row_base(index: usize) -> u32 {
    20 + index as u32 * 6
}

fn abi_source_base(index: usize) -> u32 {
    200 + index as u32 * 4
}

const ABI_FOCUS_SIGNAL: SignalId = SignalId(190);
const ABI_HOVER_SIGNAL: SignalId = SignalId(191);
const ABI_SELECTED_SOURCE_SIGNAL: SignalId = SignalId(192);
const ABI_ANCHOR_SIGNAL: SignalId = SignalId(193);
const ABI_SIDEBAR_MODE_SIGNAL: SignalId = SignalId(194);
const ABI_RENAME_SOURCE_SIGNAL: SignalId = SignalId(195);
const ABI_HISTORY_SIGNAL: SignalId = SignalId(196);
const ABI_FORWARD_SIGNAL: SignalId = SignalId(197);
const ABI_DIRECTORY_OVERRIDES_SIGNAL: SignalId = SignalId(198);
const ABI_VISIBLE_SOURCE_BASE: u32 = 300;
const ABI_SELECTED_SOURCE_BASE: u32 = 400;

fn abi_selected_source_signal(source: usize) -> SignalId {
    SignalId(ABI_SELECTED_SOURCE_BASE + source as u32)
}

fn abi_source_selected(values: &HashMap<SignalId, Value>, source: usize) -> bool {
    matches!(
        values.get(&abi_selected_source_signal(source)),
        Some(Value::Bool(true))
    )
}

fn set_abi_source_selected(values: &mut HashMap<SignalId, Value>, source: usize, selected: bool) {
    values.insert(abi_selected_source_signal(source), Value::Bool(selected));
}

fn clear_abi_source_selection(values: &mut HashMap<SignalId, Value>) {
    for source in 0..24 {
        set_abi_source_selected(values, source, false);
    }
}

fn set_abi_index_signal(
    values: &mut HashMap<SignalId, Value>,
    signal: SignalId,
    index: Option<usize>,
) {
    values.insert(
        signal,
        index
            .map(|index| Value::Number(index as f64))
            .unwrap_or(Value::Number(-1.0)),
    );
}

fn abi_index_signal(values: &HashMap<SignalId, Value>, signal: SignalId) -> Option<usize> {
    match values.get(&signal) {
        Some(Value::Number(value)) if value.is_finite() && *value >= 0.0 => Some(*value as usize),
        _ => None,
    }
}

fn set_abi_visible_source(
    values: &mut HashMap<SignalId, Value>,
    visible: usize,
    source: Option<usize>,
) {
    set_abi_index_signal(
        values,
        SignalId(ABI_VISIBLE_SOURCE_BASE + visible as u32),
        source,
    );
}

fn abi_visible_source(values: &HashMap<SignalId, Value>, visible: usize) -> Option<usize> {
    abi_index_signal(values, SignalId(ABI_VISIBLE_SOURCE_BASE + visible as u32))
}

fn abi_visible_index_for_source(values: &HashMap<SignalId, Value>, source: usize) -> Option<usize> {
    (0..12).find(|visible| abi_visible_source(values, *visible) == Some(source))
}

fn navigate_abi(values: &mut HashMap<SignalId, Value>, path: &str, remember: bool) {
    let current = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    if remember && path != current {
        push_abi_path_stack(values, ABI_HISTORY_SIGNAL, &current);
        values.insert(ABI_FORWARD_SIGNAL, String::new().into());
    }
    let directory_overrides = values
        .get(&ABI_DIRECTORY_OVERRIDES_SIGNAL)
        .map(Value::as_text)
        .unwrap_or_default();
    let (rows, location) = rows_for_path_with_overrides(path, &directory_overrides);
    set_abi_location(values, path, rows);
    set_abi_sidebar_active(values, location);
}

fn back_abi(values: &mut HashMap<SignalId, Value>) {
    let Some(path) = pop_abi_path_stack(values, ABI_HISTORY_SIGNAL) else {
        return;
    };
    let current = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    push_abi_path_stack(values, ABI_FORWARD_SIGNAL, &current);
    navigate_abi(values, &path, false);
}

fn forward_abi(values: &mut HashMap<SignalId, Value>) {
    let Some(path) = pop_abi_path_stack(values, ABI_FORWARD_SIGNAL) else {
        return;
    };
    let current = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    push_abi_path_stack(values, ABI_HISTORY_SIGNAL, &current);
    navigate_abi(values, &path, false);
}

fn push_abi_path_stack(values: &mut HashMap<SignalId, Value>, signal: SignalId, path: &str) {
    if path.is_empty() {
        return;
    }
    let mut stack = values.get(&signal).map(Value::as_text).unwrap_or_default();
    if !stack.is_empty() {
        stack.push('\n');
    }
    stack.push_str(path);
    values.insert(signal, stack.into());
}

fn pop_abi_path_stack(values: &mut HashMap<SignalId, Value>, signal: SignalId) -> Option<String> {
    let mut stack = values.get(&signal).map(Value::as_text).unwrap_or_default();
    if stack.is_empty() {
        return None;
    }
    let path = match stack.rsplit_once('\n') {
        Some((rest, path)) => {
            let path = path.to_string();
            stack = rest.to_string();
            path
        }
        None => {
            let path = stack;
            stack = String::new();
            path
        }
    };
    values.insert(signal, stack.into());
    Some(path)
}

fn set_abi_location(values: &mut HashMap<SignalId, Value>, path: &str, rows: Vec<FinderRow>) {
    values.insert(SignalId(1), path.into());
    values.insert(
        SignalId(2),
        format!("{} items · Current path: {path}", rows.len()).into(),
    );
    values.insert(SignalId(3), "0 items selected".into());
    set_abi_index_signal(values, ABI_FOCUS_SIGNAL, None);
    set_abi_index_signal(values, ABI_HOVER_SIGNAL, None);
    set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, None);
    set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, None);
    clear_abi_source_selection(values);
    for index in 0..24 {
        let source = abi_source_base(index);
        if let Some(row) = rows.get(index) {
            values.insert(SignalId(source), row.name.clone().into());
            values.insert(SignalId(source + 1), row.kind.clone().into());
            values.insert(SignalId(source + 2), row.modified.clone().into());
            values.insert(SignalId(source + 3), format_size(row).into());
        } else {
            values.insert(SignalId(source), Value::None);
            values.insert(SignalId(source + 1), Value::None);
            values.insert(SignalId(source + 2), Value::None);
            values.insert(SignalId(source + 3), Value::None);
        }
    }
    project_abi_rows(values);
    clear_abi_selection(values);
}

fn append_abi_search_text(values: &mut HashMap<SignalId, Value>, text: &str) {
    let mut query = values
        .get(&SignalId(19))
        .map(Value::as_text)
        .unwrap_or_default();
    query.push_str(text);
    values.insert(SignalId(19), query.into());
    project_abi_rows(values);
}

fn project_abi_rows(values: &mut HashMap<SignalId, Value>) {
    let query = values
        .get(&SignalId(19))
        .map(Value::as_text)
        .unwrap_or_default()
        .trim()
        .to_lowercase();
    let mut rows = Vec::new();
    for index in 0..24 {
        let source = abi_source_base(index);
        let name = values
            .get(&SignalId(source))
            .map(Value::as_text)
            .unwrap_or_default();
        let kind = values
            .get(&SignalId(source + 1))
            .map(Value::as_text)
            .unwrap_or_default();
        let modified = values
            .get(&SignalId(source + 2))
            .map(Value::as_text)
            .unwrap_or_default();
        let size = values
            .get(&SignalId(source + 3))
            .map(Value::as_text)
            .unwrap_or_default();
        if name.is_empty() {
            continue;
        }
        if query.is_empty() || name.to_lowercase().contains(&query) {
            rows.push((index, name, kind, modified, size));
        }
    }
    rows.sort_by(|a, b| match (a.2.as_str(), b.2.as_str()) {
        ("folder", "file") => std::cmp::Ordering::Less,
        ("file", "folder") => std::cmp::Ordering::Greater,
        _ => a.1.to_lowercase().cmp(&b.1.to_lowercase()),
    });
    values.insert(
        SignalId(2),
        format!(
            "{} items · Current path: {}",
            rows.len(),
            values
                .get(&SignalId(1))
                .map(Value::as_text)
                .unwrap_or_default()
        )
        .into(),
    );
    for index in 0..12 {
        let base = abi_row_base(index);
        if let Some((source, name, kind, modified, size)) = rows.get(index) {
            set_abi_visible_source(values, index, Some(*source));
            values.insert(SignalId(base), name.clone().into());
            values.insert(SignalId(base + 1), kind.clone().into());
            values.insert(SignalId(base + 2), modified.clone().into());
            values.insert(SignalId(base + 3), size.clone().into());
        } else {
            set_abi_visible_source(values, index, None);
            values.insert(SignalId(base), Value::None);
            values.insert(SignalId(base + 1), Value::None);
            values.insert(SignalId(base + 2), Value::None);
            values.insert(SignalId(base + 3), Value::None);
        }
    }
    project_abi_selection_from_sources(values);
}

fn open_abi_selected(values: &mut HashMap<SignalId, Value>) {
    let selected_index = abi_index_signal(values, ABI_FOCUS_SIGNAL)
        .or_else(|| selected_abi_index(values))
        .unwrap_or(0);
    open_abi_visible_row(values, selected_index);
}

fn open_abi_row(values: &mut HashMap<SignalId, Value>, selected_index: usize) {
    if selected_index >= abi_visible_len(values).min(12) {
        values.insert(SignalId(3), "0 items selected".into());
        return;
    }
    set_abi_selected_index(values, selected_index, true);
    open_abi_visible_row(values, selected_index);
}

fn open_abi_visible_row(values: &mut HashMap<SignalId, Value>, selected_index: usize) {
    if selected_index >= abi_visible_len(values).min(12) {
        values.insert(SignalId(3), "0 items selected".into());
        return;
    }
    let base = abi_row_base(selected_index);
    let name = values
        .get(&SignalId(base))
        .map(Value::as_text)
        .unwrap_or_default();
    let kind = values
        .get(&SignalId(base + 1))
        .map(Value::as_text)
        .unwrap_or_default();
    if name.is_empty() {
        values.insert(SignalId(3), "0 items selected".into());
        return;
    }
    if kind == "folder" {
        let path = values
            .get(&SignalId(1))
            .map(Value::as_text)
            .unwrap_or_default();
        let next_path = format!("{path}/{name}");
        navigate_abi(values, &next_path, true);
    } else {
        values.insert(SignalId(100), Value::Bool(false));
        values.insert(SignalId(101), Value::Bool(false));
        values.insert(SignalId(103), Value::Bool(true));
        values.insert(SignalId(104), name.clone().into());
        let modified = values
            .get(&SignalId(base + 2))
            .map(Value::as_text)
            .unwrap_or_else(|| "--".to_string());
        let size = values
            .get(&SignalId(base + 3))
            .map(Value::as_text)
            .unwrap_or_else(|| "--".to_string());
        values.insert(
            SignalId(105),
            format!("System file · {modified} · {size}").into(),
        );
    }
}

fn open_abi_parent(values: &mut HashMap<SignalId, Value>) {
    let path = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    let Some((parent, _child)) = path.rsplit_once('/') else {
        return;
    };
    navigate_abi(values, parent, true);
}

fn dismiss_abi_overlay(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(100), Value::Bool(false));
    values.insert(SignalId(101), Value::Bool(false));
}

fn show_abi_blank_menu(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(100), Value::Bool(false));
    values.insert(SignalId(101), Value::Bool(true));
    values.insert(SignalId(103), Value::Bool(false));
}

fn refresh_abi(values: &mut HashMap<SignalId, Value>) {
    dismiss_abi_overlay(values);
    let path = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    let rows = (0..24)
        .filter(|index| {
            !values
                .get(&SignalId(abi_source_base(*index)))
                .map(Value::as_text)
                .unwrap_or_default()
                .is_empty()
        })
        .count();
    values.insert(
        SignalId(2),
        format!("{rows} items · Current path: {path}").into(),
    );
    project_abi_rows(values);
    project_abi_selection_from_sources(values);
}

fn show_abi_row_menu(values: &mut HashMap<SignalId, Value>, index: usize) {
    show_abi_row_menu_with_modifiers(values, index, vugra_core::Modifiers::default());
}

fn show_abi_row_menu_with_modifiers(
    values: &mut HashMap<SignalId, Value>,
    index: usize,
    modifiers: vugra_core::Modifiers,
) {
    if index >= abi_visible_len(values).min(12) {
        return;
    }
    if !abi_visible_source(values, index).is_some_and(|source| abi_source_selected(values, source))
    {
        select_abi_index_with_modifiers(values, index, modifiers);
    }
    values.insert(SignalId(100), Value::Bool(true));
    values.insert(SignalId(101), Value::Bool(false));
    values.insert(SignalId(103), Value::Bool(false));
}

fn hover_abi_row(values: &mut HashMap<SignalId, Value>, index: usize) {
    if index >= abi_visible_len(values).min(12) {
        return;
    }
    set_abi_index_signal(values, ABI_HOVER_SIGNAL, Some(index));
    project_abi_visual_state(values);
}

fn close_abi_preview(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(103), Value::Bool(false));
}

fn begin_abi_rename(values: &mut HashMap<SignalId, Value>) {
    let Some(source) = first_selected_abi_source(values).or_else(|| {
        selected_abi_index(values).and_then(|selected| abi_visible_source(values, selected))
    }) else {
        return;
    };
    let base = abi_source_base(source);
    let name = values
        .get(&SignalId(base))
        .map(Value::as_text)
        .unwrap_or_default();
    if name.is_empty() {
        return;
    }
    values.insert(SignalId(102), name.into());
    set_abi_index_signal(values, ABI_RENAME_SOURCE_SIGNAL, Some(source));
    dismiss_abi_overlay(values);
    project_abi_visual_state(values);
}

fn cancel_abi_rename(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(102), String::new().into());
    set_abi_index_signal(values, ABI_RENAME_SOURCE_SIGNAL, None);
    project_abi_visual_state(values);
}

fn commit_abi_rename(values: &mut HashMap<SignalId, Value>) {
    let Some(source_index) = abi_index_signal(values, ABI_RENAME_SOURCE_SIGNAL)
        .or_else(|| abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL))
        .or_else(|| {
            selected_abi_index(values).and_then(|selected| abi_visible_source(values, selected))
        })
    else {
        return;
    };
    let name = values
        .get(&SignalId(102))
        .map(Value::as_text)
        .unwrap_or_default()
        .trim()
        .to_string();
    if name.is_empty() {
        return;
    }
    let source = abi_source_base(source_index);
    let old_name = values
        .get(&SignalId(source))
        .map(Value::as_text)
        .unwrap_or_default();
    let kind = values
        .get(&SignalId(source + 1))
        .map(Value::as_text)
        .unwrap_or_default();
    if kind == "folder" && !old_name.is_empty() && old_name != name {
        let current_path = values
            .get(&SignalId(1))
            .map(Value::as_text)
            .unwrap_or_default();
        let old_path = join_clean(&current_path, &old_name);
        let new_path = join_clean(&current_path, &name);
        rename_abi_directory_overrides(values, &old_path, &new_path);
    }
    values.insert(SignalId(source), name.into());
    cancel_abi_rename(values);
    clear_abi_source_selection(values);
    set_abi_source_selected(values, source_index, true);
    set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, Some(source_index));
    project_abi_rows(values);
    if let Some(visible) = abi_visible_index_for_source(values, source_index) {
        set_abi_index_signal(values, ABI_FOCUS_SIGNAL, Some(visible));
        set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(visible));
        project_abi_selection_from_sources(values);
    } else {
        values.insert(SignalId(3), "1 items selected".into());
        clear_abi_visible_selection(values);
    }
}

fn delete_abi_selected(values: &mut HashMap<SignalId, Value>) {
    let selected_sources = selected_abi_sources(values);
    if selected_sources.is_empty() {
        return;
    }
    let current_path = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    let selected_folder_paths = selected_sources
        .iter()
        .filter_map(|index| {
            let source = abi_source_base(*index);
            let name = values
                .get(&SignalId(source))
                .map(Value::as_text)
                .unwrap_or_default();
            let kind = values
                .get(&SignalId(source + 1))
                .map(Value::as_text)
                .unwrap_or_default();
            (kind == "folder" && !name.is_empty()).then(|| join_clean(&current_path, &name))
        })
        .collect::<Vec<_>>();
    let mut kept = Vec::new();
    for index in 0..24 {
        if selected_sources.contains(&index) {
            continue;
        }
        let source = abi_source_base(index);
        let name = values
            .get(&SignalId(source))
            .map(Value::as_text)
            .unwrap_or_default();
        if name.is_empty() {
            continue;
        }
        let kind = values
            .get(&SignalId(source + 1))
            .map(Value::as_text)
            .unwrap_or_default();
        let modified = values
            .get(&SignalId(source + 2))
            .map(Value::as_text)
            .unwrap_or_default();
        let size = values
            .get(&SignalId(source + 3))
            .map(Value::as_text)
            .unwrap_or_default();
        kept.push((name, kind, modified, size));
    }
    for index in 0..24 {
        for field in 0..4 {
            values.insert(SignalId(abi_source_base(index) + field), Value::None);
        }
    }
    for (index, (name, kind, modified, size)) in kept.into_iter().enumerate().take(24) {
        let source = abi_source_base(index);
        values.insert(SignalId(source), name.into());
        values.insert(SignalId(source + 1), kind.into());
        values.insert(SignalId(source + 2), modified.into());
        values.insert(SignalId(source + 3), size.into());
    }
    dismiss_abi_overlay(values);
    project_abi_rows(values);
    clear_abi_selection(values);
    for path in selected_folder_paths {
        remove_abi_directory_overrides(values, &path);
    }
}

fn duplicate_abi_selected(values: &mut HashMap<SignalId, Value>) {
    let selected_sources = selected_abi_sources(values);
    if selected_sources.is_empty() {
        return;
    }
    let current_path = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    for selected in selected_sources {
        let source = abi_source_base(selected);
        let name = values
            .get(&SignalId(source))
            .map(Value::as_text)
            .unwrap_or_default();
        if name.is_empty() {
            continue;
        }
        let kind = values
            .get(&SignalId(source + 1))
            .map(Value::as_text)
            .unwrap_or_else(|| "file".to_string());
        let modified = values
            .get(&SignalId(source + 2))
            .map(Value::as_text)
            .unwrap_or_else(|| "--".to_string());
        let size = values
            .get(&SignalId(source + 3))
            .map(Value::as_text)
            .unwrap_or_else(|| "--".to_string());
        let target_name = name.clone() + " copy";
        let inserted = append_abi_source_row(values, &target_name, &kind, &modified, &size);
        if inserted.is_some() && kind == "folder" {
            let source_path = join_clean(&current_path, &name);
            let target_path = join_clean(&current_path, &target_name);
            duplicate_abi_directory_overrides(values, &source_path, &target_path);
        }
    }
    dismiss_abi_overlay(values);
    project_abi_rows(values);
}

fn duplicate_abi_directory_overrides(
    values: &mut HashMap<SignalId, Value>,
    source: &str,
    target: &str,
) {
    let existing = values
        .get(&ABI_DIRECTORY_OVERRIDES_SIGNAL)
        .map(Value::as_text)
        .unwrap_or_default();
    let mut lines = existing.lines().map(ToOwned::to_owned).collect::<Vec<_>>();
    let source_prefix = format!("{source}/");
    let existing_source_paths = lines
        .iter()
        .filter_map(|line| abi_directory_override_path(line))
        .filter(|path| path == source || path.starts_with(&source_prefix))
        .collect::<Vec<_>>();
    let paths = if existing_source_paths.is_empty() {
        fixture_directory_paths_under(source)
    } else {
        existing_source_paths
    };
    let overrides = existing.as_str();
    for source_path in paths {
        let target_path = if source_path == source {
            target.to_string()
        } else {
            format!("{target}{}", source_path.trim_start_matches(source))
        };
        let (rows, _) = rows_for_path_with_overrides(&source_path, overrides);
        lines.retain(|line| abi_directory_override_path(line).as_deref() != Some(&target_path));
        lines.push(abi_encode_directory_rows(&target_path, &rows));
    }
    values.insert(ABI_DIRECTORY_OVERRIDES_SIGNAL, lines.join("\n").into());
}

fn abi_directory_override_path(line: &str) -> Option<String> {
    let rest = line.strip_prefix("dir\t")?;
    rest.split_once('\t').map(|(path, _)| path.to_string())
}

fn remove_abi_directory_overrides(values: &mut HashMap<SignalId, Value>, path: &str) {
    let existing = values
        .get(&ABI_DIRECTORY_OVERRIDES_SIGNAL)
        .map(Value::as_text)
        .unwrap_or_default();
    let prefix = format!("{path}/");
    let kept = existing
        .lines()
        .filter(|line| {
            abi_directory_override_path(line)
                .map(|candidate| candidate != path && !candidate.starts_with(&prefix))
                .unwrap_or(true)
        })
        .collect::<Vec<_>>()
        .join("\n");
    values.insert(ABI_DIRECTORY_OVERRIDES_SIGNAL, kept.into());
}

fn rename_abi_directory_overrides(
    values: &mut HashMap<SignalId, Value>,
    old_path: &str,
    new_path: &str,
) {
    let existing = values
        .get(&ABI_DIRECTORY_OVERRIDES_SIGNAL)
        .map(Value::as_text)
        .unwrap_or_default();
    let old_prefix = format!("{old_path}/");
    let rewritten = existing
        .lines()
        .map(|line| {
            let Some(path) = abi_directory_override_path(line) else {
                return line.to_string();
            };
            if path != old_path && !path.starts_with(&old_prefix) {
                return line.to_string();
            }
            let Some((_prefix, payload)) = line.rsplit_once('\t') else {
                return line.to_string();
            };
            let renamed_path = if path == old_path {
                new_path.to_string()
            } else {
                format!("{new_path}{}", path.trim_start_matches(old_path))
            };
            format!("dir\t{renamed_path}\t{payload}")
        })
        .collect::<Vec<_>>()
        .join("\n");
    values.insert(ABI_DIRECTORY_OVERRIDES_SIGNAL, rewritten.into());
}

fn fixture_directory_paths_under(path: &str) -> Vec<String> {
    let fs = FixtureFileSystem::finder_lite();
    let mut paths = Vec::new();
    collect_fixture_directory_paths(&fs, path, &mut paths);
    paths
}

fn collect_fixture_directory_paths(fs: &FixtureFileSystem, path: &str, paths: &mut Vec<String>) {
    let Ok(rows) = fs.read_dir(path) else {
        return;
    };
    paths.push(path.to_string());
    for row in rows {
        if row.kind == "folder" {
            collect_fixture_directory_paths(fs, &row.path, paths);
        }
    }
}

fn new_abi_folder(values: &mut HashMap<SignalId, Value>) {
    let inserted = append_abi_source_row(values, "Untitled Folder", "folder", "--", "--");
    dismiss_abi_overlay(values);
    project_abi_rows(values);
    if let Some(inserted) = inserted {
        clear_abi_source_selection(values);
        set_abi_source_selected(values, inserted, true);
        set_abi_index_signal(values, ABI_SELECTED_SOURCE_SIGNAL, Some(inserted));
        if let Some(visible) = abi_visible_index_for_source(values, inserted) {
            set_abi_selected_index(values, visible, true);
        } else {
            values.insert(SignalId(3), "1 items selected".into());
            if abi_visible_len(values) > 0 {
                set_abi_index_signal(values, ABI_FOCUS_SIGNAL, Some(0));
                set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, Some(0));
            } else {
                set_abi_index_signal(values, ABI_FOCUS_SIGNAL, None);
                set_abi_index_signal(values, ABI_ANCHOR_SIGNAL, None);
            }
            clear_abi_visible_selection(values);
        }
    }
}

fn append_abi_source_row(
    values: &mut HashMap<SignalId, Value>,
    name: &str,
    kind: &str,
    modified: &str,
    size: &str,
) -> Option<usize> {
    let Some(index) = (0..24).find(|index| {
        values
            .get(&SignalId(abi_source_base(*index)))
            .map(Value::as_text)
            .unwrap_or_default()
            .is_empty()
    }) else {
        return None;
    };
    let source = abi_source_base(index);
    values.insert(SignalId(source), name.to_string().into());
    values.insert(SignalId(source + 1), kind.to_string().into());
    values.insert(SignalId(source + 2), modified.to_string().into());
    values.insert(SignalId(source + 3), size.to_string().into());
    Some(index)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn visible_row_method_id(frame: &vugra_core::Frame, name: &str) -> MethodId {
        let index = frame
            .rows
            .iter()
            .position(|row| row.name == name)
            .unwrap_or_else(|| panic!("missing visible row {name}"));
        match index {
            0 => MethodId(2),
            1 => MethodId(3),
            2 => MethodId(4),
            index => MethodId(24 + (index as u32 - 3)),
        }
    }

    fn open_visible_row_method_id(frame: &vugra_core::Frame, name: &str) -> MethodId {
        let index = frame
            .rows
            .iter()
            .position(|row| row.name == name)
            .unwrap_or_else(|| panic!("missing visible row {name}"));
        MethodId(65 + index as u32)
    }

    fn visible_row<'a>(frame: &'a vugra_core::Frame, name: &str) -> &'a vugra_core::FrameRow {
        frame
            .rows
            .iter()
            .find(|row| row.name == name)
            .unwrap_or_else(|| panic!("missing visible row {name}"))
    }

    #[derive(Default)]
    struct StatErrorFileSystem {
        inner: FixtureFileSystem,
        error_paths: HashSet<String>,
    }

    impl FileSystem for StatErrorFileSystem {
        fn read_dir(&self, path: &str) -> Result<Vec<SystemEntry>, String> {
            self.inner.read_dir(path)
        }

        fn stat(&self, path: &str) -> Result<SystemEntry, FsError> {
            let path = clean_path_str(path);
            if self.error_paths.contains(&path) {
                return Err(FsError::Other(format!("{path}: permission denied")));
            }
            self.inner.stat(&path)
        }

        fn mkdir(&mut self, path: &str) -> Result<(), String> {
            self.inner.mkdir(path)
        }

        fn rename(&mut self, old_path: &str, new_path: &str) -> Result<(), String> {
            self.inner.rename(old_path, new_path)
        }

        fn remove(&mut self, path: &str) -> Result<(), String> {
            self.inner.remove(path)
        }

        fn duplicate(&mut self, source: &str, target: &str) -> Result<(), String> {
            self.inner.duplicate(source, target)
        }
    }

    #[test]
    fn direct_and_abi_finder_lite_render_same_frame() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            render_rust_api_for_test_host(&direct),
            render_for_test_host(&abi)
        );
        assert_eq!(render_direct_text(), render_for_test_host(&abi));
    }

    #[test]
    fn direct_row_methods_update_selection() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(3));
        let frame = app.inner().render_frame();
        assert!(!frame.rows[0].selected);
        assert!(frame.rows[1].selected);
        assert_eq!(frame.rows[1].select_method, MethodId(3));
        assert_eq!(
            frame.rows[1].visual_state,
            vugra_core::RowVisualState::Selected
        );
    }

    #[test]
    fn direct_row_event_modifiers_match_go_multi_select_semantics() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch_event(
            MethodId(2),
            Event {
                kind: "click".to_string(),
                ..Event::default()
            },
        );
        app.dispatch_event(
            MethodId(4),
            Event {
                kind: "click".to_string(),
                modifiers: vugra_core::Modifiers {
                    shift: true,
                    ..Default::default()
                },
                ..Event::default()
            },
        );
        let frame = app.inner().render_frame();
        assert_eq!(frame.selected_summary, "3 items selected");
        assert!(frame.rows[0].selected);
        assert!(frame.rows[1].selected);
        assert!(frame.rows[2].selected);

        app.dispatch_event(
            MethodId(3),
            Event {
                kind: "click".to_string(),
                modifiers: vugra_core::Modifiers {
                    meta: true,
                    ..Default::default()
                },
                ..Event::default()
            },
        );
        let frame = app.inner().render_frame();
        assert_eq!(frame.selected_summary, "2 items selected");
        assert!(frame.rows[0].selected);
        assert!(!frame.rows[1].selected);
        assert!(frame.rows[2].selected);
    }

    #[test]
    fn native_pointer_modifiers_reach_direct_multi_select_semantics() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        let first = render_native_frame(
            app.inner(),
            constraints,
            NativeRenderBackend::Software,
            800,
            600,
        );
        assert!(dispatch_native_pointer(
            app.inner_mut(),
            &first,
            260.0,
            104.0
        ));
        let second = render_native_frame(
            app.inner(),
            constraints,
            NativeRenderBackend::Software,
            800,
            600,
        );
        assert!(dispatch_native_pointer_with_modifiers(
            app.inner_mut(),
            &second,
            260.0,
            164.0,
            vugra_core::Modifiers {
                shift: true,
                ..Default::default()
            },
        ));

        let frame = app.inner().render_frame();
        assert_eq!(frame.selected_summary, "3 items selected");
        assert!(frame.rows[0].selected);
        assert!(frame.rows[1].selected);
        assert!(frame.rows[2].selected);
    }

    #[test]
    fn abi_row_event_modifiers_match_direct_multi_select_semantics() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let click = Event {
            kind: "click".to_string(),
            ..Event::default()
        };
        let shift_click = Event {
            kind: "click".to_string(),
            modifiers: vugra_core::Modifiers {
                shift: true,
                ..Default::default()
            },
            ..Event::default()
        };
        let meta_click = Event {
            kind: "click".to_string(),
            modifiers: vugra_core::Modifiers {
                meta: true,
                ..Default::default()
            },
            ..Event::default()
        };

        direct.dispatch_event(MethodId(2), click.clone());
        abi.dispatch_event(MethodId(2), click);
        direct.dispatch_event(MethodId(4), shift_click.clone());
        abi.dispatch_event(MethodId(4), shift_click);
        direct.dispatch_event(MethodId(3), meta_click.clone());
        abi.dispatch_event(MethodId(3), meta_click);

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = abi.render_frame();
        assert_eq!(frame.selected_summary, "2 items selected");
        assert!(frame.rows[0].selected);
        assert!(!frame.rows[1].selected);
        assert!(frame.rows[2].selected);

        direct.dispatch_event(
            MethodId(43),
            Event {
                kind: "contextmenu".to_string(),
                ..Event::default()
            },
        );
        abi.dispatch_event(
            MethodId(43),
            Event {
                kind: "contextmenu".to_string(),
                ..Event::default()
            },
        );
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = abi.render_frame();
        assert_eq!(frame.selected_summary, "2 items selected");
        assert!(frame.overlays.item_menu_open);
        assert!(frame.rows[0].selected);
        assert!(frame.rows[2].selected);
    }

    #[test]
    fn direct_keyboard_methods_move_selection() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(9));
        app.dispatch(MethodId(9));
        app.dispatch(MethodId(9));
        let frame = app.inner().render_frame();
        assert!(frame.rows[2].selected);
        app.dispatch(MethodId(8));
        let frame = app.inner().render_frame();
        assert!(frame.rows[1].selected);
    }

    #[test]
    fn native_select_all_and_delete_match_go_list_key_semantics() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        assert!(dispatch_native_key(
            direct.inner_mut(),
            NativeKey::SelectAll
        ));
        assert!(dispatch_native_key(&mut abi, NativeKey::SelectAll));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.selected_summary, "12 items selected");
        assert!(frame.rows.iter().all(|row| row.selected));

        assert!(dispatch_native_key(direct.inner_mut(), NativeKey::Delete));
        assert!(dispatch_native_key(&mut abi, NativeKey::Delete));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.selected_summary, "0 items selected");
        assert_eq!(frame.status, "0 items · Current path: Documents");
        assert!(frame.rows.is_empty());
    }

    #[test]
    fn direct_select_all_selects_all_filtered_rows_not_only_projected_rows() {
        let mut files = FixtureFileSystem::default();
        files.add_folder(
            "Documents",
            [
                fixture_file("One.md", 1),
                fixture_file("Two.md", 1),
                fixture_file("Three.md", 1),
                fixture_file("Four.md", 1),
                fixture_file("Five.md", 1),
                fixture_file("Six.md", 1),
                fixture_file("Seven.md", 1),
                fixture_file("Eight.md", 1),
                fixture_file("Nine.md", 1),
                fixture_file("Ten.md", 1),
                fixture_file("Eleven.md", 1),
                fixture_file("Twelve.md", 1),
                fixture_file("Thirteen.md", 1),
            ],
        );
        let sidebar = FinderSidebar {
            documents: "Documents".to_string(),
            downloads: "Downloads".to_string(),
            pictures: "Pictures".to_string(),
            project_a: "Project A".to_string(),
            project_b: "Project B".to_string(),
        };
        let mut app = RustApp::mount(
            finder_lite_contract(),
            FinderLiteDirect::with_fixture(files, sidebar),
        );

        app.dispatch(MethodId(80));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows.len(), 12);
        assert_eq!(frame.selected_summary, "13 items selected");
        assert!(frame.rows.iter().all(|row| row.selected));

        app.dispatch(MethodId(36));
        let frame = app.inner().render_frame();
        assert_eq!(frame.status, "0 items · Current path: Documents");
        assert!(frame.rows.is_empty());
    }

    #[test]
    fn abi_select_all_selects_all_filtered_sources_not_only_projected_rows() {
        let component = finder_lite_contract();
        let mut abi = App::new(component, finder_lite_abi());
        for index in 0..13 {
            let source = abi_source_base(index);
            abi.state_mut().set_signal(
                SignalId(source),
                format!("Parity {:02}.md", index + 1).into(),
            );
            abi.state_mut()
                .set_signal(SignalId(source + 1), "file".into());
            abi.state_mut()
                .set_signal(SignalId(source + 2), "--".into());
            abi.state_mut()
                .set_signal(SignalId(source + 3), "1 B".into());
        }
        abi.dispatch_event(
            MethodId(10),
            Event {
                kind: "text".to_string(),
                text: "parity".to_string(),
                ..Event::default()
            },
        );

        abi.dispatch(MethodId(80));
        let frame = abi.render_frame();
        assert_eq!(frame.rows.len(), 12);
        assert_eq!(frame.status, "13 items · Current path: Documents");
        assert_eq!(frame.selected_summary, "13 items selected");
        assert!(frame.rows.iter().all(|row| row.selected));

        abi.dispatch(MethodId(36));
        let frame = abi.render_frame();
        assert_eq!(frame.status, "0 items · Current path: Documents");
        assert_eq!(frame.selected_summary, "0 items selected");
        assert!(frame.rows.is_empty());
    }

    #[test]
    fn direct_hover_focus_and_editing_row_visual_states_match_go_classes() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(53));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows[0].class, "file-row-hover");
        assert_eq!(
            frame.rows[0].visual_state,
            vugra_core::RowVisualState::Hover
        );

        app.dispatch(MethodId(54));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows[1].class, "file-row-hover");
        assert_eq!(
            frame.rows[1].visual_state,
            vugra_core::RowVisualState::Hover
        );

        app.dispatch(MethodId(3));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows[1].class, "file-row-selected");
        assert_eq!(
            frame.rows[1].visual_state,
            vugra_core::RowVisualState::Selected
        );
        assert_eq!(frame.rows[0].class, "file-row");

        app.dispatch(MethodId(33));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows[1].class, "file-row-editing");
        assert_eq!(
            frame.rows[1].visual_state,
            vugra_core::RowVisualState::Editing
        );
    }

    #[test]
    fn native_hover_updates_direct_row_visual_state() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        let frame = render_native_frame(
            app.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );

        assert!(dispatch_native_hover(app.inner_mut(), &frame, 260.0, 134.0));
        let frame = app.inner().render_frame();
        assert_eq!(frame.rows[1].class, "file-row-hover");
        assert_eq!(
            frame.rows[1].visual_state,
            vugra_core::RowVisualState::Hover
        );
    }

    #[test]
    fn direct_and_abi_splitter_hover_and_drag_match_go_sidebar_semantics() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(81));
        abi.dispatch(MethodId(81));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        let splitter = frame.splitter.expect("splitter");
        assert_eq!(splitter.sidebar_class, "sidebar");
        assert_eq!(splitter.splitter_class, "splitter-hover");

        let event = Event {
            kind: "drag".to_string(),
            delta_x: 80.0,
            ..Event::default()
        };
        direct.dispatch_event(MethodId(82), event.clone());
        abi.dispatch_event(MethodId(82), event);
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        let splitter = frame.splitter.expect("splitter");
        assert_eq!(splitter.sidebar_class, "sidebar-200");
        assert_eq!(splitter.splitter_class, "splitter");
    }

    #[test]
    fn native_splitter_hover_and_drag_update_direct_and_abi_layout_state() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let constraints = Constraints {
            width: 800.0,
            height: 600.0,
        };
        let direct_frame = render_native_frame(
            direct.inner(),
            constraints,
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame =
            render_native_frame(&abi, constraints, NativeRenderBackend::Software, 800, 600);

        assert!(dispatch_native_hover(
            direct.inner_mut(),
            &direct_frame,
            242.0,
            80.0
        ));
        assert!(dispatch_native_hover(&mut abi, &abi_frame, 242.0, 80.0));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct
                .inner()
                .render_frame()
                .splitter
                .as_ref()
                .expect("splitter")
                .splitter_class,
            "splitter-hover"
        );

        let direct_frame = render_native_frame(
            direct.inner(),
            constraints,
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame =
            render_native_frame(&abi, constraints, NativeRenderBackend::Software, 800, 600);
        assert!(dispatch_native_drag(
            direct.inner_mut(),
            &direct_frame,
            242.0,
            80.0,
            80.0,
            0.0
        ));
        assert!(dispatch_native_drag(
            &mut abi, &abi_frame, 242.0, 80.0, 80.0, 0.0
        ));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct
                .inner()
                .render_frame()
                .splitter
                .as_ref()
                .expect("splitter")
                .sidebar_class,
            "sidebar-200"
        );
    }

    #[test]
    fn direct_sidebar_methods_change_location_and_rows() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(6));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Downloads");
        assert!(frame.sidebar[1].active);
        assert_eq!(frame.rows[0].name, "Receipts");
        assert_eq!(frame.selected_summary, "0 items selected");
        assert!(!frame.rows[0].selected);
        app.dispatch(visible_row_method_id(&frame, "Installer.dmg"));
        let frame = app.inner().render_frame();
        assert!(frame
            .rows
            .iter()
            .any(|row| row.name == "Installer.dmg" && row.selected));
    }

    #[test]
    fn direct_toolbar_back_and_forward_navigate_history() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        assert_eq!(app.inner().render_frame().path, "Documents");
        assert!(app.inner().render_frame().toolbar.is_some());

        app.dispatch(MethodId(6));
        assert_eq!(app.inner().render_frame().path, "Downloads");
        app.dispatch(MethodId(1));
        assert_eq!(app.inner().render_frame().path, "Documents");
        app.dispatch(MethodId(20));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Downloads");
        assert!(frame.sidebar[1].active);
        assert_eq!(frame.rows[0].name, "Receipts");
    }

    #[test]
    fn direct_state_reads_rows_from_file_system_fixture() {
        let mut files = FixtureFileSystem::default();
        files.add_folder(
            "/tmp/vugra-docs",
            [fixture_folder("Specs"), fixture_file("Notes.md", 128)],
        );
        files.add_folder(
            "/tmp/vugra-docs/Specs",
            [
                fixture_file("Layout.md", 4096),
                fixture_file("Runtime.md", 2048),
            ],
        );
        let sidebar = FinderSidebar {
            documents: "/tmp/vugra-docs".to_string(),
            downloads: "/tmp/vugra-downloads".to_string(),
            pictures: "/tmp/vugra-pictures".to_string(),
            project_a: "/tmp/vugra-project-a".to_string(),
            project_b: "/tmp/vugra-project-b".to_string(),
        };
        let mut app = RustApp::mount(
            finder_lite_contract(),
            FinderLiteDirect::with_fixture(files, sidebar),
        );

        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "/tmp/vugra-docs");
        assert_eq!(frame.rows[0].name, "Specs");
        assert_eq!(frame.rows[1].name, "Notes.md");
        assert!(frame.sidebar[0].active);

        app.dispatch(MethodId(2));
        app.dispatch(MethodId(13));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "/tmp/vugra-docs/Specs");
        assert_eq!(frame.rows[0].name, "Layout.md");
        assert_eq!(
            frame.status,
            "2 items · Current path: /tmp/vugra-docs/Specs"
        );
        assert!(frame.sidebar[0].active);
    }

    #[test]
    fn direct_navigation_rejects_non_folder_targets_like_go_system_stat() {
        let mut files = FixtureFileSystem::default();
        files.add_folder(
            "/tmp/vugra-docs",
            [fixture_file("Notes.md", 128), fixture_folder("Specs")],
        );
        files.add_folder("/tmp/vugra-docs/Specs", [fixture_file("Layout.md", 4096)]);
        let sidebar = FinderSidebar {
            documents: "/tmp/vugra-docs".to_string(),
            downloads: "/tmp/vugra-docs/Notes.md".to_string(),
            pictures: "/tmp/vugra-pictures".to_string(),
            project_a: "/tmp/vugra-project-a".to_string(),
            project_b: "/tmp/vugra-project-b".to_string(),
        };
        let mut app = RustApp::mount(
            finder_lite_contract(),
            FinderLiteDirect::with_fixture(files, sidebar),
        );

        app.dispatch(MethodId(6));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "/tmp/vugra-docs");
        assert_eq!(frame.status, "/tmp/vugra-docs/Notes.md is not a folder");
        assert!(frame.rows.iter().any(|row| row.name == "Notes.md"));
    }

    #[test]
    fn direct_new_folder_preserves_system_stat_errors_for_candidate_paths() {
        let mut files = StatErrorFileSystem::default();
        files
            .inner
            .add_folder("Documents", [fixture_file("Roadmap.md", 128)]);
        files
            .error_paths
            .insert("Documents/Untitled Folder".to_string());
        let sidebar = FinderSidebar {
            documents: "Documents".to_string(),
            downloads: "Downloads".to_string(),
            pictures: "Pictures".to_string(),
            project_a: "Project A".to_string(),
            project_b: "Project B".to_string(),
        };
        let mut app = RustApp::mount(
            finder_lite_contract(),
            FinderLiteDirect::with_sidebar_and_file_system(Box::new(files), sidebar),
        );

        app.dispatch(MethodId(38));
        let frame = app.inner().render_frame();
        assert_eq!(
            frame.selected_summary,
            "Documents/Untitled Folder: permission denied"
        );
        assert_eq!(frame.path, "Documents");
        assert!(frame.rows.iter().all(|row| row.name != "Untitled Folder"));
    }

    #[test]
    fn direct_duplicate_folder_preserves_nested_fixture_file_system_tree() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());

        app.dispatch(MethodId(2));
        app.dispatch(MethodId(37));
        app.inner_mut()
            .state_mut()
            .set_directory_path("Documents/Design copy/Assets".to_string());
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design copy/Assets");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Icon.png", "Toolbar.png"]
        );
    }

    #[test]
    fn direct_rename_folder_preserves_nested_fixture_file_system_tree() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());

        app.dispatch(MethodId(2));
        app.dispatch(MethodId(33));
        app.inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Design Renamed".into());
        app.dispatch(MethodId(35));
        let frame = app.inner().render_frame();
        assert!(frame.rows[0].selected);
        assert_eq!(frame.rows[0].name, "Design Renamed");

        app.inner_mut()
            .state_mut()
            .set_directory_path("Documents/Design Renamed/Assets".to_string());
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design Renamed/Assets");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Icon.png", "Toolbar.png"]
        );
    }

    #[test]
    fn direct_modified_dates_match_go_finder_system_format() {
        let now = DateTime::parse_from_rfc3339("2026-05-24T15:30:00+08:00")
            .unwrap()
            .with_timezone(&Local);
        let today = DateTime::parse_from_rfc3339("2026-05-24T09:05:00+08:00")
            .unwrap()
            .with_timezone(&Local);
        let yesterday = DateTime::parse_from_rfc3339("2026-05-23T23:10:00+08:00")
            .unwrap()
            .with_timezone(&Local);
        let same_year = DateTime::parse_from_rfc3339("2026-01-02T08:00:00+08:00")
            .unwrap()
            .with_timezone(&Local);
        let previous_year = DateTime::parse_from_rfc3339("2025-12-31T08:00:00+08:00")
            .unwrap()
            .with_timezone(&Local);

        assert_eq!(format_modified_at(today, now), "Today 09:05");
        assert_eq!(format_modified_at(yesterday, now), "Yesterday");
        assert_eq!(format_modified_at(same_year, now), "Jan 02");
        assert_eq!(format_modified_at(previous_year, now), "2025-12-31");
    }

    #[test]
    fn direct_search_event_filters_rows_and_backspace_restores_matches() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let select_roadmap = visible_row_method_id(&direct.inner().render_frame(), "Roadmap.md");
        direct.dispatch(select_roadmap);
        abi.dispatch(select_roadmap);
        direct.dispatch_event(
            MethodId(10),
            Event {
                kind: "text".to_string(),
                text: "road".to_string(),
                ..Event::default()
            },
        );
        abi.dispatch_event(
            MethodId(10),
            Event {
                kind: "text".to_string(),
                text: "road".to_string(),
                ..Event::default()
            },
        );
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.status, "1 items · Current path: Documents");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Roadmap.md");
        assert_eq!(frame.selected_summary, "1 items selected");
        assert!(frame.rows[0].selected);

        for _ in 0..4 {
            direct.dispatch(MethodId(11));
            abi.dispatch(MethodId(11));
        }
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "");
        assert_eq!(frame.rows.len(), 12);
        assert_eq!(frame.rows[11].name, "Roadmap.md");
        assert_eq!(frame.selected_summary, "1 items selected");
        assert!(frame.rows[11].selected);

        direct.dispatch(MethodId(12));
        abi.dispatch(MethodId(12));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().selected_summary,
            "1 items selected"
        );
    }

    #[test]
    fn direct_open_selected_enters_folder_or_updates_file_status() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(2));
        app.dispatch(MethodId(13));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design");
        assert_eq!(frame.rows[0].name, "Assets");
        assert!(frame.sidebar[0].active);

        app.dispatch(visible_row_method_id(&frame, "Components.sketch"));
        app.dispatch(MethodId(13));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design");
        assert!(frame.overlays.preview_open);
        assert_eq!(frame.overlays.preview_title, "Components.sketch");
    }

    #[test]
    fn direct_and_abi_open_row_methods_match_go_double_click_semantics() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(65));
        abi.dispatch(MethodId(65));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(direct.inner().render_frame().path, "Documents/Design");

        let open_components =
            open_visible_row_method_id(&direct.inner().render_frame(), "Components.sketch");
        direct.dispatch(open_components);
        abi.dispatch(open_components);
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert!(direct.inner().render_frame().overlays.preview_open);
        assert_eq!(
            direct.inner().render_frame().overlays.preview_title,
            "Components.sketch"
        );
    }

    #[test]
    fn direct_and_abi_blank_file_pane_click_clears_selection_without_opening_menu() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        let select_roadmap = visible_row_method_id(&direct.inner().render_frame(), "Roadmap.md");
        direct.dispatch(select_roadmap);
        abi.dispatch(select_roadmap);
        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));

        direct.dispatch_event(
            MethodId(77),
            Event {
                kind: "click".to_string(),
                ..Event::default()
            },
        );
        abi.dispatch_event(
            MethodId(77),
            Event {
                kind: "click".to_string(),
                ..Event::default()
            },
        );

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.selected_summary, "0 items selected");
        assert!(frame.rows.iter().all(|row| !row.selected));
        assert!(!frame.overlays.item_menu_open);
        assert!(!frame.overlays.blank_menu_open);
        assert!(frame.overlays.rename_text.is_empty());
    }

    #[test]
    fn native_blank_area_click_and_context_menu_use_distinct_file_pane_handlers() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        let select_roadmap = visible_row_method_id(&direct.inner().render_frame(), "Roadmap.md");
        direct.dispatch(select_roadmap);
        abi.dispatch(select_roadmap);
        let direct_frame = render_native_frame(
            direct.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame = render_native_frame(
            &abi,
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );

        assert!(dispatch_native_context_menu(
            direct.inner_mut(),
            &direct_frame,
            700.0,
            500.0
        ));
        assert!(dispatch_native_context_menu(
            &mut abi, &abi_frame, 700.0, 500.0
        ));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert!(direct.inner().render_frame().overlays.blank_menu_open);

        assert!(dispatch_native_pointer(
            direct.inner_mut(),
            &direct_frame,
            700.0,
            500.0
        ));
        assert!(dispatch_native_pointer(&mut abi, &abi_frame, 700.0, 500.0));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.selected_summary, "0 items selected");
        assert!(!frame.overlays.blank_menu_open);
        assert!(frame.rows.iter().all(|row| !row.selected));
    }

    #[test]
    fn native_double_click_opens_row_through_layout_hit_test() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component, FinderLiteDirect::new());
        let frame = render_native_frame(
            direct.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );

        assert!(dispatch_native_double_click(
            direct.inner_mut(),
            &frame,
            260.0,
            104.0
        ));
        assert_eq!(direct.inner().render_frame().path, "Documents/Design");
    }

    #[test]
    fn direct_open_parent_returns_to_previous_directory() {
        let mut app = RustApp::mount(finder_lite_contract(), FinderLiteDirect::new());
        app.dispatch(MethodId(2));
        app.dispatch(MethodId(13));
        app.dispatch(MethodId(14));
        let frame = app.inner().render_frame();
        assert_eq!(frame.path, "Documents");
        assert!(frame.sidebar[0].active);
        assert_eq!(frame.rows[0].name, "Design");
    }

    #[test]
    fn abi_back_and_forward_match_direct_history_navigation() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(65));
        abi.dispatch(MethodId(65));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(direct.inner().render_frame().path, "Documents/Design");

        direct.dispatch(MethodId(1));
        abi.dispatch(MethodId(1));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(direct.inner().render_frame().path, "Documents");

        direct.dispatch(MethodId(20));
        abi.dispatch(MethodId(20));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(direct.inner().render_frame().path, "Documents/Design");
    }

    #[test]
    fn abi_search_event_matches_direct_search_frame() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "zip".to_string(),
            ..Event::default()
        };

        direct.dispatch(MethodId(6));
        abi.dispatch(MethodId(6));
        direct.dispatch(MethodId(4));
        abi.dispatch(MethodId(4));
        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = abi.render_frame();
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Archive.zip");
    }

    #[test]
    fn direct_and_abi_search_filter_names_only_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "folder".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "folder");
        assert_eq!(frame.rows.len(), 0);
        assert_eq!(frame.status, "0 items · Current path: Documents");
    }

    #[test]
    fn direct_and_abi_search_filter_trims_query_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: " road ".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, " road ");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Roadmap.md");
        assert_eq!(frame.status, "1 items · Current path: Documents");
    }

    #[test]
    fn direct_and_abi_rename_reprojects_active_search_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));
        direct
            .inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Archive.md".into());
        abi.state_mut()
            .set_signal(SignalId(102), "Archive.md".into());
        direct.dispatch(MethodId(35));
        abi.dispatch(MethodId(35));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.status, "0 items · Current path: Documents");
        assert!(frame.rows.is_empty());
        assert_eq!(frame.selected_summary, "1 items selected");
        assert!(frame.overlays.rename_text.is_empty());
    }

    #[test]
    fn direct_and_abi_new_folder_preserves_hidden_selection_under_active_search() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.status, "1 items · Current path: Documents");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Roadmap.md");
        assert!(!frame.rows[0].selected);
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn direct_and_abi_new_folder_selects_inserted_row_when_search_matches() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "untitled".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "untitled");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Untitled Folder");
        assert!(frame.rows[0].selected);
        assert_eq!(frame.rows[0].class, "file-row-selected");
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn direct_and_abi_enter_after_hidden_new_folder_opens_focused_visible_row_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().rows[0].class,
            "file-row-focus"
        );

        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert!(frame.overlays.preview_open);
        assert_eq!(frame.overlays.preview_title, "Roadmap.md");
        assert_eq!(frame.path, "Documents");
    }

    #[test]
    fn direct_and_abi_rename_hidden_selection_under_active_search_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().overlays.rename_text,
            "Untitled Folder"
        );

        direct
            .inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Road Archive".into());
        abi.state_mut()
            .set_signal(SignalId(102), "Road Archive".into());
        direct.dispatch(MethodId(35));
        abi.dispatch(MethodId(35));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Road Archive", "Roadmap.md"]
        );
        assert!(frame.rows[0].selected);
        assert!(!frame.rows[1].selected);
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn direct_and_abi_rename_hidden_selection_keeps_visible_focus_when_still_filtered() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));
        direct
            .inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Archive".into());
        abi.state_mut().set_signal(SignalId(102), "Archive".into());
        direct.dispatch(MethodId(35));
        abi.dispatch(MethodId(35));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Roadmap.md");
        assert_eq!(frame.rows[0].class, "file-row-focus");
        assert!(!frame.rows[0].selected);
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn direct_and_abi_duplicate_hidden_selection_under_active_search_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));
        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.rows.len(), 1);
        assert_eq!(frame.rows[0].name, "Roadmap.md");
        assert_eq!(frame.rows[0].class, "file-row-focus");
        assert!(!frame.rows[0].selected);
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn direct_and_abi_duplicate_folder_preserves_openable_child_rows() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());

        let event = Event {
            kind: "text".to_string(),
            text: "copy".to_string(),
            ..Event::default()
        };
        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        assert_eq!(direct.inner().render_frame(), abi.render_frame());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design copy");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Assets", "Components.sketch", "Prototype.mov"]
        );

        let open_assets = visible_row_method_id(&direct.inner().render_frame(), "Assets");
        direct.dispatch(open_assets);
        abi.dispatch(open_assets);
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design copy/Assets");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Icon.png", "Toolbar.png"]
        );
    }

    #[test]
    fn direct_and_abi_delete_duplicate_folder_removes_copied_child_rows() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));
        let event = Event {
            kind: "text".to_string(),
            text: "copy".to_string(),
            ..Event::default()
        };
        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(36));
        abi.dispatch(MethodId(36));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "copy");
        assert!(frame.rows.is_empty());
        assert_eq!(frame.status, "0 items · Current path: Documents");
        assert_eq!(frame.selected_summary, "0 items selected");
    }

    #[test]
    fn direct_and_abi_rename_duplicate_folder_preserves_copied_child_rows() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));
        let event = Event {
            kind: "text".to_string(),
            text: "copy".to_string(),
            ..Event::default()
        };
        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));
        direct
            .inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Design Archive".into());
        abi.state_mut()
            .set_signal(SignalId(102), "Design Archive".into());
        direct.dispatch(MethodId(35));
        abi.dispatch(MethodId(35));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());

        direct.dispatch(MethodId(12));
        abi.dispatch(MethodId(12));
        let event = Event {
            kind: "text".to_string(),
            text: "Archive".to_string(),
            ..Event::default()
        };
        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct
                .inner()
                .render_frame()
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Design Archive", "Notes Archive.txt"]
        );
        let select_archive =
            visible_row_method_id(&direct.inner().render_frame(), "Design Archive");
        direct.dispatch(select_archive);
        abi.dispatch(select_archive);
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert!(direct.inner().render_frame().rows[0].selected);
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design Archive");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Assets", "Components.sketch", "Prototype.mov"]
        );

        let select_assets = visible_row_method_id(&direct.inner().render_frame(), "Assets");
        direct.dispatch(select_assets);
        abi.dispatch(select_assets);
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.path, "Documents/Design Archive/Assets");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Icon.png", "Toolbar.png"]
        );
    }

    #[test]
    fn direct_and_abi_duplicate_reprojects_active_search_like_go_finder() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());
        let event = Event {
            kind: "text".to_string(),
            text: "road".to_string(),
            ..Event::default()
        };

        direct.dispatch_event(MethodId(10), event.clone());
        abi.dispatch_event(MethodId(10), event);
        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(frame.search_query, "road");
        assert_eq!(frame.status, "2 items · Current path: Documents");
        assert_eq!(
            frame
                .rows
                .iter()
                .map(|row| row.name.as_str())
                .collect::<Vec<_>>(),
            vec!["Roadmap.md", "Roadmap.md copy"]
        );
        assert!(frame.rows[0].selected);
        assert!(!frame.rows[1].selected);
        assert_eq!(frame.selected_summary, "1 items selected");
    }

    #[test]
    fn abi_open_selected_matches_direct_open_frame() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(6));
        abi.dispatch(MethodId(6));
        let select_receipts = visible_row_method_id(&direct.inner().render_frame(), "Receipts");
        direct.dispatch(select_receipts);
        abi.dispatch(select_receipts);
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(abi.render_frame().path, "Downloads/Receipts");
    }

    #[test]
    fn abi_open_parent_matches_direct_parent_frame() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(2));
        abi.dispatch(MethodId(2));
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));
        direct.dispatch(MethodId(14));
        abi.dispatch(MethodId(14));

        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(abi.render_frame().path, "Documents");
    }

    #[test]
    fn direct_and_abi_blank_menu_paste_and_refresh_match_go_overlay_semantics() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        let select_roadmap = visible_row_method_id(&direct.inner().render_frame(), "Roadmap.md");
        direct.dispatch(select_roadmap);
        abi.dispatch(select_roadmap);
        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        assert!(direct.inner().render_frame().overlays.blank_menu_open);

        direct.dispatch(MethodId(78));
        abi.dispatch(MethodId(78));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert!(!frame.overlays.blank_menu_open);
        assert_eq!(frame.selected_summary, "1 items selected");
        assert!(visible_row(&frame, "Roadmap.md").selected);

        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        direct.dispatch(MethodId(79));
        abi.dispatch(MethodId(79));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert!(!frame.overlays.blank_menu_open);
        assert_eq!(frame.status, "12 items · Current path: Documents");
        assert_eq!(frame.selected_summary, "1 items selected");
        assert!(visible_row(&frame, "Roadmap.md").selected);
    }

    #[test]
    fn native_blank_menu_items_dispatch_paste_and_refresh_methods() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        let direct_frame = render_native_frame(
            direct.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame = render_native_frame(
            &abi,
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );

        assert!(dispatch_native_pointer(
            direct.inner_mut(),
            &direct_frame,
            270.0,
            104.0
        ));
        assert!(dispatch_native_pointer(&mut abi, &abi_frame, 270.0, 104.0));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert!(!direct.inner().render_frame().overlays.blank_menu_open);

        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        let direct_frame = render_native_frame(
            direct.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame = render_native_frame(
            &abi,
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        assert!(dispatch_native_pointer(
            direct.inner_mut(),
            &direct_frame,
            270.0,
            134.0
        ));
        assert!(dispatch_native_pointer(&mut abi, &abi_frame, 270.0, 134.0));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().status,
            "12 items · Current path: Documents"
        );
        assert!(!direct.inner().render_frame().overlays.blank_menu_open);
    }

    #[test]
    fn native_overlay_background_click_dismisses_blank_menu() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        direct.dispatch(MethodId(39));
        abi.dispatch(MethodId(39));
        let direct_frame = render_native_frame(
            direct.inner(),
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );
        let abi_frame = render_native_frame(
            &abi,
            Constraints {
                width: 800.0,
                height: 600.0,
            },
            NativeRenderBackend::Software,
            800,
            600,
        );

        assert!(dispatch_native_pointer(
            direct.inner_mut(),
            &direct_frame,
            760.0,
            540.0
        ));
        assert!(dispatch_native_pointer(&mut abi, &abi_frame, 760.0, 540.0));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert!(!direct.inner().render_frame().overlays.blank_menu_open);
    }

    #[test]
    fn direct_and_abi_overlay_actions_match() {
        let component = finder_lite_contract();
        let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
        let mut abi = App::new(component, finder_lite_abi());

        let select_roadmap = visible_row_method_id(&direct.inner().render_frame(), "Roadmap.md");
        direct.dispatch(select_roadmap);
        abi.dispatch(select_roadmap);
        direct.dispatch(MethodId(13));
        abi.dispatch(MethodId(13));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert!(frame.overlays.preview_open);
        assert_eq!(frame.overlays.preview_title, "Roadmap.md");
        assert!(frame.overlays.preview_body.contains("12 KB"));

        direct.dispatch(MethodId(40));
        abi.dispatch(MethodId(40));
        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().overlays.rename_text,
            "Roadmap.md"
        );

        assert!(dispatch_native_text(
            direct.inner_mut(),
            " Final".to_string()
        ));
        assert!(dispatch_native_text(&mut abi, " Final".to_string()));
        assert_eq!(
            direct.inner().render_frame().overlays.rename_text,
            "Roadmap.md Final"
        );
        assert!(dispatch_native_key(
            direct.inner_mut(),
            NativeKey::Backspace
        ));
        assert!(dispatch_native_key(&mut abi, NativeKey::Backspace));
        assert_eq!(
            direct.inner().render_frame().overlays.rename_text,
            "Roadmap.md Fina"
        );
        direct
            .inner_mut()
            .state_mut()
            .set_signal(SignalId(102), "Roadmap Final.md".into());
        abi.state_mut()
            .set_signal(SignalId(102), "Roadmap Final.md".into());
        assert!(dispatch_native_key(direct.inner_mut(), NativeKey::Enter));
        assert!(dispatch_native_key(&mut abi, NativeKey::Enter));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        let frame = direct.inner().render_frame();
        assert_eq!(
            visible_row(&frame, "Roadmap Final.md").name,
            "Roadmap Final.md"
        );

        direct.dispatch(MethodId(33));
        abi.dispatch(MethodId(33));
        assert!(dispatch_native_key(direct.inner_mut(), NativeKey::Escape));
        assert!(dispatch_native_key(&mut abi, NativeKey::Escape));
        assert!(direct
            .inner()
            .render_frame()
            .overlays
            .rename_text
            .is_empty());
        assert_eq!(direct.inner().render_frame(), abi.render_frame());

        direct.dispatch(MethodId(37));
        abi.dispatch(MethodId(37));
        assert_eq!(direct.inner().render_frame(), abi.render_frame());
        assert_eq!(
            direct.inner().render_frame().status,
            "13 items · Current path: Documents"
        );

        direct.dispatch(MethodId(38));
        abi.dispatch(MethodId(38));
        assert!(
            direct
                .inner()
                .render_frame()
                .rows
                .iter()
                .any(|row| row.name == "Untitled Folder" && row.kind == "folder")
                || direct.inner().render_frame().status.starts_with("14 items")
        );

        direct.dispatch(MethodId(36));
        abi.dispatch(MethodId(36));
        assert!(!direct.inner().render_frame().overlays.item_menu_open);
        assert!(!abi.render_frame().overlays.item_menu_open);
    }

    #[test]
    fn abi_native_backends_match_direct_native_frames() {
        for backend in [
            NativeRenderBackend::Software,
            NativeRenderBackend::Vello,
            NativeRenderBackend::Wgpu,
        ] {
            let component = finder_lite_contract();
            let mut direct = RustApp::mount(component.clone(), FinderLiteDirect::new());
            let mut abi = App::new(component, finder_lite_abi());
            direct.dispatch(MethodId(2));
            abi.dispatch(MethodId(2));

            let direct_frame = vugra_host_native::render_native_frame(
                direct.inner(),
                Constraints {
                    width: 800.0,
                    height: 600.0,
                },
                backend,
                800,
                600,
            );
            let abi_frame = vugra_host_native::render_native_frame(
                &abi,
                Constraints {
                    width: 800.0,
                    height: 600.0,
                },
                backend,
                800,
                600,
            );
            assert_eq!(direct_frame, abi_frame, "{backend:?}");
        }
    }
}
