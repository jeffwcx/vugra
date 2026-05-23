package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vugra/vugra/internal/project"
	"github.com/vugra/vugra/pkg/system"
)

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "vugra.config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "$schema": "https://vugra.dev/schemas/vugra.config.schema.json",
  "name": "finder-lite",
  "entry": "src/App.vue",
  "app": { "title": "Finder Lite", "width": 900, "height": 640 },
  "runtime": { "renderer": "vello-native", "layout": "css" },
  "scripts": {
    "onLaunch": [
      {
        "use": "window.setChrome",
        "titlebar": "hidden",
        "controls": { "x": 18, "y": 11, "width": 72, "height": 28 }
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Path != configPath {
		t.Fatalf("config path = %q, want %q", cfg.Path, configPath)
	}
	if cfg.EntryPath() != filepath.Join(dir, "src", "App.vue") {
		t.Fatalf("entry path = %q", cfg.EntryPath())
	}
	if cfg.App.Title != "Finder Lite" || cfg.App.Width != 900 || cfg.App.Height != 640 {
		t.Fatalf("app config = %+v", cfg.App)
	}
	if len(cfg.LaunchActions()) != 1 {
		t.Fatalf("launch actions = %+v", cfg.LaunchActions())
	}
}

func TestLoadProjectConfigRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "vugra.config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "name": "bad",
  "entry": "App.vue",
  "unknown": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := project.Load(configPath); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestLoadProjectConfigFallsBackToLegacyName(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "vuego.config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "name": "legacy",
  "entry": "App.vue"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("load legacy config: %v", err)
	}
	if cfg.Path != configPath {
		t.Fatalf("config path = %q, want %q", cfg.Path, configPath)
	}
}

func TestProjectConfigValidation(t *testing.T) {
	cfg := project.Config{Name: "bad", Entry: "App.vue"}
	cfg.Runtime.Renderer = "canvas2d"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid renderer error")
	}
	cfg.Runtime.Renderer = "software"
	cfg.System.GUI.OnLaunch = []project.SystemAction{{Use: "shell"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid system action error")
	}
}

func TestApplySystemActionsUpdatesGUIChrome(t *testing.T) {
	gui := &fakeGUI{
		chrome: system.WindowChrome{
			Titlebar: system.WindowTitlebarDefault,
			Controls: system.WindowControls{
				Frame: system.Rect{Width: 72, Height: 28},
			},
		},
	}
	x := float32(18)
	y := float32(11)
	err := project.ApplySystemActions(gui, []project.SystemAction{
		{
			Use:      "window.setChrome",
			Titlebar: "hidden",
			Controls: &project.ControlsConfig{
				X: &x,
				Y: &y,
			},
		},
	})
	if err != nil {
		t.Fatalf("apply system actions: %v", err)
	}
	if gui.sets != 1 {
		t.Fatalf("set calls = %d", gui.sets)
	}
	if gui.chrome.Titlebar != system.WindowTitlebarHidden {
		t.Fatalf("titlebar = %q", gui.chrome.Titlebar)
	}
	if !gui.chrome.Controls.Positioned || gui.chrome.Controls.Frame.X != 18 || gui.chrome.Controls.Frame.Y != 11 {
		t.Fatalf("controls = %+v", gui.chrome.Controls)
	}
}

func TestApplySystemActionsRequiresGUI(t *testing.T) {
	if err := project.ApplySystemActions(nil, []project.SystemAction{{Use: "window.setChrome"}}); err == nil {
		t.Fatal("expected missing GUI error")
	}
}

type fakeGUI struct {
	chrome system.WindowChrome
	sets   int
}

func (g *fakeGUI) WindowChrome() system.WindowChrome {
	return g.chrome
}

func (g *fakeGUI) SetWindowChrome(chrome system.WindowChrome) error {
	g.chrome = chrome
	g.sets++
	return nil
}
