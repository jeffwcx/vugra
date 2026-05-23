<template>
  <div v-if="filePaneVisible" class="file-pane" @click="ClearSelection" @contextmenu="ShowBlankMenu">
    <div class="file-header">
      <p class="name-col">Name</p>
      <p class="date-col">Modified</p>
      <p class="size-col">Size</p>
    </div>
    <div class="file-list" @click="FocusList">
      <button v-if="row1Visible" :class="row1Class" @hover="HoverRow1" @click="SelectRow1" @dblclick="OpenRow1" @contextmenu="ShowRow1Menu"><span class="file-name-cell"><span v-if="row1Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row1File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row1Name }}</span></span><span class="file-date-cell">{{ row1Modified }}</span><span class="file-size-cell">{{ row1Size }}</span></button>
      <input v-if="row1Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row2Visible" :class="row2Class" @hover="HoverRow2" @click="SelectRow2" @dblclick="OpenRow2" @contextmenu="ShowRow2Menu"><span class="file-name-cell"><span v-if="row2Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row2File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row2Name }}</span></span><span class="file-date-cell">{{ row2Modified }}</span><span class="file-size-cell">{{ row2Size }}</span></button>
      <input v-if="row2Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row3Visible" :class="row3Class" @hover="HoverRow3" @click="SelectRow3" @dblclick="OpenRow3" @contextmenu="ShowRow3Menu"><span class="file-name-cell"><span v-if="row3Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row3File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row3Name }}</span></span><span class="file-date-cell">{{ row3Modified }}</span><span class="file-size-cell">{{ row3Size }}</span></button>
      <input v-if="row3Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row4Visible" :class="row4Class" @hover="HoverRow4" @click="SelectRow4" @dblclick="OpenRow4" @contextmenu="ShowRow4Menu"><span class="file-name-cell"><span v-if="row4Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row4File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row4Name }}</span></span><span class="file-date-cell">{{ row4Modified }}</span><span class="file-size-cell">{{ row4Size }}</span></button>
      <input v-if="row4Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row5Visible" :class="row5Class" @hover="HoverRow5" @click="SelectRow5" @dblclick="OpenRow5" @contextmenu="ShowRow5Menu"><span class="file-name-cell"><span v-if="row5Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row5File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row5Name }}</span></span><span class="file-date-cell">{{ row5Modified }}</span><span class="file-size-cell">{{ row5Size }}</span></button>
      <input v-if="row5Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row6Visible" :class="row6Class" @hover="HoverRow6" @click="SelectRow6" @dblclick="OpenRow6" @contextmenu="ShowRow6Menu"><span class="file-name-cell"><span v-if="row6Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row6File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row6Name }}</span></span><span class="file-date-cell">{{ row6Modified }}</span><span class="file-size-cell">{{ row6Size }}</span></button>
      <input v-if="row6Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row7Visible" :class="row7Class" @hover="HoverRow7" @click="SelectRow7" @dblclick="OpenRow7" @contextmenu="ShowRow7Menu"><span class="file-name-cell"><span v-if="row7Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row7File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row7Name }}</span></span><span class="file-date-cell">{{ row7Modified }}</span><span class="file-size-cell">{{ row7Size }}</span></button>
      <input v-if="row7Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row8Visible" :class="row8Class" @hover="HoverRow8" @click="SelectRow8" @dblclick="OpenRow8" @contextmenu="ShowRow8Menu"><span class="file-name-cell"><span v-if="row8Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row8File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row8Name }}</span></span><span class="file-date-cell">{{ row8Modified }}</span><span class="file-size-cell">{{ row8Size }}</span></button>
      <input v-if="row8Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row9Visible" :class="row9Class" @hover="HoverRow9" @click="SelectRow9" @dblclick="OpenRow9" @contextmenu="ShowRow9Menu"><span class="file-name-cell"><span v-if="row9Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row9File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row9Name }}</span></span><span class="file-date-cell">{{ row9Modified }}</span><span class="file-size-cell">{{ row9Size }}</span></button>
      <input v-if="row9Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row10Visible" :class="row10Class" @hover="HoverRow10" @click="SelectRow10" @dblclick="OpenRow10" @contextmenu="ShowRow10Menu"><span class="file-name-cell"><span v-if="row10Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row10File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row10Name }}</span></span><span class="file-date-cell">{{ row10Modified }}</span><span class="file-size-cell">{{ row10Size }}</span></button>
      <input v-if="row10Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row11Visible" :class="row11Class" @hover="HoverRow11" @click="SelectRow11" @dblclick="OpenRow11" @contextmenu="ShowRow11Menu"><span class="file-name-cell"><span v-if="row11Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row11File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row11Name }}</span></span><span class="file-date-cell">{{ row11Modified }}</span><span class="file-size-cell">{{ row11Size }}</span></button>
      <input v-if="row11Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
      <button v-if="row12Visible" :class="row12Class" @hover="HoverRow12" @click="SelectRow12" @dblclick="OpenRow12" @contextmenu="ShowRow12Menu"><span class="file-name-cell"><span v-if="row12Folder" class="file-icon-slot"><FinderIconFolder /></span><span v-if="row12File" class="file-icon-slot"><FinderIconFile /></span><span>{{ row12Name }}</span></span><span class="file-date-cell">{{ row12Modified }}</span><span class="file-size-cell">{{ row12Size }}</span></button>
      <input v-if="row12Editing" class="rename-inline" :value="renameText" autofocus="true" @click="FocusRename" @keydown="RenameKey" />
    </div>
  </div>
</template>

<script lang="go">
import (
    FinderIconFile "./FinderIconFile.vue"
    FinderIconFolder "./FinderIconFolder.vue"

    "github.com/vugra/vugra/pkg/signal"
)

type State struct {
    FilePaneVisible signal.Bool `vugra:"filePaneVisible"`
    RenameText signal.String `vugra:"renameText"`
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
}

func (s *State) FocusList() {}
func (s *State) FocusRename() {}
func (s *State) RenameKey() {}
func (s *State) ClearSelection() {}
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
</script>
