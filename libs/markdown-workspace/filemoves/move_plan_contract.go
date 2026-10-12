package filemoves

// Move is one file or folder the user moved or renamed, as workspace-relative paths
// separated by '/'.
type Move struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Rename is a file that must move along with the user's own moves.
type Rename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// LineEdit replaces one whole line of a file. Path is where the file is before any move.
type LineEdit struct {
	Path string `json:"path"`
	// Line is zero-based.
	Line int `json:"line"`
	// Text is the new line, without its line ending.
	Text string `json:"text"`
}

// Plan is what to change besides the moves themselves.
type Plan struct {
	Renames  []Rename   `json:"renames"`
	Edits    []LineEdit `json:"edits"`
	Warnings []string   `json:"warnings"`
}
