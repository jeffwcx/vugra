<template>
  <button class="badge" @click="Pressed">
    {{ label }}
    <slot />
    <slot name="meta">default meta</slot>
  </button>
</template>

<script lang="go">
import (
    "github.com/vugra/vugra/pkg/signal"
    "github.com/vugra/vugra/pkg/vugra"
)

type State struct {
    Label signal.String `vugra:"label"`
}

func (s *State) Pressed() {
    vugra.Emit("select")
}
</script>

<style>
.badge {
  padding: 6px;
  background-color: #eef2ff;
  border-color: #c7d2fe;
  border-width: 1px;
  border-radius: 6px;
  color: #3730a3;
}
</style>
