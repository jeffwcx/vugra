package project

import (
	"fmt"

	"github.com/vugra/vugra/pkg/system"
)

func ApplySystemActions(gui system.GUI, actions []SystemAction) error {
	if len(actions) == 0 {
		return nil
	}
	if gui == nil {
		return fmt.Errorf("system GUI actions require a host that implements system.GUI")
	}
	for index, action := range actions {
		if err := applySystemAction(gui, action); err != nil {
			return fmt.Errorf("launch action[%d]: %w", index, err)
		}
	}
	return nil
}

func applySystemAction(gui system.GUI, action SystemAction) error {
	switch action.Use {
	case "window.setChrome":
		chrome := gui.WindowChrome()
		if action.Titlebar != "" {
			chrome.Titlebar = system.WindowTitlebarMode(action.Titlebar)
		}
		if action.Controls != nil {
			chrome.Controls.Visible = true
			chrome.Controls.Positioned = true
			if action.Controls.X != nil {
				chrome.Controls.Frame.X = *action.Controls.X
			}
			if action.Controls.Y != nil {
				chrome.Controls.Frame.Y = *action.Controls.Y
			}
			if action.Controls.Width != nil {
				chrome.Controls.Frame.Width = *action.Controls.Width
			}
			if action.Controls.Height != nil {
				chrome.Controls.Frame.Height = *action.Controls.Height
			}
		}
		return gui.SetWindowChrome(chrome)
	default:
		return fmt.Errorf("unsupported action %q", action.Use)
	}
}
