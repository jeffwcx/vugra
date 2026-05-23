package accessibility_test

import (
	"testing"

	"github.com/vugra/vugra/internal/accessibility"
	"github.com/vugra/vugra/internal/renderer"
)

func TestBuildAccessibilityTree(t *testing.T) {
	tree := accessibility.Build([]renderer.Command{
		{Kind: "element", ID: "root", Tag: "div", Role: "group", Rect: renderer.Rect{Width: 100, Height: 60}},
		{Kind: "element", ID: "button", Tag: "button", Role: "button", Rect: renderer.Rect{X: 10, Y: 10, Width: 80, Height: 30}},
		{Kind: "text", ID: "label", Text: "+", Role: "text", Rect: renderer.Rect{X: 12, Y: 12, Width: 8, Height: 20}},
		{Kind: "end", ID: "button", Tag: "button"},
		{Kind: "end", ID: "root", Tag: "div"},
	})
	if len(tree) != 1 {
		t.Fatalf("roots = %d", len(tree))
	}
	root := tree[0]
	if root.Role != "group" || len(root.Children) != 1 {
		t.Fatalf("root = %+v", root)
	}
	button := root.Children[0]
	if button.Role != "button" || button.Name != "+" {
		t.Fatalf("button = %+v", button)
	}
}
