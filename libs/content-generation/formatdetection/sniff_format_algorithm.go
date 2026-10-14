package formatdetection

import (
	"bytes"
	"encoding/xml"
	"path"
	"regexp"
	"strings"
)

// The generator types a file can be recognised as.
const (
	TestResults = "test-results"
	Coverage    = "coverage"
	MarkdownDoc = "markdown-docs"
	OpenAPIDocs = "openapi-docs"
	Storybook   = "storybook-docs"
)

// Formats lists what can be recognised, for messages and documentation, in a stable order.
var Formats = []string{TestResults, Coverage, OpenAPIDocs, MarkdownDoc, Storybook}

var (
	openAPIKey     = regexp.MustCompile(`(?m)^\s*\{?\s*["']?(openapi|swagger)["']?\s*:\s*["']?[23]`)
	storybookIndex = regexp.MustCompile(`(?s)"(entries|stories)"\s*:\s*\{.*?"importPath"`)
)

// Sniff names the generator type that reads a file, from its name and the first bytes of its
// content; empty when it is not a format we read. head may be cut anywhere.
func Sniff(name string, head []byte) string {
	base := strings.ToLower(path.Base(name))
	text := string(head)

	if (base == "index.json" || base == "stories.json") && storybookIndex.MatchString(text) {
		return Storybook
	}
	switch path.Ext(base) {
	case ".md", ".markdown":
		return MarkdownDoc
	case ".info", ".lcov":
		if strings.Contains(text, "SF:") || strings.Contains(text, "TN:") {
			return Coverage
		}
	case ".xml":
		switch xmlRoot(head) {
		case "testsuite", "testsuites":
			return TestResults
		case "coverage":
			return Coverage
		}
	case ".yaml", ".yml", ".json":
		if openAPIKey.MatchString(text) {
			return OpenAPIDocs
		}
	}

	return ""
}

// Container names what kind of file it is, whether or not we read what is in it: the first
// step of recognising one is to know how to open it, the second is which schema it holds.
// Empty for a file we cannot even open as a known kind.
func Container(name string) string {
	switch path.Ext(strings.ToLower(path.Base(name))) {
	case ".xml":
		return "XML"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".md", ".markdown":
		return "Markdown"
	case ".info", ".lcov":
		return "LCOV tracefile"
	}

	return ""
}

// xmlRoot is the name of the document's first element; empty when there is none in head.
func xmlRoot(head []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(head))
	decoder.Strict = false
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		if start, ok := token.(xml.StartElement); ok {
			return start.Name.Local
		}
	}
}
