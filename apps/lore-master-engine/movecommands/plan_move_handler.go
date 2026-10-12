package movecommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/markdown-workspace/filemoves"
)

// PlanMove handles files/planMove.
func PlanMove() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.FilesPlanMoveParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		moves := make([]filemoves.Move, len(params.Moves))
		for i, move := range params.Moves {
			moves[i] = filemoves.Move{From: move.From, To: move.To}
		}

		plan, err := filemoves.PlanWorkspaceMoves(ctx, params.WorkspaceRoot, moves)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "%s", err.Error())
		}

		result := rpcprotocol.FilesPlanMoveResult{
			Renames:  make([]rpcprotocol.FileMove, 0, len(plan.Renames)),
			Edits:    make([]rpcprotocol.FileLineEdit, 0, len(plan.Edits)),
			Warnings: plan.Warnings,
		}
		for _, rename := range plan.Renames {
			result.Renames = append(result.Renames, rpcprotocol.FileMove{From: rename.From, To: rename.To})
		}
		for _, edit := range plan.Edits {
			result.Edits = append(result.Edits, rpcprotocol.FileLineEdit{Path: edit.Path, Line: edit.Line, Text: edit.Text})
		}

		return result, nil
	}
}
