package clicommands

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

const pagesUsage = "usage: lore-master-engine pages build --out DIR [--workspace DIR] [--output N] [--json]\n" +
	"       lore-master-engine pages publish [--workspace DIR] [--output N] [--json]\n"

// pagesCommand builds, publishes or checks the static site of a github-pages output: build
// writes it into a folder (for a pipeline that deploys the folder itself), publish pushes it
// to the output's branch, check says whether a publish would change anything.
func pagesCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	if len(args) == 0 || (args[0] != "build" && args[0] != "publish" && args[0] != "check") {
		_, _ = fmt.Fprint(env.Stderr, pagesUsage)

		return ExitUsage
	}
	if args[0] == "build" {
		return pagesBuildCommand(ctx, env, methods, args[1:])
	}

	if args[0] == "check" {
		return pagesCheckCommand(ctx, env, methods, args[1:])
	}

	return pagesPublishCommand(ctx, env, methods, args[1:])
}

func pagesBuildCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var only intList
	var out string
	parsed, code, ok := parseFlags("pages build", env, args, func(flags *flag.FlagSet) {
		flags.Var(&only, "output", "the github-pages output at this position in the outputs list (default: the only one)")
		flags.StringVar(&out, "out", "", "the folder to write the site into (required; created, and replaced when an earlier build wrote it)")
	})
	if !ok {
		return code
	}
	if out == "" {
		_, _ = fmt.Fprintln(env.Stderr, "--out is required")

		return ExitUsage
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(env.WorkingDir, out)
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		index, err := pagesOutputIndex(ctx, engine, parsed.workspace, only)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}

		var result rpcprotocol.PagesBuildResult
		params := rpcprotocol.PagesBuildParams{WorkspaceRoot: parsed.workspace, Output: index, OutDir: out}
		if err := engine.call(ctx, rpcprotocol.MethodPagesBuild, params, &result); err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		if parsed.json {
			printJSON(env.Stdout, result)
		} else {
			printProblems(env, result.Warnings, result.Errors)
			if len(result.Errors) == 0 {
				_, _ = fmt.Fprintf(env.Stdout, "built %d files into %s\n", result.Files, result.OutDir)
			}
		}
		if len(result.Errors) > 0 {
			return ExitBlocked
		}

		return ExitOK
	})
}

func pagesPublishCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var only intList
	parsed, code, ok := parseFlags("pages publish", env, args, func(flags *flag.FlagSet) {
		flags.Var(&only, "output", "the github-pages output at this position in the outputs list (default: the only one)")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		index, err := pagesOutputIndex(ctx, engine, parsed.workspace, only)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}

		var result rpcprotocol.PagesPublishResult
		params := rpcprotocol.PagesPublishParams{WorkspaceRoot: parsed.workspace, Output: index}
		if err := engine.call(ctx, rpcprotocol.MethodPagesPublish, params, &result); err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		if parsed.json {
			printJSON(env.Stdout, result)
		} else {
			printProblems(env, result.Warnings, result.Errors)
			switch {
			case len(result.Errors) > 0:
			case !result.Changed:
				_, _ = fmt.Fprintf(env.Stdout, "%s is already up to date (%d files)\n", result.Branch, result.Files)
			default:
				_, _ = fmt.Fprintf(env.Stdout, "published %d files to %s (%s)\n", result.Files, result.Branch, result.Commit)
			}
			if result.URL != "" && len(result.Errors) == 0 {
				_, _ = fmt.Fprintln(env.Stdout, result.URL)
			}
		}
		if len(result.Errors) > 0 {
			return ExitBlocked
		}

		return ExitOK
	})
}

func pagesCheckCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var only intList
	var exitCode bool
	parsed, code, ok := parseFlags("pages check", env, args, func(flags *flag.FlagSet) {
		flags.Var(&only, "output", "the github-pages output at this position in the outputs list (default: the only one)")
		flags.BoolVar(&exitCode, "exit-code", false, "exit 2 when the published site is out of date, so a pipeline can fail on it")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		index, err := pagesOutputIndex(ctx, engine, parsed.workspace, only)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}

		var result rpcprotocol.PagesCheckResult
		params := rpcprotocol.PagesCheckParams{WorkspaceRoot: parsed.workspace, Output: index}
		if err := engine.call(ctx, rpcprotocol.MethodPagesCheck, params, &result); err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}
		if parsed.json {
			printJSON(env.Stdout, result)
		} else {
			printProblems(env, result.Warnings, result.Errors)
			printCheck(env, result)
		}
		switch {
		case len(result.Errors) > 0:
			return ExitBlocked
		case exitCode && !result.UpToDate:
			return ExitBlocked
		}

		return ExitOK
	})
}

// printCheck writes the outcome of a check: one line when up to date, otherwise a line per
// file a publish would touch.
func printCheck(env Environment, result rpcprotocol.PagesCheckResult) {
	switch {
	case len(result.Errors) > 0:
	case result.UpToDate:
		_, _ = fmt.Fprintf(env.Stdout, "%s is up to date (%d files)\n", result.Branch, result.Files)
	default:
		_, _ = fmt.Fprintf(env.Stdout, "%s is out of date: %d files would change\n", result.Branch, result.ChangesTotal)
		for _, change := range result.Changes {
			_, _ = fmt.Fprintf(env.Stdout, "  %-8s %s\n", change.Kind, change.Path)
		}
		if result.ChangesTotal > len(result.Changes) {
			_, _ = fmt.Fprintf(env.Stdout, "  … and %d more\n", result.ChangesTotal-len(result.Changes))
		}
	}
}

// pagesOutputIndex is the position of the github-pages output to use: the one named, or the
// only one the settings have. Several without a choice is an error, so a build never guesses.
func pagesOutputIndex(ctx context.Context, engine *connection, workspace string, only []int) (int, error) {
	outputs, err := selectedOutputs(ctx, engine, workspace, only)
	if err != nil {
		return 0, err
	}

	var pages []output
	for _, candidate := range outputs {
		if candidate.settings.Platform == "github-pages" || candidate.settings.Platform == "github-wiki" {
			pages = append(pages, candidate)
		}
	}
	switch {
	case len(pages) == 0:
		return 0, fmt.Errorf("no github-pages or github-wiki output in .lore-master.yaml")
	case len(pages) > 1:
		return 0, fmt.Errorf("%d github-pages or github-wiki outputs; choose one with --output N", len(pages))
	}

	return pages[0].index, nil
}

// printProblems writes a result's warnings and errors, one per line, on stderr.
func printProblems(env Environment, warnings, errors []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintln(env.Stderr, "warning:", warning)
	}
	for _, problem := range errors {
		_, _ = fmt.Fprintln(env.Stderr, "error:", problem)
	}
}
