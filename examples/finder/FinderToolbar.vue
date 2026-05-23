<template>
  <div class="toolbar">
    <button class="nav-button" @click="Back"><span class="nav-icon-slot"><FinderIconBack /></span></button>
    <button class="nav-button" @click="Forward"><span class="nav-icon-slot"><FinderIconForward /></span></button>
    <p class="path">{{ path }}</p>
    <input class="search" :value="search" />
  </div>
</template>

<script lang="go">
import (
    FinderIconBack "./FinderIconBack.vue"
    FinderIconForward "./FinderIconForward.vue"

    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    Path signal.String `vugra:"path"`
    Search signal.String `vugra:"search"`
}

func (s *State) Back() {}
func (s *State) Forward() {}
</script>
