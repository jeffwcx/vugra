//! Go binding support contracts over the stable Vugra ABI.

use vugra_abi::{
    create_finder_lite_state, set_string_signal, text_view_to_string, vugra_app_call_method,
    vugra_app_call_text_method, vugra_app_create, vugra_app_destroy, vugra_app_dispatch_native_key,
    vugra_app_dispatch_native_pointer, vugra_app_dispatch_native_text,
    vugra_app_render_native_frame, vugra_app_render_text, vugra_app_run_native_window_for_frames,
    vugra_component_destroy, vugra_component_finder_lite, vugra_state_destroy, VugraAppHandle,
    VugraComponentHandle, VugraMethodId, VugraNativeFrameView, VugraNativeWindowSmoke,
    VugraSignalId, VugraStateHandle, VUGRA_NATIVE_BACKEND_SOFTWARE, VUGRA_NATIVE_BACKEND_VELLO,
    VUGRA_NATIVE_BACKEND_WGPU, VUGRA_NATIVE_KEY_ARROW_DOWN, VUGRA_NATIVE_KEY_ARROW_UP,
    VUGRA_NATIVE_KEY_BACKSPACE, VUGRA_NATIVE_KEY_DELETE, VUGRA_NATIVE_KEY_ENTER,
    VUGRA_NATIVE_KEY_ESCAPE, VUGRA_NATIVE_KEY_SELECT_ALL,
};

pub struct GoComponent {
    handle: VugraComponentHandle,
}

impl GoComponent {
    pub fn finder_lite() -> Self {
        Self {
            handle: vugra_component_finder_lite(),
        }
    }

    pub fn handle(&self) -> VugraComponentHandle {
        self.handle
    }
}

impl Drop for GoComponent {
    fn drop(&mut self) {
        if self.handle != 0 {
            vugra_component_destroy(self.handle);
        }
    }
}

pub struct GoState {
    handle: VugraStateHandle,
}

impl GoState {
    pub fn finder_lite() -> Self {
        Self {
            handle: create_finder_lite_state(),
        }
    }

    pub fn set_string(&self, signal: VugraSignalId, value: &str) -> bool {
        set_string_signal(self.handle, signal, value)
    }

    pub fn handle(&self) -> VugraStateHandle {
        self.handle
    }
}

impl Drop for GoState {
    fn drop(&mut self) {
        if self.handle != 0 {
            vugra_state_destroy(self.handle);
        }
    }
}

pub struct GoApp {
    handle: VugraAppHandle,
}

impl GoApp {
    pub fn mount(component: &GoComponent, state: &GoState) -> Option<Self> {
        let handle = vugra_app_create(component.handle(), state.handle());
        if handle == 0 {
            return None;
        }
        Some(Self { handle })
    }

    pub fn call_method(&self, method: VugraMethodId) -> bool {
        vugra_app_call_method(self.handle, method)
    }

    pub fn call_text_method(&self, method: VugraMethodId, text: &str) -> bool {
        vugra_app_call_text_method(self.handle, method, text.as_ptr(), text.len())
    }

    pub fn render_text(&self, width: f32, height: f32) -> String {
        text_view_to_string(vugra_app_render_text(self.handle, width, height))
    }

    pub fn render_native_frame(
        &self,
        backend: GoNativeBackend,
        width: u32,
        height: u32,
    ) -> VugraNativeFrameView {
        vugra_app_render_native_frame(self.handle, backend as u32, width, height)
    }

    pub fn run_native_window_for_frames(
        &self,
        backend: GoNativeBackend,
        width: u32,
        height: u32,
        frames: u32,
    ) -> VugraNativeWindowSmoke {
        vugra_app_run_native_window_for_frames(self.handle, backend as u32, width, height, frames)
    }

    pub fn dispatch_native_pointer(&self, x: f32, y: f32) -> bool {
        vugra_app_dispatch_native_pointer(self.handle, x, y)
    }

    pub fn dispatch_native_key(&self, key: GoNativeKey) -> bool {
        vugra_app_dispatch_native_key(self.handle, key as u32)
    }

    pub fn dispatch_native_text(&self, text: &str) -> bool {
        vugra_app_dispatch_native_text(self.handle, text.as_ptr(), text.len())
    }

    pub fn handle(&self) -> VugraAppHandle {
        self.handle
    }
}

#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GoNativeBackend {
    Software = VUGRA_NATIVE_BACKEND_SOFTWARE,
    Vello = VUGRA_NATIVE_BACKEND_VELLO,
    Wgpu = VUGRA_NATIVE_BACKEND_WGPU,
}

#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GoNativeKey {
    ArrowDown = VUGRA_NATIVE_KEY_ARROW_DOWN,
    ArrowUp = VUGRA_NATIVE_KEY_ARROW_UP,
    Enter = VUGRA_NATIVE_KEY_ENTER,
    Backspace = VUGRA_NATIVE_KEY_BACKSPACE,
    Escape = VUGRA_NATIVE_KEY_ESCAPE,
    Delete = VUGRA_NATIVE_KEY_DELETE,
    SelectAll = VUGRA_NATIVE_KEY_SELECT_ALL,
}

impl Drop for GoApp {
    fn drop(&mut self) {
        if self.handle != 0 {
            vugra_app_destroy(self.handle);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn go_shaped_wrappers_mount_finder_lite_through_abi() {
        let component = GoComponent::finder_lite();
        let state = GoState::finder_lite();
        assert!(state.set_string(3, "0 items selected"));

        let app = GoApp::mount(&component, &state).expect("mount Go ABI app");
        assert_ne!(app.handle(), 0);
        assert!(app.call_method(2));
        assert!(app.call_text_method(10, "road"));

        let output = app.render_text(800.0, 600.0);
        assert!(output.contains("FinderLite"));
        assert!(output.contains("Documents"));
        assert!(output.contains("1 items selected"));
        assert!(output.contains("* Roadmap.md  --  12 KB"));
        assert!(!output.contains("Budget 2026.xlsx"));

        assert!(app.call_method(12));
        assert!(app.call_method(2));
        assert!(app.call_method(13));
        let output = app.render_text(800.0, 600.0);
        assert!(output.contains("Documents/Design"));
        assert!(output.contains("- Components.sketch  --  1.9 MB"));

        assert!(app.call_method(14));
        let output = app.render_text(800.0, 600.0);
        assert!(output.contains("Documents"));
        assert!(output.contains("- Design  --  --"));
    }

    #[test]
    fn go_shaped_wrappers_render_and_dispatch_native_frame_through_abi() {
        let component = GoComponent::finder_lite();
        let state = GoState::finder_lite();
        let app = GoApp::mount(&component, &state).expect("mount Go ABI app");

        for backend in [
            GoNativeBackend::Software,
            GoNativeBackend::Vello,
            GoNativeBackend::Wgpu,
        ] {
            let frame = app.render_native_frame(backend, 800, 600);
            assert!(frame.commands_len > 0, "{backend:?} commands");
            assert_eq!(frame.pixels_len, 800 * 600, "{backend:?} pixels");
            assert!(frame.drawn_pixels > 0, "{backend:?} drawn");
            assert_ne!(frame.pixels_ptr, 0, "{backend:?} pixels ptr");
        }

        let _ = app.render_native_frame(GoNativeBackend::Software, 800, 600);
        assert!(app.dispatch_native_pointer(260.0, 134.0));
        assert!(app
            .render_text(800.0, 600.0)
            .contains("* Roadmap.md  --  12 KB"));

        assert!(app.dispatch_native_key(GoNativeKey::ArrowDown));
        assert!(app
            .render_text(800.0, 600.0)
            .contains("* Budget 2026.xlsx  --  842 KB"));

        assert!(app.dispatch_native_text("road"));
        let output = app.render_text(800.0, 600.0);
        assert!(output.contains("* Roadmap.md  --  12 KB"));
        assert!(!output.contains("Budget 2026.xlsx"));

        for _ in 0..4 {
            assert!(app.dispatch_native_key(GoNativeKey::Backspace));
        }
        assert!(app.call_method(2));
        assert!(app.dispatch_native_key(GoNativeKey::Enter));
        let output = app.render_text(800.0, 600.0);
        assert!(output.contains("Documents/Design"));
        assert!(output.contains("- Components.sketch  --  1.9 MB"));
    }

    #[test]
    #[ignore = "opens a native window; run through finder-rust native-window-smoke on the main thread"]
    fn go_shaped_wrappers_run_short_lived_native_window_through_abi() {
        let component = GoComponent::finder_lite();
        let state = GoState::finder_lite();
        let app = GoApp::mount(&component, &state).expect("mount Go ABI app");

        let smoke = app.run_native_window_for_frames(GoNativeBackend::Software, 320, 240, 1);
        assert!(smoke.ok);
        assert_eq!(smoke.frames_presented, 1);
        assert!(smoke.commands_len > 0);
        assert_eq!(smoke.pixels_len, 320 * 240);
        assert!(smoke.drawn_pixels > 0);
    }
}
