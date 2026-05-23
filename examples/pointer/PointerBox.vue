<template>
  <div class="pointer-demo">
    <button class="target" @hover="Hover" @drag="Drag" @dblclick="Double" @contextmenu="Menu">Pointer</button>
    <p>{{ status }}</p>
  </div>
</template>

<script lang="go">
import (
    "fmt"

    "github.com/vugra/vugra/pkg/signal"
    "github.com/vugra/vugra/pkg/vugra"
)

type State struct {
    Status signal.String `vugra:"status"`
}

func (s *State) Hover() {
    s.Status.Set("hover")
}

func (s *State) Drag(event vugra.Event) {
    s.Status.Set(fmt.Sprintf("drag %d", int(event.DeltaX)))
}

func (s *State) Double() {
    s.Status.Set("double")
}

func (s *State) Menu() {
    s.Status.Set("menu")
}
</script>

<style>
.pointer-demo {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 240px;
}

.target {
  width: 120px;
  height: 42px;
  background-color: #ffffff;
  border-width: 1px;
  border-color: #2563eb;
  border-radius: 6px;
  color: #1f2937;
}
</style>
