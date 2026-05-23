package csslayout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Viewport struct {
	Width  float32  `json:"width"`
	Height *float32 `json:"height,omitempty"`
}

type Node struct {
	ID       string `json:"id"`
	Tag      string `json:"tag"`
	Class    string `json:"class,omitempty"`
	Text     string `json:"text,omitempty"`
	Children []Node `json:"children,omitempty"`
}

type Input struct {
	CSS      string   `json:"css"`
	Root     Node     `json:"root"`
	Viewport Viewport `json:"viewport"`
}

type Output struct {
	Boxes []Box `json:"boxes"`
}

type Box struct {
	ID     string  `json:"id"`
	Tag    string  `json:"tag"`
	Text   string  `json:"text,omitempty"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
	Style  Style   `json:"style"`
}

type Style struct {
	Display         string  `json:"display,omitempty"`
	FontSize        float32 `json:"font_size,omitempty"`
	LineHeight      float32 `json:"line_height,omitempty"`
	Color           string  `json:"color,omitempty"`
	BackgroundColor string  `json:"background_color,omitempty"`
	Opacity         float32 `json:"opacity,omitempty"`
	BorderWidth     float32 `json:"border_width,omitempty"`
	BorderColor     string  `json:"border_color,omitempty"`
	BorderRadius    float32 `json:"border_radius,omitempty"`
	Overflow        string  `json:"overflow,omitempty"`
}

type Engine struct {
	ManifestPath string
	BinaryPath   string
	LibraryPath  string
}

func Compute(ctx context.Context, input Input) (Output, error) {
	return Engine{}.Compute(ctx, input)
}

func (e Engine) Compute(ctx context.Context, input Input) (Output, error) {
	if input.Viewport.Width == 0 {
		return Output{}, errors.New("csslayout: viewport width is required")
	}
	if input.Root.ID == "" {
		return Output{}, errors.New("csslayout: root id is required")
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return Output{}, fmt.Errorf("csslayout: marshal input: %w", err)
	}

	return e.computePayload(ctx, payload)
}

func (e Engine) computePayload(ctx context.Context, payload []byte) (Output, error) {
	if ffiDisabled() {
		return e.computeProcess(ctx, payload)
	}
	return e.computeFFI(payload)
}

func (e Engine) computeProcess(ctx context.Context, payload []byte) (Output, error) {
	cmd, err := e.command(ctx)
	if err != nil {
		return Output{}, err
	}
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Output{}, fmt.Errorf("csslayout: run Rust engine: %w: %s", err, stderr.String())
	}
	return decodeOutput(stdout.Bytes())
}

func decodeOutput(data []byte) (Output, error) {
	var out Output
	if err := json.Unmarshal(data, &out); err != nil {
		return Output{}, fmt.Errorf("csslayout: parse engine output: %w: %s", err, string(data))
	}
	return out, nil
}

func (e Engine) command(ctx context.Context) (*exec.Cmd, error) {
	if e.BinaryPath != "" {
		return exec.CommandContext(ctx, e.BinaryPath), nil
	}
	if path := os.Getenv("VUGRA_CSS_LAYOUT_BIN"); path != "" {
		return exec.CommandContext(ctx, path), nil
	}
	manifest := e.ManifestPath
	if manifest == "" {
		var err error
		manifest, err = defaultManifestPath()
		if err != nil {
			return nil, err
		}
	}
	for _, candidate := range candidateBinaries(manifest) {
		if _, err := os.Stat(candidate); err == nil {
			return exec.CommandContext(ctx, candidate), nil
		}
	}
	return exec.CommandContext(ctx, "cargo", "run", "--quiet", "--manifest-path", manifest), nil
}

func candidateBinaries(manifest string) []string {
	root := filepath.Dir(manifest)
	name := "vugra-css-layout"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return []string{
		filepath.Join(root, "target", "release", name),
		filepath.Join(root, "target", "debug", name),
	}
}

func ffiDisabled() bool {
	return os.Getenv("VUGRA_CSS_LAYOUT_FFI") == "0"
}

func (e Engine) libraryPath() (string, error) {
	if e.LibraryPath != "" {
		return e.LibraryPath, nil
	}
	if path := os.Getenv("VUGRA_CSS_LAYOUT_LIB"); path != "" {
		return path, nil
	}
	manifest := e.ManifestPath
	if manifest == "" {
		var err error
		manifest, err = defaultManifestPath()
		if err != nil {
			return "", err
		}
	}
	for _, candidate := range candidateLibraries(manifest) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("csslayout: locate Rust FFI library; run cargo build --manifest-path %s", manifest)
}

func candidateLibraries(manifest string) []string {
	root := filepath.Dir(manifest)
	name := "libvuego_css_layout"
	ext := ".so"
	switch runtime.GOOS {
	case "darwin":
		ext = ".dylib"
	case "windows":
		name = "vuego_css_layout"
		ext = ".dll"
	}
	return []string{
		filepath.Join(root, "target", "release", name+ext),
		filepath.Join(root, "target", "debug", name+ext),
	}
}

func defaultManifestPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("csslayout: cannot resolve package path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "tools", "css-layout", "Cargo.toml")), nil
}
