package syncjournal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"lore-master/libs/documentation-sync/platformport"
)

// Options says how a rollback runs.
type Options struct {
	// DryRun reports what would happen and changes nothing.
	DryRun bool
	// Force rolls back pages and files that were edited after the sync, discarding those edits.
	Force bool
}

// Result values of one page or file.
const (
	ResultTrashed      = "trashed"
	ResultRestored     = "restored"
	ResultWouldTrash   = "would-trash"
	ResultWouldRestore = "would-restore"
	ResultGone         = "gone"
	ResultSkipped      = "skipped"
	ResultFailed       = "failed"
	ResultIrreversible = "irreversible"
)

// PageOutcome is what the rollback did to one page.
type PageOutcome struct {
	PageID string     `json:"pageId"`
	Title  string     `json:"title"`
	Kind   ChangeKind `json:"kind"`
	Result string     `json:"result"`
	Reason string     `json:"reason,omitempty"`
}

// FileOutcome is what the rollback did to one local file.
type FileOutcome struct {
	Path   string `json:"path"`
	Result string `json:"result"`
	Reason string `json:"reason,omitempty"`
}

// Outcome is everything a rollback did, in the order it did it.
type Outcome struct {
	Pages []PageOutcome `json:"pages"`
	Files []FileOutcome `json:"files"`
}

// Complete is true when nothing was skipped or failed, so the journal can be dropped.
func (outcome Outcome) Complete() bool {
	for _, page := range outcome.Pages {
		if page.Result == ResultSkipped || page.Result == ResultFailed || page.Result == ResultIrreversible {
			return false
		}
	}
	for _, file := range outcome.Files {
		if file.Result == ResultSkipped || file.Result == ResultFailed {
			return false
		}
	}

	return true
}

// Rollback undoes a journal: pages the sync created are trashed (newest first, so children go
// before their parents), pages it updated are put back as they were, and local files get
// their earlier content back. A page or file edited since the sync is skipped, with the
// reason, unless Force is set.
func Rollback(ctx context.Context, platform platformport.DocumentationPlatform, workspaceRoot string, journal Journal, options Options) Outcome {
	var outcome Outcome
	// Created pages go newest first, so a child is dealt with before its parent. A dry run trashes nothing, so a parent has to
	// be told which of its children this rollback has already trashed (or would): those are not left standing below it.
	gone := map[string]bool{}
	for index := len(journal.Pages) - 1; index >= 0; index-- {
		change := journal.Pages[index]
		if ctx.Err() != nil {
			outcome.Pages = append(outcome.Pages, PageOutcome{PageID: change.PageID, Title: change.Title, Kind: change.Kind, Result: ResultSkipped, Reason: "cancelled"})

			continue
		}
		page := rollBackPage(ctx, platform, change, options, gone)
		if page.Result == ResultTrashed || page.Result == ResultWouldTrash || page.Result == ResultGone {
			gone[change.PageID] = true
		}
		outcome.Pages = append(outcome.Pages, page)
	}
	for _, change := range journal.Files {
		outcome.Files = append(outcome.Files, rollBackFile(workspaceRoot, change, options))
	}

	return outcome
}

func rollBackPage(ctx context.Context, platform platformport.DocumentationPlatform, change PageChange, options Options, gone map[string]bool) PageOutcome {
	result := PageOutcome{PageID: change.PageID, Title: change.Title, Kind: change.Kind}
	if change.Kind == Trashed {
		result.Result = ResultIrreversible
		result.Reason = "the sync moved this page to the trash; restore it from the platform's trash"

		return result
	}

	current, err := platform.GetPage(ctx, change.PageID)
	var missing *platformport.PageNotFoundError
	switch {
	case errors.As(err, &missing):
		result.Result = ResultGone
		result.Reason = "the page no longer exists"

		return result
	case err != nil:
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	}
	if current.Version != change.AfterVersion && !options.Force {
		result.Result = ResultSkipped
		result.Reason = fmt.Sprintf("edited on the platform after the sync (version %d, the sync left %d); use force to discard that edit", current.Version, change.AfterVersion)

		return result
	}

	if change.Kind == Created {
		return trashCreated(ctx, platform, change, options, result, gone)
	}

	if change.Before == nil {
		result.Result = ResultFailed
		result.Reason = "the journal has no earlier version of this page"

		return result
	}
	if options.DryRun {
		result.Result = ResultWouldRestore

		return result
	}
	if _, err := platform.RestorePage(ctx, *change.Before, current.Version, "Rolled back by LoreMaster"); err != nil {
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	}
	result.Result = ResultRestored

	return result
}

func trashCreated(ctx context.Context, platform platformport.DocumentationPlatform, change PageChange, options Options, result PageOutcome, gone map[string]bool) PageOutcome {
	listed, err := platform.ListChildren(ctx, change.PageID)
	if err != nil {
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	}
	var children []platformport.RemotePage
	for _, child := range listed {
		if !gone[child.ID] {
			children = append(children, child)
		}
	}
	if len(children) > 0 && !options.Force {
		result.Result = ResultSkipped
		result.Reason = fmt.Sprintf("it has %d page(s) below it that this sync did not create; use force to delete it anyway", len(children))

		return result
	}
	if options.DryRun {
		result.Result = ResultWouldTrash

		return result
	}
	if err := platform.TrashPage(ctx, change.PageID); err != nil {
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	}
	result.Result = ResultTrashed

	return result
}

func rollBackFile(workspaceRoot string, change FileChange, options Options) FileOutcome {
	result := FileOutcome{Path: change.Path}
	path := filepath.Join(workspaceRoot, filepath.FromSlash(change.Path))

	current, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if !change.Existed {
			result.Result = ResultGone
			result.Reason = "the file the sync created is already gone"

			return result
		}
	case err != nil:
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	case hashOf(current) != change.AfterHash && !options.Force:
		result.Result = ResultSkipped
		result.Reason = "edited after the sync; use force to discard that edit"

		return result
	}

	if options.DryRun {
		result.Result = ResultWouldRestore

		return result
	}
	if !change.Existed {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			result.Result = ResultFailed
			result.Reason = err.Error()

			return result
		}
		result.Result = ResultRestored

		return result
	}
	if err := os.WriteFile(path, change.Before, 0o644); err != nil {
		result.Result = ResultFailed
		result.Reason = err.Error()

		return result
	}
	result.Result = ResultRestored

	return result
}
