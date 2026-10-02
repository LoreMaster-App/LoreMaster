package rpcprotocol

// MethodHostRenderDiagram is a request from the engine to the editor: draw a diagram
// with the same library the editor's Markdown preview uses, so the page shows what the
// author saw.
const MethodHostRenderDiagram = "host/renderDiagram"

// RenderDiagramParams is the diagram.
type RenderDiagramParams struct {
	// Language is "mermaid".
	Language string `json:"language"`
	Source   string `json:"source"`
}

// RenderDiagramResult is the drawing.
type RenderDiagramResult struct {
	// SVG is a complete SVG document.
	SVG string `json:"svg"`
}
