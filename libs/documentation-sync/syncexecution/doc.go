// Package syncexecution carries out a sync plan against a documentation platform:
// pages are written parents first, attachments uploaded when they changed, diagrams
// rendered, and every page's outcome reported with the annotation its file should
// carry from now on.
package syncexecution
