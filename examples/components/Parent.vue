<template>
  <div class="panel">
    <p>{{ title }}</p>
    <Badge :label="title" @select="OnBadgeClick">
      <span>default slot</span>
      <template #meta>
        <span>named slot</span>
      </template>
    </Badge>
  </div>
</template>

<script lang="go">
import (
    Badge "./Badge.vue"

    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    Title signal.String `vugra:"title"`
}

func (s *State) OnBadgeClick() {}
</script>

<style>
.panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 280px;
}
</style>
