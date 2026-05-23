<template>
  <div class="panel">
    <p class="label">Opacity</p>
  </div>
</template>

<script lang="go">
type State struct{}
</script>

<style>
.panel {
  background-color: #2563eb;
  border: 2px solid #0f172a;
  border-radius: 8px;
  opacity: 0.5;
  padding: 12px;
  width: 180px;
}

.label {
  color: #ffffff;
  font-size: 18px;
  line-height: 28px;
}
</style>
