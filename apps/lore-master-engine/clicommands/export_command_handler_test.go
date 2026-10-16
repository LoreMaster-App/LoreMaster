package clicommands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runExport(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), append([]string{"export"}, args...), Environment{Stdout: &stdout, Stderr: &stderr, WorkingDir: dir}, nil)

	return code, stdout.String(), stderr.String()
}

func TestExportWritesNextToTheFileAndIntoAFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "guide.md"), []byte("# Guide\n\nHello **world**.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runExport(t, dir, "--format", "jira-wiki", "guide.md")
	if code != ExitOK {
		t.Fatalf("code %d: %s", code, stderr)
	}
	written, err := os.ReadFile(filepath.Join(dir, "guide.jira.txt"))
	if err != nil || !strings.Contains(string(written), "Hello *world*.") || !strings.Contains(stdout, "guide.jira.txt") {
		t.Fatalf("written %q, stdout %q, err %v", written, stdout, err)
	}

	if code, _, stderr := runExport(t, dir, "--format", "txt", "--out", "dist", "guide.md"); code != ExitOK {
		t.Fatalf("code %d: %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "dist", "guide.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestExportPrintsToStdoutAndListsFormats(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, stdout, _ := runExport(t, dir, "--format", "mediawiki", "--stdout", "a.md"); !strings.Contains(stdout, "= A =") {
		t.Fatalf("stdout %q", stdout)
	}
	if _, stdout, _ := runExport(t, dir, "--list"); !strings.Contains(stdout, "jira-adf") {
		t.Fatalf("list %q", stdout)
	}
}

func TestExportRefusesWhatItCannotDo(t *testing.T) {
	dir := t.TempDir()

	if code, _, _ := runExport(t, dir, "--format", "txt"); code != ExitUsage {
		t.Errorf("no file: code %d", code)
	}
	if code, _, _ := runExport(t, dir, "--format", "nope", "missing.md"); code != ExitFailed {
		t.Errorf("missing file: code %d", code)
	}
}
