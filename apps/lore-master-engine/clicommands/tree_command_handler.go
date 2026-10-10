package clicommands

import (
	"context"
	"flag"
	"fmt"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// treeCommand shows the page tree each output would sync, parents first, with what the files
// alone say about each page: new, synced, or changed since the last sync.
func treeCommand(ctx context.Context, env Environment, methods rpcserver.Methods, args []string) int {
	var only intList
	parsed, code, ok := parseFlags("tree", env, args, func(flags *flag.FlagSet) {
		flags.Var(&only, "output", "show only the output at this position in the outputs list (repeatable; default all)")
	})
	if !ok {
		return code
	}

	return withEngine(ctx, env, methods, func(engine *connection) int {
		outputs, err := selectedOutputs(ctx, engine, parsed.workspace, only)
		if err != nil {
			_, _ = fmt.Fprintln(env.Stderr, "error:", err)

			return ExitFailed
		}

		code := ExitOK
		shown := map[string]any{}
		for _, output := range outputs {
			var tree rpcprotocol.WorkspaceTreeResult
			if err := engine.call(ctx, rpcprotocol.MethodWorkspaceTree, rpcprotocol.WorkspaceTreeParams{WorkspaceRoot: parsed.workspace, Output: output.index}, &tree); err != nil {
				_, _ = fmt.Fprintf(env.Stderr, "output %d: %v\n", output.index, err)
				code = ExitFailed

				continue
			}
			if len(tree.Problems) > 0 {
				code = ExitFailed
			}
			if parsed.json {
				shown[fmt.Sprint(output.index)] = tree

				continue
			}
			_, _ = fmt.Fprintf(env.Stdout, "%s\n", output.describe())
			for _, line := range treeLines(tree.Nodes) {
				_, _ = fmt.Fprintln(env.Stdout, line)
			}
			for _, problem := range tree.Problems {
				_, _ = fmt.Fprintln(env.Stdout, "  problem: "+problem)
			}
			for _, warning := range tree.Warnings {
				_, _ = fmt.Fprintln(env.Stdout, "  warning: "+warning)
			}
			for _, line := range leftOutLines(tree) {
				_, _ = fmt.Fprintln(env.Stdout, line)
			}
		}
		if parsed.json {
			printJSON(env.Stdout, map[string]any{"outputs": shown})
		}

		return code
	})
}

// output is one storage of the settings with its position.
type output struct {
	index    int
	settings rpcprotocol.Output
}

// describe names the storage for a heading.
func (o output) describe() string {
	switch o.settings.Platform {
	case "github-pages":
		return fmt.Sprintf("output %d: GitHub Pages", o.index)
	case "github-wiki":
		return fmt.Sprintf("output %d: GitHub Wiki", o.index)
	default:
		return fmt.Sprintf("output %d: Confluence %s (%s)", o.index, o.settings.Space, o.settings.BaseURL)
	}
}

// configured reports whether the output has what it needs to be used: a site and a space for
// Confluence (the blank scaffold a first sync would fill in is not an output yet).
func (o output) configured() bool {
	if o.settings.Platform == "github-pages" || o.settings.Platform == "github-wiki" {
		return true
	}

	return o.settings.Platform == "confluence" && o.settings.BaseURL != "" && o.settings.Space != ""
}

// selectedOutputs reads the settings and returns the outputs asked for (by position), or every
// configured output when none is named. An unknown position is an error.
func selectedOutputs(ctx context.Context, engine *connection, workspace string, only []int) ([]output, error) {
	var read rpcprotocol.SettingsReadResult
	if err := engine.call(ctx, rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: workspace}, &read); err != nil {
		return nil, err
	}

	var outputs []output
	if len(only) == 0 {
		for index, settings := range read.Settings.Outputs {
			if candidate := (output{index: index, settings: settings}); candidate.configured() {
				outputs = append(outputs, candidate)
			}
		}

		return outputs, nil
	}
	for _, index := range only {
		if index >= len(read.Settings.Outputs) {
			return nil, fmt.Errorf("output %d does not exist; the settings have %d", index, len(read.Settings.Outputs))
		}
		outputs = append(outputs, output{index: index, settings: read.Settings.Outputs[index]})
	}

	return outputs, nil
}

// leftOutLines lists the files the scan skipped, each with the rule that skipped it, under a
// count; nothing when every Markdown file is read.
func leftOutLines(tree rpcprotocol.WorkspaceTreeResult) []string {
	if len(tree.LeftOut) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("  left out (%d):", tree.LeftOutTotal)}
	for _, left := range tree.LeftOut {
		reason := left.Rule
		switch {
		case left.Rule == "outside-roots":
			reason = "outside the roots"
		case left.Source != "":
			reason = fmt.Sprintf("%s (%s: %s)", left.Rule, left.Source, left.Pattern)
		case left.Pattern != "":
			reason = fmt.Sprintf("%s (%s)", left.Rule, left.Pattern)
		}
		lines = append(lines, fmt.Sprintf("    %s  - %s", left.Path, reason))
	}
	if tree.LeftOutTotal > len(tree.LeftOut) {
		lines = append(lines, fmt.Sprintf("    ... and %d more", tree.LeftOutTotal-len(tree.LeftOut)))
	}

	return lines
}
