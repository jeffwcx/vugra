<template>
  <div class="panel">
    <p class="label">Styled</p>
  </div>
</template>

<script lang="go">
type State struct{}
</script>

<style>
.panel {
  background-color: #123456;
  border: 3px solid #abcdef;
  border-radius: 10px;
  padding: 12px;
  width: 240px;
}

.label {
  color: #fedcba;
  font-size: 18px;
  line-height: 28px;
}
</style>
