package clicommands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"lore-master/apps/lore-master-engine/rpcserver"
)

// Exit codes, so a pipeline can tell a broken run from a run that was stopped on purpose.
const (
	// ExitOK means everything asked for was done.
	ExitOK = 0
	// ExitFailed means something went wrong: settings that do not load, a credential that is
	// refused, a page that could not be written, a generator that failed.
	ExitFailed = 1
	// ExitBlocked means nothing is broken but the run stopped before changing the platform:
	// the plan has errors, or pages were edited there since the last sync and --force was not
	// given.
	ExitBlocked = 2
	// ExitUsage means the command line itself is wrong.
	ExitUsage = 64
)

// Environment is what a command needs from the process, so tests can supply their own.
type Environment struct {
	Getenv     func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
	WorkingDir string
	Version    string
}

// commands maps a subcommand to the function that runs it.
var commands = map[string]func(context.Context, Environment, rpcserver.Methods, []string) int{
	"generate": generateCommand,
	"pages":    pagesCommand,
	"sync":     syncCommand,
	"tree":     treeCommand,
	"watch":    watchCommand,
}

// IsCommand reports whether name is a command of the command line, so the binary can tell it
// from the flags of the editor protocol (no arguments, or --mcp).
func IsCommand(name string) bool {
	_, found := commands[name]

	return found || name == "help" || name == "version"
}

// Run executes one command line (args[0] is the command) against the engine's methods and
// returns the process's exit code.
func Run(ctx context.Context, args []string, env Environment, methods rpcserver.Methods) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		_, _ = fmt.Fprint(env.Stdout, usage)

		return ExitOK
	}
	if args[0] == "version" {
		_, _ = fmt.Fprintln(env.Stdout, env.Version)

		return ExitOK
	}
	command, found := commands[args[0]]
	if !found {
		_, _ = fmt.Fprintf(env.Stderr, "unknown command %q\n\n%s", args[0], usage)

		return ExitUsage
	}

	return command(ctx, env, methods, args[1:])
}

const usage = `lore-master-engine: sync a workspace's Markdown, and the pages generated from code, to a documentation platform.

Usage:
  lore-master-engine generate [--workspace DIR] [--generator N]... [--json]
  lore-master-engine tree     [--workspace DIR] [--output N]... [--json]
  lore-master-engine sync     [--workspace DIR] [--output N]... [--scope PATH]... [--generate]
                              [--yes] [--dry-run] [--force] [--prune] [--json]
  lore-master-engine watch    [--workspace DIR] [--output N]... [--yes] [--force]
                              [--debounce 2s] [--poll 1s]
  lore-master-engine pages build   --out DIR [--workspace DIR] [--output N] [--json]
  lore-master-engine pages publish [--workspace DIR] [--output N] [--json]
  lore-master-engine version

generate   run the generators of .lore-master.yaml (test results, Go, OpenAPI, TypeScript docs)
tree       show the page tree each output would sync, with each page's local status
sync       plan the sync of each output and, with --yes, apply it. Without --yes it only shows the plan.
           --generate runs the generators first. --force overwrites pages edited on the platform;
           --prune moves pages whose file is gone to the trash.

pages      build writes a github-pages output's static site into --out (replaced when an earlier build wrote it,
           refused when the folder holds other files); publish pushes it to the output's branch.

watch       keep the storage up to date: when files change and stay quiet for --debounce, regenerate only the
           generators that read them and sync only the pages that changed. Applies nothing without --yes.

Credentials come from the environment, never from a file:
  LORE_MASTER_EMAIL and LORE_MASTER_TOKEN   Confluence Cloud (email and API token)
  LORE_MASTER_PAT                           Confluence Data Center or Server (personal access token)
  LORE_MASTER_USER and LORE_MASTER_PASSWORD Confluence Server (basic)

Exit codes: 0 done, 1 failed, 2 stopped before changing the platform (plan errors, or pages
edited on the platform and no --force), 64 usage.
`

// intList is a flag that may be given several times, each a whole number.
type intList []int

func (l *intList) String() string { return fmt.Sprint(*l) }

func (l *intList) Set(value string) error {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 {
		return fmt.Errorf("%q is not a whole number", value)
	}
	*l = append(*l, number)

	return nil
}

// stringList is a flag that may be given several times.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	*l = append(*l, value)

	return nil
}

// common are the flags every command takes.
type common struct {
	workspace string
	json      bool
}

// parseFlags parses args for a command: the common flags, then whatever define adds. It
// returns the workspace folder as an absolute path, or an exit code when the line is not valid.
func parseFlags(name string, env Environment, args []string, define func(*flag.FlagSet)) (common, int, bool) {
	flags := flag.NewFlagSet("lore-master-engine "+name, flag.ContinueOnError)
	flags.SetOutput(env.Stderr)
	var parsed common
	flags.StringVar(&parsed.workspace, "workspace", "", "the workspace folder (default: the current folder)")
	flags.BoolVar(&parsed.json, "json", false, "print a machine-readable result instead of text")
	define(flags)
	if err := flags.Parse(args); err != nil {
		return common{}, ExitUsage, false
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(env.Stderr, "unexpected argument %q\n", flags.Arg(0))

		return common{}, ExitUsage, false
	}

	root := parsed.workspace
	if root == "" {
		root = env.WorkingDir
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		_, _ = fmt.Fprintf(env.Stderr, "workspace %q: %v\n", root, err)

		return common{}, ExitUsage, false
	}
	parsed.workspace = absolute

	return parsed, ExitOK, true
}

// withEngine runs fn against a connection to the engine, closing it afterwards. Progress lines
// go to stderr so they never mix with a result printed for a machine.
func withEngine(ctx context.Context, env Environment, methods rpcserver.Methods, fn func(*connection) int) int {
	connection := connect(ctx, methods, slog.New(slog.NewTextHandler(io.Discard, nil)), env.Stderr)
	defer connection.close()

	return fn(connection)
}

// printJSON writes a value as indented JSON on one document.
func printJSON(out io.Writer, value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(out, "{\"error\": %q}\n", err.Error())

		return
	}
	_, _ = fmt.Fprintln(out, string(encoded))
}

// worst is the more serious of two exit codes: a failure outranks a stop, a stop outranks success.
func worst(a int, b int) int {
	order := []int{ExitOK, ExitBlocked, ExitFailed}
	if slices.Index(order, a) >= slices.Index(order, b) {
		return a
	}

	return b
}
