// Package clicommands is the command line of the engine binary: generate, sync and tree, run
// unattended for CI and git hooks. It does not reimplement anything. It talks to the very same
// method handlers the editors call, over an in-process JSON-RPC connection, so a sync from the
// command line plans, previews and writes exactly what the same sync from an editor would.
// Credentials come from the environment, never from a file.
package clicommands
