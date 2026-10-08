package pythondocs

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"lore-master/libs/content-generation/externaltool"
)

// Runner runs an external tool. The generator takes one so that tests need no Python.
type Runner func(ctx context.Context, command externaltool.Command) (externaltool.Result, error)

const (
	// pydocTimeout bounds one run over a project.
	pydocTimeout = 10 * time.Minute
	maxWarnings  = 10
	installHint  = "install it with: pip install pydoc-markdown (or pipx install pydoc-markdown)"
)

var warningWord = regexp.MustCompile(`(?i)\bwarning\b`)

// locatePydocMarkdown finds the tool in a virtual environment of the project or of the workspace
// (.venv, venv), else on the PATH. Not finding it is a MissingToolError saying what to install.
func locatePydocMarkdown(workspaceRoot string, projectDir string) (string, error) {
	scripts, exe := "bin", "pydoc-markdown"
	if runtime.GOOS == "windows" {
		scripts, exe = "Scripts", "pydoc-markdown.exe"
	}
	for _, base := range []string{projectDir, workspaceRoot} {
		for _, env := range []string{".venv", "venv"} {
			candidate := filepath.Join(base, env, scripts, exe)
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	if found, err := exec.LookPath("pydoc-markdown"); err == nil {
		return found, nil
	}

	return "", &externaltool.MissingToolError{Tool: "pydoc-markdown", Hint: installHint}
}

// pydocArguments documents the packages and modules found under the search path, to stdout.
func pydocArguments(found sources) []string {
	args := []string{"-I", found.SearchPath}
	for _, name := range found.Packages {
		args = append(args, "-p", name)
	}
	for _, name := range found.Modules {
		args = append(args, "-m", name)
	}

	return args
}

// runPydocMarkdown runs the tool in the project's folder and returns what it printed and its
// warnings. UTF-8 is forced so docstrings come out the same on every OS.
func runPydocMarkdown(ctx context.Context, runner Runner, tool string, projectDir string, found sources) (string, []string, error) {
	result, err := runner(ctx, externaltool.Command{
		Name:    tool,
		Args:    pydocArguments(found),
		Dir:     projectDir,
		Env:     []string{"PYTHONUTF8=1", "PYTHONIOENCODING=utf-8"},
		Timeout: pydocTimeout,
	})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", nil, &externaltool.MissingToolError{Tool: "pydoc-markdown", Hint: installHint}
		}

		return "", nil, err
	}
	if failure := result.Failure("pydoc-markdown"); failure != "" {
		return "", nil, errors.New(failure)
	}

	return result.Stdout, warningsIn(result.Stderr), nil
}

// warningsIn picks the warning lines out of what the tool logged, bounded.
func warningsIn(logged string) []string {
	var warnings []string
	for _, line := range strings.Split(logged, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !warningWord.MatchString(line) {
			continue
		}
		if len(warnings) == maxWarnings {
			warnings = append(warnings, "pydoc-markdown: more warnings were left out")

			break
		}
		warnings = append(warnings, "pydoc-markdown: "+line)
	}

	return warnings
}
