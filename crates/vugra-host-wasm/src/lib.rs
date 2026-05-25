//! Wasm host boundary for the Rust kernel path.

use vugra_core::{App, ComponentState};
use vugra_layout::{layout_frame, Constraints};
use vugra_render::{Renderer, TestRenderer};
use vugra_scene::build_scene;

pub struct WasmHost {
    canvas_id: String,
    constraints: Constraints,
}

impl WasmHost {
    pub fn new(canvas_id: impl Into<String>, constraints: Constraints) -> Self {
        Self {
            canvas_id: canvas_id.into(),
            constraints,
        }
    }

    pub fn canvas_id(&self) -> &str {
        &self.canvas_id
    }

    pub fn render<S: ComponentState>(&self, app: &App<S>) -> TestRenderer {
        let frame = app.render_frame();
        let layout = layout_frame(&frame, self.constraints);
        let scene = build_scene(&layout);
        let mut renderer = TestRenderer::default();
        renderer.render(&scene);
        renderer
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_core::{finder_lite_contract, App, ComponentState, MethodId, SignalId, Value};

    struct EmptyState;

    impl ComponentState for EmptyState {
        fn get_signal(&self, _: SignalId) -> Value {
            Value::None
        }

        fn set_signal(&mut self, _: SignalId, _: Value) {}

        fn call_method(&mut self, _: MethodId) {}
    }

    #[test]
    fn wasm_host_keeps_canvas_identity_and_renders() {
        let host = WasmHost::new(
            "vugra-canvas",
            Constraints {
                width: 320.0,
                height: 240.0,
            },
        );
        let app = App::new(finder_lite_contract(), EmptyState);
        assert_eq!(host.canvas_id(), "vugra-canvas");
        assert!(host.render(&app).to_string().contains("FinderLite"));
    }
}
