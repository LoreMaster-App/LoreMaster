package synccommands

import (
	"context"
	"fmt"
	"time"

	"lore-master/apps/lore-master-engine/hostbridge"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncexecution"
)

// ExecuteSync handles sync/execute: carry the plan out, then write the annotations
// back. A cancelled sync stops between pages and still reports, and still writes back
// what it did, so the files agree with the pages.
func ExecuteSync(sessions *sessionlifecycle.Store, plans *PlanStore, renderTimeout time.Duration) rpcserver.Method {
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

		var renderer platformport.DiagramRenderer
		if stored.output.MermaidMode == "image" {
			renderer = hostbridge.NewDiagramRenderer(call.Editor, renderTimeout)
		}
		report := syncexecution.ExecutePlan(ctx, session.Platform.ForOutput(stored.output), syncexecution.ExecuteInput{
			Plan: stored.plan, Prepared: stored.prepared, Output: stored.output, Space: stored.space,
			Read: fileReader(stored.root), Renderer: renderer,
			Options:  syncexecution.Options{Force: params.Force, Prune: params.Prune},
			Progress: hostbridge.Progress(ctx, call.Editor, stored.id),
		})
		written := syncexecution.WriteAnnotations(stored.root, report)

		result := rpcprotocol.SyncExecuteResult{Warnings: append(report.Warnings, written.Warnings...)}
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
