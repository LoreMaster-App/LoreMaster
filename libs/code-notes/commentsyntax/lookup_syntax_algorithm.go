package commentsyntax

import (
	"path"
	"slices"
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

// Extensions are the file extensions (lower case, with the dot) whose comments are read, sorted.
func Extensions() []string {
	extensions := make([]string, 0, len(byExtension))
	for extension := range byExtension {
		extensions = append(extensions, extension)
	}
	slices.Sort(extensions)

	return extensions
}

// Names are the whole file names (lower case) whose comments are read, sorted.
func Names() []string {
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, strings.ToLower(name))
	}
	slices.Sort(names)

	return names
}
