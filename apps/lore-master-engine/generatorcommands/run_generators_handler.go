package generatorcommands

import (
	"context"
	"path/filepath"
	"slices"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/generatorregistry"
	"lore-master/libs/content-generation/generatorrunning"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// RunGenerators handles generators/run. Settings that are not valid are an invalid-settings
// error naming each problem; a generator that fails is reported in its own entry so the others
// still run.
func RunGenerators() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.GeneratorsRunParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		configured := loaded.Settings.Generators

		indexes, err := selected(params.Generators, len(configured))
		if err != nil {
			return nil, err
		}
		result := rpcprotocol.GeneratorsRunResult{Runs: []rpcprotocol.GeneratorRun{}}
		for _, index := range indexes {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			result.Runs = append(result.Runs, run(ctx, params.WorkspaceRoot, index, configured[index]))
		}

		return result, nil
	}
}

// selected is the generator indexes to run: the requested ones, each checked, or all.
func selected(requested []int, count int) ([]int, error) {
	if len(requested) == 0 {
		all := make([]int, count)
		for i := range all {
			all[i] = i
		}

		return all, nil
	}
	indexes := slices.Clone(requested)
	slices.Sort(indexes)
	indexes = slices.Compact(indexes)
	for _, index := range indexes {
		if index < 0 || index >= count {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "generator %d does not exist; %s has %d", index, workspacesettings.FileName, count)
		}
	}

	return indexes, nil
}

func run(ctx context.Context, workspaceRoot string, index int, generator workspacesettings.Generator) rpcprotocol.GeneratorRun {
	entry := rpcprotocol.GeneratorRun{Index: index, Type: generator.Type, Output: generator.Output}
	generate, found := generatorregistry.For(generator.Type)
	if !found {
		entry.Error = "there is no generator for type " + generator.Type
		return entry
	}
	spec := generatedfile.Spec{Type: generator.Type, Input: generator.Input, Output: generator.Output, Title: generator.Title}
	outcome, err := generatorrunning.Run(ctx, workspaceRoot, spec, generate)
	if err != nil {
		entry.Error = err.Error()
		return entry
	}
	entry.Written, entry.Unchanged, entry.Removed, entry.Warnings = outcome.Written, outcome.Unchanged, outcome.Removed, outcome.Warnings

	return entry
}
