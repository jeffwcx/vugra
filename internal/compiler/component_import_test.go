package compiler_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vugra/vugra/internal/compiler"
	"github.com/vugra/vugra/internal/ir"
)

func TestCompileFileExpandsImportedVugraComponent(t *testing.T) {
	dir := t.TempDir()
	childPath := filepath.Join(dir, "Child.vue")
	parentPath := filepath.Join(dir, "Parent.vue")
	if err := os.WriteFile(childPath, []byte(`<template><p>From child</p></template>
<script lang="go">
type State struct {}
</script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parentPath, []byte(`<template><div><Child /></div></template>
<script lang="go">
import Child "./Child.vue"

type State struct {}
</script>`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := compiler.CompileFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	root := result.IR.Nodes[0].(*ir.Element)
	instance, ok := root.Children[0].(*ir.ComponentInstance)
	if !ok {
		t.Fatalf("expected component instance, got %T", root.Children[0])
	}
	childRoot := instance.Nodes[0].(*ir.Element)
	text := childRoot.Children[0].(*ir.Text)
	if text.Value != "From child" {
		t.Fatalf("child text = %q", text.Value)
	}
}

func TestCompileFileFallsBackFromLegacyVugraPathToVue(t *testing.T) {
	dir := t.TempDir()
	childPath := filepath.Join(dir, "Child.vue")
	parentPath := filepath.Join(dir, "Parent.vue")
	if err := os.WriteFile(childPath, []byte(`<template><p>From fallback child</p></template>
<script lang="go">
type State struct {}
</script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parentPath, []byte(`<template><div><Child /></div></template>
<script lang="go">
import Child "./Child.vugra"

type State struct {}
</script>`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := compiler.CompileFile(filepath.Join(dir, "Parent.vugra"))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := result.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	root := result.IR.Nodes[0].(*ir.Element)
	instance := root.Children[0].(*ir.ComponentInstance)
	childRoot := instance.Nodes[0].(*ir.Element)
	text := childRoot.Children[0].(*ir.Text)
	if text.Value != "From fallback child" {
		t.Fatalf("child text = %q", text.Value)
	}
}
