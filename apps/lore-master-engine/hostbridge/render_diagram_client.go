package hostbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/platformport"
)

// DefaultRenderTimeout is how long the editor has to draw one diagram.
const DefaultRenderTimeout = 30 * time.Second

// DiagramRenderer draws diagrams by asking the editor (host/renderDiagram). It
// implements the sync's DiagramRenderer. Make one per sync: once the editor has missed
// a deadline it is not asked again, so a page full of diagrams costs one timeout, not
// one each. A diagram that is not drawn stays as code, with the sync's warning.
type DiagramRenderer struct {
	editor  rpcserver.Peer
	timeout time.Duration
	gaveUp  atomic.Bool
}

var _ platformport.DiagramRenderer = (*DiagramRenderer)(nil)

// NewDiagramRenderer asks editor, giving it timeout per diagram (DefaultRenderTimeout
// when zero).
func NewDiagramRenderer(editor rpcserver.Peer, timeout time.Duration) *DiagramRenderer {
	if timeout <= 0 {
		timeout = DefaultRenderTimeout
	}

	return &DiagramRenderer{editor: editor, timeout: timeout}
}

// Render implements platformport.DiagramRenderer.
func (r *DiagramRenderer) Render(ctx context.Context, language string, source string) ([]byte, error) {
	if r.gaveUp.Load() {
		return nil, errors.New("the editor did not answer an earlier diagram in time")
	}
	asking, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var drawn rpcprotocol.RenderDiagramResult
	err := r.editor.Call(asking, rpcprotocol.MethodHostRenderDiagram, rpcprotocol.RenderDiagramParams{Language: language, Source: source}, &drawn)
	switch {
	case ctx.Err() != nil:
		return nil, ctx.Err()
	case errors.Is(asking.Err(), context.DeadlineExceeded):
		r.gaveUp.Store(true)

		return nil, fmt.Errorf("the editor did not draw the diagram within %s", r.timeout)
	case err != nil:
		return nil, fmt.Errorf("the editor could not draw the diagram: %w", err)
	}
	svg := bytes.TrimSpace([]byte(drawn.SVG))
	if !bytes.Contains(svg, []byte("<svg")) {
		return nil, errors.New("the editor answered with something that is not an SVG")
	}

	return svg, nil
}
