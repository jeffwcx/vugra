package scene

import "github.com/vugra/vugra/internal/renderer"

type Scene struct {
	DisplayList []DisplayItem   `json:"displayList"`
	Clips       []ClipNode      `json:"clips,omitempty"`
	Scrolls     []ScrollNode    `json:"scrolls,omitempty"`
	Dirty       []renderer.Rect `json:"dirty,omitempty"`
}

type DisplayItem struct {
	Command renderer.Command `json:"command"`
	ClipID  string           `json:"clipId,omitempty"`
}

type ClipNode struct {
	ID     string        `json:"id"`
	Rect   renderer.Rect `json:"rect"`
	Parent string        `json:"parent,omitempty"`
}

type ScrollNode struct {
	ID        string        `json:"id"`
	Rect      renderer.Rect `json:"rect"`
	MaxOffset float32       `json:"maxOffset"`
	Offset    float32       `json:"offset"`
	ClipID    string        `json:"clipId"`
}

type Previous struct {
	Bounds map[string]renderer.Rect
}

func Build(commands []renderer.Command, scrollOffsets map[string]float32, previous *Previous) (Scene, Previous) {
	var out Scene
	next := Previous{Bounds: map[string]renderer.Rect{}}
	clipStack := []ClipNode{}
	for _, command := range commands {
		switch command.Kind {
		case "element":
			currentClip := ""
			if len(clipStack) > 0 {
				currentClip = clipStack[len(clipStack)-1].ID
			}
			out.DisplayList = append(out.DisplayList, DisplayItem{Command: command, ClipID: currentClip})
			next.Bounds[command.ID] = command.Rect
			if changed(previous, command.ID, command.Rect) {
				out.Dirty = append(out.Dirty, command.Rect)
			}
			if clipsOverflow(command.Style.Overflow) {
				clip := ClipNode{ID: command.ID + ":clip", Rect: command.Rect, Parent: currentClip}
				out.Clips = append(out.Clips, clip)
				clipStack = append(clipStack, clip)
				if command.Style.Overflow == "scroll" {
					out.Scrolls = append(out.Scrolls, ScrollNode{
						ID:     command.ID,
						Rect:   command.Rect,
						Offset: scrollOffsets[command.ID],
						ClipID: clip.ID,
					})
				}
			}
		case "text", "svg":
			currentClip := ""
			if len(clipStack) > 0 {
				currentClip = clipStack[len(clipStack)-1].ID
			}
			out.DisplayList = append(out.DisplayList, DisplayItem{Command: command, ClipID: currentClip})
			next.Bounds[command.ID] = command.Rect
			if changed(previous, command.ID, command.Rect) {
				out.Dirty = append(out.Dirty, command.Rect)
			}
		case "end":
			if len(clipStack) > 0 {
				clipStack = clipStack[:len(clipStack)-1]
			}
		}
	}
	return out, next
}

func changed(previous *Previous, id string, rect renderer.Rect) bool {
	if previous == nil || previous.Bounds == nil {
		return true
	}
	return previous.Bounds[id] != rect
}

func clipsOverflow(overflow string) bool {
	return overflow == "hidden" || overflow == "scroll"
}
