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
}

// formats lists every format, in the order a picker shows them.
var formats = []Format{
	{ID: "html", Label: "HTML (one self-contained file)", Extension: ".html"},
	{ID: "txt", Label: "Plain text (Markdown without the tags)", Extension: ".txt"},
	{ID: "docx", Label: "Word document", Extension: ".docx"},
	{ID: "odt", Label: "OpenDocument text (LibreOffice)", Extension: ".odt"},
	{ID: "jira-wiki", Label: "Jira wiki markup", Extension: ".jira.txt"},
	{ID: "jira-adf", Label: "Jira Cloud document (ADF JSON)", Extension: ".adf.json"},
	{ID: "mediawiki", Label: "MediaWiki / Wikipedia", Extension: ".wiki"},
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
