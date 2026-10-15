// Package mermaidrendering draws Mermaid diagrams inside the engine, in a headless Chromium
// based browser (Chrome, Chromium or Edge) it starts itself, with the same Mermaid library the
// editor's preview uses (the gzip next to this file; refresh it from the extension's mermaid
// dependency when that is upgraded). That is what lets image mode work from the command line and
// CI, and keeps one diagram that never finishes from holding up the rest: it fails alone, and the
// browser tab is replaced for the next one.
//
// When no browser can be found the renderer says so with platformport.ErrRendererUnavailable, so
// the caller can ask the editor instead.
package mermaidrendering
