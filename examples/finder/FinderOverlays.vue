<template>
  <div>
    <div v-if="itemMenuOpen" class="overlay" @click="DismissOverlay">
      <div class="menu">
        <button class="menu-item" @click="OpenSelected">Open</button>
        <button class="menu-item" @click="BeginRename">Rename</button>
        <button class="menu-item" @click="DeleteSelected">Delete</button>
        <button class="menu-item" @click="DuplicateSelected">Duplicate</button>
      </div>
    </div>

    <div v-if="blankMenuOpen" class="overlay" @click="DismissOverlay">
      <div class="menu">
        <button class="menu-item" @click="NewFolder">New Folder</button>
        <button class="menu-item" @click="Paste">Paste</button>
        <button class="menu-item" @click="Refresh">Refresh</button>
      </div>
    </div>

    <div v-if="previewOpen" class="dialog-layer" focus-scope="modal">
      <div class="dialog">
        <p class="dialog-title">{{ previewTitle }}</p>
        <p class="preview-copy">{{ previewBody }}</p>
        <div class="dialog-actions">
          <button class="primary" @click="ClosePreview">Done</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="go">
import "github.com/vugra/vugra/pkg/signal"

type State struct {
    ItemMenuOpen signal.Bool `vugra:"itemMenuOpen"`
    BlankMenuOpen signal.Bool `vugra:"blankMenuOpen"`
    PreviewOpen signal.Bool `vugra:"previewOpen"`
    PreviewTitle signal.String `vugra:"previewTitle"`
    PreviewBody signal.String `vugra:"previewBody"`
}

func (s *State) DismissOverlay() {}
func (s *State) OpenSelected() {}
func (s *State) BeginRename() {}
func (s *State) DeleteSelected() {}
func (s *State) DuplicateSelected() {}
func (s *State) NewFolder() {}
func (s *State) Paste() {}
func (s *State) Refresh() {}
func (s *State) ClosePreview() {}
</script>
