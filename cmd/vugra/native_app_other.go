//go:build !darwin

package main

import "fmt"

func runNativeApp(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	return fmt.Errorf("native-app is only available on macOS")
}
