package mcpserver

import (
	"fmt"
	"strings"

	"lore-master/libs/content-generation/generatorregistry"
	"lore-master/libs/documentation-sync/editorsettings"
	"lore-master/libs/documentation-sync/workspacesettings"
)

const listGeneratorsToolName = "list_generators"

const listGeneratorsToolDescription = "Lists the generators LoreMaster has (test results, coverage, OpenAPI, Markdown imports, Storybook, and code documentation for Go, TypeScript, Python, C# and Dart) with what each reads and writes, and the generators configured in this workspace with the folder each writes to. Pages in those folders are written by a generator and overwritten on every run, so edit the source instead. Writes nothing."

func listGeneratorsTool() toolDescriptor {
	return toolDescriptor{
		Name:        listGeneratorsToolName,
		Description: listGeneratorsToolDescription,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// configuredGenerator is one generator of the workspace's settings, as the agent sees it.
type configuredGenerator struct {
	Type    string   `json:"type,omitempty"`
	Input   []string `json:"input,omitempty"`
	Source  string   `json:"source,omitempty"`
	Output  string   `json:"output"`
	Title   string   `json:"title,omitempty"`
	Prepare string   `json:"prepare,omitempty"`
}

type generatorsReport struct {
	Available  []generatorregistry.Entry `json:"available"`
	Configured []configuredGenerator     `json:"configured"`
	Warnings   []string                  `json:"warnings,omitempty"`
}

// listGeneratorsResult answers without a workspace too: the catalogue does not need one, only
// the configured list does.
func listGeneratorsResult(workspaceRoot string) toolCallResult {
	report := generatorsReport{Available: generatorregistry.Catalogue(), Configured: []configuredGenerator{}}

	if workspaceRoot != "" {
		loaded, err := editorsettings.LoadConfiguration(workspaceRoot)
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s could not be read: %v", workspacesettings.ConfigurationName, err))
		}
		for _, generator := range loaded.Settings.Generators {
			report.Configured = append(report.Configured, configuredGenerator{
				Type: generator.Type, Input: generator.Input, Source: generator.Source,
				Output: generator.Output, Title: generator.Title, Prepare: generator.Prepare,
			})
		}
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: describeGenerators(report, workspaceRoot != "")}},
		StructuredContent: report,
	}
}

func describeGenerators(report generatorsReport, hasWorkspace bool) string {
	var text strings.Builder

	text.WriteString("Generators write Markdown pages from what the project already produces; the pages then sync like any other. Leave `type` out and the format is recognised from what `input` selects.\n")
	for _, entry := range report.Available {
		fmt.Fprintf(&text, "\n- %s: reads %s; writes %s.", entry.Type, entry.Reads, entry.Writes)
		if entry.Needs != "" {
			fmt.Fprintf(&text, " Needs %s.", entry.Needs)
		}
	}

	switch {
	case !hasWorkspace:
		text.WriteString("\n\nNo workspace is open, so the generators configured for it are not known.")
	case len(report.Configured) == 0:
		text.WriteString("\n\nThis workspace configures no generators (the `loreMaster.generators` setting).")
	default:
		text.WriteString("\n\nConfigured in this workspace (the pages in each output folder are generated, so do not edit them by hand):")
		for _, generator := range report.Configured {
			kind := generator.Type
			if kind == "" {
				kind = "(recognised from its output)"
			}
			fmt.Fprintf(&text, "\n- %s writes to `%s`", kind, generator.Output)
			if generator.Source != "" {
				fmt.Fprintf(&text, " from `%s`", generator.Source)
			}
			if len(generator.Input) > 0 {
				fmt.Fprintf(&text, " reading %s", strings.Join(generator.Input, ", "))
			}
			text.WriteString(".")
		}
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(&text, "\nNote: %s", warning)
	}

	return text.String()
}
