//go:build !darwin || !cgo || !vuego_native_window

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const nativeAutoRelaunchEnv = "VUGRA_NATIVE_AUTORELAUNCHED"

func runNativeWindow(args []string) error {
	if len(args) != 1 {
		return usage()
	}
	if canAutoRelaunchNative() {
		return relaunchWithNativeWindowTag(append([]string{"native-window"}, args...))
	}
	return fmt.Errorf("native-window is not enabled; build on darwin with cgo and -tags vuego_native_window")
}

func runProject(args []string) error {
	if len(args) > 1 {
		return usage()
	}
	if canAutoRelaunchNative() {
		return relaunchWithNativeWindowTag(append([]string{"run"}, args...))
	}
	return fmt.Errorf("run is not enabled; build on darwin with cgo and -tags vuego_native_window")
}

func canAutoRelaunchNative() bool {
	return runtime.GOOS == "darwin" && os.Getenv("CGO_ENABLED") != "0" && os.Getenv(nativeAutoRelaunchEnv) == ""
}

func relaunchWithNativeWindowTag(args []string) error {
	commandArgs := []string{"run", "-tags", "vuego_native_window", "./cmd/vugra"}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command("go", commandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), nativeAutoRelaunchEnv+"=1")
	fmt.Fprintf(os.Stderr, "vugra %s: enabling native window backend\n", args[0])
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("native relaunch failed: go %s: %w", strings.Join(commandArgs, " "), err)
	}
	return nil
}
