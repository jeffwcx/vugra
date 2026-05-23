package vugra_test

import "github.com/vugra/vugra/pkg/vugra"

func useTextSelectionAPI(app *vugra.App) {
	selection := vugra.TextSelection{Start: 0, End: 1}
	_ = selection.Collapsed()
	_ = selection.Caret()
	app.SetTextSelection(selection)
	app.CollapseTextSelection(0)
	app.CollapseTextSelectionFor("input", 0)
	app.TextSelection()
	app.TextSelectionFor("input")
	app.SelectedText()
	app.SelectedTextFor("input")
	app.DocumentTextSelection()
	app.SelectedDocumentText()
	app.ClearDocumentTextSelection()
}
