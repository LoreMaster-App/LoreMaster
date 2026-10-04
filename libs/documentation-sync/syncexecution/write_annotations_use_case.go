package syncexecution

import (
	"fmt"
	"path/filepath"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// WriteBack is what writing the annotations did.
type WriteBack struct {
	// Rewritten are the files whose annotation changed.
	Rewritten []documentdiscovery.DocumentPath
	// Warnings name the files that could not be written. Their pages are synced all
	// the same.
	Warnings []string
}

// WriteAnnotations writes each written page's annotation into its file under
// workspaceRoot, and for a pulled page replaces the body below the annotation with the
// one pulled from the platform. Only those files are opened, and one that would be
// identical is left untouched, so after a sync git shows exactly what synced. A file
// that cannot be written is reported, never fatal: the page is on the platform already.
func WriteAnnotations(workspaceRoot string, report SyncReport) WriteBack {
	var result WriteBack
	for _, page := range report.Pages {
		if page.Annotation == nil {
			continue
		}
		path := filepath.Join(workspaceRoot, filepath.FromSlash(string(page.Path)))
		changed, err := writeFile(path, page)
		switch {
		case err != nil:
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: the page was synced (%s), but the file could not be updated (%v); the next sync takes the page back by its title",
				page.Path, page.URL, err))
		case changed:
			result.Rewritten = append(result.Rewritten, page.Path)
		}
	}

	return result
}

// writeFile updates a file's annotation, and — on a pull — the body below it too.
func writeFile(path string, page PageResult) (bool, error) {
	if page.PulledBody != nil {
		return syncannotation.WriteWithBody(path, *page.Annotation, page.PulledBody)
	}

	return syncannotation.Write(path, *page.Annotation)
}
