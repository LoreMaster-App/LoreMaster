package syncexecution

import (
	"fmt"
	"os"
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
		for _, warning := range writeAttachments(workspaceRoot, page) {
			result.Warnings = append(result.Warnings, warning)
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

// writeAttachments writes a pull's downloaded images into the workspace, creating folders as
// needed. A file that cannot be written is a warning, never fatal: the page's text still lands.
func writeAttachments(workspaceRoot string, page PageResult) []string {
	var warnings []string
	for _, file := range page.PulledAttachments {
		target := filepath.Join(workspaceRoot, filepath.FromSlash(string(file.Path)))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: the image %s could not be written (%v)", page.Path, file.Path, err))

			continue
		}
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: the image %s could not be written (%v)", page.Path, file.Path, err))
		}
	}

	return warnings
}
