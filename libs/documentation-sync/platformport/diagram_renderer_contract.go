package platformport

import "context"

// DiagramRenderer turns diagram source into an image. The editor implements it: a
// VS Code webview renders Mermaid with the same library the Markdown preview uses, so
// what is uploaded is what the author saw.
type DiagramRenderer interface {
	// Render returns an SVG document for source written in language ("mermaid").
	Render(ctx context.Context, language string, source string) ([]byte, error)
}
