package godocs

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/format"
	"go/token"
	"strings"

	"lore-master/libs/content-generation/markdownwriting"
)

// renderPackage writes a package's page: its import path, its comment, then constants,
// variables, functions and types with their constructors and methods, as `go doc` orders
// them. Each declaration is shown without its body or its comment, which follow it as text.
func renderPackage(p goPackage) string {
	docs := p.docs
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", p.folder.ImportPath)
	if docs.Name == "main" {
		out.WriteString("This is a command (package main), not an importable package.\n\n")
	} else {
		out.WriteString(markdownwriting.CodeBlock("go", fmt.Sprintf("import %q", p.folder.ImportPath)) + "\n")
	}
	writeText(&out, docs.Doc)

	writeValues(&out, p.fset, "Constants", docs.Consts)
	writeValues(&out, p.fset, "Variables", docs.Vars)
	if len(docs.Funcs) > 0 {
		out.WriteString("## Functions\n\n")
		for _, function := range docs.Funcs {
			writeFunc(&out, p.fset, "###", function)
		}
	}
	if len(docs.Types) > 0 {
		out.WriteString("## Types\n\n")
		for _, typ := range docs.Types {
			writeType(&out, p.fset, typ)
		}
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func writeValues(out *strings.Builder, fset *token.FileSet, heading string, values []*doc.Value) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(out, "## %s\n\n", heading)
	for _, value := range values {
		writeValue(out, fset, value)
	}
}

func writeValue(out *strings.Builder, fset *token.FileSet, value *doc.Value) {
	out.WriteString(markdownwriting.CodeBlock("go", printDecl(fset, value.Decl)) + "\n")
	writeText(out, value.Doc)
}

func writeFunc(out *strings.Builder, fset *token.FileSet, level string, function *doc.Func) {
	title := "func " + function.Name
	if function.Recv != "" {
		title = fmt.Sprintf("func (%s) %s", function.Recv, function.Name)
	}
	fmt.Fprintf(out, "%s %s\n\n", level, title)
	out.WriteString(markdownwriting.CodeBlock("go", printDecl(fset, function.Decl)) + "\n")
	writeText(out, function.Doc)
}

func writeType(out *strings.Builder, fset *token.FileSet, typ *doc.Type) {
	fmt.Fprintf(out, "### type %s\n\n", typ.Name)
	out.WriteString(markdownwriting.CodeBlock("go", printDecl(fset, typ.Decl)) + "\n")
	writeText(out, typ.Doc)
	for _, value := range typ.Consts {
		writeValue(out, fset, value)
	}
	for _, value := range typ.Vars {
		writeValue(out, fset, value)
	}
	for _, function := range typ.Funcs {
		writeFunc(out, fset, "####", function)
	}
	for _, method := range typ.Methods {
		writeFunc(out, fset, "####", method)
	}
}

// writeText writes a doc comment as Markdown followed by a blank line; nothing for none.
func writeText(out *strings.Builder, text string) {
	if markdown := commentToMarkdown(text); markdown != "" {
		out.WriteString(markdown + "\n\n")
	}
}

// commentToMarkdown renders a Go doc comment with the standard library's own converter:
// paragraphs, lists, code blocks and headings keep their meaning.
func commentToMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var parser comment.Parser
	// Headings without ids: "{#id}" is not Markdown and would show up literally on a wiki.
	printer := comment.Printer{HeadingID: func(*comment.Heading) string { return "" }}

	return strings.TrimSpace(string(printer.Markdown(parser.Parse(text))))
}

// printDecl formats a declaration as gofmt would, without its comment and, for a function,
// without its body.
func printDecl(fset *token.FileSet, decl ast.Decl) string {
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		shown := *typed
		shown.Doc, shown.Body = nil, nil
		decl = &shown
	case *ast.GenDecl:
		shown := *typed
		shown.Doc = nil
		decl = &shown
	}
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, decl); err != nil {
		return fmt.Sprintf("// the declaration could not be printed: %v", err)
	}

	return buffer.String()
}
