// Package syncstate keeps what a sync knows about its pages in one file, lore-master.json at
// the workspace root, instead of in an annotation at the top of every Markdown file.
//
// The file is the tree of each storage: per storage name, per Markdown file, the page it
// became (id, parent, version, content and render hashes, attachment hashes). A sync then
// changes one file in git, and moving a storage's parent page is one edit. What the author
// owns (title, parent and generated in the annotation) stays in the Markdown file.
//
// Files that still carry the old state in their annotation are read as before and migrated
// by the first sync that writes back: the state moves into lore-master.json and the file
// keeps only what the author set.
package syncstate
