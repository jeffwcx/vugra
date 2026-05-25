//! Renderer-neutral commands and test renderer for the Rust kernel path.

use std::fmt;

use vugra_layout::Rect;
use vugra_scene::{Scene, SceneCommand};

#[derive(Clone, Debug, PartialEq)]
pub enum RenderCommand {
    Element {
        id: String,
        role: String,
        rect: Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
        method: Option<vugra_ir::MethodId>,
    },
    Text {
        id: String,
        text: String,
        role: String,
        rect: Rect,
        selected: bool,
        visual_state: vugra_layout::RowVisualState,
    },
    End {
        id: String,
    },
}

pub trait Renderer {
    fn render(&mut self, scene: &Scene);
}

#[derive(Default)]
pub struct TestRenderer {
    commands: Vec<RenderCommand>,
}

impl TestRenderer {
    pub fn commands(&self) -> &[RenderCommand] {
        &self.commands
    }
}

impl Renderer for TestRenderer {
    fn render(&mut self, scene: &Scene) {
        self.commands = scene
            .commands
            .iter()
            .map(|command| match command {
                SceneCommand::Begin {
                    id,
                    role,
                    rect,
                    selected,
                    visual_state,
                    method,
                } => RenderCommand::Element {
                    id: id.clone(),
                    role: role.clone(),
                    rect: *rect,
                    selected: *selected,
                    visual_state: *visual_state,
                    method: *method,
                },
                SceneCommand::Text {
                    id,
                    text,
                    role,
                    rect,
                    selected,
                    visual_state,
                } => RenderCommand::Text {
                    id: id.clone(),
                    text: text.clone(),
                    role: role.clone(),
                    rect: *rect,
                    selected: *selected,
                    visual_state: *visual_state,
                },
                SceneCommand::End { id } => RenderCommand::End { id: id.clone() },
            })
            .collect();
    }
}

impl fmt::Display for TestRenderer {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let mut index = 0;
        while index < self.commands.len() {
            if let Some((line, next)) = row_line_at(&self.commands, index) {
                writeln!(f, "{line}")?;
                index = next;
                continue;
            }
            if let RenderCommand::Text { id, text, .. } = &self.commands[index] {
                if !id.contains("-icon:text")
                    && !id.contains("-name:text")
                    && !id.contains("-modified:text")
                    && !id.contains("-size:text")
                {
                    writeln!(f, "{text}")?;
                }
            }
            index += 1;
        }
        Ok(())
    }
}

fn row_line_at(commands: &[RenderCommand], index: usize) -> Option<(String, usize)> {
    let RenderCommand::Element {
        id, role, selected, ..
    } = commands.get(index)?
    else {
        return None;
    };
    if role != "row" {
        return None;
    }
    let mut name = String::new();
    let mut modified = String::new();
    let mut size = String::new();
    let mut next = index + 1;
    while next < commands.len() {
        match &commands[next] {
            RenderCommand::End { id: end } if end == id => {
                next += 1;
                break;
            }
            RenderCommand::Text { id, text, .. } if id.contains("-name:text") => {
                name = text.clone();
            }
            RenderCommand::Text { id, text, .. } if id.contains("-modified:text") => {
                modified = text.clone();
            }
            RenderCommand::Text { id, text, .. } if id.contains("-size:text") => {
                size = text.clone();
            }
            _ => {}
        }
        next += 1;
    }
    if name.is_empty() {
        return None;
    }
    let marker = if *selected { "*" } else { "-" };
    Some((format!("{marker} {name}  {modified}  {size}"), next))
}

#[cfg(test)]
mod tests {
    use super::*;
    use vugra_scene::SceneCommand;

    #[test]
    fn test_renderer_projects_scene_to_text_snapshot() {
        let rect = Rect {
            x: 0.0,
            y: 0.0,
            width: 10.0,
            height: 10.0,
        };
        let scene = Scene::from_commands(vec![SceneCommand::Text {
            id: "title:text".to_string(),
            text: "FinderLite".to_string(),
            role: "heading".to_string(),
            rect,
            selected: false,
            visual_state: vugra_layout::RowVisualState::Normal,
        }]);
        let mut renderer = TestRenderer::default();
        renderer.render(&scene);
        assert_eq!(renderer.to_string(), "FinderLite\n");
    }
}
