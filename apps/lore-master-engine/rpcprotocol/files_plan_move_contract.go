package rpcprotocol

// MethodFilesPlanMove says what else must change when Markdown files or folders are moved
// or renamed: the dotted children that nest under a moved file by name are renamed to
// follow it, and "parent:" annotations that name a moved file, or sit in one, are
// rewritten. It reads the workspace and writes nothing; the editor applies the plan with
// the move as one undoable edit.
const MethodFilesPlanMove = "files/planMove"

// FilesPlanMoveParams names the workspace and the moves.
type FilesPlanMoveParams struct {
	// WorkspaceRoot is the workspace folder, as an absolute path.
	WorkspaceRoot string `json:"workspaceRoot"`
	// Moves are workspace-relative, '/'-separated; a move may name a file or a folder.
	Moves []FileMove `json:"moves"`
}

// FileMove is one moved or renamed file or folder.
type FileMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// FilesPlanMoveResult is what to change besides the moves themselves.
type FilesPlanMoveResult struct {
	// Renames are files to move along: each From is where the file is now.
	Renames []FileMove `json:"renames"`
	// Edits replace whole lines of files; Path is where the file is before any move.
	Edits []FileLineEdit `json:"edits"`
	// Warnings are renames that were skipped and why.
	Warnings []string `json:"warnings"`
}

// FileLineEdit replaces one line of a file with Text (no line ending).
type FileLineEdit struct {
	Path string `json:"path"`
	// Line is zero-based.
	Line int    `json:"line"`
	Text string `json:"text"`
}
