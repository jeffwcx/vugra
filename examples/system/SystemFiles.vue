<template>
  <div class="system-demo">
    <p class="title">System Files</p>
    <input class="path" :value="path" />
    <button class="button" @click="Load">Load</button>
    <p class="status">{{ status }}</p>
  </div>
</template>

<script lang="go">
import (
    "fmt"

    "github.com/vugra/vugra/pkg/signal"
    "github.com/vugra/vugra/pkg/system"
)

type State struct {
    Path signal.String `vugra:"path"`
    Status signal.String `vugra:"status"`
}

func (s *State) Load() {
    path := s.Path.Get()
    if path == "" {
        path = "."
    }
    entries, err := system.ReadDir(path)
    if err != nil {
        s.Status.Set(err.Error())
        return
    }
    s.Status.Set(fmt.Sprintf("%d entries in %s", len(entries), path))
}
</script>

<style>
.system-demo {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
  width: 360px;
}

.title {
  font-size: 18px;
  line-height: 28px;
}

.path {
  height: 34px;
}

.button {
  height: 36px;
}

.status {
  line-height: 24px;
}
</style>
