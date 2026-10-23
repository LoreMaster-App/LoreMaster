// Package commentsyntax knows how each kind of file writes comments, so a scan reads comments
// and not strings that happen to hold a tag.
package commentsyntax

// BlockPair is a comment that opens and closes, possibly over several lines.
type BlockPair struct {
	Open  string
	Close string
}

// Syntax is how one kind of file writes comments and strings.
type Syntax struct {
	// Line are the markers that start a comment running to the end of the line.
	Line []string
	// Block are the comments with an open and a close. A longer Open is tried before a shorter one.
	Block []BlockPair
	// Quotes are the characters that start a string on one line; a backslash escapes inside it.
	Quotes string
}
