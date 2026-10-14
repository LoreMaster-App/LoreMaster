package syncannotation

import (
	"bytes"
	"strings"
)

// ParentField is where an annotation's "parent:" line sits in a file, so an editor can
// rewrite just that line.
type ParentField struct {
	// Line is the zero-based line number in the file (after any byte-order mark).
	Line int
	// Text is the whole line without its line ending.
	Text string
	// Value is the parent as written, trimmed.
	Value string
}

// LocateParent finds the "parent:" line of the annotation block. It reports false when the
// file has no annotation, the annotation is not closed, or it has no parent. It does not
// validate the rest of the block: that is Read's job, and a move must not be blocked by an
// unrelated problem in a file.
func LocateParent(content []byte) (ParentField, bool) {
	rest := bytes.TrimPrefix(content, byteOrderMark)
	start := frontMatterEnd(rest)
	if line, _ := lineAt(rest, start); strings.TrimRight(line, " \t") != OpeningLine {
		return ParentField{}, false
	}

	lineNumber := bytes.Count(rest[:start], []byte("\n"))
	for offset := nextLine(rest, start); offset < len(rest); offset = nextLine(rest, offset) {
		lineNumber++
		line, _ := lineAt(rest, offset)
		trimmed := strings.TrimSpace(line)
		if trimmed == ClosingLine {
			return ParentField{}, false
		}
		key, value, isField := strings.Cut(trimmed, ":")
		if isField && strings.TrimSpace(key) == KeyParent {
			value = strings.TrimSpace(value)
			if value == "" {
				return ParentField{}, false
			}

			return ParentField{Line: lineNumber, Text: line, Value: value}, true
		}
	}

	return ParentField{}, false
}
