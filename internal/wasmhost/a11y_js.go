//go:build js && wasm

package wasmhost

import (
	"strconv"
	"syscall/js"

	"github.com/vugra/vugra/internal/accessibility"
	"github.com/vugra/vugra/internal/renderer"
)

func SyncAccessibility(containerID string, commands []renderer.Command, focusedID string) {
	document := js.Global().Get("document")
	container := document.Call("getElementById", containerID)
	if !container.Truthy() {
		return
	}
	container.Set("innerHTML", "")
	for _, node := range accessibility.Build(commands) {
		appendAccessibilityNode(document, container, node, focusedID)
	}
}

type AccessibilityEvents struct {
	Click func(x, y int)
	Focus func(id string)
	Key   func(key string)
	Text  func(text string)
}

func InstallAccessibilityEvents(containerID string, dispatch func(x, y int)) js.Func {
	return InstallAccessibilityEventHandlers(containerID, AccessibilityEvents{
		Click: dispatch,
	})
}

func InstallAccessibilityEventHandlers(containerID string, handlers AccessibilityEvents) js.Func {
	document := js.Global().Get("document")
	container := document.Call("getElementById", containerID)
	if !container.Truthy() {
		return js.Func{}
	}
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		switch event.Get("type").String() {
		case "focusin":
			if handlers.Focus == nil {
				return nil
			}
			target := event.Get("target")
			for target.Truthy() && !sameNode(target, container) {
				if target.Get("getAttribute").Truthy() {
					id := target.Call("getAttribute", "data-vugra-id").String()
					if id != "" {
						handlers.Focus(id)
						stopEvent(event)
						return nil
					}
				}
				target = target.Get("parentElement")
			}
			return nil
		case "keydown":
			if handlers.Key == nil {
				return nil
			}
			key := event.Get("key").String()
			if jsBool(event.Get("shiftKey")) && key == "Tab" {
				key = "Shift+Tab"
			}
			if (jsBool(event.Get("ctrlKey")) || jsBool(event.Get("metaKey"))) && (key == "a" || key == "A") {
				key = "Mod+A"
			}
			handlers.Key(key)
			stopEvent(event)
			return nil
		case "input":
			if handlers.Text == nil {
				return nil
			}
			text := event.Get("data").String()
			if text == "" {
				text = event.Get("target").Get("value").String()
			}
			if text == "" {
				return nil
			}
			handlers.Text(text)
			stopEvent(event)
			return nil
		}
		target := event.Get("target")
		for target.Truthy() && !sameNode(target, container) {
			if target.Get("getAttribute").Truthy() {
				x, okX := attrFloat32(target, "data-vugra-x")
				y, okY := attrFloat32(target, "data-vugra-y")
				width, okW := attrFloat32(target, "data-vugra-width")
				height, okH := attrFloat32(target, "data-vugra-height")
				if okX && okY && okW && okH && handlers.Click != nil {
					handlers.Click(int(x+width/2), int(y+height/2))
					stopEvent(event)
					return nil
				}
			}
			target = target.Get("parentElement")
		}
		return nil
	})
	container.Call("addEventListener", "click", handler)
	container.Call("addEventListener", "focusin", handler)
	container.Call("addEventListener", "keydown", handler)
	container.Call("addEventListener", "input", handler)
	return handler
}

func stopEvent(event js.Value) {
	if event.Get("preventDefault").Truthy() {
		event.Call("preventDefault")
	}
	if event.Get("stopPropagation").Truthy() {
		event.Call("stopPropagation")
	}
}

func appendAccessibilityNode(document js.Value, parent js.Value, node accessibility.Node, focusedID string) {
	element := document.Call("createElement", "div")
	role := node.Role
	if role == "" {
		role = "group"
	}
	element.Call("setAttribute", "role", role)
	if node.ID != "" {
		element.Call("setAttribute", "data-vugra-id", node.ID)
	}
	if node.ID != "" && node.ID == focusedID {
		element.Call("setAttribute", "aria-current", "true")
		element.Call("setAttribute", "data-vugra-focused", "true")
	}
	if role == "button" || role == "checkbox" || role == "textbox" {
		element.Call("setAttribute", "tabindex", "0")
	}
	if node.Name != "" {
		element.Call("setAttribute", "aria-label", node.Name)
	}
	if role == "textbox" {
		element.Set("textContent", node.Name)
	}
	if node.Props != nil {
		if node.Props["checked"] != "" {
			element.Call("setAttribute", "aria-checked", node.Props["checked"])
		}
		if disabled, ok := node.Props["disabled"]; ok && disabled != "false" && disabled != "0" {
			element.Call("setAttribute", "aria-disabled", "true")
		}
	}
	element.Call("setAttribute", "data-vugra-x", formatFloat32(node.Rect.X))
	element.Call("setAttribute", "data-vugra-y", formatFloat32(node.Rect.Y))
	element.Call("setAttribute", "data-vugra-width", formatFloat32(node.Rect.Width))
	element.Call("setAttribute", "data-vugra-height", formatFloat32(node.Rect.Height))
	parent.Call("appendChild", element)
	for _, child := range node.Children {
		appendAccessibilityNode(document, element, child, focusedID)
	}
}

func formatFloat32(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

func attrFloat32(element js.Value, name string) (float32, bool) {
	raw := element.Call("getAttribute", name)
	if !raw.Truthy() {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw.String(), 32)
	if err != nil {
		return 0, false
	}
	return float32(value), true
}

func sameNode(a, b js.Value) bool {
	if a.Get("isSameNode").Truthy() {
		return a.Call("isSameNode", b).Bool()
	}
	return a.Equal(b)
}
