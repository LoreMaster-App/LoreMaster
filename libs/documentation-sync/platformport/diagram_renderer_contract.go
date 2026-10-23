package platformport

import (
	"context"
	"errors"
)

// DiagramRenderer turns diagram source into an image. The editor implements it: a
// VS Code webview renders Mermaid with the same library the Markdown preview uses, so
// what is uploaded is what the author saw.
type DiagramRenderer interface {
	// Render returns an SVG document for source written in language ("mermaid").
	Render(ctx context.Context, language string, source string) ([]byte, error)
}

// ErrRendererUnavailable says a renderer cannot draw at all on this machine (no browser to
// draw in, say), as opposed to a diagram it could not draw. A caller that has another renderer
// asks that one instead; a diagram that failed is not asked twice.
var ErrRendererUnavailable = errors.New("the renderer is not available on this machine")

// DiagramProblem is one thing wrong with a diagram's source, as the author sees it in the editor.
type DiagramProblem struct {
	// Line and Column are 1-based within the diagram's own source; 0 when the language does not say.
	Line   int
	Column int
	// Message is the language's own explanation.
	Message string
}
