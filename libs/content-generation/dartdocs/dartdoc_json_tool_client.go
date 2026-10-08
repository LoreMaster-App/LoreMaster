package dartdocs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"lore-master/libs/content-generation/externaltool"
)

// Runner runs an external tool. The generator takes one so that tests need no Dart.
type Runner func(ctx context.Context, command externaltool.Command) (externaltool.Result, error)

const (
	toolTimeout = 10 * time.Minute
	// filesPerRun bounds how many files one run is given, so a big package stays under the
	// command line length limit.
	filesPerRun = 60

	dartHint = "install the Dart SDK (https://dart.dev/get-dart), or Flutter, and run the generator again"
	toolHint = "install it with: dart pub global activate dartdoc_json"
)

// checkTool makes sure Dart and dartdoc_json are there before any file is parsed.
func checkTool(ctx context.Context, runner Runner, workspaceRoot string) error {
	result, err := runner(ctx, externaltool.Command{Name: "dart", Args: []string{"pub", "global", "list"}, Dir: workspaceRoot, Timeout: toolTimeout})
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return &externaltool.MissingToolError{Tool: "dart", Hint: dartHint}
		}

		return err
	}
	if failure := result.Failure("dart pub global list"); failure != "" {
		return errors.New(failure)
	}
	if !strings.Contains(result.Stdout, "dartdoc_json") {
		return &externaltool.MissingToolError{Tool: "dartdoc_json", Hint: toolHint}
	}

	return nil
}

// parseFiles runs dartdoc_json over the project's files (project-relative), a few at a time, and
// returns every parsed unit. The parsing is syntactic: nothing is analysed or run.
func parseFiles(ctx context.Context, runner Runner, projectDir string, files []string) ([]unit, error) {
	scratch, err := os.MkdirTemp("", "lore-master-dartdoc-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(scratch) }()

	var units []unit
	for start := 0; start < len(files); start += filesPerRun {
		chunk := files[start:min(start+filesPerRun, len(files))]
		output := filepath.Join(scratch, fmt.Sprintf("chunk-%d.json", start))
		args := append([]string{"pub", "global", "run", "dartdoc_json:main", "-r", projectDir, "-o", output}, chunk...)
		result, err := runner(ctx, externaltool.Command{Name: "dart", Args: args, Dir: projectDir, Timeout: toolTimeout})
		if err != nil {
			return nil, err
		}
		if failure := result.Failure("dartdoc_json"); failure != "" {
			return nil, errors.New(failure)
		}
		parsed, err := readUnits(output)
		if err != nil {
			return nil, err
		}
		units = append(units, parsed...)
	}

	return units, nil
}

func readUnits(file string) ([]unit, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("dartdoc_json wrote no result: %w", err)
	}
	var units []unit
	if err := json.Unmarshal(content, &units); err != nil {
		return nil, fmt.Errorf("reading what dartdoc_json wrote: %w", err)
	}

	return units, nil
}
