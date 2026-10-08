// Package changedetection tells which files of a workspace changed between two looks at it. A
// look is a Snapshot: the size, modification time and content hash of every file that can matter
// to the sync (Markdown, the configuration, and the sources, reports and specs the generators
// read). Polling snapshots needs no operating-system watcher, so it behaves the same on every
// OS and on network folders; the hash makes a file that was only touched, or saved without a
// change, count as unchanged.
package changedetection
