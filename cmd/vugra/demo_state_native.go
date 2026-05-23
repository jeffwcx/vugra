//go:build !js || !wasm

package main

import "os"

func defaultFinderHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return home
}

func defaultFinderWorkDir(fallback string) string {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return fallback
	}
	return wd
}
