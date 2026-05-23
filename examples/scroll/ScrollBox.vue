<template>
  <div class="scroll-demo">
    <p>Alpha</p>
    <p>Beta</p>
    <p>Gamma</p>
    <p>Delta</p>
  </div>
</template>

<script lang="go">
type State struct{}
</script>

<style>
.scroll-demo {
  background-color: #f8fafc;
  border: 2px solid #64748b;
  border-radius: 6px;
  height: 44px;
  overflow: scroll;
  padding: 4px;
  width: 180px;
}
</style>
