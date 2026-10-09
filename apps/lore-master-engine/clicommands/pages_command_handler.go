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

// pagesCommand builds or publishes the static site of a github-pages output: build writes it
// into a folder (for a pipeline that deploys the folder itself), publish pushes it to the
// output's branch.
func pagesCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	if len(args) == 0 || (args[0] != "build" && args[0] != "publish") {
		_, _ = fmt.Fprint(env.Stderr, pagesUsage)

		return ExitUsage
	}
	if args[0] == "build" {
		return pagesBuildCommand(ctx, env, methods, args[1:])
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

// pagesOutputIndex is the position of the github-pages output to use: the one named, or the
// only one the settings have. Several without a choice is an error, so a build never guesses.
func pagesOutputIndex(ctx context.Context, engine *connection, workspace string, only []int) (int, error) {
	outputs, err := selectedOutputs(ctx, engine, workspace, only)
	if err != nil {
		return 0, err
	}

	var pages []output
	for _, candidate := range outputs {
		if candidate.settings.Platform == "github-pages" {
			pages = append(pages, candidate)
		}
	}
	switch {
	case len(pages) == 0:
		return 0, fmt.Errorf("no github-pages output in .lore-master.yaml")
	case len(pages) > 1:
		return 0, fmt.Errorf("%d github-pages outputs; choose one with --output N", len(pages))
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
