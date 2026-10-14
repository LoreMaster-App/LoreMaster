// Package d2rendering draws D2 diagrams to SVG inside the engine, with the D2 library and its
// dagre layout. Unlike Mermaid, which needs a browser and so the editor's webview, D2 needs
// nothing outside the process: a diagram is drawn the same way by the extension, the command
// line on CI and watch mode, and the SVG is deterministic, so an unchanged diagram uploads
// nothing new.
package d2rendering
