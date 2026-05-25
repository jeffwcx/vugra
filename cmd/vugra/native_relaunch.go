package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const nativeAutoRelaunchEnv = "VUGRA_NATIVE_AUTORELAUNCHED"

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
