package externaltool

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// The tests run this test binary as the tool: with HELPER_MODE set it behaves as the helper
// the case asks for instead of running the tests, so no other program has to be installed.
func TestMain(m *testing.M) {
	switch os.Getenv("HELPER_MODE") {
	case "":
		os.Exit(m.Run())
	case "print":
		_, _ = fmt.Fprint(os.Stdout, "out:", os.Getenv("HELPER_WORD"))
		_, _ = fmt.Fprint(os.Stderr, "err:", os.Getenv("HELPER_WORD"))
	case "fail":
		_, _ = fmt.Fprintln(os.Stderr, "something went wrong")
		os.Exit(3)
	case "failquiet":
		_, _ = fmt.Fprintln(os.Stdout, "only on stdout")
		os.Exit(2)
	case "cwd":
		dir, _ := os.Getwd()
		_, _ = fmt.Fprint(os.Stdout, dir)
	case "sleep":
		time.Sleep(time.Minute)
	case "flood":
		chunk := strings.Repeat("x", 64<<10)
		for range 64 {
			_, _ = fmt.Fprint(os.Stdout, chunk)
		}
	}
	os.Exit(0)
}

func helper(mode string, extra ...string) Command {
	return Command{Name: os.Args[0], Env: append([]string{"HELPER_MODE=" + mode}, extra...)}
}

func TestRunCapturesOutputAndPassesTheEnvironment(t *testing.T) {
	result, err := Run(context.Background(), helper("print", "HELPER_WORD=hello"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Stdout != "out:hello" || result.Stderr != "err:hello" || result.ExitCode != 0 {
		t.Fatalf("result %+v", result)
	}
	if result.Failure("tool") != "" {
		t.Fatalf("a successful run has no failure: %q", result.Failure("tool"))
	}
}

func TestRunReportsANonZeroExitInTheResultNotAsAnError(t *testing.T) {
	result, err := Run(context.Background(), helper("fail"))
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode != 3 {
		t.Fatalf("exit code %d", result.ExitCode)
	}
	if got := result.Failure("tool"); got != "tool exited with status 3: something went wrong" {
		t.Fatalf("failure %q", got)
	}
	quiet, _ := Run(context.Background(), helper("failquiet"))
	if got := quiet.Failure("tool"); !strings.Contains(got, "only on stdout") {
		t.Fatalf("a tool that talks on stdout still explains itself: %q", got)
	}
}

func TestFailureKeepsOnlyTheEndOfALongMessage(t *testing.T) {
	var lines []string
	for i := range 30 {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}

	got := Result{ExitCode: 1, Stderr: strings.Join(lines, "\n")}.Failure("tool")

	if !strings.Contains(got, "line 29") || strings.Contains(got, "line 0\n") || !strings.Contains(got, "…") {
		t.Fatalf("failure %q", got)
	}
}

func TestRunStartsTheToolInTheGivenFolder(t *testing.T) {
	dir := t.TempDir()
	command := helper("cwd")
	command.Dir = dir

	result, err := Run(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.EqualFold(strings.ReplaceAll(result.Stdout, `\`, "/"), strings.ReplaceAll(dir, `\`, "/")) {
		t.Fatalf("ran in %q, want %q", result.Stdout, dir)
	}
}

func TestRunStopsATimedOutToolAndSaysSo(t *testing.T) {
	command := helper("sleep")
	command.Timeout = 300 * time.Millisecond

	started := time.Now()
	_, err := Run(context.Background(), command)

	if err == nil || !strings.Contains(err.Error(), "did not finish within 300ms") {
		t.Fatalf("got %v", err)
	}
	if time.Since(started) > 10*time.Second {
		t.Fatalf("the tool was not stopped: %s", time.Since(started))
	}
}

func TestRunStopsWhenTheCallerCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := Run(ctx, helper("sleep"))

	if err == nil {
		t.Fatal("expected the cancellation")
	}
	if time.Since(started) > 10*time.Second {
		t.Fatalf("the tool was not stopped: %s", time.Since(started))
	}
}

func TestRunKeepsOnlyTheStartOfAFloodOfOutput(t *testing.T) {
	result, err := Run(context.Background(), helper("flood"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Stdout) != maxCapturedBytes {
		t.Fatalf("kept %d bytes, want %d", len(result.Stdout), maxCapturedBytes)
	}
}

func TestRunRefusesAToolThatIsNotThere(t *testing.T) {
	_, err := Run(context.Background(), Command{Name: "definitely-not-a-real-tool-xyz"})

	if err == nil || !strings.Contains(err.Error(), "cannot run definitely-not-a-real-tool-xyz") {
		t.Fatalf("got %v", err)
	}
	if Available("definitely-not-a-real-tool-xyz") {
		t.Fatal("a missing tool is available")
	}
	if !Available(os.Args[0]) {
		t.Fatal("the test binary is not available")
	}
}

func TestMissingToolErrorSaysWhatToInstall(t *testing.T) {
	err := &MissingToolError{Tool: "typedoc", Hint: "install it with: npm i -D typedoc"}

	if err.Error() != "typedoc was not found; install it with: npm i -D typedoc" {
		t.Fatalf("got %q", err.Error())
	}
}
