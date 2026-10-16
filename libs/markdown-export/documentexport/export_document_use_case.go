package documentexport

import (
	"fmt"
	"path/filepath"
	"strings"

	"lore-master/libs/markdown-export/exportbuilding"
	"lore-master/libs/markdown-export/htmlexport"
	"lore-master/libs/markdown-export/jiraadfexport"
	"lore-master/libs/markdown-export/jirawikiexport"
	"lore-master/libs/markdown-export/mediawikiexport"
	"lore-master/libs/markdown-export/plaintextexport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// Options say where the document lives, which a format may need to resolve what it refers to.
type Options struct {
	// BaseDir is the folder the document's relative image paths are read from.
	BaseDir string
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

	var text string
	switch format.ID {
	case "html":
		text = htmlexport.Render(document, htmlexport.Options{BaseDir: options.BaseDir})
	case "txt":
		text = plaintextexport.Render(document)
	case "jira-wiki":
		text = jirawikiexport.Render(document)
	case "jira-adf":
		text, err = jiraadfexport.Render(document)
		if err != nil {
			return File{}, err
		}
	case "mediawiki":
		text = mediawikiexport.Render(document)
	}

	return File{Name: resultName(path, format), Content: []byte(text)}, nil
}

// resultName is the file name an export is written under: the document's, with the format's ending.
func resultName(path string, format Format) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	return name + format.Extension
}
