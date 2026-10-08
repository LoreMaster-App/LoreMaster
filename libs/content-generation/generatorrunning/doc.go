// Package generatorrunning runs one generator against a workspace and writes its pages to
// disk: new files are created, files whose content changed are rewritten with their sync
// annotation intact, unchanged files are left byte for byte alone, stale generated files are
// removed, and a file that was not generated is never touched.
package generatorrunning
