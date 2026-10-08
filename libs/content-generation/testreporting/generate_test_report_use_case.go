package testreporting

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"lore-master/libs/content-generation/generatedfile"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "Test results"

// Generate reads every JUnit report spec selects and returns the pages for them. A report
// that cannot be read or is not JUnit is a warning, and the others are still rendered. With
// no reports at all it still returns the index page saying so, so the output folder never
// keeps results from a run that no longer exists.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	reports, err := findReports(ctx, workspaceRoot, spec.Input)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("searching for test reports: %w", err)
	}

	var output generatedfile.Output
	var suites []Suite
	for _, report := range reports {
		data, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(report)))
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", report, err))

			continue
		}
		parsed, err := ParseJUnit(data, report)
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", report, err))

			continue
		}
		suites = append(suites, parsed...)
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	output.Files = RenderReport(title, suites)

	return output, nil
}
