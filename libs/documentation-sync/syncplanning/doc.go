// Package syncplanning decides, before anything is written, what a sync will do to
// every page: create, update, move, retitle, adopt, leave alone, report as a conflict,
// or report as an orphan. The plan is plain data, so the editor can show it for
// approval before it runs.
package syncplanning
