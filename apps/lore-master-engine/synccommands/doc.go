// Package synccommands plans and executes syncs for the editor: sync/plan reads the
// workspace and its settings and plans one output without writing anything; sync/execute
// carries a plan out, asking the editor to draw diagrams and telling it the progress,
// then writes the annotations back into the files.
package synccommands
