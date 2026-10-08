package agentcommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/agentinstructions"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documenttree"
)

// AgentInstructions handles agent/instructions. Settings that are not valid are an
// invalid-settings error naming each problem, because instructions built on settings the
// sync would refuse would teach the agent something untrue.
func AgentInstructions() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.AgentInstructionsParams
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

		return rpcprotocol.AgentInstructionsResult{
			Instructions: agentinstructions.Compose(loaded.Settings, loaded.Exists, documenttree.NestingConventions()),
			HasConfig:    loaded.Exists,
		}, nil
	}
}
