<template>
  <div class="counter">
    <p>{{ count }}</p>
    <button class="button" @click="Inc">+</button>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Count signal.Int `vugra:"count"`
}

func (s *State) Inc() {
    s.Count.Set(s.Count.Get() + 1)
}
</script>

<style>
.counter {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 260px;
}

.button {
  height: 48px;
}
</style>
