package coveragereporting

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"lore-master/libs/content-generation/generatedfile"
)

// Generate reads every coverage report spec selects, merges them by file and returns the pages.
// A report that cannot be read or is in no known format is a warning, and the others are still
// rendered. With no reports at all it still returns the index saying so.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	reports, err := findReports(ctx, workspaceRoot, spec.Input)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("searching for coverage reports: %w", err)
	}

	var output generatedfile.Output
	var parsed [][]FileCoverage
	for _, report := range reports {
		data, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(report)))
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", report, err))

			continue
		}
		files, err := ParseCoverage(data, workspaceRoot)
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", report, err))

			continue
		}
		parsed = append(parsed, files)
	}

	output.Files = RenderCoverage(spec.Title, MergeCoverage(parsed))

	return output, nil
}
