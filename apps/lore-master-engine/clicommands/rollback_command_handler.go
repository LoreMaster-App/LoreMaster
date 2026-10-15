package clicommands

import (
	"context"
	"flag"
	"fmt"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// rollbackReport is what --json prints.
type rollbackReport struct {
	Journals []rpcprotocol.JournalSummary    `json:"journals,omitempty"`
	Result   *rpcprotocol.SyncRollbackResult `json:"result,omitempty"`
	Stopped  string                          `json:"stopped,omitempty"`
}

// rollbackCommand undoes a sync: the pages it created are deleted, the pages it updated are
// put back as they were and the local files get their earlier content back. Without --yes it
// only shows what would happen; pages and files edited after the sync are left alone unless
// --force says to discard those edits.
func rollbackCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var journalID string
	var list, yes, force bool
	parsed, code, ok := parseFlags("rollback", env, args, func(flags *flag.FlagSet) {
		flags.StringVar(&journalID, "journal", "", "the sync to undo, by id (default: the latest)")
		flags.BoolVar(&list, "list", false, "list the syncs that can be rolled back and stop")
		flags.BoolVar(&yes, "yes", false, "apply the rollback; without it only what would happen is shown")
		flags.BoolVar(&force, "force", false, "also roll back pages and files edited after the sync, discarding those edits")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		report := &rollbackReport{}
		code := runRollback(ctx, env, engine, parsed, journalID, list, yes, force, report)
		if parsed.json {
			printJSON(env.Stdout, report)
		}

		return code
	})
}

func runRollback(ctx context.Context, env Environment, engine *connection, parsed common, journalID string, list bool, yes bool, force bool, report *rollbackReport) int {
	say := func(format string, args ...any) {
		if !parsed.json {
			_, _ = fmt.Fprintf(env.Stdout, format+"\n", args...)
		}
	}

	var journals rpcprotocol.SyncJournalsResult
	if err := engine.call(ctx, rpcprotocol.MethodSyncJournals, rpcprotocol.SyncJournalsParams{WorkspaceRoot: parsed.workspace}, &journals); err != nil {
		_, _ = fmt.Fprintln(env.Stderr, "error:", err)

		return ExitFailed
	}
	report.Journals = journals.Journals
	if list {
		if len(journals.Journals) == 0 {
			say("no sync to roll back in this workspace")
		}
		for _, journal := range journals.Journals {
			say("%s  %s  %s/%s: %d created, %d updated, %d trashed, %d files",
				journal.ID, journal.StartedAt.Local().Format("2006-01-02 15:04"), journal.BaseURL, journal.Space, journal.Created, journal.Updated, journal.Trashed, journal.Files)
		}

		return ExitOK
	}

	target, found := pickJournal(journals.Journals, journalID)
	if !found {
		_, _ = fmt.Fprintln(env.Stderr, "error: there is no sync to roll back in this workspace (see --list)")

		return ExitFailed
	}

	credential, err := credentialFor(ctx, env, engine, target.BaseURL)
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, "error:", err)

		return ExitFailed
	}
	var session rpcprotocol.SessionOpenResult
	if err := engine.call(ctx, rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{BaseURL: target.BaseURL, Credential: credential}, &session); err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "error: signing in to %s: %v\n", target.BaseURL, err)

		return ExitFailed
	}
	defer func() {
		_ = engine.call(ctx, rpcprotocol.MethodSessionClose, rpcprotocol.SessionCloseParams{SessionID: session.SessionID}, nil)
	}()

	say("sync %s of %s: %d created, %d updated, %d files", target.ID, target.StartedAt.Local().Format("2006-01-02 15:04"), target.Created, target.Updated, target.Files)
	params := rpcprotocol.SyncRollbackParams{SessionID: session.SessionID, WorkspaceRoot: parsed.workspace, JournalID: target.ID, DryRun: !yes, Force: force}
	var result rpcprotocol.SyncRollbackResult
	if err := engine.call(ctx, rpcprotocol.MethodSyncRollback, params, &result); err != nil {
		_, _ = fmt.Fprintln(env.Stderr, "error:", err)

		return ExitFailed
	}
	report.Result = &result
	for _, page := range result.Pages {
		say("  page %-13s %s (%s)%s", page.Result, page.Title, page.PageID, reasonSuffix(page.Reason))
	}
	for _, file := range result.Files {
		say("  file %-13s %s%s", file.Result, file.Path, reasonSuffix(file.Reason))
	}
	if !yes {
		say("  (nothing changed: pass --yes to roll back)")

		return ExitOK
	}
	if !result.Complete {
		report.Stopped = "some pages or files were left alone; see above, or pass --force to discard the edits made since the sync"
		_, _ = fmt.Fprintln(env.Stderr, "  stopped:", report.Stopped)

		return ExitBlocked
	}
	say("rolled back")

	return ExitOK
}

func pickJournal(journals []rpcprotocol.JournalSummary, id string) (rpcprotocol.JournalSummary, bool) {
	for _, journal := range journals {
		if id == "" || journal.ID == id {
			return journal, true
		}
	}

	return rpcprotocol.JournalSummary{}, false
}

func reasonSuffix(reason string) string {
	if reason == "" {
		return ""
	}

	return ": " + reason
}
