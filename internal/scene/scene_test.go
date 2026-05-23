package scene_test

import (
	"testing"

	"github.com/vugra/vugra/internal/renderer"
	"github.com/vugra/vugra/internal/scene"
)

func TestBuildRetainedSceneDisplayClipsScrollAndDirty(t *testing.T) {
	commands := []renderer.Command{
		{
			Kind:  "element",
			ID:    "viewport",
			Rect:  renderer.Rect{X: 0.5, Y: 1.25, Width: 100.5, Height: 50.25},
			Style: renderer.Style{Overflow: "scroll"},
		},
		{
			Kind: "text",
			ID:   "label",
			Text: "Hello",
			Rect: renderer.Rect{X: 2.5, Y: 3.5, Width: 40.25, Height: 20.75},
		},
		{Kind: "end", ID: "viewport"},
	}
	first, previous := scene.Build(commands, map[string]float32{"viewport": 7.5}, nil)
	if len(first.DisplayList) != 2 {
		t.Fatalf("display list = %+v", first.DisplayList)
	}
	if first.DisplayList[1].ClipID != "viewport:clip" {
		t.Fatalf("text clip id = %q", first.DisplayList[1].ClipID)
	}
	if len(first.Clips) != 1 || first.Clips[0].Rect.X != 0.5 {
		t.Fatalf("clips = %+v", first.Clips)
	}
	if len(first.Scrolls) != 1 || first.Scrolls[0].Offset != 7.5 || first.Scrolls[0].ClipID != "viewport:clip" {
		t.Fatalf("scrolls = %+v", first.Scrolls)
	}
	if len(first.Dirty) != 2 {
		t.Fatalf("first dirty = %+v", first.Dirty)
	}

	second, _ := scene.Build(commands, map[string]float32{"viewport": 7.5}, &previous)
	if len(second.Dirty) != 0 {
		t.Fatalf("unchanged dirty = %+v", second.Dirty)
	}
	commands[1].Rect.X = 4.75
	third, _ := scene.Build(commands, map[string]float32{"viewport": 7.5}, &previous)
	if len(third.Dirty) != 1 || third.Dirty[0].X != 4.75 {
		t.Fatalf("changed dirty = %+v", third.Dirty)
	}
}
