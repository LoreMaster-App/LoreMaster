package generatorpreparing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunExecutesTheCommandInTheWorkspaceFolder(t *testing.T) {
	root := t.TempDir()
	if err := Run(context.Background(), root, "echo built > built.txt", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "built.txt")); err != nil {
		t.Fatalf("the command did not run in the workspace folder: %v", err)
	}
}

func TestRunReportsAFailureWithTheCommandsOutput(t *testing.T) {
	err := Run(context.Background(), t.TempDir(), "echo the reason & exit 3", 0)
	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "the reason") {
		t.Fatalf("error %q lacks the command's output", err)
	}
}

func TestRunRefusesAnEmptyCommand(t *testing.T) {
	if err := Run(context.Background(), t.TempDir(), "   ", 0); err == nil {
		t.Fatal("want an error for an empty command")
	}
}

func TestRunStopsACommandThatOutlivesItsTimeout(t *testing.T) {
	command := "sleep 5"
	if os.PathSeparator == '\\' {
		command = "ping -n 6 127.0.0.1 > nul"
	}
	start := time.Now()
	err := Run(context.Background(), t.TempDir(), command, 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatalf("took %s, the command was not stopped", time.Since(start))
	}
}
