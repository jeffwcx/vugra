//! Rust user-facing API over the Vugra kernel.

pub use vugra_core::{ComponentState, Event, MethodId, Modifiers, SignalId, Value};
pub use vugra_ir::Component;
pub use vugra_layout::Constraints;
pub use vugra_render::TestRenderer;

use vugra_core::App as CoreApp;
use vugra_host_native::{
    render_test_frame, run_app_window, run_app_window_for_frames, NativeWindowConfig,
    NativeWindowSmoke,
};

pub struct App<S> {
    inner: CoreApp<S>,
}

impl<S: ComponentState> App<S> {
    pub fn mount(component: Component, state: S) -> Self {
        Self {
            inner: CoreApp::new(component, state),
        }
    }

    pub fn dispatch(&mut self, method: MethodId) {
        self.inner.dispatch(method);
    }

    pub fn dispatch_event(&mut self, method: MethodId, event: Event) {
        self.inner.dispatch_event(method, event);
    }

    pub fn render_test(&self, constraints: Constraints) -> TestRenderer {
        render_test_frame(&self.inner, constraints)
    }

    pub fn run_native(&mut self, config: NativeWindowConfig) -> Result<(), minifb::Error> {
        run_app_window(&mut self.inner, config)
    }

    pub fn run_native_for_frames(
        &mut self,
        config: NativeWindowConfig,
        frames: usize,
    ) -> Result<NativeWindowSmoke, minifb::Error> {
        run_app_window_for_frames(&mut self.inner, config, frames)
    }

    pub fn inner(&self) -> &CoreApp<S> {
        &self.inner
    }

    pub fn inner_mut(&mut self) -> &mut CoreApp<S> {
        &mut self.inner
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_core::finder_lite_contract;

    struct State;

    impl ComponentState for State {
        fn get_signal(&self, id: SignalId) -> Value {
            match id.0 {
                1 => "Documents".into(),
                2 => "0 items".into(),
                3 => "0 selected".into(),
                _ => Value::None,
            }
        }

        fn set_signal(&mut self, _: SignalId, _: Value) {}

        fn call_method(&mut self, _: MethodId) {}
    }

    #[test]
    fn rust_api_mounts_direct_component_state() {
        let app = App::mount(finder_lite_contract(), State);
        let output = app
            .render_test(Constraints {
                width: 800.0,
                height: 600.0,
            })
            .to_string();
        assert!(output.contains("FinderLite"));
        assert!(output.contains("Documents"));
    }
}
