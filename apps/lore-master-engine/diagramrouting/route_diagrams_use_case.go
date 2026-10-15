package diagramrouting

import (
	"context"
	"errors"
	"strings"

	"lore-master/libs/documentation-sync/d2rendering"
	"lore-master/libs/documentation-sync/platformport"
)

// router sends D2 to the engine's own renderer, Mermaid to the engine's browser when it has one
// that works, and the rest (and Mermaid, when there is no browser) to the editor.
type router struct {
	native  platformport.DiagramRenderer
	browser platformport.DiagramRenderer
	editor  platformport.DiagramRenderer
}

// Route answers a renderer that draws D2 in the engine and asks editor for the rest. editor
// may be nil (a process with no editor, such as the command line): the other languages are
// then refused, and the sync shows them as code with its warning.
func Route(editor platformport.DiagramRenderer) platformport.DiagramRenderer {
	return RouteWith(editor, nil)
}

// RouteWith is Route with a browser renderer for Mermaid, asked first. When it says it is
// unavailable (no browser installed) the editor draws the diagram, as without it; a diagram it
// could not draw is not asked of the editor again.
func RouteWith(editor platformport.DiagramRenderer, browser platformport.DiagramRenderer) platformport.DiagramRenderer {
	return router{native: d2rendering.New(), browser: browser, editor: editor}
}

// Render implements platformport.DiagramRenderer.
func (r router) Render(ctx context.Context, language string, source string) ([]byte, error) {
	if strings.EqualFold(language, d2rendering.Language) {
		return r.native.Render(ctx, language, source)
	}
	if r.browser != nil && strings.EqualFold(language, "mermaid") {
		svg, err := r.browser.Render(ctx, language, source)
		if err == nil || !errors.Is(err, platformport.ErrRendererUnavailable) {
			return svg, err
		}
		if r.editor == nil {
			return nil, err
		}
	}
	if r.editor == nil {
		return nil, errors.New("no editor is available to draw " + language + " diagrams")
	}

	return r.editor.Render(ctx, language, source)
}
