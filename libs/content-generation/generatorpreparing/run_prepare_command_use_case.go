package generatorpreparing

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout bounds a prepare command: builds are slow, but not unbounded.
const DefaultTimeout = 30 * time.Minute

// outputTail is how much of a failing command's output the error carries.
const outputTail = 4000

// Run executes command in workspaceRoot through the shell (cmd on Windows, sh elsewhere) and
// waits for it. timeout of zero means DefaultTimeout. The error of a failing command ends with
// the tail of its combined output.
func Run(ctx context.Context, workspaceRoot string, command string, timeout time.Duration) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("the prepare command is empty")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var shell *exec.Cmd
	if runtime.GOOS == "windows" {
		shell = exec.CommandContext(runCtx, "cmd", "/C", command)
	} else {
		shell = exec.CommandContext(runCtx, "sh", "-c", command)
	}
	shell.Dir = workspaceRoot
	// Stopping the shell must stop what it started, or the command's output pipe stays open.
	shell.Cancel = func() error {
		if runtime.GOOS == "windows" && shell.Process != nil {
			_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(shell.Process.Pid)).Run()
		}
		if shell.Process == nil {
			return nil
		}

		return shell.Process.Kill()
	}
	shell.WaitDelay = 2 * time.Second
	var output bytes.Buffer
	shell.Stdout = &output
	shell.Stderr = &output

	err := shell.Run()
	if err == nil {
		return nil
	}
	reason := err.Error()
	switch {
	case runCtx.Err() == context.DeadlineExceeded:
		reason = fmt.Sprintf("timed out after %s", timeout)
	case ctx.Err() != nil:
		reason = "cancelled"
	}
	tail := strings.TrimSpace(output.String())
	if len(tail) > outputTail {
		tail = "..." + tail[len(tail)-outputTail:]
	}
	if tail == "" {
		return fmt.Errorf("%s: %s", command, reason)
	}

	return fmt.Errorf("%s: %s\n%s", command, reason, tail)
}
