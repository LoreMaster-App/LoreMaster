package clicommands

import (
	"context"
	"flag"
	"fmt"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// generateCommand runs the generators of the loreMaster.generators setting: each writes ordinary Markdown into
// its output folder, which the next sync publishes like any other page.
func generateCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var only intList
	var prepare bool
	parsed, code, ok := parseFlags("generate", env, args, func(flags *flag.FlagSet) {
		flags.Var(&only, "generator", "run only the generator at this position in the generators list (repeatable; default all)")
		flags.BoolVar(&prepare, "prepare", false, "run each generator's prepare command first (the build or test run that produces what it reads)")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		runs, err := runGenerators(ctx, engine, parsed.workspace, only, prepare)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}

		code := ExitOK
		for _, run := range runs {
			if run.Error != "" {
				code = ExitFailed
			}
		}
		if parsed.json {
			printJSON(env.Stdout, map[string]any{"generators": runs})

			return code
		}
		if len(runs) == 0 {
			_, _ = fmt.Fprintln(env.Stdout, "no generators are configured")
		}
		for _, line := range generatorLines(runs) {
			_, _ = fmt.Fprintln(env.Stdout, line)
		}

		return code
	})
}

// runGenerators asks the engine to run the generators at the given positions (all when none).
func runGenerators(ctx context.Context, engine *connection, workspace string, only []int, prepare bool) ([]rpcprotocol.GeneratorRun, error) {
	var result rpcprotocol.GeneratorsRunResult
	if err := engine.call(ctx, rpcprotocol.MethodGeneratorsRun, rpcprotocol.GeneratorsRunParams{WorkspaceRoot: workspace, Generators: only, Prepare: prepare}, &result); err != nil {
		return nil, err
	}

	return result.Runs, nil
}
