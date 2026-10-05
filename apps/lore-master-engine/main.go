// Command lore-master-engine is the sidecar every LoreMaster editor extension spawns.
// It speaks JSON-RPC 2.0 on stdin and stdout (see package rpcprotocol), logs to
// stderr, and exits when the editor closes stdin.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"lore-master/apps/lore-master-engine/catalogqueries"
	"lore-master/apps/lore-master-engine/editiondetect"
	"lore-master/apps/lore-master-engine/hostbridge"
	"lore-master/apps/lore-master-engine/mcpserver"
	"lore-master/apps/lore-master-engine/pagescommands"
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/apps/lore-master-engine/settingscommands"
	"lore-master/apps/lore-master-engine/synccommands"
)

// version is stamped at build time: -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lore-master-engine", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print the version and exit")
	mcpMode := flags.Bool("mcp", false, "run as an MCP server over stdio, exposing LoreMaster's documentation rules")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)

		return 0
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil)).With("version", version)

	if *mcpMode {
		if err := mcpserver.Serve(ctx, stdin, stdout, version, logger); err != nil {
			logger.Error("mcp server stopped", "error", err.Error())

			return 1
		}

		return 0
	}

	if err := rpcserver.Serve(ctx, stdio{Reader: stdin, Writer: stdout}, engineMethods(logger), logger); err != nil {
		logger.Error("stopped", "error", err.Error())

		return 1
	}

	return 0
}

// engineMethods is every method the editor can call, wired to its slice.
func engineMethods(logger *slog.Logger) rpcserver.Methods {
	sessions := sessionlifecycle.NewStore()
	plans := synccommands.NewPlanStore()

	return rpcserver.Methods{
		rpcprotocol.MethodPing:          rpcserver.Ping(version),
		rpcprotocol.MethodEditionDetect: editiondetect.DetectEdition(logger),
		rpcprotocol.MethodSessionOpen:   sessionlifecycle.OpenSession(sessions, sessionlifecycle.Environment{Logger: logger}),
		rpcprotocol.MethodSessionClose:  sessionlifecycle.CloseSession(sessions),
		rpcprotocol.MethodSpaceList:     catalogqueries.ListSpaces(sessions),
		rpcprotocol.MethodPageChildren:  catalogqueries.ListChildren(sessions),
		rpcprotocol.MethodPageSearch:    catalogqueries.SearchPages(sessions),
		rpcprotocol.MethodSyncPlan:      synccommands.PlanSync(sessions, plans),
		rpcprotocol.MethodSyncExecute:   synccommands.ExecuteSync(sessions, plans, hostbridge.DefaultRenderTimeout),
		rpcprotocol.MethodSettingsRead:  settingscommands.ReadSettings(),
		rpcprotocol.MethodSettingsSave:  settingscommands.SaveSettings(),
		rpcprotocol.MethodPagesPublish:  pagescommands.PublishPages(),
	}
}

// stdio joins stdin and stdout into the stream the server reads and writes. Closing it
// closes neither: the process owns them.
type stdio struct {
	io.Reader
	io.Writer
}

func (stdio) Close() error { return nil }
