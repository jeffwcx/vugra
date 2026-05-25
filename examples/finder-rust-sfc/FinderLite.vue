<template>
  <div class="finder">
    <button @click="OpenDocuments">{{ path }}</button>
    <button :class="splitterClass" @hover="HoverSplitter" @drag="ResizeSidebar"></button>
    <input :value="searchQuery" @input="SearchInput" />
    <p>{{ status }}</p>
    <ul>
      <li @click="SelectRow1">{{ row1Name }}</li>
      <li @click="SelectRow2">{{ row2Name }}</li>
      <li @click="SelectRow3">{{ row3Name }}</li>
    </ul>
  </div>
</template>

<script lang="rust">
pub struct State {
    pub path: String,
    pub status: String,
    pub selectedSummary: String,
    pub row1Name: String,
    pub row1Kind: String,
    pub row2Name: String,
    pub row2Kind: String,
    pub row3Name: String,
    pub row3Kind: String,
    pub row1Selected: bool,
    pub row2Selected: bool,
    pub row3Selected: bool,
    pub documentsLabel: String,
    pub downloadsLabel: String,
    pub picturesLabel: String,
    pub documentsActive: bool,
    pub downloadsActive: bool,
    pub picturesActive: bool,
    pub searchQuery: String,
    pub sidebarClass: String,
    pub splitterClass: String,
}

impl State {
    pub fn Back(&mut self) {}
    pub fn Forward(&mut self) {}
    pub fn SelectRow1(&mut self) {}
    pub fn SelectRow2(&mut self) {}
    pub fn SelectRow3(&mut self) {}
    pub fn OpenDocuments(&mut self) {}
    pub fn OpenDownloads(&mut self) {}
    pub fn OpenPictures(&mut self) {}
    pub fn SelectPrevious(&mut self) {}
    pub fn SelectNext(&mut self) {}
    pub fn SearchInput(&mut self, event: Event) {}
    pub fn SearchBackspace(&mut self) {}
    pub fn SearchClear(&mut self) {}
    pub fn OpenSelected(&mut self) {}
    pub fn OpenParent(&mut self) {}
    pub fn HoverSplitter(&mut self) {}
    pub fn ResizeSidebar(&mut self, event: Event) {}
}
</script>
