package compiler

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vugra/vugra/internal/componentfile"
	"github.com/vugra/vugra/internal/goanalysis"
	"github.com/vugra/vugra/internal/ir"
	"github.com/vugra/vugra/internal/sfc"
	"github.com/vugra/vugra/internal/style"
	"github.com/vugra/vugra/internal/template"
)

type Result struct {
	SFC      *sfc.File           `json:"sfc"`
	Template *template.Document  `json:"template,omitempty"`
	Style    *style.Stylesheet   `json:"style,omitempty"`
	StyleCSS string              `json:"styleCSS,omitempty"`
	Go       goanalysis.Metadata `json:"go"`
	IR       *ir.Component       `json:"ir,omitempty"`
	Imports  []ImportResult      `json:"-"`
}

type resolvedImport struct {
	Import ir.Import
	Result *Result
}

type ImportResult struct {
	Alias  string
	Path   string
	Result *Result
}

func CompileFile(path string) (*Result, error) {
	resolvedPath, source, err := readComponentFile(path)
	if err != nil {
		return nil, err
	}
	return compile(resolvedPath, source, map[string]bool{}), nil
}

func Compile(path string, source []byte) *Result {
	return compile(path, source, map[string]bool{})
}

func compile(path string, source []byte, stack map[string]bool) *Result {
	sfcFile := sfc.Parse(path, source)
	result := &Result{SFC: sfcFile}
	if sfcFile.Template != nil {
		result.Template = template.ParseWithBase(sfcFile.Template.Content, template.BasePosition{
			Offset: sfcFile.Template.ContentSpan.Start.Offset,
			Line:   sfcFile.Template.ContentSpan.Start.Line,
			Column: sfcFile.Template.ContentSpan.Start.Column,
		})
	}
	if sfcFile.Script != nil {
		result.Go = goanalysis.Analyze(sfcFile.Script.Content, goanalysis.BasePosition{
			Offset: sfcFile.Script.ContentSpan.Start.Offset,
			Line:   sfcFile.Script.ContentSpan.Start.Line,
			Column: sfcFile.Script.ContentSpan.Start.Column,
		})
	}
	if sfcFile.Style != nil {
		result.StyleCSS = sfcFile.Style.Content
		result.Style = style.Parse(sfcFile.Style.Content, style.BasePosition{
			Offset: sfcFile.Style.ContentSpan.Start.Offset,
			Line:   sfcFile.Style.ContentSpan.Start.Line,
			Column: sfcFile.Style.ContentSpan.Start.Column,
		})
	}
	resolvedImports := resolveComponentImports(path, result.Go, stack)
	result.Imports = importResults(resolvedImports)
	mergeImportedStyles(result, resolvedImports)
	result.IR = ir.Build(ir.BuildInput{
		Name:     path,
		Template: result.Template,
		Go:       result.Go,
		Imports:  irImports(resolvedImports),
	})
	return result
}

func importResults(imports []resolvedImport) []ImportResult {
	out := make([]ImportResult, 0, len(imports))
	for _, imported := range imports {
		out = append(out, ImportResult{
			Alias:  imported.Import.Alias,
			Path:   imported.Import.Path,
			Result: imported.Result,
		})
	}
	return out
}

func resolveComponentImports(path string, metadata goanalysis.Metadata, stack map[string]bool) []resolvedImport {
	var imports []resolvedImport
	for _, imported := range metadata.Imports {
		if imported.Path == "" {
			continue
		}
		resolved := imported.Path
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(filepath.Dir(path), resolved)
		}
		resolved = filepath.Clean(resolved)
		if stack[resolved] {
			out := ir.Import{Alias: imported.Alias, Path: resolved}
			imports = append(imports, resolvedImport{Import: out})
			continue
		}
		resolved, source, err := readComponentFile(resolved)
		out := ir.Import{Alias: imported.Alias, Path: resolved}
		var child *Result
		if err == nil {
			nextStack := copyStack(stack)
			nextStack[filepath.Clean(path)] = true
			nextStack[resolved] = true
			child = compile(resolved, source, nextStack)
			out.Component = child.IR
		}
		imports = append(imports, resolvedImport{Import: out, Result: child})
	}
	return imports
}

func readComponentFile(path string) (string, []byte, error) {
	source, err := os.ReadFile(path)
	if err == nil {
		return path, source, nil
	}
	if !os.IsNotExist(err) || !componentfile.IsComponentPath(path) {
		return path, nil, err
	}
	alternate := componentfile.AlternatePath(path)
	if alternate == path {
		return path, nil, err
	}
	source, alternateErr := os.ReadFile(alternate)
	if alternateErr == nil {
		return alternate, source, nil
	}
	return path, nil, err
}

func irImports(imports []resolvedImport) []ir.Import {
	out := make([]ir.Import, 0, len(imports))
	for _, imported := range imports {
		out = append(out, imported.Import)
	}
	return out
}

func copyStack(stack map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, value := range stack {
		out[key] = value
	}
	return out
}

func mergeImportedStyles(result *Result, imports []resolvedImport) {
	for _, imported := range imports {
		childResult := imported.Result
		if childResult == nil {
			continue
		}
		if childResult.Style != nil {
			if result.Style == nil {
				result.Style = &style.Stylesheet{}
			}
			result.Style.Rules = append(result.Style.Rules, childResult.Style.Rules...)
		}
		if strings.TrimSpace(childResult.StyleCSS) != "" {
			if strings.TrimSpace(result.StyleCSS) == "" {
				result.StyleCSS = childResult.StyleCSS
			} else {
				result.StyleCSS += "\n" + childResult.StyleCSS
			}
		}
	}
}

func (r *Result) Diagnostics() []Diagnostic {
	var out []Diagnostic
	if r.SFC != nil {
		for _, diag := range r.SFC.Diagnostics {
			out = append(out, Diagnostic{
				Code:     diag.Code,
				Message:  diag.Message,
				Severity: diag.Severity,
				Span: Span{
					Start: Position(diag.Span.Start),
					End:   Position(diag.Span.End),
				},
			})
		}
	}
	if r.Template != nil {
		for _, diag := range r.Template.Diagnostics {
			out = append(out, Diagnostic{
				Code:     diag.Code,
				Message:  diag.Message,
				Severity: diag.Severity,
				Span: Span{
					Start: Position(diag.Span.Start),
					End:   Position(diag.Span.End),
				},
			})
		}
	}
	if r.Style != nil {
		for _, diag := range r.Style.Diagnostics {
			out = append(out, Diagnostic{
				Code:     diag.Code,
				Message:  diag.Message,
				Severity: diag.Severity,
				Span: Span{
					Start: Position(diag.Span.Start),
					End:   Position(diag.Span.End),
				},
			})
		}
	}
	for _, diag := range r.Go.Diagnostics {
		out = append(out, Diagnostic{
			Code:     diag.Code,
			Message:  diag.Message,
			Severity: diag.Severity,
			Span: Span{
				Start: Position(diag.Span.Start),
				End:   Position(diag.Span.End),
			},
		})
	}
	if r.IR != nil {
		for _, diag := range r.IR.Diagnostics {
			out = append(out, Diagnostic{
				Code:     diag.Code,
				Message:  diag.Message,
				Severity: diag.Severity,
				Span: Span{
					Start: Position(diag.Span.Start),
					End:   Position(diag.Span.End),
				},
			})
		}
	}
	return out
}

type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Span     Span   `json:"span"`
}
