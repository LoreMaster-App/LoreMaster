package diagramcommands

import (
	"context"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/d2rendering"
)

// RenderDiagram handles diagram/render: draw a D2 diagram to SVG. A diagram D2 rejects comes back
// as an invalid-params error carrying D2's own message, so the preview can show it.
func RenderDiagram() rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.DiagramRenderParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !strings.EqualFold(params.Language, d2rendering.Language) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "the engine draws %q diagrams here, not %q", d2rendering.Language, params.Language)
		}
		svg, err := d2rendering.RenderSVG(ctx, params.Source, 0)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "%s", err.Error())
		}

		return rpcprotocol.DiagramRenderResult{SVG: string(svg)}, nil
	}
}
