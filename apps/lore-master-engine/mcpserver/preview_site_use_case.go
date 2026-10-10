package mcpserver

import (
	"encoding/json"
	"context"
	"fmt"
	"sort"
	"strings"

	"lore-master/apps/lore-master-engine/pagescommands"
	"lore-master/libs/documentation-sync/workspacesettings"
)

const previewSiteToolName = "preview_site"

const previewSiteToolDescription = "Renders the GitHub Pages site of the workspace in memory and reports what it would contain: page and file counts, the files, and the Markdown problems and warnings that would stop or degrade a build. Writes nothing; build it for real with `lore-master-engine pages build --out DIR` or the pages action."

// maxPreviewFiles caps the file list returned to the model; the count is always complete.
const maxPreviewFiles = 60

type previewSiteView struct {
	Output   int      `json:"output"`
	Pages    int      `json:"pages"`
	Files    int      `json:"files"`
	Listed   []string `json:"listed,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func previewSiteTool() toolDescriptor {
	return toolDescriptor{
		Name:        previewSiteToolName,
		Description: previewSiteToolDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"output": map[string]any{
					"type":        "integer",
					"description": "Position of the github-pages output in .lore-master.yaml's outputs list. Optional when there is exactly one.",
				},
			},
		},
	}
}

func previewSiteResult(ctx context.Context, workspaceRoot string, arguments []byte) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}
	index, err := pagesOutputIndex(workspaceRoot, arguments)
	if err != nil {
		return errorResult("%v", err)
	}

	site, err := pagescommands.RenderSite(ctx, workspaceRoot, index)
	if err != nil {
		return errorResult("could not render the site: %v", err)
	}

	view := previewSiteView{Output: index, Warnings: site.Warnings, Errors: site.Problems, Files: len(site.Files)}
	paths := make([]string, 0, len(site.Files))
	for _, file := range site.Files {
		paths = append(paths, file.Path)
		if strings.HasSuffix(file.Path, ".html") {
			view.Pages++
		}
	}
	sort.Strings(paths)
	if len(paths) > maxPreviewFiles {
		paths = paths[:maxPreviewFiles]
	}
	view.Listed = paths

	var text strings.Builder
	if len(site.Problems) > 0 {
		fmt.Fprintf(&text, "The site would NOT build: %d problem(s) in the Markdown.\n", len(site.Problems))
		for _, problem := range site.Problems {
			fmt.Fprintf(&text, "! %s\n", problem)
		}
	} else {
		fmt.Fprintf(&text, "The site would build: %d pages, %d files in all.\n", view.Pages, view.Files)
		for _, path := range view.Listed {
			fmt.Fprintf(&text, "  %s\n", path)
		}
		if view.Files > len(view.Listed) {
			fmt.Fprintf(&text, "  … and %d more\n", view.Files-len(view.Listed))
		}
	}
	for _, warning := range site.Warnings {
		fmt.Fprintf(&text, "warning: %s\n", warning)
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: strings.TrimRight(text.String(), "\n")}},
		StructuredContent: view,
		IsError:           len(site.Problems) > 0,
	}
}

// pagesOutputIndex is the output to render: the argument when given, otherwise the only
// github-pages output; none or several without a choice is an error that says what to do.
func pagesOutputIndex(workspaceRoot string, arguments []byte) (int, error) {
	loaded, err := workspacesettings.LoadSettings(workspaceRoot)
	if err != nil {
		return 0, err
	}
	var positions []int
	for index, output := range loaded.Settings.Outputs {
		if output.Platform == "github-pages" || output.Platform == "github-wiki" {
			positions = append(positions, index)
		}
	}

	if chosen, given := outputArgument(arguments); given {
		for _, position := range positions {
			if position == chosen {
				return chosen, nil
			}
		}

		return 0, fmt.Errorf("output %d is not a github-pages or github-wiki output", chosen)
	}
	switch len(positions) {
	case 0:
		return 0, fmt.Errorf("%s has no github-pages output; call site_publishing_plan for the one to add", workspacesettings.FileName)
	case 1:
		return positions[0], nil
	default:
		return 0, fmt.Errorf("%d github-pages outputs; pass output: one of %v", len(positions), positions)
	}
}

// outputArgument reads the optional "output" argument.
func outputArgument(arguments []byte) (int, bool) {
	if len(arguments) == 0 {
		return 0, false
	}
	var parsed struct {
		Output *int `json:"output"`
	}
	if err := json.Unmarshal(arguments, &parsed); err != nil || parsed.Output == nil {
		return 0, false
	}

	return *parsed.Output, true
}
