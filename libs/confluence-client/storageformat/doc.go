// Package storageformat renders a resolved document into Confluence storage format, the
// XHTML dialect every edition accepts. Its input is its own small node tree, not a
// Markdown AST: the sync maps Markdown into it after resolving links to page titles and
// attachments, so this library never imports the Markdown side.
package storageformat
