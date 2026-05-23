package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const DefaultConfigFile = "vugra.config.json"
const LegacyConfigFile = "vuego.config.json"
const CurrentSchema = "https://vugra.dev/schemas/vugra.config.schema.json"

type Config struct {
	Schema  string        `json:"$schema,omitempty"`
	Name    string        `json:"name"`
	Version string        `json:"version,omitempty"`
	Entry   string        `json:"entry"`
	App     AppConfig     `json:"app,omitempty"`
	Runtime RuntimeConfig `json:"runtime,omitempty"`
	Scripts ScriptsConfig `json:"scripts,omitempty"`
	System  SystemConfig  `json:"system,omitempty"`

	Path string `json:"-"`
	Dir  string `json:"-"`
}

type AppConfig struct {
	Title  string `json:"title,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type RuntimeConfig struct {
	Renderer string `json:"renderer,omitempty"`
	Layout   string `json:"layout,omitempty"`
}

type ScriptsConfig struct {
	OnLaunch []SystemAction `json:"onLaunch,omitempty"`
}

type SystemConfig struct {
	GUI GUIConfig `json:"gui,omitempty"`
}

type GUIConfig struct {
	OnLaunch []SystemAction `json:"onLaunch,omitempty"`
}

type SystemAction struct {
	Use      string          `json:"use"`
	Titlebar string          `json:"titlebar,omitempty"`
	Controls *ControlsConfig `json:"controls,omitempty"`
}

type ControlsConfig struct {
	X      *float32 `json:"x,omitempty"`
	Y      *float32 `json:"y,omitempty"`
	Width  *float32 `json:"width,omitempty"`
	Height *float32 `json:"height,omitempty"`
}

func Load(path string) (Config, error) {
	resolved, err := ResolveConfigPath(path)
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return Config{}, fmt.Errorf("read project config %s: %w", resolved, err)
	}
	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse project config %s: %w", resolved, err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, fmt.Errorf("parse project config %s: %w", resolved, err)
	}
	cfg.Path = resolved
	cfg.Dir = filepath.Dir(resolved)
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid project config %s: %w", resolved, err)
	}
	return cfg, nil
}

func ResolveConfigPath(path string) (string, error) {
	if path == "" {
		path = DefaultConfigFile
	}
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		candidate := filepath.Join(path, DefaultConfigFile)
		if candidateInfo, candidateErr := os.Stat(candidate); candidateErr == nil && !candidateInfo.IsDir() {
			path = candidate
			err = nil
		} else {
			path = filepath.Join(path, LegacyConfigFile)
			info, err = os.Stat(path)
		}
	}
	if err != nil && os.IsNotExist(err) && filepath.Base(path) != DefaultConfigFile && filepath.Base(path) != LegacyConfigFile {
		for _, name := range []string{DefaultConfigFile, LegacyConfigFile} {
			candidate := filepath.Join(path, name)
			if candidateInfo, candidateErr := os.Stat(candidate); candidateErr == nil && !candidateInfo.IsDir() {
				path = candidate
				err = nil
				break
			}
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat project config %s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve project config %s: %w", path, err)
	}
	return abs, nil
}

func (c *Config) applyDefaults() {
	if c.App.Title == "" {
		c.App.Title = "Vugra"
	}
	if c.App.Width <= 0 {
		c.App.Width = 800
	}
	if c.App.Height <= 0 {
		c.App.Height = 600
	}
}

func (c Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Entry == "" {
		return fmt.Errorf("entry is required")
	}
	if c.App.Width <= 0 {
		return fmt.Errorf("app.width must be greater than zero")
	}
	if c.App.Height <= 0 {
		return fmt.Errorf("app.height must be greater than zero")
	}
	switch c.Runtime.Renderer {
	case "", "vello-native", "vello", "software":
	default:
		return fmt.Errorf("runtime.renderer must be vello-native, vello, or software")
	}
	switch c.Runtime.Layout {
	case "", "go", "css":
	default:
		return fmt.Errorf("runtime.layout must be go or css")
	}
	for index, action := range c.LaunchActions() {
		if err := validateAction(action); err != nil {
			return fmt.Errorf("launch action[%d]: %w", index, err)
		}
	}
	return nil
}

func validateAction(action SystemAction) error {
	switch action.Use {
	case "window.setChrome":
	default:
		return fmt.Errorf("use must be window.setChrome")
	}
	switch action.Titlebar {
	case "", "default", "hidden":
	default:
		return fmt.Errorf("titlebar must be default or hidden")
	}
	if action.Controls != nil {
		if action.Controls.Width != nil && *action.Controls.Width <= 0 {
			return fmt.Errorf("controls.width must be greater than zero")
		}
		if action.Controls.Height != nil && *action.Controls.Height <= 0 {
			return fmt.Errorf("controls.height must be greater than zero")
		}
	}
	return nil
}

func (c Config) EntryPath() string {
	if filepath.IsAbs(c.Entry) {
		return filepath.Clean(c.Entry)
	}
	return filepath.Clean(filepath.Join(c.Dir, c.Entry))
}

func (c Config) LaunchActions() []SystemAction {
	out := make([]SystemAction, 0, len(c.Scripts.OnLaunch)+len(c.System.GUI.OnLaunch))
	out = append(out, c.Scripts.OnLaunch...)
	out = append(out, c.System.GUI.OnLaunch...)
	return out
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}
