// Package filemoves plans what else must change when Markdown files or folders are moved
// or renamed: the dotted children that nest under a moved file by name follow it, and
// "parent:" annotations that name a moved file (or sit in one) are rewritten. It only
// plans; the editor applies the plan as one undoable edit.
package filemoves
