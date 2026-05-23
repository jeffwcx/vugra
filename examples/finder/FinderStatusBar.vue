<template>
  <div class="statusbar">
    <p>{{ status }}</p>
    <p>{{ selectedSummary }}</p>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    Status signal.String `vugra:"status"`
    SelectedSummary signal.String `vugra:"selectedSummary"`
}
</script>
