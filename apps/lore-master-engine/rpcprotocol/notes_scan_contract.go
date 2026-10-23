package rpcprotocol

// MethodNotesScan finds the notes left in the comments of a workspace's source files (TODO, FIXME...). It needs no
// session and no settings of its own; it reads files by the ignore list a sync reads them by.
const MethodNotesScan = "notes/scan"

// NotesScanParams says where to look and for what.
type NotesScanParams struct {
	// WorkspaceRoot is the workspace folder, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Tags are the tags to find, upper case; empty means the engine's default list.
	Tags []string `json:"tags,omitempty"`
}

// NotesScanResult is what was found, in path then line order.
type NotesScanResult struct {
	Notes []NoteEntry `json:"notes"`
	// FilesScanned counts the files read; FilesSkipped the source files left unread (too big or binary).
	FilesScanned int `json:"filesScanned"`
	FilesSkipped int `json:"filesSkipped"`
	// Tags are the tags that were looked for, so a shell can show a group even for one with no notes.
	Tags []string `json:"tags"`
}

// NoteEntry is one tagged comment.
type NoteEntry struct {
	// Path is workspace-relative and '/'-separated; Line and Column are 1-based.
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Tag    string `json:"tag"`
	Text   string `json:"text"`
	Owner  string `json:"owner,omitempty"`
	Issue  string `json:"issue,omitempty"`
	Due    string `json:"due,omitempty"`
}
