// Command lore-master-engine is the sidecar every Lore Master editor extension spawns.
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
	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
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
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version)

		return 0
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil)).With("version", version)
	sessions := sessionlifecycle.NewStore()
	network := sessionlifecycle.Environment{Logger: logger}
	methods := rpcserver.Methods{
		rpcprotocol.MethodPing:         rpcserver.Ping(version),
		rpcprotocol.MethodSessionOpen:  sessionlifecycle.OpenSession(sessions, network),
		rpcprotocol.MethodSessionClose: sessionlifecycle.CloseSession(sessions),
		rpcprotocol.MethodSpaceList:    catalogqueries.ListSpaces(sessions),
		rpcprotocol.MethodPageChildren: catalogqueries.ListChildren(sessions),
		rpcprotocol.MethodPageSearch:   catalogqueries.SearchPages(sessions),
	}
	if err := rpcserver.Serve(ctx, stdio{Reader: stdin, Writer: stdout}, methods, logger); err != nil {
		logger.Error("stopped", "error", err.Error())

		return 1
	}

	return 0
}

// stdio joins stdin and stdout into the stream the server reads and writes. Closing it
// closes neither: the process owns them.
type stdio struct {
	io.Reader
	io.Writer
}

func (stdio) Close() error { return nil }
