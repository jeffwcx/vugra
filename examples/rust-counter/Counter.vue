<template>
  <button @click="Inc">{{ count }}</button>
</template>

<script lang="rust">
pub struct State {
    pub count: Signal<i32>,
}

impl State {
    pub fn Inc(&mut self) {}
}
</script>
