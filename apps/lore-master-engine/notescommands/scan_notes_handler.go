// Package notescommands answers notes/scan: the tagged comments of a workspace's source files.
package notescommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/code-notes/notescanning"
	"lore-master/libs/code-notes/notetags"
	"lore-master/libs/documentation-sync/editorsettings"
)

// ScanNotes handles notes/scan. It reads files by the workspace's ignore list and .gitignore files, as a sync does; a
// workspace with no LoreMaster settings, or settings that are not valid, is scanned with the defaults, since the notes
// do not depend on a storage being set up: the ignore list counts even where nothing is synced.
func ScanNotes() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.NotesScanParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		options := notescanning.Options{WorkspaceRoot: params.WorkspaceRoot, Tags: params.Tags}
		if scope, err := editorsettings.LoadDiscoveryScope(params.WorkspaceRoot, editorsettings.CurrentUserSettingsPath()); err == nil {
			options.Excludes, options.IncludeGitignored = scope.Ignore, !scope.SkipGitignored
		}
		found, err := notescanning.Scan(ctx, options)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "scan the notes: %s", err.Error())
		}

		tags := params.Tags
		if len(tags) == 0 {
			tags = notetags.DefaultTags
		}
		result := rpcprotocol.NotesScanResult{Notes: make([]rpcprotocol.NoteEntry, len(found.Notes)), FilesScanned: found.FilesScanned, FilesSkipped: found.FilesSkipped, Tags: tags}
		for i, note := range found.Notes {
			result.Notes[i] = rpcprotocol.NoteEntry{Path: note.Path, Line: note.Line, Column: note.Column, Tag: note.Tag, Text: note.Text, Owner: note.Owner, Issue: note.Issue, Due: note.Due}
		}

		return result, nil
	}
}
