package accessibility

import "github.com/vugra/vugra/internal/renderer"

type Node struct {
	ID       string            `json:"id,omitempty"`
	Role     string            `json:"role"`
	Name     string            `json:"name,omitempty"`
	Rect     renderer.Rect     `json:"rect"`
	Children []Node            `json:"children,omitempty"`
	Props    map[string]string `json:"props,omitempty"`
}

func Build(commands []renderer.Command) []Node {
	var roots []Node
	var stack []*Node
	for _, command := range commands {
		switch command.Kind {
		case "element":
			node := Node{
				ID:    command.ID,
				Role:  role(command),
				Rect:  command.Rect,
				Props: command.Props,
			}
			if len(stack) == 0 {
				roots = append(roots, node)
				stack = append(stack, &roots[len(roots)-1])
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
				stack = append(stack, &parent.Children[len(parent.Children)-1])
			}
		case "text":
			node := Node{
				ID:   command.ID,
				Role: role(command),
				Name: command.Text,
				Rect: command.Rect,
			}
			if len(stack) == 0 {
				roots = append(roots, node)
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
				if parent.Name == "" {
					parent.Name = command.Text
				}
			}
		case "end":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return roots
}

func role(command renderer.Command) string {
	if command.Role != "" {
		return command.Role
	}
	if command.Kind == "text" {
		return "text"
	}
	return "group"
}
