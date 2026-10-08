package watchcommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/generatorrouting"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// RouteChanges handles watch/route. Settings that are not valid are an invalid-settings error
// naming each problem.
func RouteChanges() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.WatchRouteParams
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
		specs := make([]generatedfile.Spec, len(loaded.Settings.Generators))
		for i, generator := range loaded.Settings.Generators {
			specs[i] = generatedfile.Spec{Type: generator.Type, Input: generator.Input, Output: generator.Output, Title: generator.Title}
		}

		plan := generatorrouting.Route(params.Changed, specs)

		return rpcprotocol.WatchRouteResult{Everything: plan.Everything, Generators: nonNil(plan.Generators), Markdown: nonNilStrings(plan.Markdown)}, nil
	}
}

func nonNil(values []int) []int {
	if values == nil {
		return []int{}
	}

	return values
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
