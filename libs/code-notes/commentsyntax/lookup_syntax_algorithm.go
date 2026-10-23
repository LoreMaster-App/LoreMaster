package commentsyntax

import (
	"path"
	"strings"
)

// SyntaxFor is how the file at rel (workspace-relative, '/'-separated) writes comments, and whether
// the kind of file is known. Files of an unknown kind are not scanned: a tag found in text that is not
// known to be code would too often be prose.
func SyntaxFor(rel string) (Syntax, bool) {
	base := path.Base(rel)
	if syntax, found := byName[base]; found {
		return syntax, true
	}
	syntax, found := byExtension[strings.ToLower(path.Ext(base))]

	return syntax, found
}
