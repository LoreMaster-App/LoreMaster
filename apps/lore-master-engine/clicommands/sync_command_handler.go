package clicommands

import (
	"context"
	"flag"
	"fmt"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// syncOptions are the flags of the sync command.
type syncOptions struct {
	outputs  intList
	scope    stringList
	generate bool
	yes      bool
	dryRun   bool
	force    bool
	prune    bool
}

// syncCommand plans the sync of each output and, with --yes, applies it. Without --yes it only
// shows the plan, so running it by hand is safe and a pipeline has to say it means it.
func syncCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var options syncOptions
	parsed, code, ok := parseFlags("sync", env, args, func(flags *flag.FlagSet) {
		flags.Var(&options.outputs, "output", "sync only the output at this position in the outputs list (repeatable; default all)")
		flags.Var(&options.scope, "scope", "limit a Confluence sync to this workspace-relative file and the ancestors it needs (repeatable)")
		flags.BoolVar(&options.generate, "generate", false, "run the generators first, so generated pages are as fresh as the code")
		flags.BoolVar(&options.yes, "yes", false, "apply the plan; without it the plan is only shown")
		flags.BoolVar(&options.dryRun, "dry-run", false, "show the plan and change nothing, even with --yes")
		flags.BoolVar(&options.force, "force", false, "overwrite pages edited on the platform since the last sync")
		flags.BoolVar(&options.prune, "prune", false, "move pages whose file is gone to the trash")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		report := &syncReport{Outputs: []outputReport{}}
		code := runSync(ctx, env, engine, parsed, options, report)
		if parsed.json {
			printJSON(env.Stdout, report)
		}

		return code
	})
}

// syncReport is what --json prints.
type syncReport struct {
	Generators []rpcprotocol.GeneratorRun `json:"generators,omitempty"`
	Outputs    []outputReport             `json:"outputs"`
}

// outputReport is one output's part of the report.
type outputReport struct {
	Index    int                             `json:"index"`
	Platform string                          `json:"platform"`
	Plan     *rpcprotocol.SyncPlanResult     `json:"plan,omitempty"`
	Result   *rpcprotocol.SyncExecuteResult  `json:"result,omitempty"`
	Pages    *rpcprotocol.PagesPublishResult `json:"pages,omitempty"`
	// Stopped says why the run did not change the platform: plan errors, or conflicts.
	Stopped string `json:"stopped,omitempty"`
	Error   string `json:"error,omitempty"`
}

func runSync(ctx context.Context, env Environment, engine *connection, parsed common, options syncOptions, report *syncReport) int {
	say := func(format string, args ...any) {
		if !parsed.json {
			_, _ = fmt.Fprintf(env.Stdout, format+"\n", args...)
		}
	}

	if options.generate {
		runs, err := runGenerators(ctx, engine, parsed.workspace, nil)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		report.Generators = runs
		failed := false
		for _, run := range runs {
			failed = failed || run.Error != ""
		}
		for _, line := range generatorLines(runs) {
			say("%s", line)
		}
		if failed {
			_, _ = fmt.Fprintln(env.Stderr, "a generator failed, so the pages it writes may be missing or stale; nothing was synced")

			return ExitFailed
		}
	}

	outputs, err := selectedOutputs(ctx, engine, parsed.workspace, options.outputs)
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, "error:", err)

		return ExitFailed
	}
	if len(outputs) == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "error: there is no configured output to sync; run a sync from an editor first, or edit .lore-master.yaml")

		return ExitFailed
	}

	code := ExitOK
	for _, output := range outputs {
		say("%s", output.describe())
		entry := outputReport{Index: output.index, Platform: output.settings.Platform}
		var outputCode int
		if output.settings.Platform == "github-pages" {
			outputCode = publishSite(ctx, env, engine, parsed, options, output, &entry, say)
		} else {
			outputCode = syncConfluence(ctx, env, engine, parsed, options, output, &entry, say)
		}
		report.Outputs = append(report.Outputs, entry)
		code = worst(code, outputCode)
	}

	return code
}

// publishSite publishes a github-pages output; it needs no session, since git carries the
// credentials.
func publishSite(ctx context.Context, env Environment, engine *connection, parsed common, options syncOptions, output output, entry *outputReport, say func(string, ...any)) int {
	if options.dryRun || !options.yes {
		say("  would publish the site to the output's branch (%s)", applyHint(options))

		return ExitOK
	}
	var published rpcprotocol.PagesPublishResult
	if err := engine.call(ctx, rpcprotocol.MethodPagesPublish, rpcprotocol.PagesPublishParams{WorkspaceRoot: parsed.workspace, Output: output.index}, &published); err != nil {
		entry.Error = err.Error()
		_, _ = fmt.Fprintf(env.Stderr, "output %d: %v\n", output.index, err)

		return ExitFailed
	}
	entry.Pages = &published
	for _, warning := range published.Warnings {
		say("  warning: %s", warning)
	}
	if len(published.Errors) > 0 {
		for _, problem := range published.Errors {
			_, _ = fmt.Fprintf(env.Stderr, "  error: %s\n", problem)
		}

		return ExitFailed
	}
	if published.Changed {
		say("  published %d files to %s (%s)", published.Files, published.Branch, published.Commit)
	} else {
		say("  the site already matched %s; nothing to publish", published.Branch)
	}

	return ExitOK
}

// syncConfluence plans one Confluence output and, when told to, applies the plan.
func syncConfluence(ctx context.Context, env Environment, engine *connection, parsed common, options syncOptions, output output, entry *outputReport, say func(string, ...any)) int {
	fail := func(err error) int {
		entry.Error = err.Error()
		_, _ = fmt.Fprintf(env.Stderr, "output %d: %v\n", output.index, err)

		return ExitFailed
	}

	credential, err := credentialFromEnvironment(env.Getenv)
	if err != nil {
		return fail(err)
	}
	var session rpcprotocol.SessionOpenResult
	if err := engine.call(ctx, rpcprotocol.MethodSessionOpen, rpcprotocol.SessionOpenParams{BaseURL: output.settings.BaseURL, Credential: credential}, &session); err != nil {
		return fail(fmt.Errorf("signing in to %s: %w", output.settings.BaseURL, err))
	}
	defer func() {
		_ = engine.call(ctx, rpcprotocol.MethodSessionClose, rpcprotocol.SessionCloseParams{SessionID: session.SessionID}, nil)
	}()

	var plan rpcprotocol.SyncPlanResult
	if err := engine.call(ctx, rpcprotocol.MethodSyncPlan, rpcprotocol.SyncPlanParams{SessionID: session.SessionID, WorkspaceRoot: parsed.workspace, Output: output.index, Scope: options.scope}, &plan); err != nil {
		return fail(err)
	}
	entry.Plan = &plan
	say("  plan: %s", countLine(plan.Counts))
	for _, line := range planLines(plan) {
		say("%s", line)
	}
	for _, warning := range plan.Warnings {
		say("  warning: %s", warning)
	}
	if len(plan.Errors) > 0 {
		for _, problem := range plan.Errors {
			_, _ = fmt.Fprintf(env.Stderr, "  error: %s\n", problem)
		}
		entry.Stopped = "the plan has errors; fix them and run again"

		return ExitBlocked
	}
	if conflicts := plan.Counts["conflict"]; conflicts > 0 && !options.force {
		entry.Stopped = fmt.Sprintf("%d page(s) were edited on the platform since the last sync; pass --force to overwrite them", conflicts)
		_, _ = fmt.Fprintf(env.Stderr, "  stopped: %s\n", entry.Stopped)

		return ExitBlocked
	}
	if options.dryRun || !options.yes {
		say("  (%s)", applyHint(options))

		return ExitOK
	}

	var result rpcprotocol.SyncExecuteResult
	if err := engine.call(ctx, rpcprotocol.MethodSyncExecute, rpcprotocol.SyncExecuteParams{PlanID: plan.PlanID, Force: options.force, Prune: options.prune}, &result); err != nil {
		return fail(err)
	}
	entry.Result = &result
	for _, line := range outcomeLines(result) {
		say("%s", line)
	}
	for _, warning := range result.Warnings {
		say("  warning: %s", warning)
	}
	for _, page := range result.Pages {
		if page.Outcome == "failed" {
			return ExitFailed
		}
	}

	return ExitOK
}

// applyHint tells the user what would make the run change anything.
func applyHint(options syncOptions) string {
	if options.dryRun {
		return "dry run: nothing was changed"
	}

	return "pass --yes to apply"
}
