//! Stable ABI-shaped values, handles, and an adapter for the Rust kernel contract.

use std::collections::HashMap;
use std::sync::{Mutex, OnceLock};

pub use vugra_core::{ComponentState, Event};
pub use vugra_ir::{MethodId, SignalId};

use vugra_core::{finder_lite_contract, App, Value};
use vugra_host_native::{
    dispatch_native_key, dispatch_native_pointer, dispatch_native_text, render_native_frame,
    render_test_frame, run_app_window_for_frames, NativeFrame, NativeKey, NativeRenderBackend,
    NativeWindowConfig,
};
use vugra_layout::Constraints;

pub type VugraAppHandle = u64;
pub type VugraComponentHandle = u64;
pub type VugraStateHandle = u64;
pub type VugraSignalId = u32;
pub type VugraMethodId = u32;

pub const VUGRA_VALUE_NONE: u32 = 0;
pub const VUGRA_VALUE_BOOL: u32 = 1;
pub const VUGRA_VALUE_NUMBER: u32 = 2;
pub const VUGRA_VALUE_STRING: u32 = 3;

pub const VUGRA_NATIVE_BACKEND_SOFTWARE: u32 = 0;
pub const VUGRA_NATIVE_BACKEND_VELLO: u32 = 1;
pub const VUGRA_NATIVE_BACKEND_WGPU: u32 = 2;

pub const VUGRA_NATIVE_KEY_ARROW_DOWN: u32 = 1;
pub const VUGRA_NATIVE_KEY_ARROW_UP: u32 = 2;
pub const VUGRA_NATIVE_KEY_ENTER: u32 = 3;
pub const VUGRA_NATIVE_KEY_BACKSPACE: u32 = 4;
pub const VUGRA_NATIVE_KEY_ESCAPE: u32 = 5;
pub const VUGRA_NATIVE_KEY_DELETE: u32 = 6;
pub const VUGRA_NATIVE_KEY_SELECT_ALL: u32 = 7;

#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct VugraValue {
    pub kind: u32,
    pub flags: u32,
    pub number: f64,
    pub string_ptr: u64,
    pub string_len: u64,
}

#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct VugraEvent {
    pub kind: u32,
    pub key: u32,
    pub x: f32,
    pub y: f32,
    pub delta_x: f32,
    pub delta_y: f32,
    pub modifiers: u32,
}

#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct VugraTextView {
    pub ptr: u64,
    pub len: u64,
}

#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct VugraNativeFrameView {
    pub commands_len: u64,
    pub pixels_ptr: u64,
    pub pixels_len: u64,
    pub drawn_pixels: u64,
}

#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct VugraNativeWindowSmoke {
    pub frames_presented: u64,
    pub commands_len: u64,
    pub pixels_len: u64,
    pub drawn_pixels: u64,
    pub ok: bool,
}

#[derive(Default)]
pub struct AbiState {
    values: HashMap<SignalId, Value>,
    calls: HashMap<MethodId, MethodCall>,
    event_calls: HashMap<MethodId, EventMethodCall>,
}

type MethodCall = fn(&mut HashMap<SignalId, Value>);
type EventMethodCall = fn(&mut HashMap<SignalId, Value>, Event);

impl AbiState {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_signal(mut self, id: SignalId, value: impl Into<Value>) -> Self {
        self.values.insert(id, value.into());
        self
    }

    pub fn with_method(mut self, id: MethodId, call: MethodCall) -> Self {
        self.calls.insert(id, call);
        self
    }

    pub fn with_event_method(mut self, id: MethodId, call: EventMethodCall) -> Self {
        self.event_calls.insert(id, call);
        self
    }
}

impl Clone for AbiState {
    fn clone(&self) -> Self {
        Self {
            values: self.values.clone(),
            calls: self.calls.clone(),
            event_calls: self.event_calls.clone(),
        }
    }
}

impl ComponentState for AbiState {
    fn get_signal(&self, id: SignalId) -> Value {
        self.values.get(&id).cloned().unwrap_or(Value::None)
    }

    fn set_signal(&mut self, id: SignalId, value: Value) {
        self.values.insert(id, value);
    }

    fn call_method(&mut self, id: MethodId) {
        if let Some(call) = self.calls.get_mut(&id) {
            call(&mut self.values);
        }
    }

    fn call_event_method(&mut self, id: MethodId, event: Event) {
        if let Some(call) = self.event_calls.get_mut(&id) {
            call(&mut self.values, event);
        } else {
            self.call_method(id);
        }
    }
}

impl From<VugraValue> for Value {
    fn from(value: VugraValue) -> Self {
        match value.kind {
            VUGRA_VALUE_BOOL => Value::Bool(value.flags != 0),
            VUGRA_VALUE_NUMBER => Value::Number(value.number),
            _ => Value::None,
        }
    }
}

pub fn encode_owned_string(value: &str, arena: &mut Vec<String>) -> VugraValue {
    arena.push(value.to_string());
    let stored = arena.last().unwrap();
    VugraValue {
        kind: VUGRA_VALUE_STRING,
        flags: 0,
        number: 0.0,
        string_ptr: stored.as_ptr() as u64,
        string_len: stored.len() as u64,
    }
}

pub fn encode_value(value: &Value, arena: &mut Vec<String>) -> VugraValue {
    match value {
        Value::Bool(value) => VugraValue {
            kind: VUGRA_VALUE_BOOL,
            flags: u32::from(*value),
            ..VugraValue::default()
        },
        Value::Number(value) => VugraValue {
            kind: VUGRA_VALUE_NUMBER,
            number: *value,
            ..VugraValue::default()
        },
        Value::String(value) => encode_owned_string(value, arena),
        Value::None => VugraValue::default(),
    }
}

#[derive(Default)]
struct Registry {
    next: u64,
    components: HashMap<VugraComponentHandle, vugra_ir::Component>,
    states: HashMap<VugraStateHandle, AbiState>,
    apps: HashMap<VugraAppHandle, App<AbiState>>,
    text: HashMap<VugraAppHandle, String>,
    frames: HashMap<VugraAppHandle, NativeFrame>,
}

impl Registry {
    fn alloc(&mut self) -> u64 {
        self.next += 1;
        self.next
    }
}

fn registry() -> &'static Mutex<Registry> {
    static REGISTRY: OnceLock<Mutex<Registry>> = OnceLock::new();
    REGISTRY.get_or_init(|| Mutex::new(Registry::default()))
}

#[no_mangle]
pub extern "C" fn vugra_component_finder_lite() -> VugraComponentHandle {
    let mut registry = registry().lock().unwrap();
    let handle = registry.alloc();
    registry.components.insert(handle, finder_lite_contract());
    handle
}

#[no_mangle]
pub extern "C" fn vugra_component_destroy(handle: VugraComponentHandle) {
    let mut registry = registry().lock().unwrap();
    registry.components.remove(&handle);
}

#[no_mangle]
pub extern "C" fn vugra_state_create() -> VugraStateHandle {
    let mut registry = registry().lock().unwrap();
    let handle = registry.alloc();
    registry.states.insert(handle, AbiState::new());
    handle
}

#[no_mangle]
pub extern "C" fn vugra_state_destroy(handle: VugraStateHandle) {
    let mut registry = registry().lock().unwrap();
    registry.states.remove(&handle);
}

#[no_mangle]
pub extern "C" fn vugra_state_set_signal(
    state: VugraStateHandle,
    signal: VugraSignalId,
    value: VugraValue,
) -> bool {
    let mut registry = registry().lock().unwrap();
    let Some(state) = registry.states.get_mut(&state) else {
        return false;
    };
    state.set_signal(SignalId(signal), value.into());
    true
}

#[no_mangle]
pub extern "C" fn vugra_state_set_string_signal(
    state: VugraStateHandle,
    signal: VugraSignalId,
    ptr: *const u8,
    len: usize,
) -> bool {
    if ptr.is_null() {
        return false;
    }
    let value = unsafe {
        let bytes = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(bytes).to_string()
    };
    let mut registry = registry().lock().unwrap();
    let Some(state) = registry.states.get_mut(&state) else {
        return false;
    };
    state.set_signal(SignalId(signal), Value::String(value));
    true
}

#[no_mangle]
pub extern "C" fn vugra_state_get_signal(
    state: VugraStateHandle,
    signal: VugraSignalId,
) -> VugraValue {
    let registry = registry().lock().unwrap();
    let Some(state) = registry.states.get(&state) else {
        return VugraValue::default();
    };
    match state.get_signal(SignalId(signal)) {
        Value::Bool(value) => VugraValue {
            kind: VUGRA_VALUE_BOOL,
            flags: u32::from(value),
            ..VugraValue::default()
        },
        Value::Number(value) => VugraValue {
            kind: VUGRA_VALUE_NUMBER,
            number: value,
            ..VugraValue::default()
        },
        Value::String(_) | Value::None => VugraValue::default(),
    }
}

#[no_mangle]
pub extern "C" fn vugra_app_create(
    component: VugraComponentHandle,
    state: VugraStateHandle,
) -> VugraAppHandle {
    let mut registry = registry().lock().unwrap();
    let Some(component) = registry.components.get(&component).cloned() else {
        return 0;
    };
    let Some(state) = registry.states.get(&state).cloned() else {
        return 0;
    };
    let handle = registry.alloc();
    registry.apps.insert(handle, App::new(component, state));
    handle
}

#[no_mangle]
pub extern "C" fn vugra_app_destroy(handle: VugraAppHandle) {
    let mut registry = registry().lock().unwrap();
    registry.apps.remove(&handle);
    registry.text.remove(&handle);
    registry.frames.remove(&handle);
}

#[no_mangle]
pub extern "C" fn vugra_app_call_method(app: VugraAppHandle, method: VugraMethodId) -> bool {
    let mut registry = registry().lock().unwrap();
    let Some(app) = registry.apps.get_mut(&app) else {
        return false;
    };
    app.dispatch(MethodId(method));
    true
}

#[no_mangle]
pub extern "C" fn vugra_app_call_text_method(
    app: VugraAppHandle,
    method: VugraMethodId,
    ptr: *const u8,
    len: usize,
) -> bool {
    if ptr.is_null() {
        return false;
    }
    let text = unsafe {
        let bytes = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(bytes).to_string()
    };
    let mut registry = registry().lock().unwrap();
    let Some(app) = registry.apps.get_mut(&app) else {
        return false;
    };
    app.dispatch_event(
        MethodId(method),
        Event {
            kind: "text".to_string(),
            text,
            ..Event::default()
        },
    );
    true
}

#[no_mangle]
pub extern "C" fn vugra_app_render_text(
    app: VugraAppHandle,
    width: f32,
    height: f32,
) -> VugraTextView {
    let mut registry = registry().lock().unwrap();
    let Some(app_ref) = registry.apps.get(&app) else {
        return VugraTextView::default();
    };
    let output = render_test_frame(app_ref, Constraints { width, height }).to_string();
    registry.text.insert(app, output);
    let stored = registry.text.get(&app).unwrap();
    VugraTextView {
        ptr: stored.as_ptr() as u64,
        len: stored.len() as u64,
    }
}

#[no_mangle]
pub extern "C" fn vugra_app_render_native_frame(
    app: VugraAppHandle,
    backend: u32,
    width: u32,
    height: u32,
) -> VugraNativeFrameView {
    let mut registry = registry().lock().unwrap();
    let Some(app_ref) = registry.apps.get(&app) else {
        return VugraNativeFrameView::default();
    };
    let Some(backend) = native_backend_from_abi(backend) else {
        return VugraNativeFrameView::default();
    };
    let width = width.max(1) as usize;
    let height = height.max(1) as usize;
    let frame = render_native_frame(
        app_ref,
        Constraints {
            width: width as f32,
            height: height as f32,
        },
        backend,
        width,
        height,
    );
    let view = native_frame_view(&frame);
    registry.frames.insert(app, frame);
    view
}

#[no_mangle]
pub extern "C" fn vugra_app_run_native_window_for_frames(
    app: VugraAppHandle,
    backend: u32,
    width: u32,
    height: u32,
    frames: u32,
) -> VugraNativeWindowSmoke {
    std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        let mut registry = registry().lock().unwrap();
        let Some(app_ref) = registry.apps.get_mut(&app) else {
            return VugraNativeWindowSmoke::default();
        };
        let Some(backend) = native_backend_from_abi(backend) else {
            return VugraNativeWindowSmoke::default();
        };
        let config = NativeWindowConfig {
            title: "Vugra ABI native window smoke".to_string(),
            width: width.max(1) as usize,
            height: height.max(1) as usize,
            backend,
        };
        let Ok(smoke) = run_app_window_for_frames(app_ref, config, frames.max(1) as usize) else {
            return VugraNativeWindowSmoke::default();
        };
        VugraNativeWindowSmoke {
            frames_presented: smoke.frames_presented as u64,
            commands_len: smoke.commands as u64,
            pixels_len: smoke.pixels as u64,
            drawn_pixels: smoke.drawn_pixels as u64,
            ok: true,
        }
    }))
    .unwrap_or_default()
}

#[no_mangle]
pub extern "C" fn vugra_app_dispatch_native_pointer(app: VugraAppHandle, x: f32, y: f32) -> bool {
    let mut registry = registry().lock().unwrap();
    let Some(frame) = registry.frames.get(&app).cloned() else {
        return false;
    };
    let Some(app_ref) = registry.apps.get_mut(&app) else {
        return false;
    };
    dispatch_native_pointer(app_ref, &frame, x, y)
}

#[no_mangle]
pub extern "C" fn vugra_app_dispatch_native_key(app: VugraAppHandle, key: u32) -> bool {
    let Some(key) = native_key_from_abi(key) else {
        return false;
    };
    let mut registry = registry().lock().unwrap();
    let Some(app_ref) = registry.apps.get_mut(&app) else {
        return false;
    };
    dispatch_native_key(app_ref, key)
}

#[no_mangle]
pub extern "C" fn vugra_app_dispatch_native_text(
    app: VugraAppHandle,
    ptr: *const u8,
    len: usize,
) -> bool {
    if ptr.is_null() {
        return false;
    }
    let text = unsafe {
        let bytes = std::slice::from_raw_parts(ptr, len);
        String::from_utf8_lossy(bytes).to_string()
    };
    let mut registry = registry().lock().unwrap();
    let Some(app_ref) = registry.apps.get_mut(&app) else {
        return false;
    };
    dispatch_native_text(app_ref, text)
}

fn native_frame_view(frame: &NativeFrame) -> VugraNativeFrameView {
    VugraNativeFrameView {
        commands_len: frame.commands.len() as u64,
        pixels_ptr: frame.pixels.as_ptr() as u64,
        pixels_len: frame.pixels.len() as u64,
        drawn_pixels: frame
            .pixels
            .iter()
            .filter(|pixel| **pixel != 0x00f7f7f8)
            .count() as u64,
    }
}

fn native_backend_from_abi(value: u32) -> Option<NativeRenderBackend> {
    match value {
        VUGRA_NATIVE_BACKEND_SOFTWARE => Some(NativeRenderBackend::Software),
        VUGRA_NATIVE_BACKEND_VELLO => Some(NativeRenderBackend::Vello),
        VUGRA_NATIVE_BACKEND_WGPU => Some(NativeRenderBackend::Wgpu),
        _ => None,
    }
}

fn native_key_from_abi(value: u32) -> Option<NativeKey> {
    match value {
        VUGRA_NATIVE_KEY_ARROW_DOWN => Some(NativeKey::ArrowDown),
        VUGRA_NATIVE_KEY_ARROW_UP => Some(NativeKey::ArrowUp),
        VUGRA_NATIVE_KEY_ENTER => Some(NativeKey::Enter),
        VUGRA_NATIVE_KEY_BACKSPACE => Some(NativeKey::Backspace),
        VUGRA_NATIVE_KEY_ESCAPE => Some(NativeKey::Escape),
        VUGRA_NATIVE_KEY_DELETE => Some(NativeKey::Delete),
        VUGRA_NATIVE_KEY_SELECT_ALL => Some(NativeKey::SelectAll),
        _ => None,
    }
}

pub fn text_view_to_string(view: VugraTextView) -> String {
    if view.ptr == 0 || view.len == 0 {
        return String::new();
    }
    unsafe {
        let bytes = std::slice::from_raw_parts(view.ptr as *const u8, view.len as usize);
        String::from_utf8_lossy(bytes).to_string()
    }
}

pub fn set_string_signal(handle: VugraStateHandle, signal: VugraSignalId, value: &str) -> bool {
    vugra_state_set_string_signal(handle, signal, value.as_ptr(), value.len())
}

pub fn create_finder_lite_state() -> VugraStateHandle {
    let state = vugra_state_create();
    set_string_signal(state, 1, "Documents");
    set_string_signal(state, 2, "3 items · Current path: Documents");
    set_string_signal(state, 3, "0 items selected");
    set_string_signal(state, 13, "Documents");
    set_string_signal(state, 14, "Downloads");
    set_string_signal(state, 15, "Pictures");
    set_bool_signal(state, 16, true);
    set_bool_signal(state, 17, false);
    set_bool_signal(state, 18, false);
    set_string_signal(state, 19, "");
    set_string_signal(state, 92, "Favorites");
    set_string_signal(state, 93, "Workspace");
    set_bool_signal(state, 94, true);
    set_bool_signal(state, 95, true);
    set_string_signal(state, 96, "Current Project");
    set_string_signal(state, 97, "Parent Folder");
    set_bool_signal(state, 98, false);
    set_bool_signal(state, 99, false);
    set_bool_signal(state, 100, false);
    set_bool_signal(state, 101, false);
    set_string_signal(state, 102, "");
    set_bool_signal(state, 103, false);
    set_string_signal(state, 104, "");
    set_string_signal(state, 105, "");
    set_string_signal(state, 106, "sidebar");
    set_string_signal(state, 107, "splitter");
    set_string_signal(state, 20, "Design");
    set_string_signal(state, 21, "folder");
    set_string_signal(state, 22, "--");
    set_string_signal(state, 23, "--");
    set_string_signal(state, 24, "file-row");
    set_bool_signal(state, 25, false);
    set_string_signal(state, 26, "Roadmap.md");
    set_string_signal(state, 27, "file");
    set_string_signal(state, 28, "--");
    set_string_signal(state, 29, "12 KB");
    set_string_signal(state, 30, "file-row");
    set_bool_signal(state, 31, false);
    set_string_signal(state, 32, "Budget 2026.xlsx");
    set_string_signal(state, 33, "file");
    set_string_signal(state, 34, "--");
    set_string_signal(state, 35, "842 KB");
    set_string_signal(state, 36, "file-row");
    set_bool_signal(state, 37, false);
    set_string_signal(state, 200, "Design");
    set_string_signal(state, 201, "folder");
    set_string_signal(state, 202, "--");
    set_string_signal(state, 203, "--");
    set_string_signal(state, 204, "Roadmap.md");
    set_string_signal(state, 205, "file");
    set_string_signal(state, 206, "--");
    set_string_signal(state, 207, "12 KB");
    set_string_signal(state, 208, "Budget 2026.xlsx");
    set_string_signal(state, 209, "file");
    set_string_signal(state, 210, "--");
    set_string_signal(state, 211, "842 KB");
    {
        let mut registry = registry().lock().unwrap();
        if let Some(state) = registry.states.get_mut(&state) {
            state.calls.insert(MethodId(2), |values| {
                select_index(values, 0);
            });
            state.calls.insert(MethodId(3), |values| {
                select_index(values, 1);
            });
            state.calls.insert(MethodId(4), |values| {
                select_index(values, 2);
            });
            state.calls.insert(MethodId(5), |values| {
                set_location(values, "Documents", rows_for_path("Documents"));
                set_sidebar_active(values, Some("Documents"));
            });
            state.calls.insert(MethodId(6), |values| {
                set_location(values, "Downloads", rows_for_path("Downloads"));
                set_sidebar_active(values, Some("Downloads"));
            });
            state.calls.insert(MethodId(7), |values| {
                set_location(values, "Pictures", rows_for_path("Pictures"));
                set_sidebar_active(values, Some("Pictures"));
            });
            state.calls.insert(MethodId(8), |values| {
                select_delta(values, -1);
            });
            state.calls.insert(MethodId(9), |values| {
                select_delta(values, 1);
            });
            state.calls.insert(MethodId(11), |values| {
                let mut query = values
                    .get(&SignalId(19))
                    .map(Value::as_text)
                    .unwrap_or_default();
                query.pop();
                values.insert(SignalId(19), query.into());
                project_rows(values);
            });
            state.calls.insert(MethodId(12), |values| {
                values.insert(SignalId(19), String::new().into());
                project_rows(values);
            });
            state.calls.insert(MethodId(13), |values| {
                open_selected(values);
            });
            state.calls.insert(MethodId(14), |values| {
                open_parent(values);
            });
            state.calls.insert(MethodId(15), |values| {
                let next = !matches!(values.get(&SignalId(94)), Some(Value::Bool(true)));
                values.insert(SignalId(94), Value::Bool(next));
            });
            state.calls.insert(MethodId(16), |values| {
                let next = !matches!(values.get(&SignalId(95)), Some(Value::Bool(true)));
                values.insert(SignalId(95), Value::Bool(next));
            });
            state.calls.insert(MethodId(17), |values| {
                set_location(values, "Project A", rows_for_path("Project A"));
                set_sidebar_active(values, Some("Project A"));
            });
            state.calls.insert(MethodId(18), |values| {
                set_location(values, "Project B", rows_for_path("Project B"));
                set_sidebar_active(values, Some("Project B"));
            });
            state.calls.insert(MethodId(19), dismiss_overlay);
            state.event_calls.insert(MethodId(10), |values, event| {
                let mut query = values
                    .get(&SignalId(19))
                    .map(Value::as_text)
                    .unwrap_or_default();
                query.push_str(&event.text);
                values.insert(SignalId(19), query.into());
                project_rows(values);
            });
            state
                .calls
                .insert(MethodId(24), |values| select_index(values, 3));
            state
                .calls
                .insert(MethodId(25), |values| select_index(values, 4));
            state
                .calls
                .insert(MethodId(26), |values| select_index(values, 5));
            state
                .calls
                .insert(MethodId(27), |values| select_index(values, 6));
            state
                .calls
                .insert(MethodId(28), |values| select_index(values, 7));
            state
                .calls
                .insert(MethodId(29), |values| select_index(values, 8));
            state
                .calls
                .insert(MethodId(30), |values| select_index(values, 9));
            state
                .calls
                .insert(MethodId(31), |values| select_index(values, 10));
            state
                .calls
                .insert(MethodId(32), |values| select_index(values, 11));
            state.calls.insert(MethodId(33), begin_rename);
            state.calls.insert(MethodId(34), cancel_rename);
            state.calls.insert(MethodId(35), commit_rename);
            state.calls.insert(MethodId(36), delete_selected);
            state.calls.insert(MethodId(37), duplicate_selected);
            state.calls.insert(MethodId(38), new_folder);
            state.calls.insert(MethodId(39), show_blank_menu);
            state.calls.insert(MethodId(40), close_preview);
            state.calls.insert(MethodId(80), select_all);
            state.calls.insert(MethodId(81), hover_splitter);
            state.event_calls.insert(MethodId(82), |values, event| {
                resize_sidebar(values, event.delta_x);
            });
            state
                .calls
                .insert(MethodId(41), |values| show_row_menu(values, 0));
            state
                .calls
                .insert(MethodId(42), |values| show_row_menu(values, 1));
            state
                .calls
                .insert(MethodId(43), |values| show_row_menu(values, 2));
            state
                .calls
                .insert(MethodId(44), |values| show_row_menu(values, 3));
            state
                .calls
                .insert(MethodId(45), |values| show_row_menu(values, 4));
            state
                .calls
                .insert(MethodId(46), |values| show_row_menu(values, 5));
            state
                .calls
                .insert(MethodId(47), |values| show_row_menu(values, 6));
            state
                .calls
                .insert(MethodId(48), |values| show_row_menu(values, 7));
            state
                .calls
                .insert(MethodId(49), |values| show_row_menu(values, 8));
            state
                .calls
                .insert(MethodId(50), |values| show_row_menu(values, 9));
            state
                .calls
                .insert(MethodId(51), |values| show_row_menu(values, 10));
            state
                .calls
                .insert(MethodId(52), |values| show_row_menu(values, 11));
            state
                .calls
                .insert(MethodId(53), |values| hover_row(values, 0));
            state
                .calls
                .insert(MethodId(54), |values| hover_row(values, 1));
            state
                .calls
                .insert(MethodId(55), |values| hover_row(values, 2));
            state
                .calls
                .insert(MethodId(56), |values| hover_row(values, 3));
            state
                .calls
                .insert(MethodId(57), |values| hover_row(values, 4));
            state
                .calls
                .insert(MethodId(58), |values| hover_row(values, 5));
            state
                .calls
                .insert(MethodId(59), |values| hover_row(values, 6));
            state
                .calls
                .insert(MethodId(60), |values| hover_row(values, 7));
            state
                .calls
                .insert(MethodId(61), |values| hover_row(values, 8));
            state
                .calls
                .insert(MethodId(62), |values| hover_row(values, 9));
            state
                .calls
                .insert(MethodId(63), |values| hover_row(values, 10));
            state
                .calls
                .insert(MethodId(64), |values| hover_row(values, 11));
            state
                .calls
                .insert(MethodId(82), |values| resize_sidebar(values, 0.0));
        }
    }
    state
}

fn set_bool_signal(handle: VugraStateHandle, signal: VugraSignalId, value: bool) -> bool {
    vugra_state_set_signal(
        handle,
        signal,
        VugraValue {
            kind: VUGRA_VALUE_BOOL,
            flags: u32::from(value),
            ..VugraValue::default()
        },
    )
}

fn set_location(values: &mut HashMap<SignalId, Value>, path: &str, rows: Vec<FinderAbiRow>) {
    values.insert(SignalId(1), path.into());
    values.insert(
        SignalId(2),
        format!("{} items · Current path: {path}", rows.len()).into(),
    );
    values.insert(SignalId(3), "0 items selected".into());
    values.insert(SignalId(19), String::new().into());
    set_index_signal(values, FOCUS_SIGNAL, None);
    set_index_signal(values, HOVER_SIGNAL, None);
    for index in 0..24 {
        let source = source_signal_base(index);
        if let Some(row) = rows.get(index) {
            values.insert(SignalId(source), row.name.into());
            values.insert(SignalId(source + 1), row.kind.into());
            values.insert(SignalId(source + 2), row.modified.into());
            values.insert(SignalId(source + 3), row.size.into());
        } else {
            values.insert(SignalId(source), Value::None);
            values.insert(SignalId(source + 1), Value::None);
            values.insert(SignalId(source + 2), Value::None);
            values.insert(SignalId(source + 3), Value::None);
        }
    }
    project_rows(values);
    clear_selection(values);
}

fn set_sidebar_active(values: &mut HashMap<SignalId, Value>, location: Option<&str>) {
    values.insert(SignalId(16), Value::Bool(location == Some("Documents")));
    values.insert(SignalId(17), Value::Bool(location == Some("Downloads")));
    values.insert(SignalId(18), Value::Bool(location == Some("Pictures")));
    values.insert(SignalId(98), Value::Bool(location == Some("Project A")));
    values.insert(SignalId(99), Value::Bool(location == Some("Project B")));
}

fn project_rows(values: &mut HashMap<SignalId, Value>) {
    let query = values
        .get(&SignalId(19))
        .map(Value::as_text)
        .unwrap_or_default()
        .to_lowercase();
    let mut rows: Vec<(String, String, String, String)> = Vec::new();
    for index in 0..24 {
        let source = source_signal_base(index);
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
        if query.is_empty()
            || name.to_lowercase().contains(&query)
            || kind.to_lowercase().contains(&query)
        {
            rows.push((name, kind, modified, size));
        }
    }
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
        let base = row_signal_base(index);
        if let Some((name, kind, modified, size)) = rows.get(index) {
            values.insert(SignalId(base), name.clone().into());
            values.insert(SignalId(base + 1), kind.clone().into());
            values.insert(SignalId(base + 2), modified.clone().into());
            values.insert(SignalId(base + 3), size.clone().into());
        } else {
            values.insert(SignalId(base), Value::None);
            values.insert(SignalId(base + 1), Value::None);
            values.insert(SignalId(base + 2), Value::None);
            values.insert(SignalId(base + 3), Value::None);
        }
    }
    if let Some(selected) = selected_index(values) {
        set_selected_index(values, selected, false);
    } else {
        clear_selection(values);
    }
}

fn open_selected(values: &mut HashMap<SignalId, Value>) {
    let selected_index = index_signal(values, FOCUS_SIGNAL)
        .or_else(|| selected_index(values))
        .unwrap_or(0);
    let base = row_signal_base(selected_index);
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
        set_location(values, &next_path, folder_rows(&name));
        set_sidebar_active(values, sidebar_location(&next_path));
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

fn open_parent(values: &mut HashMap<SignalId, Value>) {
    let path = values
        .get(&SignalId(1))
        .map(Value::as_text)
        .unwrap_or_default();
    let Some((parent, _child)) = path.rsplit_once('/') else {
        return;
    };
    let rows = rows_for_path(parent);
    set_location(values, parent, rows);
    set_sidebar_active(values, sidebar_location(parent));
}

fn dismiss_overlay(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(100), Value::Bool(false));
    values.insert(SignalId(101), Value::Bool(false));
}

fn show_blank_menu(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(100), Value::Bool(false));
    values.insert(SignalId(101), Value::Bool(true));
    values.insert(SignalId(103), Value::Bool(false));
}

fn show_row_menu(values: &mut HashMap<SignalId, Value>, index: usize) {
    if index >= visible_len(values).min(12) {
        return;
    }
    select_index(values, index);
    values.insert(SignalId(100), Value::Bool(true));
    values.insert(SignalId(101), Value::Bool(false));
    values.insert(SignalId(103), Value::Bool(false));
}

fn hover_row(values: &mut HashMap<SignalId, Value>, index: usize) {
    if index >= visible_len(values).min(12) {
        return;
    }
    set_index_signal(values, HOVER_SIGNAL, Some(index));
    project_visual_state(values);
}

fn close_preview(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(103), Value::Bool(false));
}

fn begin_rename(values: &mut HashMap<SignalId, Value>) {
    let Some(selected) = selected_index(values) else {
        return;
    };
    let base = row_signal_base(selected);
    let name = values
        .get(&SignalId(base))
        .map(Value::as_text)
        .unwrap_or_default();
    if name.is_empty() {
        return;
    }
    values.insert(SignalId(102), name.into());
    dismiss_overlay(values);
    project_visual_state(values);
}

fn cancel_rename(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(102), String::new().into());
    project_visual_state(values);
}

fn commit_rename(values: &mut HashMap<SignalId, Value>) {
    let Some(selected) = selected_index(values) else {
        return;
    };
    let base = row_signal_base(selected);
    let name = values
        .get(&SignalId(102))
        .map(Value::as_text)
        .unwrap_or_default()
        .trim()
        .to_string();
    if name.is_empty() {
        return;
    }
    values.insert(SignalId(base), name.clone().into());
    let source = source_signal_base(selected);
    values.insert(SignalId(source), name.into());
    cancel_rename(values);
    select_index(values, selected);
}

fn delete_selected(values: &mut HashMap<SignalId, Value>) {
    let Some(selected) = selected_index(values) else {
        return;
    };
    for index in selected..23 {
        let next = source_signal_base(index + 1);
        let source = source_signal_base(index);
        for field in 0..4 {
            let value = values
                .get(&SignalId(next + field))
                .cloned()
                .unwrap_or(Value::None);
            values.insert(SignalId(source + field), value);
        }
    }
    let last = source_signal_base(23);
    for field in 0..4 {
        values.insert(SignalId(last + field), Value::None);
    }
    dismiss_overlay(values);
    project_rows(values);
    clear_selection(values);
}

fn duplicate_selected(values: &mut HashMap<SignalId, Value>) {
    let Some(selected) = selected_index(values) else {
        return;
    };
    let base = row_signal_base(selected);
    let name = values
        .get(&SignalId(base))
        .map(Value::as_text)
        .unwrap_or_default();
    if name.is_empty() {
        return;
    }
    let kind = values
        .get(&SignalId(base + 1))
        .map(Value::as_text)
        .unwrap_or_else(|| "file".to_string());
    let modified = values
        .get(&SignalId(base + 2))
        .map(Value::as_text)
        .unwrap_or_else(|| "--".to_string());
    let size = values
        .get(&SignalId(base + 3))
        .map(Value::as_text)
        .unwrap_or_else(|| "--".to_string());
    let inserted = append_source_row(values, &(name + " copy"), &kind, &modified, &size);
    dismiss_overlay(values);
    project_rows(values);
    if let Some(inserted) = inserted {
        if inserted < visible_len(values).min(12) {
            select_index(values, inserted);
        } else {
            values.insert(SignalId(3), "1 items selected".into());
            set_index_signal(values, FOCUS_SIGNAL, Some(inserted));
            clear_visible_selection(values);
        }
    }
}

fn new_folder(values: &mut HashMap<SignalId, Value>) {
    let inserted = append_source_row(values, "Untitled Folder", "folder", "--", "--");
    dismiss_overlay(values);
    project_rows(values);
    if let Some(inserted) = inserted {
        if inserted < visible_len(values).min(12) {
            select_index(values, inserted);
        } else {
            values.insert(SignalId(3), "1 items selected".into());
            set_index_signal(values, FOCUS_SIGNAL, Some(inserted));
            clear_visible_selection(values);
        }
    }
}

fn append_source_row(
    values: &mut HashMap<SignalId, Value>,
    name: &str,
    kind: &str,
    modified: &str,
    size: &str,
) -> Option<usize> {
    let Some(index) = (0..24).find(|index| {
        values
            .get(&SignalId(source_signal_base(*index)))
            .map(Value::as_text)
            .unwrap_or_default()
            .is_empty()
    }) else {
        return None;
    };
    let source = source_signal_base(index);
    values.insert(SignalId(source), name.to_string().into());
    values.insert(SignalId(source + 1), kind.to_string().into());
    values.insert(SignalId(source + 2), modified.to_string().into());
    values.insert(SignalId(source + 3), size.to_string().into());
    Some(index)
}

fn sidebar_location(path: &str) -> Option<&'static str> {
    [
        "Documents",
        "Downloads",
        "Pictures",
        "Project A",
        "Project B",
    ]
    .into_iter()
    .find(|base| path == *base || path.strip_prefix(&format!("{base}/")).is_some())
}

#[derive(Clone)]
struct FinderAbiRow {
    name: &'static str,
    kind: &'static str,
    modified: &'static str,
    size: &'static str,
}

fn abi_folder(name: &'static str) -> FinderAbiRow {
    FinderAbiRow {
        name,
        kind: "folder",
        modified: "--",
        size: "--",
    }
}

fn abi_file(name: &'static str, size: &'static str) -> FinderAbiRow {
    FinderAbiRow {
        name,
        kind: "file",
        modified: "--",
        size,
    }
}

fn folder_rows(folder: &str) -> Vec<FinderAbiRow> {
    match folder {
        "Design" => vec![
            abi_file("Components.sketch", "1.9 MB"),
            abi_folder("Assets"),
            abi_file("Prototype.mov", "4.8 MB"),
        ],
        "Receipts" => vec![
            abi_file("May.pdf", "210 KB"),
            abi_file("April.pdf", "198 KB"),
            abi_folder("Archive"),
        ],
        "Screenshots" => vec![
            abi_file("Desktop.png", "1.1 MB"),
            abi_file("Window.png", "860 KB"),
            abi_folder("Exports"),
        ],
        "Assets" => vec![
            abi_file("Icon.png", "56 KB"),
            abi_file("Toolbar.png", "74 KB"),
        ],
        "Archive" => vec![
            abi_file("2025.pdf", "184 KB"),
            abi_file("2024.pdf", "193 KB"),
        ],
        "Exports" => vec![
            abi_file("Header.png", "64 KB"),
            abi_file("Sidebar.png", "67 KB"),
        ],
        _ => vec![abi_file("Readme.md", "0 B")],
    }
}

fn rows_for_path(path: &str) -> Vec<FinderAbiRow> {
    match path {
        "Documents" => vec![
            abi_folder("Design"),
            abi_file("Roadmap.md", "12 KB"),
            abi_file("Budget 2026.xlsx", "842 KB"),
        ],
        "Downloads" => vec![
            abi_file("Installer.dmg", "3.4 MB"),
            abi_folder("Receipts"),
            abi_file("Archive.zip", "721 KB"),
        ],
        "Pictures" => vec![
            abi_file("Vacation.jpg", "2.1 MB"),
            abi_folder("Screenshots"),
            abi_file("Profile.png", "98 KB"),
        ],
        "Project A" => vec![
            abi_file("Cargo.toml", "4 KB"),
            abi_folder("crates"),
            abi_folder("examples"),
        ],
        "Project B" => vec![abi_folder("vugra"), abi_file("README.md", "8 KB")],
        _ => path
            .rsplit_once('/')
            .map(|(_, folder)| folder_rows(folder))
            .unwrap_or_default(),
    }
}

fn select_delta(values: &mut HashMap<SignalId, Value>, delta: isize) {
    let Some(current) = index_signal(values, FOCUS_SIGNAL).or_else(|| selected_index(values))
    else {
        select_index(values, 0);
        return;
    };
    let visible_len = visible_len(values);
    let next = (current as isize + delta).clamp(0, visible_len.saturating_sub(1) as isize) as usize;
    select_index(values, next);
}

fn select_index(values: &mut HashMap<SignalId, Value>, selected: usize) {
    set_selected_index(values, selected, true);
}

fn set_selected_index(values: &mut HashMap<SignalId, Value>, selected: usize, update_focus: bool) {
    let visible_len = visible_len(values);
    values.insert(
        SignalId(3),
        if visible_len == 0 {
            "0 items selected"
        } else {
            "1 items selected"
        }
        .into(),
    );
    if visible_len == 0 {
        set_index_signal(values, FOCUS_SIGNAL, None);
        for index in 0..12 {
            let base = row_signal_base(index);
            values.insert(SignalId(base + 5), Value::None);
        }
        project_visual_state(values);
        return;
    }
    let selected = selected.min(visible_len.min(12).saturating_sub(1));
    if update_focus {
        set_index_signal(values, FOCUS_SIGNAL, (visible_len > 0).then_some(selected));
    }
    for index in 0..12 {
        let base = row_signal_base(index);
        let is_selected = visible_len > index && selected == index;
        if visible_len > index {
            values.insert(SignalId(base + 5), Value::Bool(is_selected));
        } else {
            values.insert(SignalId(base + 5), Value::None);
        }
    }
    project_visual_state(values);
}

fn clear_selection(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(3), "0 items selected".into());
    set_index_signal(values, FOCUS_SIGNAL, None);
    clear_visible_selection(values);
}

fn select_all(values: &mut HashMap<SignalId, Value>) {
    let visible_len = visible_len(values).min(12);
    values.insert(SignalId(3), format!("{visible_len} items selected").into());
    if visible_len == 0 {
        set_index_signal(values, FOCUS_SIGNAL, None);
    } else {
        set_index_signal(values, FOCUS_SIGNAL, Some(0));
    }
    for index in 0..12 {
        let base = row_signal_base(index);
        if index < visible_len {
            values.insert(SignalId(base + 5), Value::Bool(true));
        } else {
            values.insert(SignalId(base + 5), Value::None);
        }
    }
    project_visual_state(values);
}

fn clear_visible_selection(values: &mut HashMap<SignalId, Value>) {
    for index in 0..12 {
        let base = row_signal_base(index);
        if index < visible_len(values).min(12) {
            values.insert(SignalId(base + 5), Value::Bool(false));
        } else {
            values.insert(SignalId(base + 5), Value::None);
        }
    }
    project_visual_state(values);
}

fn hover_splitter(values: &mut HashMap<SignalId, Value>) {
    values.insert(SignalId(107), "splitter-hover".into());
}

fn resize_sidebar(values: &mut HashMap<SignalId, Value>, delta_x: f32) {
    let mut mode = index_signal(values, SIDEBAR_MODE_SIGNAL).unwrap_or(0);
    if delta_x < -8.0 && mode > 1 {
        mode -= 1;
    } else if delta_x < -8.0 && mode == 1 {
        mode = 0;
    } else if delta_x > 8.0 && mode < 3 {
        mode += 1;
    }
    set_index_signal(values, SIDEBAR_MODE_SIGNAL, Some(mode));
    values.insert(
        SignalId(106),
        ["sidebar", "sidebar-200", "sidebar-280", "sidebar-320"][mode].into(),
    );
    values.insert(SignalId(107), "splitter".into());
}

fn selected_index(values: &HashMap<SignalId, Value>) -> Option<usize> {
    (0..12).find(|index| {
        matches!(
            values.get(&SignalId(row_signal_base(*index) + 5)),
            Some(Value::Bool(true))
        )
    })
}

fn visible_len(values: &HashMap<SignalId, Value>) -> usize {
    (0..12)
        .filter(|index| {
            values
                .get(&SignalId(row_signal_base(*index)))
                .map(Value::as_text)
                .is_some_and(|value| !value.is_empty())
        })
        .count()
}

const FOCUS_SIGNAL: SignalId = SignalId(190);
const HOVER_SIGNAL: SignalId = SignalId(191);
const SIDEBAR_MODE_SIGNAL: SignalId = SignalId(194);

fn set_index_signal(values: &mut HashMap<SignalId, Value>, signal: SignalId, index: Option<usize>) {
    values.insert(
        signal,
        index
            .map(|index| Value::Number(index as f64))
            .unwrap_or(Value::Number(-1.0)),
    );
}

fn index_signal(values: &HashMap<SignalId, Value>, signal: SignalId) -> Option<usize> {
    match values.get(&signal) {
        Some(Value::Number(value)) if value.is_finite() && *value >= 0.0 => Some(*value as usize),
        _ => None,
    }
}

fn project_visual_state(values: &mut HashMap<SignalId, Value>) {
    let visible_len = visible_len(values).min(12);
    let selected = selected_index(values);
    let focus = index_signal(values, FOCUS_SIGNAL);
    let hover = index_signal(values, HOVER_SIGNAL);
    let editing = if values
        .get(&SignalId(102))
        .map(Value::as_text)
        .unwrap_or_default()
        .is_empty()
    {
        None
    } else {
        selected
    };
    for index in 0..12 {
        let base = row_signal_base(index);
        if index >= visible_len {
            values.insert(SignalId(base + 4), Value::None);
            continue;
        }
        let class = if editing == Some(index) {
            "file-row-editing"
        } else if selected == Some(index) {
            "file-row-selected"
        } else if focus == Some(index) {
            "file-row-focus"
        } else if hover == Some(index) {
            "file-row-hover"
        } else {
            "file-row"
        };
        values.insert(SignalId(base + 4), class.into());
    }
}

fn row_signal_base(index: usize) -> u32 {
    20 + index as u32 * 6
}

fn source_signal_base(index: usize) -> u32 {
    200 + index as u32 * 4
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn abi_state_satisfies_component_contract() {
        let mut state = AbiState::new()
            .with_signal(SignalId(1), "Documents")
            .with_method(MethodId(1), |values| {
                values.insert(SignalId(1), "Home".into());
            });
        assert_eq!(state.get_signal(SignalId(1)), Value::from("Documents"));
        state.call_method(MethodId(1));
        assert_eq!(state.get_signal(SignalId(1)), Value::from("Home"));
    }

    #[test]
    fn handle_abi_renders_finder_lite_without_exposing_rust_objects() {
        let component = vugra_component_finder_lite();
        let state = create_finder_lite_state();
        let app = vugra_app_create(component, state);

        assert_ne!(component, 0);
        assert_ne!(state, 0);
        assert_ne!(app, 0);
        assert!(vugra_app_call_method(app, 2));

        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("FinderLite"));
        assert!(text.contains("Documents"));
        assert!(text.contains("1 items selected"));
        assert!(text.contains("* Design  --  --"));
        assert!(text.contains("- Roadmap.md  --  12 KB"));

        assert!(vugra_app_call_method(app, 6));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("Downloads"));
        assert!(text.contains("- Installer.dmg  --  3.4 MB"));
        assert!(vugra_app_call_method(app, 2));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Installer.dmg  --  3.4 MB"));
        assert!(vugra_app_call_method(app, 9));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Receipts  --  --"));
        assert!(vugra_app_call_text_method(app, 10, "zip".as_ptr(), 3));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Archive.zip  --  721 KB"));
        assert!(!text.contains("Installer.dmg"));
        assert!(vugra_app_call_method(app, 11));
        assert!(vugra_app_call_method(app, 11));
        assert!(vugra_app_call_method(app, 11));
        assert!(vugra_app_call_method(app, 3));
        assert!(vugra_app_call_method(app, 13));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("Downloads/Receipts"));
        assert!(text.contains("- May.pdf  --  210 KB"));
        assert!(vugra_app_call_method(app, 14));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("Downloads"));
        assert!(text.contains("- Installer.dmg  --  3.4 MB"));
        assert!(text.contains("- Receipts  --  --"));

        vugra_app_destroy(app);
        vugra_state_destroy(state);
        vugra_component_destroy(component);
    }

    #[test]
    fn handle_abi_renders_and_dispatches_native_frame() {
        let component = vugra_component_finder_lite();
        let state = create_finder_lite_state();
        let app = vugra_app_create(component, state);

        let frame = vugra_app_render_native_frame(app, VUGRA_NATIVE_BACKEND_SOFTWARE, 800, 600);
        assert_eq!(frame.pixels_len, 800 * 600);
        assert!(frame.commands_len > 0);
        assert!(frame.drawn_pixels > 0);
        assert_ne!(frame.pixels_ptr, 0);

        assert!(vugra_app_dispatch_native_pointer(app, 260.0, 134.0));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Roadmap.md  --  12 KB"));

        assert!(vugra_app_dispatch_native_key(
            app,
            VUGRA_NATIVE_KEY_ARROW_DOWN
        ));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Budget 2026.xlsx  --  842 KB"));

        assert!(vugra_app_dispatch_native_text(app, "road".as_ptr(), 4));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("* Roadmap.md  --  12 KB"));
        assert!(!text.contains("Budget 2026.xlsx"));

        assert!(vugra_app_dispatch_native_key(
            app,
            VUGRA_NATIVE_KEY_BACKSPACE
        ));
        assert!(vugra_app_dispatch_native_key(
            app,
            VUGRA_NATIVE_KEY_BACKSPACE
        ));
        assert!(vugra_app_dispatch_native_key(
            app,
            VUGRA_NATIVE_KEY_BACKSPACE
        ));
        assert!(vugra_app_dispatch_native_key(
            app,
            VUGRA_NATIVE_KEY_BACKSPACE
        ));
        assert!(vugra_app_call_method(app, 3));
        assert!(vugra_app_dispatch_native_key(app, VUGRA_NATIVE_KEY_ENTER));
        let text = text_view_to_string(vugra_app_render_text(app, 800.0, 600.0));
        assert!(text.contains("Roadmap.md"));
        assert!(text.contains("* Roadmap.md  --  12 KB"));

        let frame = vugra_app_render_native_frame(app, VUGRA_NATIVE_BACKEND_VELLO, 800, 600);
        assert_eq!(frame.pixels_len, 800 * 600);
        assert!(frame.drawn_pixels > 0);
        let frame = vugra_app_render_native_frame(app, VUGRA_NATIVE_BACKEND_WGPU, 800, 600);
        assert_eq!(frame.pixels_len, 800 * 600);
        assert!(frame.drawn_pixels > 0);

        vugra_app_destroy(app);
        vugra_state_destroy(state);
        vugra_component_destroy(component);
    }

    #[test]
    #[ignore = "opens a native window; run through finder-rust native-window-smoke on the main thread"]
    fn handle_abi_runs_short_lived_native_window() {
        let component = vugra_component_finder_lite();
        let state = create_finder_lite_state();
        let app = vugra_app_create(component, state);

        let smoke =
            vugra_app_run_native_window_for_frames(app, VUGRA_NATIVE_BACKEND_SOFTWARE, 320, 240, 1);
        assert!(smoke.ok);
        assert_eq!(smoke.frames_presented, 1);
        assert!(smoke.commands_len > 0);
        assert_eq!(smoke.pixels_len, 320 * 240);
        assert!(smoke.drawn_pixels > 0);

        vugra_app_destroy(app);
        vugra_state_destroy(state);
        vugra_component_destroy(component);
    }
}
