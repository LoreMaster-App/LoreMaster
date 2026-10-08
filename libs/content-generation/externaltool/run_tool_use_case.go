package externaltool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// maxCapturedBytes bounds what is kept of each output stream: a tool that prints without end
// must not exhaust memory.
const maxCapturedBytes = 1 << 20

// gracePeriod is how long a cancelled tool gets to exit before it is killed outright.
const gracePeriod = 3 * time.Second

// Command is one run of a tool.
type Command struct {
	// Name is the program to run, looked up on the PATH (or a path).
	Name string
	Args []string
	// Dir is the working directory; empty means the current one.
	Dir string
	// Env is added to the environment the tool inherits, as "KEY=value".
	Env []string
	// Timeout bounds the run; zero means no limit beyond the caller's context.
	Timeout time.Duration
}

// Result is what a finished tool left behind.
type Result struct {
	Stdout string
	Stderr string
	// ExitCode is the tool's exit status; zero means success.
	ExitCode int
}

// Failure describes a non-zero run for a message: the status and the end of what the tool
// said on stderr (or stdout when it said nothing there). Empty when the run succeeded.
func (r Result) Failure(name string) string {
	if r.ExitCode == 0 {
		return ""
	}
	said := strings.TrimSpace(r.Stderr)
	if said == "" {
		said = strings.TrimSpace(r.Stdout)
	}
	if lines := strings.Split(said, "\n"); len(lines) > 12 {
		said = "…\n" + strings.Join(lines[len(lines)-12:], "\n")
	}

	return fmt.Sprintf("%s exited with status %d: %s", name, r.ExitCode, said)
}

// Available reports whether the program can be found.
func Available(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

// Run executes the command and waits for it. A tool that cannot be started or that outlives its
// Timeout is an error; one that exits non-zero is not: its status and output come back in the
// Result for the caller to judge. When ctx is cancelled or the Timeout passes, the tool and
// everything it started are stopped.
func Run(ctx context.Context, command Command) (Result, error) {
	path, err := exec.LookPath(command.Name)
	if err != nil {
		return Result{}, fmt.Errorf("cannot run %s: %w", command.Name, err)
	}
	if command.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, command.Timeout)
		defer cancel()
	}

	run := exec.CommandContext(ctx, path, command.Args...)
	run.Dir = command.Dir
	run.Env = append(os.Environ(), command.Env...)
	var stdout, stderr limitedBuffer
	run.Stdout, run.Stderr = &stdout, &stderr
	run.Cancel = func() error { return stopTree(run.Process) }
	run.WaitDelay = gracePeriod

	err = run.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCodeOf(run)}
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return result, fmt.Errorf("%s did not finish within %s", command.Name, command.Timeout)
		}

		return result, ctx.Err()
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return result, fmt.Errorf("running %s: %w", command.Name, err)
	}

	return result, nil
}

func exitCodeOf(run *exec.Cmd) int {
	if run.ProcessState == nil {
		return -1
	}

	return run.ProcessState.ExitCode()
}

// stopTree ends the process and what it started: on Windows with taskkill, which follows the
// tree (npx starts node, and the shell starts both); elsewhere with the signal a terminal
// would send, which npm and node pass on to their children.
func stopTree(process *os.Process) error {
	if process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(process.Pid)).Run(); err == nil {
			return nil
		}
	}

	return process.Kill()
}

// limitedBuffer keeps the first maxCapturedBytes written to it and drops the rest.
type limitedBuffer struct {
	buffer bytes.Buffer
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := maxCapturedBytes - b.buffer.Len(); room > 0 {
		b.buffer.Write(p[:min(len(p), room)])
	}

	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
}
