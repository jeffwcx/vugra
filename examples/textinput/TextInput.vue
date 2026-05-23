<template>
  <div class="text-input-demo">
    <input class="field" :value="name" />
    <p>{{ name }}</p>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Name signal.String `vugra:"name"`
}
</script>

<style>
.text-input-demo {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  width: 320px;
}

.field {
  height: 34px;
}
</style>
