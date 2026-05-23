<template>
  <div class="finder" @click="DismissOverlay">
    <FinderToolbar :path="path" :search="search" />

    <div class="main">
      <FinderSidebar
        :sidebarClass="sidebarClass"
        :favoritesChevron="favoritesChevron"
        :workspaceChevron="workspaceChevron"
        :favoritesOpen="favoritesOpen"
        :workspaceOpen="workspaceOpen"
        :favoritesClosed="favoritesClosed"
        :workspaceClosed="workspaceClosed"
        :documentsClass="documentsClass"
        :downloadsClass="downloadsClass"
        :picturesClass="picturesClass"
        :projectAClass="projectAClass"
        :projectBClass="projectBClass"
      />
      <button :class="splitterClass" @hover="HoverSplitter" @drag="ResizeSidebar"></button>

      <div class="workspace">
        <FinderFilePane
          :filePaneVisible="filePaneVisible"
          :renameText="renameText"
          :row1="row1" :row1Name="row1Name" :row1Modified="row1Modified" :row1Size="row1Size" :row1Class="row1Class" :row1Visible="row1Visible" :row1Editing="row1Editing"
          :row2="row2" :row2Name="row2Name" :row2Modified="row2Modified" :row2Size="row2Size" :row2Class="row2Class" :row2Visible="row2Visible" :row2Editing="row2Editing"
          :row3="row3" :row3Name="row3Name" :row3Modified="row3Modified" :row3Size="row3Size" :row3Class="row3Class" :row3Visible="row3Visible" :row3Editing="row3Editing"
          :row4="row4" :row4Name="row4Name" :row4Modified="row4Modified" :row4Size="row4Size" :row4Class="row4Class" :row4Visible="row4Visible" :row4Editing="row4Editing"
          :row5="row5" :row5Name="row5Name" :row5Modified="row5Modified" :row5Size="row5Size" :row5Class="row5Class" :row5Visible="row5Visible" :row5Editing="row5Editing"
          :row6="row6" :row6Name="row6Name" :row6Modified="row6Modified" :row6Size="row6Size" :row6Class="row6Class" :row6Visible="row6Visible" :row6Editing="row6Editing"
          :row7="row7" :row7Name="row7Name" :row7Modified="row7Modified" :row7Size="row7Size" :row7Class="row7Class" :row7Visible="row7Visible" :row7Editing="row7Editing"
          :row8="row8" :row8Name="row8Name" :row8Modified="row8Modified" :row8Size="row8Size" :row8Class="row8Class" :row8Visible="row8Visible" :row8Editing="row8Editing"
          :row9="row9" :row9Name="row9Name" :row9Modified="row9Modified" :row9Size="row9Size" :row9Class="row9Class" :row9Visible="row9Visible" :row9Editing="row9Editing"
          :row10="row10" :row10Name="row10Name" :row10Modified="row10Modified" :row10Size="row10Size" :row10Class="row10Class" :row10Visible="row10Visible" :row10Editing="row10Editing"
          :row11="row11" :row11Name="row11Name" :row11Modified="row11Modified" :row11Size="row11Size" :row11Class="row11Class" :row11Visible="row11Visible" :row11Editing="row11Editing"
          :row12="row12" :row12Name="row12Name" :row12Modified="row12Modified" :row12Size="row12Size" :row12Class="row12Class" :row12Visible="row12Visible" :row12Editing="row12Editing"
          :row1Folder="row1Folder" :row1File="row1File"
          :row2Folder="row2Folder" :row2File="row2File"
          :row3Folder="row3Folder" :row3File="row3File"
          :row4Folder="row4Folder" :row4File="row4File"
          :row5Folder="row5Folder" :row5File="row5File"
          :row6Folder="row6Folder" :row6File="row6File"
          :row7Folder="row7Folder" :row7File="row7File"
          :row8Folder="row8Folder" :row8File="row8File"
          :row9Folder="row9Folder" :row9File="row9File"
          :row10Folder="row10Folder" :row10File="row10File"
          :row11Folder="row11Folder" :row11File="row11File"
          :row12Folder="row12Folder" :row12File="row12File"
        />

        <FinderOverlays
          :itemMenuOpen="itemMenuOpen"
          :blankMenuOpen="blankMenuOpen"
          :previewOpen="previewOpen"
          :previewTitle="previewTitle"
          :previewBody="previewBody"
        />
      </div>
    </div>

    <FinderStatusBar :status="status" :selectedSummary="selectedSummary" />

  </div>
</template>

<script lang="go">
import (
    FinderFilePane "./FinderFilePane.vue"
    FinderOverlays "./FinderOverlays.vue"
    FinderSidebar "./FinderSidebar.vue"
    FinderStatusBar "./FinderStatusBar.vue"
    FinderToolbar "./FinderToolbar.vue"

    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    Path signal.String `vugra:"path"`
    Search signal.String `vugra:"search"`
    Status signal.String `vugra:"status"`
    SelectedSummary signal.String `vugra:"selectedSummary"`
    FavoritesOpen signal.Bool `vugra:"favoritesOpen"`
    WorkspaceOpen signal.Bool `vugra:"workspaceOpen"`
    FavoritesClosed signal.Bool `vugra:"favoritesClosed"`
    WorkspaceClosed signal.Bool `vugra:"workspaceClosed"`
    FavoritesChevron signal.String `vugra:"favoritesChevron"`
    WorkspaceChevron signal.String `vugra:"workspaceChevron"`
    DocumentsClass signal.String `vugra:"documentsClass"`
    DownloadsClass signal.String `vugra:"downloadsClass"`
    PicturesClass signal.String `vugra:"picturesClass"`
    ProjectAClass signal.String `vugra:"projectAClass"`
    ProjectBClass signal.String `vugra:"projectBClass"`
    SidebarClass signal.String `vugra:"sidebarClass"`
    SplitterClass signal.String `vugra:"splitterClass"`
    Row1 signal.String `vugra:"row1"`
    Row2 signal.String `vugra:"row2"`
    Row3 signal.String `vugra:"row3"`
    Row4 signal.String `vugra:"row4"`
    Row5 signal.String `vugra:"row5"`
    Row6 signal.String `vugra:"row6"`
    Row7 signal.String `vugra:"row7"`
    Row8 signal.String `vugra:"row8"`
    Row9 signal.String `vugra:"row9"`
    Row10 signal.String `vugra:"row10"`
    Row11 signal.String `vugra:"row11"`
    Row12 signal.String `vugra:"row12"`
    Row1Name signal.String `vugra:"row1Name"`
    Row1Modified signal.String `vugra:"row1Modified"`
    Row1Size signal.String `vugra:"row1Size"`
    Row2Name signal.String `vugra:"row2Name"`
    Row2Modified signal.String `vugra:"row2Modified"`
    Row2Size signal.String `vugra:"row2Size"`
    Row3Name signal.String `vugra:"row3Name"`
    Row3Modified signal.String `vugra:"row3Modified"`
    Row3Size signal.String `vugra:"row3Size"`
    Row4Name signal.String `vugra:"row4Name"`
    Row4Modified signal.String `vugra:"row4Modified"`
    Row4Size signal.String `vugra:"row4Size"`
    Row5Name signal.String `vugra:"row5Name"`
    Row5Modified signal.String `vugra:"row5Modified"`
    Row5Size signal.String `vugra:"row5Size"`
    Row6Name signal.String `vugra:"row6Name"`
    Row6Modified signal.String `vugra:"row6Modified"`
    Row6Size signal.String `vugra:"row6Size"`
    Row7Name signal.String `vugra:"row7Name"`
    Row7Modified signal.String `vugra:"row7Modified"`
    Row7Size signal.String `vugra:"row7Size"`
    Row8Name signal.String `vugra:"row8Name"`
    Row8Modified signal.String `vugra:"row8Modified"`
    Row8Size signal.String `vugra:"row8Size"`
    Row9Name signal.String `vugra:"row9Name"`
    Row9Modified signal.String `vugra:"row9Modified"`
    Row9Size signal.String `vugra:"row9Size"`
    Row10Name signal.String `vugra:"row10Name"`
    Row10Modified signal.String `vugra:"row10Modified"`
    Row10Size signal.String `vugra:"row10Size"`
    Row11Name signal.String `vugra:"row11Name"`
    Row11Modified signal.String `vugra:"row11Modified"`
    Row11Size signal.String `vugra:"row11Size"`
    Row12Name signal.String `vugra:"row12Name"`
    Row12Modified signal.String `vugra:"row12Modified"`
    Row12Size signal.String `vugra:"row12Size"`
    Row1Class signal.String `vugra:"row1Class"`
    Row2Class signal.String `vugra:"row2Class"`
    Row3Class signal.String `vugra:"row3Class"`
    Row4Class signal.String `vugra:"row4Class"`
    Row5Class signal.String `vugra:"row5Class"`
    Row6Class signal.String `vugra:"row6Class"`
    Row7Class signal.String `vugra:"row7Class"`
    Row8Class signal.String `vugra:"row8Class"`
    Row9Class signal.String `vugra:"row9Class"`
    Row10Class signal.String `vugra:"row10Class"`
    Row11Class signal.String `vugra:"row11Class"`
    Row12Class signal.String `vugra:"row12Class"`
    Row1Visible signal.Bool `vugra:"row1Visible"`
    Row2Visible signal.Bool `vugra:"row2Visible"`
    Row3Visible signal.Bool `vugra:"row3Visible"`
    Row4Visible signal.Bool `vugra:"row4Visible"`
    Row5Visible signal.Bool `vugra:"row5Visible"`
    Row6Visible signal.Bool `vugra:"row6Visible"`
    Row7Visible signal.Bool `vugra:"row7Visible"`
    Row8Visible signal.Bool `vugra:"row8Visible"`
    Row9Visible signal.Bool `vugra:"row9Visible"`
    Row10Visible signal.Bool `vugra:"row10Visible"`
    Row11Visible signal.Bool `vugra:"row11Visible"`
    Row12Visible signal.Bool `vugra:"row12Visible"`
    Row1Folder signal.Bool `vugra:"row1Folder"`
    Row2Folder signal.Bool `vugra:"row2Folder"`
    Row3Folder signal.Bool `vugra:"row3Folder"`
    Row4Folder signal.Bool `vugra:"row4Folder"`
    Row5Folder signal.Bool `vugra:"row5Folder"`
    Row6Folder signal.Bool `vugra:"row6Folder"`
    Row7Folder signal.Bool `vugra:"row7Folder"`
    Row8Folder signal.Bool `vugra:"row8Folder"`
    Row9Folder signal.Bool `vugra:"row9Folder"`
    Row10Folder signal.Bool `vugra:"row10Folder"`
    Row11Folder signal.Bool `vugra:"row11Folder"`
    Row12Folder signal.Bool `vugra:"row12Folder"`
    Row1File signal.Bool `vugra:"row1File"`
    Row2File signal.Bool `vugra:"row2File"`
    Row3File signal.Bool `vugra:"row3File"`
    Row4File signal.Bool `vugra:"row4File"`
    Row5File signal.Bool `vugra:"row5File"`
    Row6File signal.Bool `vugra:"row6File"`
    Row7File signal.Bool `vugra:"row7File"`
    Row8File signal.Bool `vugra:"row8File"`
    Row9File signal.Bool `vugra:"row9File"`
    Row10File signal.Bool `vugra:"row10File"`
    Row11File signal.Bool `vugra:"row11File"`
    Row12File signal.Bool `vugra:"row12File"`
    Row1Editing signal.Bool `vugra:"row1Editing"`
    Row2Editing signal.Bool `vugra:"row2Editing"`
    Row3Editing signal.Bool `vugra:"row3Editing"`
    Row4Editing signal.Bool `vugra:"row4Editing"`
    Row5Editing signal.Bool `vugra:"row5Editing"`
    Row6Editing signal.Bool `vugra:"row6Editing"`
    Row7Editing signal.Bool `vugra:"row7Editing"`
    Row8Editing signal.Bool `vugra:"row8Editing"`
    Row9Editing signal.Bool `vugra:"row9Editing"`
    Row10Editing signal.Bool `vugra:"row10Editing"`
    Row11Editing signal.Bool `vugra:"row11Editing"`
    Row12Editing signal.Bool `vugra:"row12Editing"`
    FilePaneVisible signal.Bool `vugra:"filePaneVisible"`
    ItemMenuOpen signal.Bool `vugra:"itemMenuOpen"`
    BlankMenuOpen signal.Bool `vugra:"blankMenuOpen"`
    RenameText signal.String `vugra:"renameText"`
    PreviewOpen signal.Bool `vugra:"previewOpen"`
    PreviewTitle signal.String `vugra:"previewTitle"`
    PreviewBody signal.String `vugra:"previewBody"`
}

func (s *State) Back() {}
func (s *State) Forward() {}
func (s *State) ToggleFavorites() {}
func (s *State) ToggleWorkspace() {}
func (s *State) OpenDocuments() {}
func (s *State) OpenDownloads() {}
func (s *State) OpenPictures() {}
func (s *State) OpenProjectA() {}
func (s *State) OpenProjectB() {}
func (s *State) HoverSplitter() {}
func (s *State) ResizeSidebar() {}
func (s *State) FocusList() {}
func (s *State) FocusRename() {}
func (s *State) RenameKey() {}
func (s *State) ClearSelection() {}
func (s *State) DismissOverlay() {}
func (s *State) ShowBlankMenu() {}
func (s *State) SelectRow1() {}
func (s *State) SelectRow2() {}
func (s *State) SelectRow3() {}
func (s *State) SelectRow4() {}
func (s *State) SelectRow5() {}
func (s *State) SelectRow6() {}
func (s *State) SelectRow7() {}
func (s *State) SelectRow8() {}
func (s *State) SelectRow9() {}
func (s *State) SelectRow10() {}
func (s *State) SelectRow11() {}
func (s *State) SelectRow12() {}
func (s *State) HoverRow1() {}
func (s *State) HoverRow2() {}
func (s *State) HoverRow3() {}
func (s *State) HoverRow4() {}
func (s *State) HoverRow5() {}
func (s *State) HoverRow6() {}
func (s *State) HoverRow7() {}
func (s *State) HoverRow8() {}
func (s *State) HoverRow9() {}
func (s *State) HoverRow10() {}
func (s *State) HoverRow11() {}
func (s *State) HoverRow12() {}
func (s *State) OpenRow1() {}
func (s *State) OpenRow2() {}
func (s *State) OpenRow3() {}
func (s *State) OpenRow4() {}
func (s *State) OpenRow5() {}
func (s *State) OpenRow6() {}
func (s *State) OpenRow7() {}
func (s *State) OpenRow8() {}
func (s *State) OpenRow9() {}
func (s *State) OpenRow10() {}
func (s *State) OpenRow11() {}
func (s *State) OpenRow12() {}
func (s *State) ShowRow1Menu() {}
func (s *State) ShowRow2Menu() {}
func (s *State) ShowRow3Menu() {}
func (s *State) ShowRow4Menu() {}
func (s *State) ShowRow5Menu() {}
func (s *State) ShowRow6Menu() {}
func (s *State) ShowRow7Menu() {}
func (s *State) ShowRow8Menu() {}
func (s *State) ShowRow9Menu() {}
func (s *State) ShowRow10Menu() {}
func (s *State) ShowRow11Menu() {}
func (s *State) ShowRow12Menu() {}
func (s *State) OpenSelected() {}
func (s *State) BeginRename() {}
func (s *State) CancelRename() {}
func (s *State) CommitRename() {}
func (s *State) DeleteSelected() {}
func (s *State) DuplicateSelected() {}
func (s *State) NewFolder() {}
func (s *State) Paste() {}
func (s *State) Refresh() {}
func (s *State) ClosePreview() {}
</script>

<style>
.finder {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background-color: #f7f7f8;
  color: #1f2328;
  font-family: system-ui;
  font-size: 13px;
  line-height: 18px;
  overflow: hidden;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 52px;
  padding: 10px;
  padding-left: env(vugra-window-controls-left, 10px);
  background-color: #f2f2f4;
  border-width: 1px;
  border-color: #d8d8dc;
}

.nav-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 30px;
  border-radius: 6px;
  border-width: 1px;
  border-color: #c7c7cc;
  background-color: #ffffff;
  color: #25292e;
}

.nav-icon {
  width: 18px;
  height: 30px;
}

.nav-icon-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 30px;
}

.path {
  flex: 1;
  height: 30px;
  padding: 6px;
  border-radius: 6px;
  background-color: #ffffff;
  border-width: 1px;
  border-color: #d8d8dc;
  color: #4b5563;
}

.search {
  width: 220px;
  height: 30px;
  padding: 6px;
  border-radius: 6px;
  border-width: 1px;
  border-color: #c7c7cc;
  background-color: #ffffff;
}

.main {
  display: flex;
  flex: 1;
  min-height: 0px;
  background-color: #ffffff;
}

.sidebar {
  width: 240px;
  height: 100%;
  padding: 12px;
  background-color: #ececf0;
  border-width: 1px;
  border-color: #d1d1d6;
  overflow: scroll;
}

.sidebar-200 {
  width: 200px;
  height: 100%;
  padding: 12px;
  background-color: #ececf0;
  border-width: 1px;
  border-color: #d1d1d6;
  overflow: scroll;
}

.sidebar-280 {
  width: 280px;
  height: 100%;
  padding: 12px;
  background-color: #ececf0;
  border-width: 1px;
  border-color: #d1d1d6;
  overflow: scroll;
}

.sidebar-320 {
  width: 320px;
  height: 100%;
  padding: 12px;
  background-color: #ececf0;
  border-width: 1px;
  border-color: #d1d1d6;
  overflow: scroll;
}

.splitter {
  width: 6px;
  height: 100%;
  border-width: 0px;
  background-color: #d1d1d6;
}

.splitter-hover {
  width: 6px;
  height: 100%;
  border-width: 0px;
  background-color: #8bb8f7;
}

.file-pane {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 360px;
  height: 100%;
  background-color: #ffffff;
  overflow: hidden;
}

.workspace {
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: 1fr;
  flex: 1;
  min-width: 360px;
  height: 100%;
  background-color: #ffffff;
}

.section {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 28px;
  border-width: 0px;
  background-color: #ececf0;
  color: #6b7280;
  text-align: left;
  font-weight: 600;
}

.tree-chevron {
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tree-group {
  display: flex;
  flex-direction: column;
  gap: 3px;
  margin: 0px;
}

.tree-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 28px;
  padding: 6px;
  border-width: 0px;
  border-radius: 6px;
  background-color: #ececf0;
  color: #374151;
  text-align: left;
}

.tree-item-selected {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 28px;
  padding: 6px;
  border-width: 0px;
  border-radius: 6px;
  background-color: #d9e8ff;
  color: #0f3d74;
  text-align: left;
}

.finder-icon {
  width: 18px;
  height: 18px;
}

.file-icon-slot {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-header {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  height: 34px;
  padding: 8px;
  background-color: #fbfbfc;
  border-width: 1px;
  border-color: #e4e4e7;
  color: #6b7280;
  font-weight: 600;
}

.name-col {
  grid-column: 1;
  height: 18px;
}

.date-col {
  grid-column: 2;
  height: 18px;
}

.size-col {
  grid-column: 3;
  height: 18px;
}

.file-list {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0px;
  padding: 6px;
  overflow: scroll;
  background-color: #ffffff;
}

.file-row {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 0px;
  border-radius: 5px;
  background-color: #ffffff;
  color: #1f2937;
  text-align: left;
  font-size: 13px;
}

.file-row-hover {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 0px;
  border-radius: 5px;
  background-color: #f4f7fb;
  color: #1f2937;
  text-align: left;
  font-size: 13px;
}

.file-row-selected {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 0px;
  border-radius: 5px;
  background-color: #0a84ff;
  color: #ffffff;
  text-align: left;
  font-size: 13px;
}

.file-row-editing {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 1px;
  border-color: #0a84ff;
  border-radius: 5px;
  background-color: #eef6ff;
  color: #0f3d74;
  text-align: left;
  font-size: 13px;
}

.rename-inline {
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 1px;
  border-color: #0a84ff;
  border-radius: 5px;
  background-color: #ffffff;
  color: #111827;
  font-size: 13px;
}

.file-row-focus {
  display: grid;
  grid-template-columns: 1fr 150px 90px;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 1px;
  border-color: #6aa8ff;
  border-radius: 5px;
  background-color: #edf6ff;
  color: #0f3d74;
  text-align: left;
  font-size: 13px;
}

.file-name-cell {
  display: flex;
  grid-column: 1;
  align-items: center;
  gap: 8px;
  min-width: 0px;
  height: 18px;
}

.file-date-cell {
  grid-column: 2;
  height: 18px;
  color: #4b5563;
}

.file-size-cell {
  grid-column: 3;
  height: 18px;
  color: #4b5563;
  text-align: right;
}

.statusbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 28px;
  padding: 6px;
  background-color: #f5f5f7;
  border-width: 1px;
  border-color: #d8d8dc;
  color: #4b5563;
}

.overlay {
  display: flex;
  grid-column: 1;
  grid-row: 1;
  width: 554px;
  height: 520px;
  padding: 14px;
  background-color: #f7f9fc;
  border-width: 1px;
  border-color: #c7c7cc;
}

.menu {
  width: 180px;
  padding: 6px;
  border-width: 1px;
  border-color: #c7c7cc;
  border-radius: 8px;
  background-color: #ffffff;
}

.menu-item {
  width: 100%;
  height: 30px;
  padding: 6px;
  border-width: 0px;
  background-color: #ffffff;
  text-align: left;
  color: #1f2328;
}

.dialog-layer {
  display: flex;
  grid-column: 1;
  grid-row: 1;
  width: 554px;
  height: 520px;
  align-items: center;
  justify-content: center;
  background-color: #eef2f7;
  border-width: 1px;
  border-color: #d8d8dc;
}

.dialog {
  width: 360px;
  padding: 14px;
  border-width: 1px;
  border-color: #c7c7cc;
  border-radius: 8px;
  background-color: #ffffff;
}

.dialog-title {
  height: 24px;
  color: #111827;
  font-weight: 600;
  font-size: 15px;
}

.preview-copy {
  min-height: 48px;
  color: #4b5563;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.secondary {
  width: 90px;
  height: 32px;
  border-width: 1px;
  border-color: #c7c7cc;
  border-radius: 6px;
  background-color: #ffffff;
  color: #374151;
}

.primary {
  width: 90px;
  height: 32px;
  border-width: 1px;
  border-color: #0a84ff;
  border-radius: 6px;
  background-color: #0a84ff;
  color: #ffffff;
}
</style>
