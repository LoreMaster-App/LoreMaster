// Package godocs is the go-docs generator: it reads the Go packages in the workspace and
// writes their documentation as Markdown — an index page and one page per package, nested
// like the directories — from the same doc comments `go doc` shows. Build constraints are
// ignored, so the pages are the same on every platform.
package godocs
