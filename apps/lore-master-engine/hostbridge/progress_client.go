package hostbridge

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

// Progress returns a callback that tells the editor how far the plan's execution got
// (host/progress). A notification that cannot be sent is dropped: progress is a
// courtesy, and the report at the end is the record.
func Progress(ctx context.Context, editor rpcserver.Peer, planID string) func(done int, total int, message string) {
	ctx = context.WithoutCancel(ctx)

	return func(done int, total int, message string) {
		_ = editor.Notify(ctx, rpcprotocol.MethodHostProgress, rpcprotocol.ProgressParams{PlanID: planID, Message: message, Done: done, Total: total})
	}
}
