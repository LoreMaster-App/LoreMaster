package d2rendering

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	d2log "oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
	"oss.terrastruct.com/util-go/go2"

	"lore-master/libs/documentation-sync/platformport"
)

// Language is the diagram language this renderer draws.
const Language = "d2"

// DefaultTimeout bounds one diagram: a layout that does not settle must not stall a sync.
const DefaultTimeout = 30 * time.Second

// padding is the margin around the drawing, in pixels. D2's own default (100) leaves a wide
// empty frame, which reads badly inside a page.
const padding = 20

// Renderer draws D2 diagrams. It implements the sync's DiagramRenderer for the "d2" language
// and refuses any other, so it can sit behind a router that sends Mermaid to the editor.
type Renderer struct {
	timeout time.Duration
}

var _ platformport.DiagramRenderer = (*Renderer)(nil)

// New makes a renderer with DefaultTimeout.
func New() *Renderer {
	return &Renderer{timeout: DefaultTimeout}
}

// Render implements platformport.DiagramRenderer.
func (r *Renderer) Render(ctx context.Context, language string, source string) ([]byte, error) {
	if !strings.EqualFold(language, Language) {
		return nil, fmt.Errorf("the D2 renderer does not draw %q", language)
	}

	return RenderSVG(ctx, source, r.timeout)
}

// ruler measures text for the layout. Loading its fonts is slow and it keeps state, so one is
// made per process and used by one diagram at a time.
var (
	rulerOnce sync.Once
	ruler     *textmeasure.Ruler
	rulerErr  error
	drawing   sync.Mutex
)

// RenderSVG compiles source, lays it out with dagre and returns the SVG. A timeout of zero
// means DefaultTimeout. An invalid diagram is an error carrying D2's own message.
func RenderSVG(ctx context.Context, source string, timeout time.Duration) ([]byte, error) {
	if strings.TrimSpace(source) == "" {
		return nil, fmt.Errorf("the D2 diagram is empty")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	rulerOnce.Do(func() { ruler, rulerErr = textmeasure.NewRuler() })
	if rulerErr != nil {
		return nil, fmt.Errorf("preparing the D2 text measurer: %w", rulerErr)
	}

	type outcome struct {
		svg []byte
		err error
	}
	done := make(chan outcome, 1)
	// D2's layout runs in a JavaScript runtime that cannot be cancelled, so the wait is bounded
	// here; a diagram that never settles leaves its goroutine behind, not the sync.
	go func() {
		drawing.Lock()
		defer drawing.Unlock()
		svg, err := draw(ctx, source)
		done <- outcome{svg: svg, err: err}
	}()

	select {
	case result := <-done:
		return result.svg, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("the D2 diagram was not laid out within %s", timeout)
	}
}

func draw(ctx context.Context, source string) ([]byte, error) {
	// D2 logs through a logger carried in the context and complains with a stack when there is
	// none; an engine's stderr is not the place for its debug lines.
	ctx = d2log.With(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	diagram, _, err := d2lib.Compile(ctx, source, &d2lib.CompileOptions{
		Ruler: ruler,
		LayoutResolver: func(string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	return d2svg.Render(diagram, &d2svg.RenderOpts{
		Pad:     go2.Pointer(int64(padding)),
		ThemeID: go2.Pointer(int64(0)),
	})
}
