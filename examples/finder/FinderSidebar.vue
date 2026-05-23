<template>
  <div :class="sidebarClass">
    <button class="section" @click="ToggleFavorites">
      <span v-if="favoritesOpen" class="tree-chevron"><FinderIconChevronDown /></span>
      <span v-if="favoritesClosed" class="tree-chevron"><FinderIconChevronRight /></span>
      <span>Favorites</span>
    </button>
    <div v-if="favoritesOpen" class="tree-group">
      <button :class="documentsClass" @click="OpenDocuments"><FinderIconFolder /><span>Documents</span></button>
      <button :class="downloadsClass" @click="OpenDownloads"><FinderIconDownload /><span>Downloads</span></button>
      <button :class="picturesClass" @click="OpenPictures"><FinderIconPicture /><span>Pictures</span></button>
    </div>
    <button class="section" @click="ToggleWorkspace">
      <span v-if="workspaceOpen" class="tree-chevron"><FinderIconChevronDown /></span>
      <span v-if="workspaceClosed" class="tree-chevron"><FinderIconChevronRight /></span>
      <span>Workspace</span>
    </button>
    <div v-if="workspaceOpen" class="tree-group">
      <button :class="projectAClass" @click="OpenProjectA"><FinderIconProject /><span>Current Project</span></button>
      <button :class="projectBClass" @click="OpenProjectB"><FinderIconFolder /><span>Parent Folder</span></button>
    </div>
  </div>
</template>

<script lang="go">
import (
    FinderIconChevronDown "./FinderIconChevronDown.vue"
    FinderIconChevronRight "./FinderIconChevronRight.vue"
    FinderIconDownload "./FinderIconDownload.vue"
    FinderIconFolder "./FinderIconFolder.vue"
    FinderIconPicture "./FinderIconPicture.vue"
    FinderIconProject "./FinderIconProject.vue"

    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    SidebarClass signal.String `vugra:"sidebarClass"`
    FavoritesChevron signal.String `vugra:"favoritesChevron"`
    WorkspaceChevron signal.String `vugra:"workspaceChevron"`
    FavoritesOpen signal.Bool `vugra:"favoritesOpen"`
    WorkspaceOpen signal.Bool `vugra:"workspaceOpen"`
    FavoritesClosed signal.Bool `vugra:"favoritesClosed"`
    WorkspaceClosed signal.Bool `vugra:"workspaceClosed"`
    DocumentsClass signal.String `vugra:"documentsClass"`
    DownloadsClass signal.String `vugra:"downloadsClass"`
    PicturesClass signal.String `vugra:"picturesClass"`
    ProjectAClass signal.String `vugra:"projectAClass"`
    ProjectBClass signal.String `vugra:"projectBClass"`
}

func (s *State) ToggleFavorites() {}
func (s *State) ToggleWorkspace() {}
func (s *State) OpenDocuments() {}
func (s *State) OpenDownloads() {}
func (s *State) OpenPictures() {}
func (s *State) OpenProjectA() {}
func (s *State) OpenProjectB() {}
</script>
