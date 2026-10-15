package syncexecution

import (
	"fmt"
	"os"
	"path/filepath"

	"lore-master/libs/documentation-sync/syncstate"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// StateTarget says where the sync state of a write-back goes.
type StateTarget struct {
	// Key is the storage's name in lore-master.json.
	Key    string
	Output workspacesettings.Output
	// Loaded is each file's annotation as the file carried it before the sync (see
	// syncstate.Apply). The state of a file that still carries it is moved into the state
	// file, and the file keeps only what the author set.
	Loaded map[documentdiscovery.DocumentPath]*syncannotation.Annotation
}

// WriteBack is what writing the annotations did.
type WriteBack struct {
	// Rewritten are the files whose annotation changed.
	Rewritten []documentdiscovery.DocumentPath
	// Warnings name the files that could not be written. Their pages are synced all
	// the same.
	Warnings []string
}

// WriteAnnotations records each written page in lore-master.json (under target.Key) and, for
// a pulled page, replaces the file's body with the one pulled from the platform. A file
// keeps only the annotation fields its author owns (title, parent, generated); the state it
// still carries from the old per-file annotation is moved into the state file, so a sync
// changes one file in git. A file or state that would be identical is left untouched. A file
// that cannot be written is reported, never fatal: the page is on the platform already.
func WriteAnnotations(workspaceRoot string, report SyncReport, target StateTarget) WriteBack {
	var result WriteBack
	state, err := syncstate.Read(workspaceRoot)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"the sync state was not saved: %v; the pages were synced, and the next sync takes them back by their titles", err))

		return result
	}
	storage := state.StorageNamed(target.Key)
	storage.Platform, storage.BaseURL, storage.Space = target.Output.Platform, target.Output.BaseURL, target.Output.Space

	handled := map[documentdiscovery.DocumentPath]bool{}
	for _, page := range report.Pages {
		if page.Annotation == nil {
			continue
		}
		handled[page.Path] = true
		result.Warnings = append(result.Warnings, writeAttachments(workspaceRoot, page)...)
		storage.Pages[string(page.Path)] = syncstate.PageFrom(*page.Annotation)
		path := filepath.Join(workspaceRoot, filepath.FromSlash(string(page.Path)))
		changed, err := writeFile(path, page, target)
		switch {
		case err != nil:
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"%s: the page was synced (%s), but the file could not be updated (%v)", page.Path, page.URL, err))
		case changed:
			result.Rewritten = append(result.Rewritten, page.Path)
		}
	}

	// One-time migration: what the files not written above still carry moves into the state.
	for path, loaded := range target.Loaded {
		if handled[path] || !syncstate.CarriesState(loaded, target.Output.BaseURL, target.Output.Space) {
			continue
		}
		if _, known := storage.Pages[string(path)]; !known {
			storage.Pages[string(path)] = syncstate.PageFrom(*loaded)
		}
		changed, err := syncannotation.Write(filepath.Join(workspaceRoot, filepath.FromSlash(string(path))), syncannotation.AuthorFields(*loaded))
		switch {
		case err != nil:
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: its state was moved to %s, but the old annotation could not be removed (%v)", path, syncstate.FileName, err))
		case changed:
			result.Rewritten = append(result.Rewritten, path)
		}
	}

	// A file that is gone takes its entry with it.
	for path := range storage.Pages {
		if _, err := os.Stat(filepath.Join(workspaceRoot, filepath.FromSlash(path))); err != nil && os.IsNotExist(err) {
			delete(storage.Pages, path)
		}
	}

	if len(storage.Pages) == 0 {
		delete(state.Storages, target.Key)
	} else {
		state.Storages[target.Key] = storage
	}
	changed, err := syncstate.Write(workspaceRoot, state)
	switch {
	case err != nil:
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"%s could not be written (%v); the pages were synced, and the next sync takes them back by their titles", syncstate.FileName, err))
	case changed:
		result.Rewritten = append(result.Rewritten, documentdiscovery.DocumentPath(syncstate.FileName))
	}

	return result
}

// WriteTargets are the workspace-relative files WriteAnnotations may change for report: each
// written page, the files whose old annotation is migrated, and the state file.
func WriteTargets(report SyncReport, target StateTarget) []string {
	var paths []string
	seen := map[string]bool{}
	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	for _, page := range report.Pages {
		if page.Annotation == nil {
			continue
		}
		add(string(page.Path))
		for _, attachment := range page.PulledAttachments {
			add(string(attachment.Path))
		}
	}
	for path, loaded := range target.Loaded {
		if syncstate.CarriesState(loaded, target.Output.BaseURL, target.Output.Space) {
			add(string(path))
		}
	}
	add(syncstate.FileName)

	return paths
}

// writeFile updates a file's annotation to the fields its author owns, and — on a pull — the
// body below it too. A file whose annotation holds the state of another target keeps that
// annotation whole: that other target's sync moves it.
func writeFile(path string, page PageResult, target StateTarget) (bool, error) {
	annotation := syncannotation.AuthorFields(*page.Annotation)
	if loaded := target.Loaded[page.Path]; loaded != nil && syncannotation.HasState(*loaded) &&
		!syncstate.MatchesTarget(*loaded, target.Output.BaseURL, target.Output.Space) {
		annotation = *loaded
	}
	if page.PulledBody != nil {
		return syncannotation.WriteWithBody(path, annotation, page.PulledBody)
	}

	return syncannotation.Write(path, annotation)
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
