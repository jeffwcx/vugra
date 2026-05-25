package main

import (
	"fmt"
	"os"
	"os/exec"
)

var rustFinderCommand = func(variant string) *exec.Cmd {
	if variant == "vello-device-smoke" {
		return exec.Command("cargo", "run", "-q", "-p", "finder-rust", "--features", "vugra-render-vello/vello-device", "--", variant)
	}
	if variant == "wgpu-device-smoke" {
		return exec.Command("cargo", "run", "-q", "-p", "finder-rust", "--features", "vugra-render-wgpu/wgpu-device", "--", variant)
	}
	return exec.Command("cargo", "run", "-q", "-p", "finder-rust", "--", variant)
}

func runRustFinderLite(args []string) error {
	if len(args) > 1 {
		return usage()
	}
	variant := "direct"
	if len(args) == 1 {
		variant = args[0]
	}
	if !isRustFinderVariant(variant) {
		return fmt.Errorf("rust-finder-lite variant must be direct, abi, generated-adapter-smoke, parity-summary, native, native-software, native-vello, native-wgpu, abi-native, abi-native-software, abi-native-vello, abi-native-wgpu, native-smoke, native-window-smoke, native-software-window-smoke, native-vello-window-smoke, native-wgpu-window-smoke, abi-window-smoke, abi-software-window-smoke, abi-vello-window-smoke, abi-wgpu-window-smoke, vello-device-smoke, or wgpu-device-smoke")
	}
	cmd := rustFinderCommand(variant)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run rust finder lite %s: %w", variant, err)
	}
	return nil
}

func isRustFinderVariant(variant string) bool {
	switch variant {
	case "direct", "abi", "generated-adapter-smoke", "parity-summary", "native", "direct-native", "native-software", "native-vello", "native-wgpu", "abi-native", "abi-native-software", "abi-native-vello", "abi-native-wgpu", "native-smoke", "native-window-smoke", "native-software-window-smoke", "native-vello-window-smoke", "native-wgpu-window-smoke", "abi-window-smoke", "abi-software-window-smoke", "abi-vello-window-smoke", "abi-wgpu-window-smoke", "vello-device-smoke", "wgpu-device-smoke":
		return true
	default:
		return false
	}
}
