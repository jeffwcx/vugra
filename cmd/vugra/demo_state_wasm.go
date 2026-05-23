//go:build js && wasm

package main

func defaultFinderHomeDir() string {
	return "."
}

func defaultFinderWorkDir(fallback string) string {
	return "Workspace/Vugra"
}
