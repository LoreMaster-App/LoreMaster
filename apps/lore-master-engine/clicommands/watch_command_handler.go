package clicommands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
	"time"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/watchcommands"
	"lore-master/libs/markdown-workspace/changedetection"
)

// watchCommand keeps the storage up to date while the workspace changes: it watches the
// Markdown, the settings and the inputs of the generators, and once nothing has changed for a
// moment it regenerates only the generators that read what changed and syncs only the pages
// that changed. It runs until interrupted. Like sync, it applies nothing without --yes.
func watchCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var options syncOptions
	var quiet, poll time.Duration
	parsed, code, ok := parseFlags("watch", env, args, func(flags *flag.FlagSet) {
		flags.Var(&options.outputs, "output", "watch only the output at this position in the outputs list (repeatable; default all)")
		flags.BoolVar(&options.yes, "yes", false, "apply each batch; without it each batch is only planned and shown")
		flags.BoolVar(&options.force, "force", false, "overwrite pages edited on the platform since the last sync")
		flags.DurationVar(&quiet, "debounce", 2*time.Second, "how long nothing may change before the changes are applied")
		flags.DurationVar(&poll, "poll", time.Second, "how often the workspace is looked at")
	})
	if !ok {
		return code
	}
	if parsed.json {
		_, _ = fmt.Fprintln(env.Stderr, "--json is not available for watch")

		return ExitUsage
	}
	if quiet <= 0 || poll <= 0 {
		_, _ = fmt.Fprintln(env.Stderr, "--debounce and --poll must be positive")

		return ExitUsage
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		watcher := &workspaceWatcher{env: env, engine: engine, parsed: parsed, options: options}
		if err := watcher.start(ctx); err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		watcher.say("watching %s (every %s, applying after %s of quiet%s); Ctrl+C stops", parsed.workspace, poll, quiet, dryNote(options))

		err := watchcommands.Watch(ctx, watcher.source, watcher.apply,
			watchcommands.Options{Interval: poll, Quiet: quiet, MaxBackoff: max(time.Minute, quiet)}, watcher)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		watcher.say("stopped")

		return ExitOK
	})
}

func dryNote(options syncOptions) string {
	if options.yes {
		return ""
	}

	return "; without --yes batches are only planned"
}

// workspaceWatcher holds what watching needs between batches: the last look at the workspace.
type workspaceWatcher struct {
	env     Environment
	engine  *connection
	parsed  common
	options syncOptions
	current changedetection.Snapshot
}

func (w *workspaceWatcher) say(format string, args ...any) {
	_, _ = fmt.Fprintf(w.env.Stdout, "[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

// start takes the first look, which is the baseline changes are measured from.
func (w *workspaceWatcher) start(ctx context.Context) error {
	snapshot, err := changedetection.Take(ctx, w.parsed.workspace, changedetection.Snapshot{})
	if err != nil {
		return err
	}
	w.current = snapshot

	return nil
}

// source looks at the workspace and returns what changed since the last look.
func (w *workspaceWatcher) source(ctx context.Context) ([]string, error) {
	next, err := changedetection.Take(ctx, w.parsed.workspace, w.current)
	if err != nil {
		return nil, err
	}
	changed := changedetection.Changed(w.current, next)
	w.current = next

	return changed, nil
}

// apply handles one batch: it asks the engine what the changes call for, regenerates only those
// generators, and syncs only the pages that changed or were rewritten. A generator that fails is
// reported but does not stop the sync of the rest. A sync that fails is an error, so the batch is
// tried again later; one that stops on plan errors or on pages edited on the platform is not,
// since trying again changes nothing until the files do.
func (w *workspaceWatcher) apply(ctx context.Context, changed []string) error {
	var route rpcprotocol.WatchRouteResult
	if err := w.engine.call(ctx, rpcprotocol.MethodWatchRoute, rpcprotocol.WatchRouteParams{WorkspaceRoot: w.parsed.workspace, Changed: changed}, &route); err != nil {
		return err
	}

	written := map[string]bool{}
	if len(route.Generators) > 0 {
		runs, err := runGenerators(ctx, w.engine, w.parsed.workspace, route.Generators)
		if err != nil {
			return err
		}
		for _, line := range generatorLines(runs) {
			w.say("%s", line)
		}
		for _, run := range runs {
			for _, file := range run.Written {
				written[file] = true
			}
		}
	}

	scope := slices.Clone(route.Markdown)
	for file := range written {
		if strings.HasSuffix(strings.ToLower(file), ".md") && !slices.Contains(scope, file) {
			scope = append(scope, file)
		}
	}
	slices.Sort(scope)
	if len(scope) == 0 && !route.Everything {
		w.say("nothing to sync for %s", summary(changed))
		w.settle(ctx, written, scope, false)

		return nil
	}

	options := w.options
	if route.Everything {
		options.scope = nil
		w.say("settings changed: syncing everything")
	} else {
		options.scope = scope
		w.say("syncing %s", summary(scope))
	}
	report := &syncReport{Outputs: []outputReport{}}
	code := runSync(ctx, w.env, w.engine, w.parsed, options, report)
	w.settle(ctx, written, scope, route.Everything)

	switch code {
	case ExitFailed:
		return errors.New("the sync failed (see above)")
	case ExitBlocked:
		w.say("stopped before changing anything (see above); it is tried again when the files change")
	}

	return nil
}

// settle makes what the batch itself wrote, the generated pages and the annotations a sync adds
// to pages, part of the baseline, so it is not mistaken for the next change. Anything else that
// differs, such as an edit made while the batch ran, is left to be seen at the next look. After a
// sync of everything every Markdown file may have been annotated, so all of them are adopted.
func (w *workspaceWatcher) settle(ctx context.Context, written map[string]bool, scope []string, everything bool) {
	after, err := changedetection.Take(ctx, w.parsed.workspace, w.current)
	if err != nil {
		return
	}
	adopt := func(file string) {
		if state, found := after.Files[file]; found {
			w.current.Files[file] = state
		} else {
			delete(w.current.Files, file)
		}
	}
	for file := range written {
		adopt(file)
	}
	for _, file := range scope {
		adopt(file)
	}
	if everything {
		for file := range after.Files {
			if strings.HasSuffix(strings.ToLower(file), ".md") {
				adopt(file)
			}
		}
		for file := range w.current.Files {
			if _, found := after.Files[file]; !found && strings.HasSuffix(strings.ToLower(file), ".md") {
				delete(w.current.Files, file)
			}
		}
	}
}

// summary names up to three files, then counts the rest.
func summary(files []string) string {
	if len(files) <= 3 {
		return strings.Join(files, ", ")
	}

	return fmt.Sprintf("%s and %d more", strings.Join(files[:3], ", "), len(files)-3)
}

// Changes implements watchcommands.Reporter.
func (w *workspaceWatcher) Changes(changed []string) {
	w.say("changed: %s", summary(changed))
}

// Applied implements watchcommands.Reporter.
func (w *workspaceWatcher) Applied(changed []string) {
	w.say("done (%d file(s))", len(changed))
}

// Failed implements watchcommands.Reporter.
func (w *workspaceWatcher) Failed(_ []string, err error, retryIn time.Duration) {
	w.say("failed: %v; trying again in %s", err, retryIn.Round(time.Second))
}
