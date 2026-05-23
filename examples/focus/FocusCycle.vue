<template>
  <div class="focus-demo">
    <button class="button" @click="First">First</button>
    <button class="button" @click="Second">Second</button>
    <p>{{ selected }}</p>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Selected signal.String `vugra:"selected"`
}

func (s *State) First() {
    s.Selected.Set("first")
}

func (s *State) Second() {
    s.Selected.Set("second")
}
</script>

<style>
.focus-demo {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 260px;
}

.button {
  height: 40px;
}
</style>
