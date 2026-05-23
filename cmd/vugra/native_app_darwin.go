//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runNativeApp(args []string) error {
	if len(args) < 2 {
		return usage()
	}
	binaryPath := args[0]
	appPath := args[1]
	launchArgs := args[2:]
	name := strings.TrimSuffix(filepath.Base(appPath), ".app")
	if name == "" {
		name = "Vugra"
	}
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		return fmt.Errorf("create app bundle: %w", err)
	}
	executableName := name
	executablePath := filepath.Join(macosDir, executableName)
	if len(launchArgs) > 0 {
		executableName = name + "-bin"
		executablePath = filepath.Join(macosDir, executableName)
	}
	if err := copyFile(binaryPath, executablePath, 0o755); err != nil {
		return err
	}
	if len(launchArgs) > 0 {
		if err := writeLauncher(filepath.Join(macosDir, name), executableName, launchArgs); err != nil {
			return err
		}
		executableName = name
	}
	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>%s</string>
  <key>CFBundleIdentifier</key>
  <string>dev.vugra.%s</string>
  <key>CFBundleName</key>
  <string>%s</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
`, executableName, strings.ToLower(name), name)
	if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), []byte(infoPlist), 0o644); err != nil {
		return fmt.Errorf("write Info.plist: %w", err)
	}
	fmt.Println(appPath)
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dst, content, mode); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func writeLauncher(path, binaryName string, args []string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("DIR=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\n")
	b.WriteString("cd ")
	b.WriteString(shellSingleQuote(workingDir))
	b.WriteByte('\n')
	b.WriteString("exec \"$DIR/")
	b.WriteString(shellEscapeDoubleQuoted(binaryName))
	b.WriteString("\"")
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(shellSingleQuote(arg))
	}
	b.WriteByte('\n')
	if err := os.WriteFile(path, []byte(b.String()), 0o755); err != nil {
		return fmt.Errorf("write launcher %s: %w", path, err)
	}
	return nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellEscapeDoubleQuoted(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "$", "\\$")
	value = strings.ReplaceAll(value, "`", "\\`")
	return value
}
