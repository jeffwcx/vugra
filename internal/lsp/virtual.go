package lsp

import "github.com/vugra/vugra/internal/compiler"

type VirtualFileSet struct {
	Template    VirtualFile           `json:"template"`
	Script      VirtualFile           `json:"script"`
	Style       VirtualFile           `json:"style"`
	Metadata    VirtualFile           `json:"metadata"`
	Diagnostics []compiler.Diagnostic `json:"diagnostics,omitempty"`
}

type VirtualFile struct {
	FileName string        `json:"fileName"`
	Language string        `json:"language"`
	Content  string        `json:"content"`
	Span     compiler.Span `json:"span"`
}

func BuildVirtualFiles(name string, result *compiler.Result) VirtualFileSet {
	out := VirtualFileSet{
		Template: VirtualFile{FileName: name + ".template.html", Language: "html"},
		Script:   VirtualFile{FileName: name + ".script.go", Language: "go"},
		Style:    VirtualFile{FileName: name + ".style.css", Language: "css"},
		Metadata: VirtualFile{FileName: name + ".meta.json", Language: "json", Content: "{}"},
	}
	if result == nil {
		return out
	}
	if result.SFC != nil {
		if block := result.SFC.Template; block != nil {
			out.Template.Content = block.Content
			out.Template.Span = compiler.Span{Start: compiler.Position(block.ContentSpan.Start), End: compiler.Position(block.ContentSpan.End)}
		}
		if block := result.SFC.Script; block != nil {
			out.Script.Content = block.Content
			if block.Lang == "rust" {
				out.Script.FileName = name + ".script.rs"
				out.Script.Language = "rust"
			}
			out.Script.Span = compiler.Span{Start: compiler.Position(block.ContentSpan.Start), End: compiler.Position(block.ContentSpan.End)}
		}
		if block := result.SFC.Style; block != nil {
			out.Style.Content = block.Content
			out.Style.Span = compiler.Span{Start: compiler.Position(block.ContentSpan.Start), End: compiler.Position(block.ContentSpan.End)}
		}
	}
	out.Diagnostics = result.Diagnostics()
	return out
}
