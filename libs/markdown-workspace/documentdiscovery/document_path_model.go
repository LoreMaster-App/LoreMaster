package documentdiscovery

import "strings"

// DocumentPath is a Markdown file's path relative to the workspace root, always
// '/'-separated so it reads the same on every OS and can be stored in annotations.
type DocumentPath string

// Discovery is the outcome of one scan: the documents in byte order, and what the user
// should be told about files that were found but cannot be used as they are.
type Discovery struct {
	Documents []DocumentPath
	Warnings  []string
	byFolded  map[string][]DocumentPath
}

func newDiscovery(documents []DocumentPath, warnings []string) Discovery {
	byFolded := make(map[string][]DocumentPath, len(documents))
	for _, document := range documents {
		folded := strings.ToLower(string(document))
		byFolded[folded] = append(byFolded[folded], document)
	}

	return Discovery{Documents: documents, Warnings: warnings, byFolded: byFolded}
}

// Lookup resolves a '/'-separated workspace-relative path to a discovered document. An
// exact match wins; otherwise a case-insensitive match is accepted only when it is
// unambiguous, because links written on Windows or macOS often differ in case from
// the file on disk.
func (d Discovery) Lookup(path string) (DocumentPath, bool) {
	candidates := d.byFolded[strings.ToLower(path)]
	for _, candidate := range candidates {
		if string(candidate) == path {
			return candidate, true
		}
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}

	return "", false
}
