package clicommands

import (
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcserver"
)

func TestTheCommandsAreRecognised(t *testing.T) {
	for _, name := range []string{"generate", "sync", "tree", "help", "version"} {
		if !IsCommand(name) {
			t.Errorf("%s is not recognised", name)
		}
	}
	for _, name := range []string{"", "--mcp", "-version", "syncs", "deploy"} {
		if IsCommand(name) {
			t.Errorf("%q is recognised", name)
		}
	}
}

func TestHelpAndVersionNeedNoEngine(t *testing.T) {
	double := newEngineDouble(map[string]func(rpcserver.Call) (any, error){})

	help, _, code := double.run(t, nil, "help")
	if code != ExitOK || !strings.Contains(help, "lore-master-engine sync") || !strings.Contains(help, "LORE_MASTER_TOKEN") || !strings.Contains(help, "Exit codes") {
		t.Fatalf("help (%d): %s", code, help)
	}
	if bare, _, bareCode := double.run(t, nil); bareCode != ExitOK || bare != help {
		t.Fatalf("no arguments should print the help too")
	}
	version, _, code := double.run(t, nil, "version")
	if code != ExitOK || version != "1.2.3\n" {
		t.Fatalf("version (%d): %q", code, version)
	}
	if len(double.calls) != 0 {
		t.Fatalf("the engine was called: %v", double.calls)
	}
}

func TestAnUnknownCommandIsAUsageError(t *testing.T) {
	_, stderr, code := newEngineDouble(nil).run(t, nil, "deploy")

	if code != ExitUsage || !strings.Contains(stderr, `unknown command "deploy"`) {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestABadFlagOrAStrayArgumentIsAUsageError(t *testing.T) {
	double := newEngineDouble(nil)
	for _, args := range [][]string{{"generate", "--nope"}, {"generate", "extra"}, {"tree", "--output", "x"}, {"tree", "--output", "-1"}, {"sync", "--yes=maybe"}} {
		if _, _, code := double.run(t, nil, args...); code != ExitUsage {
			t.Errorf("%v: exit %d", args, code)
		}
	}
}

func TestTheMoreSeriousExitCodeWins(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{ExitOK, ExitOK, ExitOK}, {ExitOK, ExitBlocked, ExitBlocked}, {ExitBlocked, ExitOK, ExitBlocked},
		{ExitBlocked, ExitFailed, ExitFailed}, {ExitFailed, ExitBlocked, ExitFailed}, {ExitFailed, ExitOK, ExitFailed},
	}
	for _, tc := range cases {
		if got := worst(tc.a, tc.b); got != tc.want {
			t.Errorf("worst(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
