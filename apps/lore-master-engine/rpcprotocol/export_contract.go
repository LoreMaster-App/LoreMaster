package rpcprotocol

// MethodExportFormats lists the formats a Markdown document can be exported to.
const MethodExportFormats = "export/formats"

// ExportFormatsResult is the list of formats, in the order a picker shows them.
type ExportFormatsResult struct {
	Formats []ExportFormat `json:"formats"`
}

// ExportFormat is one format a document can be exported to.
type ExportFormat struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Extension is the file ending, with its dot.
	Extension string `json:"extension"`
	// Binary says the result is not text: it can be written to a file but not copied.
	Binary bool `json:"binary,omitempty"`
}

// MethodExportRun converts one Markdown document to another format and either writes the result
// into a folder or returns it as text. It reads no settings and needs no session.
const MethodExportRun = "export/run"

// ExportRunParams asks to export one document.
type ExportRunParams struct {
	// Path is the Markdown file, as an absolute path. Its folder is where relative images are read from.
	Path string `json:"path"`
	// Content is the document's text when the editor holds changes that are not saved; the file on
	// disk is read when it is absent.
	Content *string `json:"content,omitempty"`
	// Format is the id of a format from MethodExportFormats.
	Format string `json:"format"`
	// OutDir is the absolute folder to write the result into. It is created when missing and
	// defaults to the document's own folder. It is not used with Inline.
	OutDir string `json:"outDir,omitempty"`
	// Inline returns the result as text in the reply instead of writing a file. Only text formats
	// can be returned.
	Inline bool `json:"inline,omitempty"`
	// Preview makes html for a preview pane: links to other Markdown files are left as they are.
	Preview bool `json:"preview,omitempty"`
	// BrowserPath is the Chrome, Edge or Chromium that prints pdf and png; one is looked for when empty.
	BrowserPath string `json:"browserPath,omitempty"`
	// MermaidPath is the Mermaid library (mermaid.min.js) pdf and png draw diagrams with; they are printed as code without one.
	MermaidPath string `json:"mermaidPath,omitempty"`
	// ReferencePath is a template (a Word file for docx, an OpenDocument file for odt), as an absolute path: the document takes
	// its styles, headers, footers and page setup, and its own text is not used.
	ReferencePath string `json:"referencePath,omitempty"`
	// PageSize is the paper of a pdf (A4, A3, A5, Letter, Legal); A4 when empty.
	PageSize string `json:"pageSize,omitempty"`
	// ImageWidth is the width of a png in pixels; 1000 when zero.
	ImageWidth int `json:"imageWidth,omitempty"`
}

// ExportRunResult is what the export produced.
type ExportRunResult struct {
	// Name is the result's file name.
	Name string `json:"name"`
	// Path is where the file was written, as an absolute path. Empty with Inline.
	Path string `json:"path,omitempty"`
	// Text is the result when Inline asked for it.
	Text string `json:"text,omitempty"`
}
