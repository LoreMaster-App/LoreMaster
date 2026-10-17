package rpcprotocol

// MethodDiagramRender draws one diagram to SVG with the engine's own renderer. It needs no
// session; the editor's Markdown preview uses it for D2, which no editor can draw by itself.
const MethodDiagramRender = "diagram/render"

// DiagramRenderParams is the diagram to draw.
type DiagramRenderParams struct {
	// Language is the fence language; only "d2" is drawn here (Mermaid is drawn by the editor).
	Language string `json:"language"`
	// Source is the diagram text.
	Source string `json:"source"`
}

// DiagramRenderResult is the drawing.
type DiagramRenderResult struct {
	// SVG is the complete SVG document.
	SVG string `json:"svg"`
}
