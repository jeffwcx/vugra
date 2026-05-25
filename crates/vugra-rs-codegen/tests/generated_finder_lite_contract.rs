pub fn generated_finder_lite_contract() -> vugra_ir::Component {
    let mut component = vugra_ir::Component::new("FinderLite");
    component.signals = vec![
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(1),
            name: "path".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(2),
            name: "status".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(3),
            name: "selectedSummary".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(13),
            name: "documentsLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(14),
            name: "downloadsLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(15),
            name: "picturesLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(16),
            name: "documentsActive".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(17),
            name: "downloadsActive".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(18),
            name: "picturesActive".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(19),
            name: "searchQuery".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(92),
            name: "favoritesLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(93),
            name: "workspaceLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(94),
            name: "favoritesOpen".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(95),
            name: "workspaceOpen".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(96),
            name: "projectALabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(97),
            name: "projectBLabel".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(98),
            name: "projectAActive".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(99),
            name: "projectBActive".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(100),
            name: "itemMenuOpen".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(101),
            name: "blankMenuOpen".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(102),
            name: "renameText".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(103),
            name: "previewOpen".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(104),
            name: "previewTitle".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(105),
            name: "previewBody".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(106),
            name: "sidebarClass".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(107),
            name: "splitterClass".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(20),
            name: "row1Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(21),
            name: "row1Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(22),
            name: "row1Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(23),
            name: "row1Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(24),
            name: "row1Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(25),
            name: "row1Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(26),
            name: "row2Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(27),
            name: "row2Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(28),
            name: "row2Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(29),
            name: "row2Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(30),
            name: "row2Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(31),
            name: "row2Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(32),
            name: "row3Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(33),
            name: "row3Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(34),
            name: "row3Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(35),
            name: "row3Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(36),
            name: "row3Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(37),
            name: "row3Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(38),
            name: "row4Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(39),
            name: "row4Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(40),
            name: "row4Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(41),
            name: "row4Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(42),
            name: "row4Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(43),
            name: "row4Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(44),
            name: "row5Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(45),
            name: "row5Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(46),
            name: "row5Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(47),
            name: "row5Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(48),
            name: "row5Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(49),
            name: "row5Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(50),
            name: "row6Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(51),
            name: "row6Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(52),
            name: "row6Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(53),
            name: "row6Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(54),
            name: "row6Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(55),
            name: "row6Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(56),
            name: "row7Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(57),
            name: "row7Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(58),
            name: "row7Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(59),
            name: "row7Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(60),
            name: "row7Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(61),
            name: "row7Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(62),
            name: "row8Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(63),
            name: "row8Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(64),
            name: "row8Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(65),
            name: "row8Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(66),
            name: "row8Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(67),
            name: "row8Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(68),
            name: "row9Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(69),
            name: "row9Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(70),
            name: "row9Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(71),
            name: "row9Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(72),
            name: "row9Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(73),
            name: "row9Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(74),
            name: "row10Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(75),
            name: "row10Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(76),
            name: "row10Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(77),
            name: "row10Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(78),
            name: "row10Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(79),
            name: "row10Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(80),
            name: "row11Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(81),
            name: "row11Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(82),
            name: "row11Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(83),
            name: "row11Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(84),
            name: "row11Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(85),
            name: "row11Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(86),
            name: "row12Name".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(87),
            name: "row12Kind".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(88),
            name: "row12Modified".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(89),
            name: "row12Size".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(90),
            name: "row12Class".to_string(),
            kind: vugra_ir::ValueKind::String,
        },
        vugra_ir::SignalDef {
            id: vugra_ir::SignalId(91),
            name: "row12Selected".to_string(),
            kind: vugra_ir::ValueKind::Bool,
        },
    ];
    component.methods = vec![
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(1),
            name: "Back".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(2),
            name: "SelectRow1".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(3),
            name: "SelectRow2".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(4),
            name: "SelectRow3".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(5),
            name: "OpenDocuments".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(6),
            name: "OpenDownloads".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(7),
            name: "OpenPictures".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(8),
            name: "SelectPrevious".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(9),
            name: "SelectNext".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(10),
            name: "SearchInput".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(11),
            name: "SearchBackspace".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(12),
            name: "SearchClear".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(13),
            name: "OpenSelected".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(14),
            name: "OpenParent".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(15),
            name: "ToggleFavorites".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(16),
            name: "ToggleWorkspace".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(17),
            name: "OpenProjectA".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(18),
            name: "OpenProjectB".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(19),
            name: "DismissOverlay".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(20),
            name: "Forward".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(33),
            name: "BeginRename".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(34),
            name: "CancelRename".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(35),
            name: "CommitRename".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(36),
            name: "DeleteSelected".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(37),
            name: "DuplicateSelected".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(38),
            name: "NewFolder".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(39),
            name: "ShowBlankMenu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(40),
            name: "ClosePreview".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(77),
            name: "ClearSelection".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(78),
            name: "Paste".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(79),
            name: "Refresh".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(80),
            name: "SelectAll".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(81),
            name: "HoverSplitter".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(82),
            name: "ResizeSidebar".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(24),
            name: "SelectRow4".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(25),
            name: "SelectRow5".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(26),
            name: "SelectRow6".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(27),
            name: "SelectRow7".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(28),
            name: "SelectRow8".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(29),
            name: "SelectRow9".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(30),
            name: "SelectRow10".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(31),
            name: "SelectRow11".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(32),
            name: "SelectRow12".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(41),
            name: "ShowRow1Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(42),
            name: "ShowRow2Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(43),
            name: "ShowRow3Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(44),
            name: "ShowRow4Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(45),
            name: "ShowRow5Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(46),
            name: "ShowRow6Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(47),
            name: "ShowRow7Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(48),
            name: "ShowRow8Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(49),
            name: "ShowRow9Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(50),
            name: "ShowRow10Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(51),
            name: "ShowRow11Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(52),
            name: "ShowRow12Menu".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(53),
            name: "HoverRow1".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(54),
            name: "HoverRow2".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(55),
            name: "HoverRow3".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(56),
            name: "HoverRow4".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(57),
            name: "HoverRow5".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(58),
            name: "HoverRow6".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(59),
            name: "HoverRow7".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(60),
            name: "HoverRow8".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(61),
            name: "HoverRow9".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(62),
            name: "HoverRow10".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(63),
            name: "HoverRow11".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(64),
            name: "HoverRow12".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(65),
            name: "OpenRow1".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(66),
            name: "OpenRow2".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(67),
            name: "OpenRow3".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(68),
            name: "OpenRow4".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(69),
            name: "OpenRow5".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(70),
            name: "OpenRow6".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(71),
            name: "OpenRow7".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(72),
            name: "OpenRow8".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(73),
            name: "OpenRow9".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(74),
            name: "OpenRow10".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(75),
            name: "OpenRow11".to_string(),
        },
        vugra_ir::MethodDef {
            id: vugra_ir::MethodId(76),
            name: "OpenRow12".to_string(),
        },
    ];
    component.rows = vec![
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(20),
            kind: vugra_ir::SignalId(21),
            modified: vugra_ir::SignalId(22),
            size: vugra_ir::SignalId(23),
            class: vugra_ir::SignalId(24),
            selected: vugra_ir::SignalId(25),
            select_method: vugra_ir::MethodId(2),
            hover_method: vugra_ir::MethodId(53),
            open_method: vugra_ir::MethodId(65),
            context_menu_method: vugra_ir::MethodId(41),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(26),
            kind: vugra_ir::SignalId(27),
            modified: vugra_ir::SignalId(28),
            size: vugra_ir::SignalId(29),
            class: vugra_ir::SignalId(30),
            selected: vugra_ir::SignalId(31),
            select_method: vugra_ir::MethodId(3),
            hover_method: vugra_ir::MethodId(54),
            open_method: vugra_ir::MethodId(66),
            context_menu_method: vugra_ir::MethodId(42),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(32),
            kind: vugra_ir::SignalId(33),
            modified: vugra_ir::SignalId(34),
            size: vugra_ir::SignalId(35),
            class: vugra_ir::SignalId(36),
            selected: vugra_ir::SignalId(37),
            select_method: vugra_ir::MethodId(4),
            hover_method: vugra_ir::MethodId(55),
            open_method: vugra_ir::MethodId(67),
            context_menu_method: vugra_ir::MethodId(43),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(38),
            kind: vugra_ir::SignalId(39),
            modified: vugra_ir::SignalId(40),
            size: vugra_ir::SignalId(41),
            class: vugra_ir::SignalId(42),
            selected: vugra_ir::SignalId(43),
            select_method: vugra_ir::MethodId(24),
            hover_method: vugra_ir::MethodId(56),
            open_method: vugra_ir::MethodId(68),
            context_menu_method: vugra_ir::MethodId(44),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(44),
            kind: vugra_ir::SignalId(45),
            modified: vugra_ir::SignalId(46),
            size: vugra_ir::SignalId(47),
            class: vugra_ir::SignalId(48),
            selected: vugra_ir::SignalId(49),
            select_method: vugra_ir::MethodId(25),
            hover_method: vugra_ir::MethodId(57),
            open_method: vugra_ir::MethodId(69),
            context_menu_method: vugra_ir::MethodId(45),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(50),
            kind: vugra_ir::SignalId(51),
            modified: vugra_ir::SignalId(52),
            size: vugra_ir::SignalId(53),
            class: vugra_ir::SignalId(54),
            selected: vugra_ir::SignalId(55),
            select_method: vugra_ir::MethodId(26),
            hover_method: vugra_ir::MethodId(58),
            open_method: vugra_ir::MethodId(70),
            context_menu_method: vugra_ir::MethodId(46),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(56),
            kind: vugra_ir::SignalId(57),
            modified: vugra_ir::SignalId(58),
            size: vugra_ir::SignalId(59),
            class: vugra_ir::SignalId(60),
            selected: vugra_ir::SignalId(61),
            select_method: vugra_ir::MethodId(27),
            hover_method: vugra_ir::MethodId(59),
            open_method: vugra_ir::MethodId(71),
            context_menu_method: vugra_ir::MethodId(47),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(62),
            kind: vugra_ir::SignalId(63),
            modified: vugra_ir::SignalId(64),
            size: vugra_ir::SignalId(65),
            class: vugra_ir::SignalId(66),
            selected: vugra_ir::SignalId(67),
            select_method: vugra_ir::MethodId(28),
            hover_method: vugra_ir::MethodId(60),
            open_method: vugra_ir::MethodId(72),
            context_menu_method: vugra_ir::MethodId(48),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(68),
            kind: vugra_ir::SignalId(69),
            modified: vugra_ir::SignalId(70),
            size: vugra_ir::SignalId(71),
            class: vugra_ir::SignalId(72),
            selected: vugra_ir::SignalId(73),
            select_method: vugra_ir::MethodId(29),
            hover_method: vugra_ir::MethodId(61),
            open_method: vugra_ir::MethodId(73),
            context_menu_method: vugra_ir::MethodId(49),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(74),
            kind: vugra_ir::SignalId(75),
            modified: vugra_ir::SignalId(76),
            size: vugra_ir::SignalId(77),
            class: vugra_ir::SignalId(78),
            selected: vugra_ir::SignalId(79),
            select_method: vugra_ir::MethodId(30),
            hover_method: vugra_ir::MethodId(62),
            open_method: vugra_ir::MethodId(74),
            context_menu_method: vugra_ir::MethodId(50),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(80),
            kind: vugra_ir::SignalId(81),
            modified: vugra_ir::SignalId(82),
            size: vugra_ir::SignalId(83),
            class: vugra_ir::SignalId(84),
            selected: vugra_ir::SignalId(85),
            select_method: vugra_ir::MethodId(31),
            hover_method: vugra_ir::MethodId(63),
            open_method: vugra_ir::MethodId(75),
            context_menu_method: vugra_ir::MethodId(51),
        },
        vugra_ir::RowBinding {
            name: vugra_ir::SignalId(86),
            kind: vugra_ir::SignalId(87),
            modified: vugra_ir::SignalId(88),
            size: vugra_ir::SignalId(89),
            class: vugra_ir::SignalId(90),
            selected: vugra_ir::SignalId(91),
            select_method: vugra_ir::MethodId(32),
            hover_method: vugra_ir::MethodId(64),
            open_method: vugra_ir::MethodId(76),
            context_menu_method: vugra_ir::MethodId(52),
        },
    ];
    component.sidebar = vec![
        vugra_ir::SidebarBinding {
            label: vugra_ir::SignalId(13),
            kind: vugra_ir::SidebarItemKind::Folder,
            active: vugra_ir::SignalId(16),
            open_method: vugra_ir::MethodId(5),
        },
        vugra_ir::SidebarBinding {
            label: vugra_ir::SignalId(14),
            kind: vugra_ir::SidebarItemKind::Download,
            active: vugra_ir::SignalId(17),
            open_method: vugra_ir::MethodId(6),
        },
        vugra_ir::SidebarBinding {
            label: vugra_ir::SignalId(15),
            kind: vugra_ir::SidebarItemKind::Picture,
            active: vugra_ir::SignalId(18),
            open_method: vugra_ir::MethodId(7),
        },
    ];
    component.sidebar_sections = vec![
        vugra_ir::SidebarSectionBinding {
            label: vugra_ir::SignalId(92),
            open: vugra_ir::SignalId(94),
            toggle_method: vugra_ir::MethodId(15),
            items: vec![
                vugra_ir::SidebarBinding {
                    label: vugra_ir::SignalId(13),
                    kind: vugra_ir::SidebarItemKind::Folder,
                    active: vugra_ir::SignalId(16),
                    open_method: vugra_ir::MethodId(5),
                },
                vugra_ir::SidebarBinding {
                    label: vugra_ir::SignalId(14),
                    kind: vugra_ir::SidebarItemKind::Download,
                    active: vugra_ir::SignalId(17),
                    open_method: vugra_ir::MethodId(6),
                },
                vugra_ir::SidebarBinding {
                    label: vugra_ir::SignalId(15),
                    kind: vugra_ir::SidebarItemKind::Picture,
                    active: vugra_ir::SignalId(18),
                    open_method: vugra_ir::MethodId(7),
                },
            ],
        },
        vugra_ir::SidebarSectionBinding {
            label: vugra_ir::SignalId(93),
            open: vugra_ir::SignalId(95),
            toggle_method: vugra_ir::MethodId(16),
            items: vec![
                vugra_ir::SidebarBinding {
                    label: vugra_ir::SignalId(96),
                    kind: vugra_ir::SidebarItemKind::Project,
                    active: vugra_ir::SignalId(98),
                    open_method: vugra_ir::MethodId(17),
                },
                vugra_ir::SidebarBinding {
                    label: vugra_ir::SignalId(97),
                    kind: vugra_ir::SidebarItemKind::Folder,
                    active: vugra_ir::SignalId(99),
                    open_method: vugra_ir::MethodId(18),
                },
            ],
        },
    ];
    component.toolbar = Some(vugra_ir::ToolbarBinding {
        back_method: vugra_ir::MethodId(1),
        forward_method: vugra_ir::MethodId(20),
    });
    component.splitter = Some(vugra_ir::SplitterBinding {
        sidebar_class: vugra_ir::SignalId(106),
        splitter_class: vugra_ir::SignalId(107),
        hover_method: vugra_ir::MethodId(81),
        drag_method: vugra_ir::MethodId(82),
    });
    component.search = Some(vugra_ir::SearchBinding {
        query: vugra_ir::SignalId(19),
        input_method: vugra_ir::MethodId(10),
        backspace_method: vugra_ir::MethodId(11),
        clear_method: vugra_ir::MethodId(12),
    });
    component.overlays = Some(vugra_ir::OverlayBinding {
        item_menu_open: vugra_ir::SignalId(100),
        blank_menu_open: vugra_ir::SignalId(101),
        rename_text: vugra_ir::SignalId(102),
        preview_open: vugra_ir::SignalId(103),
        preview_title: vugra_ir::SignalId(104),
        preview_body: vugra_ir::SignalId(105),
        open_selected_method: vugra_ir::MethodId(13),
        begin_rename_method: vugra_ir::MethodId(33),
        cancel_rename_method: vugra_ir::MethodId(34),
        commit_rename_method: vugra_ir::MethodId(35),
        delete_selected_method: vugra_ir::MethodId(36),
        duplicate_selected_method: vugra_ir::MethodId(37),
        new_folder_method: vugra_ir::MethodId(38),
        dismiss_overlay_method: vugra_ir::MethodId(19),
        clear_selection_method: vugra_ir::MethodId(77),
        show_blank_menu_method: vugra_ir::MethodId(39),
        paste_method: vugra_ir::MethodId(78),
        refresh_method: vugra_ir::MethodId(79),
        close_preview_method: vugra_ir::MethodId(40),
    });
    component
}
