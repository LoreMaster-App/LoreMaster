package tsdocs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lore-master/libs/content-generation/externaltool"
	"lore-master/libs/content-generation/generatedfile"
)

// Runner runs an external tool. The generator takes one so that tests need no Node.
type Runner func(ctx context.Context, command externaltool.Command) (externaltool.Result, error)

// typedocTimeout bounds one TypeDoc run: a large project takes a minute, not ten.
const typedocTimeout = 10 * time.Minute

// maxWarnings bounds how many of TypeDoc's own warnings are passed on per project.
const maxWarnings = 10

var ansi = regexp.MustCompile("\x1b\\[[0-9;]*m")

// locateTypeDoc finds TypeDoc's script and its Markdown plugin in the node_modules of the
// project or of any folder above it up to the workspace root, which is where an npm
// workspace hoists them. A missing piece is a MissingToolError saying what to install.
func locateTypeDoc(workspaceRoot string, projectDir string) (string, error) {
	hint := "install it in the workspace with: npm i -D typedoc typedoc-plugin-markdown"
	script, typedocFound := findPackageFile(workspaceRoot, projectDir, filepath.Join("typedoc", "bin", "typedoc"))
	if !typedocFound {
		return "", &externaltool.MissingToolError{Tool: "TypeDoc", Hint: hint}
	}
	if _, pluginFound := findPackageFile(workspaceRoot, projectDir, filepath.Join("typedoc-plugin-markdown", "package.json")); !pluginFound {
		return "", &externaltool.MissingToolError{Tool: "typedoc-plugin-markdown", Hint: hint}
	}

	return script, nil
}

// findPackageFile looks for node_modules/<relative> from dir up to root.
func findPackageFile(root string, dir string, relative string) (string, bool) {
	for current := dir; ; current = filepath.Dir(current) {
		candidate := filepath.Join(current, "node_modules", relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		if current == root || current == filepath.Dir(current) {
			return "", false
		}
	}
}

// typedocArguments is the command line for one project. The page header, the breadcrumbs, the
// generator footer and the source links are switched off: they change with the code's layout or
// the commit and add nothing a wiki page needs.
func typedocArguments(script string, outDir string, tsconfig string, choice entryChoice) []string {
	args := []string{
		script, "--plugin", "typedoc-plugin-markdown", "--out", outDir,
		"--readme", "none", "--disableSources", "--excludePrivate", "--excludeInternal",
		"--skipErrorChecking", "--logLevel", "Warn",
		"--hidePageHeader", "--hideBreadcrumbs", "--hideGenerator",
	}
	if tsconfig != "" {
		args = append(args, "--tsconfig", tsconfig)
	}
	if choice.UseOwnConfig {
		return args
	}
	args = append(args, "--entryPoints")
	args = append(args, choice.Paths...)
	if choice.Expand {
		args = append(args, "--entryPointStrategy", "expand", "--exclude", "**/*.spec.ts", "**/*.test.ts", "**/*.d.ts", "**/node_modules/**")
	}

	return args
}

// runTypeDoc runs TypeDoc in the project's folder, writing its Markdown to outDir, and returns
// the warnings it printed. A tool that cannot start because Node is missing says so; a run that
// exits non-zero is an error carrying the end of what TypeDoc said.
func runTypeDoc(ctx context.Context, runner Runner, script string, projectDir string, outDir string, tsconfig string, choice entryChoice) ([]string, error) {
	result, err := runner(ctx, externaltool.Command{
		Name:    "node",
		Args:    typedocArguments(script, outDir, tsconfig, choice),
		Dir:     projectDir,
		Timeout: typedocTimeout,
	})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, &externaltool.MissingToolError{Tool: "node", Hint: "install Node.js (https://nodejs.org) and run the generator again"}
		}

		return nil, err
	}
	if failure := result.Failure("typedoc"); failure != "" {
		return nil, errors.New(ansi.ReplaceAllString(failure, ""))
	}

	return warningsIn(result.Stdout + "\n" + result.Stderr), nil
}

// warningsIn picks the warning lines out of what TypeDoc printed, without colour codes,
// bounded.
func warningsIn(output string) []string {
	var warnings []string
	for _, line := range strings.Split(ansi.ReplaceAllString(output, ""), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[warning]") {
			if len(warnings) == maxWarnings {
				warnings = append(warnings, "typedoc: more warnings were left out")

				break
			}
			warnings = append(warnings, "typedoc: "+strings.TrimSpace(strings.TrimPrefix(line, "[warning]")))
		}
	}

	return warnings
}

// readPages reads the Markdown TypeDoc wrote, prefixing every path with prefix (the project's
// folder, or empty for one at the workspace root).
func readPages(outDir string, prefix string) ([]generatedfile.File, error) {
	var pages []generatedfile.File
	err := filepath.WalkDir(outDir, func(full string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(full), ".md") {
			return nil
		}
		relative, err := filepath.Rel(outDir, full)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return err
		}
		pages = append(pages, generatedfile.File{Path: path.Join(prefix, filepath.ToSlash(relative)), Body: content})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading what TypeDoc wrote: %w", err)
	}

	return pages, nil
}
