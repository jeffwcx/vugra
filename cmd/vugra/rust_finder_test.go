package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunRustFinderLiteVariants(t *testing.T) {
	for _, variant := range []string{"direct", "abi"} {
		t.Run(variant, func(t *testing.T) {
			out := captureStdout(t, func() {
				if err := runRustFinderLite([]string{variant}); err != nil {
					t.Fatalf("run rust finder lite: %v", err)
				}
			})
			for _, want := range []string{
				"Documents",
				"12 items · Current path: Documents",
				"1 items selected",
				"* Design  --  --",
				"- Roadmap.md  --  12 KB",
			} {
				if !strings.Contains(out, want) {
					t.Fatalf("output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestRunRustFinderLiteRejectsUnknownVariant(t *testing.T) {
	if err := runRustFinderLite([]string{"go"}); err == nil || !strings.Contains(err.Error(), "direct, abi, generated-adapter-smoke, parity-summary, native, native-software, native-vello, native-wgpu, abi-native, abi-native-software, abi-native-vello, abi-native-wgpu, native-smoke, native-window-smoke, native-software-window-smoke, native-vello-window-smoke, native-wgpu-window-smoke, abi-window-smoke, abi-software-window-smoke, abi-vello-window-smoke, abi-wgpu-window-smoke, vello-device-smoke, or wgpu-device-smoke") {
		t.Fatalf("expected variant error, got %v", err)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	old := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	t.Cleanup(func() {
		os.Stdout = old
	})

	run()

	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(read); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRustFinderCommandShape(t *testing.T) {
	cmd := rustFinderCommand("abi")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi" {
		t.Fatalf("command args = %q", got)
	}
	cmd = rustFinderCommand("native")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native" {
		t.Fatalf("native command args = %q", got)
	}
	cmd = rustFinderCommand("native-software")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-software" {
		t.Fatalf("native-software command args = %q", got)
	}
	cmd = rustFinderCommand("generated-adapter-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- generated-adapter-smoke" {
		t.Fatalf("generated-adapter-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("parity-summary")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- parity-summary" {
		t.Fatalf("parity-summary command args = %q", got)
	}
	cmd = rustFinderCommand("native-vello")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-vello" {
		t.Fatalf("native-vello command args = %q", got)
	}
	cmd = rustFinderCommand("native-wgpu")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-wgpu" {
		t.Fatalf("native-wgpu command args = %q", got)
	}
	cmd = rustFinderCommand("abi-native")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-native" {
		t.Fatalf("abi-native command args = %q", got)
	}
	cmd = rustFinderCommand("abi-native-software")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-native-software" {
		t.Fatalf("abi-native-software command args = %q", got)
	}
	cmd = rustFinderCommand("abi-native-vello")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-native-vello" {
		t.Fatalf("abi-native-vello command args = %q", got)
	}
	cmd = rustFinderCommand("abi-native-wgpu")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-native-wgpu" {
		t.Fatalf("abi-native-wgpu command args = %q", got)
	}
	cmd = rustFinderCommand("native-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-smoke" {
		t.Fatalf("native-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("native-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-window-smoke" {
		t.Fatalf("native-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("native-software-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-software-window-smoke" {
		t.Fatalf("native-software-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("native-vello-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-vello-window-smoke" {
		t.Fatalf("native-vello-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("native-wgpu-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- native-wgpu-window-smoke" {
		t.Fatalf("native-wgpu-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("abi-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-window-smoke" {
		t.Fatalf("abi-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("abi-software-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-software-window-smoke" {
		t.Fatalf("abi-software-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("abi-vello-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-vello-window-smoke" {
		t.Fatalf("abi-vello-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("abi-wgpu-window-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust -- abi-wgpu-window-smoke" {
		t.Fatalf("abi-wgpu-window-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("vello-device-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust --features vugra-render-vello/vello-device -- vello-device-smoke" {
		t.Fatalf("vello-device-smoke command args = %q", got)
	}
	cmd = rustFinderCommand("wgpu-device-smoke")
	if got := strings.Join(cmd.Args, " "); got != "cargo run -q -p finder-rust --features vugra-render-wgpu/wgpu-device -- wgpu-device-smoke" {
		t.Fatalf("wgpu-device-smoke command args = %q", got)
	}
}

func TestRunRustFinderLiteUsesCommandHook(t *testing.T) {
	old := rustFinderCommand
	t.Cleanup(func() { rustFinderCommand = old })
	rustFinderCommand = func(variant string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=TestRustFinderLiteHelperProcess", "--", variant)
	}
	t.Setenv("VUGRA_RUST_FINDER_HELPER", "1")

	out := captureStdout(t, func() {
		if err := runRustFinderLite(nil); err != nil {
			t.Fatalf("run rust finder lite: %v", err)
		}
	})
	if strings.TrimSpace(out) != "helper variant=direct" {
		t.Fatalf("output = %q", out)
	}
}

func TestRustFinderLiteHelperProcess(t *testing.T) {
	if os.Getenv("VUGRA_RUST_FINDER_HELPER") != "1" {
		return
	}
	args := os.Args
	variant := args[len(args)-1]
	if variant == "--" {
		variant = "missing"
	}
	os.Stdout.WriteString("helper variant=" + variant + "\n")
	os.Exit(0)
}
