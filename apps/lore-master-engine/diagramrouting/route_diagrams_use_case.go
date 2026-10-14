package diagramrouting

import (
	"context"
	"errors"
	"strings"

	"lore-master/libs/documentation-sync/d2rendering"
	"lore-master/libs/documentation-sync/platformport"
)

// router sends D2 to the engine's own renderer and every other language to the editor.
type router struct {
	native platformport.DiagramRenderer
	editor platformport.DiagramRenderer
}

// Route answers a renderer that draws D2 in the engine and asks editor for the rest. editor
// may be nil (a process with no editor, such as the command line): the other languages are
// then refused, and the sync shows them as code with its warning.
func Route(editor platformport.DiagramRenderer) platformport.DiagramRenderer {
	return router{native: d2rendering.New(), editor: editor}
}

// Render implements platformport.DiagramRenderer.
func (r router) Render(ctx context.Context, language string, source string) ([]byte, error) {
	if strings.EqualFold(language, d2rendering.Language) {
		return r.native.Render(ctx, language, source)
	}
	if r.editor == nil {
		return nil, errors.New("no editor is available to draw " + language + " diagrams")
	}

	return r.editor.Render(ctx, language, source)
}
