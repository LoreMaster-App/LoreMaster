// Package syncjournal keeps track of what a sync changed, so it can be undone: the pages it
// created, the pages it updated (with how they were before), and the local files it
// rewrote (with their content before). A rollback deletes the created pages, puts the
// updated ones back and restores the files, refusing anything edited since the sync unless
// it is forced.
package syncjournal
