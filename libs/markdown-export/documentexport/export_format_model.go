// Package documentexport exports one Markdown document to any of the formats LoreMaster writes,
// and says which formats those are.
package documentexport

// Format is one thing a document can be exported to.
type Format struct {
	// ID is what a command line or a setting names the format by.
	ID    string
	Label string
	// Extension is the file name's ending, with its dot.
	Extension string
	// Binary says the result is not text, so it can be written to a file but not shown or copied.
	Binary bool
}

// formats lists every format, in the order a picker shows them.
var formats = []Format{
	{ID: "html", Label: "HTML (one self-contained file)", Extension: ".html"},
	{ID: "txt", Label: "Plain text (Markdown without the tags)", Extension: ".txt"},
	{ID: "pdf", Label: "PDF (printed by Chrome, Edge or Chromium)", Extension: ".pdf", Binary: true},
	{ID: "png", Label: "PNG image of the page (Chrome, Edge or Chromium)", Extension: ".png", Binary: true},
	{ID: "docx", Label: "Word document", Extension: ".docx", Binary: true},
	{ID: "odt", Label: "OpenDocument text (LibreOffice)", Extension: ".odt", Binary: true},
	{ID: "jira-wiki", Label: "Jira wiki markup", Extension: ".jira.txt"},
	{ID: "jira-adf", Label: "Jira Cloud document (ADF JSON)", Extension: ".adf.json"},
	{ID: "mediawiki", Label: "MediaWiki / Wikipedia", Extension: ".wiki"},
	{ID: "asciidoc", Label: "AsciiDoc (Asciidoctor, Antora)", Extension: ".adoc"},
	{ID: "rst", Label: "reStructuredText (Sphinx)", Extension: ".rst"},
	{ID: "man", Label: "Man page (roff)", Extension: ".7"},
	{ID: "rtf", Label: "Rich Text Format", Extension: ".rtf"},
	{ID: "mdx", Label: "MDX (Docusaurus, with front matter)", Extension: ".mdx"},
	{ID: "slack", Label: "Slack message (mrkdwn)", Extension: ".slack.txt"},
	{ID: "org", Label: "Org-mode (Emacs)", Extension: ".org"},
	{ID: "latex", Label: "LaTeX (a file that compiles)", Extension: ".tex"},
	{ID: "outline", Label: "JSON outline (headings, links, tasks)", Extension: ".outline.json"},
}

// Formats returns the formats a document can be exported to.
func Formats() []Format {
	return append([]Format(nil), formats...)
}

// FormatByID finds a format by its id.
func FormatByID(id string) (Format, bool) {
	for _, format := range formats {
		if format.ID == id {
			return format, true
		}
	}

	return Format{}, false
}
