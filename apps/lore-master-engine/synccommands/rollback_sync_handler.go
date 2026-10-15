package synccommands

import (
	"context"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/syncjournal"
)

// ListJournals handles sync/journals: the syncs of a workspace that can be rolled back.
func ListJournals(journalCache string) rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SyncJournalsParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		result := rpcprotocol.SyncJournalsResult{Journals: []rpcprotocol.JournalSummary{}}
		if journalCache == "" {
			return result, nil
		}
		journals, err := syncjournal.NewRepository(syncjournal.DirFor(journalCache, params.WorkspaceRoot)).List()
		if err != nil {
			return nil, err
		}
		for _, journal := range journals {
			summary := journal.Summarize()
			result.Journals = append(result.Journals, rpcprotocol.JournalSummary{
				ID: journal.ID, StartedAt: journal.StartedAt, BaseURL: journal.Output.BaseURL, Space: journal.Output.Space,
				ParentPageID: journal.Output.ParentPageID, Created: summary.Created, Updated: summary.Updated,
				Trashed: summary.Trashed, Files: summary.Files,
			})
		}

		return result, nil
	}
}

// RollbackSync handles sync/rollback: it undoes one journalled sync through a session on the
// site the sync wrote to. A complete rollback drops the journal; a partial one keeps it, so
// what was skipped can be forced later.
func RollbackSync(sessions *sessionlifecycle.Store, journalCache string) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SyncRollbackParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		session, err := sessions.Get(params.SessionID)
		if err != nil {
			return nil, err
		}
		if journalCache == "" {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "this engine keeps no sync journals")
		}
		repository := syncjournal.NewRepository(syncjournal.DirFor(journalCache, params.WorkspaceRoot))
		journal, err := pickJournal(repository, params.JournalID)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(strings.TrimRight(journal.Output.BaseURL, "/"), strings.TrimRight(session.BaseURL, "/")) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams,
				"that sync wrote to %s but the session is on %s; open a session on the site it wrote to", journal.Output.BaseURL, session.BaseURL)
		}

		outcome := syncjournal.Rollback(ctx, session.Platform.ForOutput(journal.Output), params.WorkspaceRoot, journal,
			syncjournal.Options{DryRun: params.DryRun, Force: params.Force})
		if !params.DryRun && outcome.Complete() {
			_ = repository.Remove(journal.ID)
		}

		result := rpcprotocol.SyncRollbackResult{
			JournalID: journal.ID, StartedAt: journal.StartedAt, DryRun: params.DryRun, Complete: outcome.Complete(),
			Pages: []rpcprotocol.RollbackPage{}, Files: []rpcprotocol.RollbackFile{},
		}
		for _, page := range outcome.Pages {
			result.Pages = append(result.Pages, rpcprotocol.RollbackPage{PageID: page.PageID, Title: page.Title, Kind: string(page.Kind), Result: page.Result, Reason: page.Reason})
		}
		for _, file := range outcome.Files {
			result.Files = append(result.Files, rpcprotocol.RollbackFile{Path: file.Path, Result: file.Result, Reason: file.Reason})
		}

		return result, nil
	}
}

func pickJournal(repository *syncjournal.Repository, id string) (syncjournal.Journal, error) {
	if id != "" {
		journal, err := repository.Load(id)
		if err != nil {
			return syncjournal.Journal{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "%v", err)
		}

		return journal, nil
	}
	journals, err := repository.List()
	if err != nil {
		return syncjournal.Journal{}, err
	}
	if len(journals) == 0 {
		return syncjournal.Journal{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "there is no sync to roll back in this workspace")
	}

	return journals[0], nil
}
