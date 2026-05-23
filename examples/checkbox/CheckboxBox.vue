<template>
  <div class="panel">
    <input class="check" type="checkbox" :checked="enabled" />
    <p>{{ enabled }}</p>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Enabled signal.Bool `vugra:"enabled"`
}
</script>

<style>
.panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 180px;
}

.check {
  height: 22px;
  width: 22px;
}
</style>
