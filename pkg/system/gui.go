package system

const (
	WindowControlsXToken      = "vugra-window-controls-x"
	WindowControlsYToken      = "vugra-window-controls-y"
	WindowControlsWidthToken  = "vugra-window-controls-width"
	WindowControlsHeightToken = "vugra-window-controls-height"
	WindowControlsLeftToken   = "vugra-window-controls-left"
	WindowControlsTopToken    = "vugra-window-controls-top"
)

type Rect struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type WindowTitlebarMode string

const (
	WindowTitlebarDefault WindowTitlebarMode = "default"
	WindowTitlebarHidden  WindowTitlebarMode = "hidden"
)

type WindowControls struct {
	Visible    bool
	Positioned bool
	Frame      Rect
}

type WindowChrome struct {
	Titlebar WindowTitlebarMode
	Controls WindowControls
}

type GUI interface {
	WindowChrome() WindowChrome
	SetWindowChrome(WindowChrome) error
}

func (chrome WindowChrome) SystemTokens() map[string]float32 {
	if chrome.Titlebar != WindowTitlebarHidden || !chrome.Controls.Visible {
		return nil
	}
	frame := chrome.Controls.Frame
	return map[string]float32{
		WindowControlsXToken:      frame.X,
		WindowControlsYToken:      frame.Y,
		WindowControlsWidthToken:  frame.Width,
		WindowControlsHeightToken: frame.Height,
		WindowControlsLeftToken:   frame.X + frame.Width,
		WindowControlsTopToken:    frame.Y + frame.Height,
	}
}
