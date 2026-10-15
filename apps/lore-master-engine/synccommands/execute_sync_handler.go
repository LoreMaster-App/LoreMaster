package synccommands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lore-master/apps/lore-master-engine/diagramrouting"
	"lore-master/apps/lore-master-engine/hostbridge"
	"lore-master/apps/lore-master-engine/mermaidrendering"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/documentation-sync/syncjournal"
)

// mermaidRendererVariable set to "editor" makes the editor draw Mermaid even when the engine could
// do it in a browser of its own.
const mermaidRendererVariable = "LORE_MASTER_MERMAID"

// ExecuteSync handles sync/execute: carry the plan out, then write the annotations
// back. A cancelled sync stops between pages and still reports, and still writes back
// what it did, so the files agree with the pages.
//
// A Confluence sync is journalled under journalCache (empty disables it): the pages it created
// and updated, with how they were, and the files it rewrote, so sync/rollback can undo it.
func ExecuteSync(sessions *sessionlifecycle.Store, plans *PlanStore, renderTimeout time.Duration, journalCache string) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SyncExecuteParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		stored, err := plans.take(params.PlanID)
		if err != nil {
			return nil, err
		}
		if err := planHasErrors(stored.plan); err != nil {
			return nil, err
		}
		session, err := sessions.Get(stored.sessionID)
		if err != nil {
			return nil, err
		}

		// D2 is drawn here whatever the mode; Mermaid is asked of the editor, and only when the
		// output shows it as an image (the sync does not ask otherwise).
		var editor platformport.DiagramRenderer
		if stored.output.MermaidMode == "image" {
			editor = hostbridge.NewDiagramRenderer(call.Editor, renderTimeout)
		}
		// Mermaid is drawn in the engine's own browser when there is one, so it also works from the
		// command line; the editor is the fallback. The browser only starts on the first diagram.
		var browser platformport.DiagramRenderer
		if stored.output.MermaidMode == "image" && os.Getenv(mermaidRendererVariable) != "editor" {
			drawing := mermaidrendering.New()
			defer drawing.Close()
			browser = drawing
		}
		renderer := diagramrouting.RouteWith(editor, browser)
		platform := session.Platform.ForOutputWithTemplate(stored.output, stored.template)
		var recorder *syncjournal.Recorder
		if journalCache != "" {
			recorder = syncjournal.NewRecorder(platform)
			platform = recorder
		}
		started := time.Now()
		report := syncexecution.ExecutePlan(ctx, platform, syncexecution.ExecuteInput{
			Plan: stored.plan, Prepared: stored.prepared, Output: stored.output, Space: stored.space,
			Read: fileReader(stored.root), Renderer: renderer,
			Options:  syncexecution.Options{Force: params.Force, Prune: params.Prune},
			Progress: hostbridge.Progress(ctx, call.Editor, stored.id),
		})
		before := syncjournal.CaptureFiles(stored.root, touchedFiles(report))
		written := syncexecution.WriteAnnotations(stored.root, report)

		result := rpcprotocol.SyncExecuteResult{Warnings: append(report.Warnings, written.Warnings...)}
		if recorder != nil {
			result.JournalID, result.Warnings = keepJournal(journalCache, started, stored, recorder, before, result.Warnings)
		}
		if ctx.Err() != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("the sync was cancelled; %d page(s) were written before it stopped", report.Count(syncexecution.Written)))
		}
		for _, path := range written.Rewritten {
			result.Rewritten = append(result.Rewritten, string(path))
		}
		for _, page := range report.Pages {
			result.Pages = append(result.Pages, rpcprotocol.PageOutcome{
				Path: string(page.Path), Title: page.Title, Planned: string(page.Planned), Outcome: string(page.Outcome),
				PageID: page.PageID, Version: page.Version, URL: page.URL, Error: page.Error,
			})
		}

		return result, nil
	}
}

// touchedFiles are the workspace-relative files the write-back may change: each written
// page's file and the images a pull downloads.
func touchedFiles(report syncexecution.SyncReport) []string {
	var paths []string
	for _, page := range report.Pages {
		if page.Annotation == nil {
			continue
		}
		paths = append(paths, string(page.Path))
		for _, attachment := range page.PulledAttachments {
			paths = append(paths, string(attachment.Path))
		}
	}

	return paths
}

// keepJournal saves what the sync changed and returns the journal's id. A journal that
// cannot be saved is a warning: the sync itself went through, it just cannot be rolled back.
func keepJournal(journalCache string, started time.Time, stored *storedPlan, recorder *syncjournal.Recorder, before map[string]syncjournal.FileState, warnings []string) (string, []string) {
	journal := syncjournal.Journal{
		ID: syncjournal.NewID(started), StartedAt: started.UTC(), WorkspaceRoot: filepath.ToSlash(stored.root),
		Output: stored.output, Pages: recorder.Changes(), Files: syncjournal.ChangedFiles(stored.root, before),
	}
	if journal.Empty() {
		return "", warnings
	}
	if err := syncjournal.NewRepository(syncjournal.DirFor(journalCache, stored.root)).Save(journal); err != nil {
		return "", append(warnings, fmt.Sprintf("this sync could not be journalled, so it cannot be rolled back: %v", err))
	}

	return journal.ID, warnings
}
