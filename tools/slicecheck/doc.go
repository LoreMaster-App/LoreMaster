// Package slicecheck enforces the file-layout half of the vertical-slice
// rules for the Go projects in this repository (docs/architecture/
// vertical-feature-slices.md). The Go compiler already enforces the other
// half: unexported identifiers are the slice's barrel, and import cycles are
// compile errors. It is a test, not a linter, and the convention stands
// without it (MarkDoc issue #47).
package slicecheck
