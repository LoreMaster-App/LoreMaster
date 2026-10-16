package documentexport

import (
	"fmt"
	"path/filepath"
	"strings"

	"lore-master/libs/markdown-export/browserexport"
	"lore-master/libs/markdown-export/docxexport"
	"lore-master/libs/markdown-export/exportbuilding"
	"lore-master/libs/markdown-export/exportmodel"
	"lore-master/libs/markdown-export/htmlexport"
	"lore-master/libs/markdown-export/jiraadfexport"
	"lore-master/libs/markdown-export/jirawikiexport"
	"lore-master/libs/markdown-export/mediawikiexport"
	"lore-master/libs/markdown-export/odtexport"
	"lore-master/libs/markdown-export/plaintextexport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// Options say where the document lives, which a format may need to resolve what it refers to.
type Options struct {
	// BaseDir is the folder the document's relative image paths are read from.
	BaseDir string
	// BrowserPath is the browser PDF and PNG are printed with; one is looked for when empty.
	BrowserPath string
	// PageSize is the PDF's paper (A4, Letter...); A4 when empty.
	PageSize string
	// Width is the PNG's width in pixels; 1000 when zero.
	Width int
}

// Export converts one Markdown file's content to the format with the given id. path is the file's
// path, used for its title when it has no heading and to name the result.
func Export(path string, content []byte, formatID string, options Options) (File, error) {
	format, found := FormatByID(formatID)
	if !found {
		return File{}, fmt.Errorf("unknown export format %q", formatID)
	}
	parsed, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(filepath.ToSlash(path)), content)
	if err != nil {
		return File{}, err
	}
	document := exportbuilding.BuildDocument(parsed)

	result, err := render(format.ID, document, options)
	if err != nil {
		return File{}, err
	}

	return File{Name: resultName(path, format), Content: result}, nil
}

func render(id string, document exportmodel.Document, options Options) ([]byte, error) {
	switch id {
	case "html":
		return []byte(htmlexport.Render(document, htmlexport.Options{BaseDir: options.BaseDir})), nil
	case "txt":
		return []byte(plaintextexport.Render(document)), nil
	case "pdf":
		return browserexport.RenderPDF(htmlexport.Render(document, htmlexport.Options{BaseDir: options.BaseDir}), browserexport.Options{BrowserPath: options.BrowserPath, PageSize: options.PageSize})
	case "png":
		return browserexport.RenderPNG(htmlexport.Render(document, htmlexport.Options{BaseDir: options.BaseDir}), browserexport.Options{BrowserPath: options.BrowserPath, Width: options.Width})
	case "docx":
		return docxexport.Render(document, docxexport.Options{BaseDir: options.BaseDir})
	case "odt":
		return odtexport.Render(document, odtexport.Options{BaseDir: options.BaseDir})
	case "jira-wiki":
		return []byte(jirawikiexport.Render(document)), nil
	case "jira-adf":
		text, err := jiraadfexport.Render(document)

		return []byte(text), err
	case "mediawiki":
		return []byte(mediawikiexport.Render(document)), nil
	default:
		return nil, fmt.Errorf("unknown export format %q", id)
	}
}

// resultName is the file name an export is written under: the document's, with the format's ending.
func resultName(path string, format Format) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	return name + format.Extension
}
