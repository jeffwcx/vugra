<template>
  <div class="demo">
    <img class="icon" src="./icon.svg" />
    <svg class="icon" viewBox="0 0 32 32">
      <circle cx="16" cy="16" r="12" fill="#f97316" />
      <path d="M10 17l4 4 9-11" stroke="#ffffff" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </div>
</template>

<script lang="go">
package svgdemo

type State struct{}
</script>

<style>
.demo {
  display: flex;
  gap: 12px;
  padding: 16px;
  width: 120px;
  height: 64px;
}

.icon {
  width: 32px;
  height: 32px;
}
</style>
