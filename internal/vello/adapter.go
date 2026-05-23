package vello

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vugra/vugra/internal/renderer"
)

type Op struct {
	Kind   string
	Rect   renderer.Rect
	Text   string
	SVG    string
	Lines  []renderer.LineBox
	Glyphs []renderer.GlyphRun
	Role   string
	Tag    string
	Props  map[string]string
	Style  renderer.Style
}

type Renderer struct {
	Ops []Op
}

type NativeRenderer struct {
	Width    int
	Height   int
	Commands []renderer.Command
	Pixels   []byte
	Status   string
	Fallback *renderer.SoftwareRenderer
}

func New() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Render(commands []renderer.Command) {
	r.Ops = Translate(commands)
}

func Translate(commands []renderer.Command) []Op {
	ops := make([]Op, 0, len(commands))
	clipStack := []bool{}
	for _, command := range commands {
		switch command.Kind {
		case "element":
			ops = append(ops, Op{
				Kind:  "fill-rect",
				Rect:  command.Rect,
				Role:  command.Role,
				Tag:   command.Tag,
				Props: command.Props,
				Style: command.Style,
			})
			ops = append(ops, Op{
				Kind:  "stroke-rect",
				Rect:  command.Rect,
				Role:  command.Role,
				Tag:   command.Tag,
				Props: command.Props,
				Style: command.Style,
			})
			clipped := clipsOverflow(command.Style.Overflow)
			clipStack = append(clipStack, clipped)
			if clipped {
				ops = append(ops, Op{
					Kind:  "begin-clip",
					Rect:  command.Rect,
					Role:  command.Role,
					Tag:   command.Tag,
					Props: command.Props,
					Style: command.Style,
				})
			}
		case "text":
			ops = append(ops, Op{
				Kind:   "text",
				Rect:   command.Rect,
				Text:   command.Text,
				Lines:  command.Lines,
				Glyphs: command.Glyphs,
				Role:   command.Role,
				Tag:    command.Tag,
				Props:  command.Props,
				Style:  command.Style,
			})
		case "selection":
			ops = append(ops, Op{
				Kind:  "fill-rect",
				Rect:  command.Rect,
				Role:  command.Role,
				Tag:   command.Tag,
				Props: command.Props,
				Style: command.Style,
			})
		case "svg":
			ops = append(ops, Op{
				Kind:  "svg",
				Rect:  command.Rect,
				SVG:   command.SVG,
				Role:  command.Role,
				Tag:   command.Tag,
				Props: command.Props,
				Style: command.Style,
			})
		case "end":
			if len(clipStack) == 0 {
				continue
			}
			clipped := clipStack[len(clipStack)-1]
			clipStack = clipStack[:len(clipStack)-1]
			if clipped {
				ops = append(ops, Op{Kind: "end-clip"})
			}
		}
	}
	return ops
}

func clipsOverflow(overflow string) bool {
	return overflow == "hidden" || overflow == "scroll"
}

func (r *Renderer) BackendStatus() error {
	return fmt.Errorf("vello is available through the Rust sidecar; use NativeRenderer or the vello-png CLI for the executable Vello/wgpu path")
}

func NewNativeRenderer(width, height int) *NativeRenderer {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return &NativeRenderer{
		Width:    width,
		Height:   height,
		Fallback: renderer.NewSoftware(width, height),
	}
}

func (r *NativeRenderer) Render(commands []renderer.Command) {
	r.Commands = append([]renderer.Command(nil), commands...)
	pixels, status, err := RenderPixels(commands, r.Width, r.Height)
	if err == nil && len(pixels) == r.Width*r.Height*4 {
		r.Pixels = pixels
		r.Status = status
		return
	}
	r.Fallback.Render(commands)
	if r.Fallback.Image != nil {
		r.Pixels = append(r.Pixels[:0], r.Fallback.Image.Pix...)
	}
	if err != nil {
		r.Status = "fallback: " + err.Error()
	} else {
		r.Status = "fallback: invalid Vello pixel buffer"
	}
}

func (r *NativeRenderer) Resize(width, height int) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	if r.Width == width && r.Height == height {
		return
	}
	r.Width = width
	r.Height = height
	r.Fallback = renderer.NewSoftware(width, height)
	r.Pixels = nil
}

func EncodeOps(commands []renderer.Command) ([]byte, error) {
	ops := Translate(commands)
	encoded, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("encode Vello ops: %w", err)
	}
	return encoded, nil
}

func RenderPixels(commands []renderer.Command, width, height int) ([]byte, string, error) {
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	ops := Translate(commands)
	encoded, err := json.MarshalIndent(ops, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode Vello ops: %w", err)
	}
	dir, err := os.MkdirTemp("", "vugra-vello-*")
	if err != nil {
		return nil, "", fmt.Errorf("create Vello temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	rawPath := filepath.Join(dir, "frame.rgba")
	size := fmt.Sprintf("%dx%d", width, height)
	manifest, err := sidecarManifestPath()
	if err != nil {
		return nil, "", err
	}
	cmd := exec.Command("cargo", "run", "--manifest-path", manifest, "--", "--size", size, "--raw", rawPath)
	cmd.Stdin = bytes.NewReader(encoded)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, stdout.String(), fmt.Errorf("run Vello sidecar: %w: %s", err, stderr.String())
	}
	pixels, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, stdout.String(), fmt.Errorf("read Vello raw frame: %w", err)
	}
	return pixels, stdout.String(), nil
}

func sidecarManifestPath() (string, error) {
	if path := os.Getenv("VUGRA_VELLO_SIDECAR_MANIFEST"); path != "" {
		return path, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, "tools", "vello-sidecar", "Cargo.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("locate tools/vello-sidecar/Cargo.toml")
}
