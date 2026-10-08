package csharpdocs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"lore-master/libs/content-generation/externaltool"
	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/inputselection"
	"lore-master/libs/content-generation/languagedetection"
	"lore-master/libs/content-generation/markdownnormalising"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "C# API reference"

const indexPath = "README.md"

// Generate documents the C# projects spec selects. It satisfies generatedfile.GenerateFunc.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	return generateWith(ctx, externaltool.Run, externaltool.Available, workspaceRoot, spec)
}

// generateWith is Generate with the tool runner, and the check for a tool being installed, supplied.
//
// The projects are those language detection finds (a folder with a .csproj), narrowed by
// spec.Input (gitignore-style patterns against project folders, "!" leaving one out); test
// projects are skipped. Each is built and documented in turn, its pages going under its own
// folder. DefaultDocumentation not being installed fails the whole run with what to install,
// before anything is built.
func generateWith(ctx context.Context, runner Runner, available func(string) bool, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	projects, err := languagedetection.Detect(ctx, workspaceRoot)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("finding C# projects: %w", err)
	}
	selector := inputselection.NewSelector(spec.Input, nil)

	var chosen []languagedetection.Project
	for _, project := range projects {
		if project.Language == languagedetection.CSharp && !isTestProject(project.MarkerFile) && (project.Dir == "." || selector.Selects(project.Dir, true)) {
			chosen = append(chosen, project)
		}
	}
	if len(chosen) > 0 && !available("defaultdocumentation") {
		return generatedfile.Output{}, &externaltool.MissingToolError{Tool: "defaultdocumentation", Hint: toolHint}
	}

	var output generatedfile.Output
	var pages []generatedfile.File
	for _, project := range chosen {
		projectPages, err := documentProject(ctx, runner, workspaceRoot, project)
		if err != nil {
			return generatedfile.Output{}, err
		}
		pages = append(pages, projectPages...)
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	pages = append(pages, generatedfile.File{Path: indexPath, Body: []byte(projectIndex(title, pages))})
	normalised, notes := markdownnormalising.NormalisePages(pages, markdownnormalising.Options{IndexTitle: title})
	output.Files, output.Warnings = normalised, notes

	return output, nil
}

// documentProject builds one project and documents its assembly, returning its pages under the
// project's folder.
func documentProject(ctx context.Context, runner Runner, workspaceRoot string, project languagedetection.Project) ([]generatedfile.File, error) {
	projectDir := filepath.Join(workspaceRoot, filepath.FromSlash(project.Dir))
	built, err := buildProject(ctx, runner, projectDir, project.MarkerFile)
	if err != nil {
		return nil, fmt.Errorf("building %s: %w", path.Join(project.Dir, project.MarkerFile), err)
	}
	outDir, err := os.MkdirTemp("", "lore-master-defaultdoc-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	raw, err := runDefaultDocumentation(ctx, runner, projectDir, built, outDir)
	if err != nil {
		return nil, fmt.Errorf("documenting %s: %w", project.Dir, err)
	}
	prefix := project.Dir
	if prefix == "." {
		prefix = ""
	}

	return convertPages(raw, prefix), nil
}

// projectIndex lists every namespace page, so the reference opens on the namespace tree rather
// than on every type.
func projectIndex(title string, pages []generatedfile.File) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	listed := 0
	for _, page := range pages {
		if path.Base(page.Path) == "README.md" {
			fmt.Fprintf(&out, "- [%s](%s)\n", strings.ReplaceAll(path.Dir(page.Path), "/", "."), page.Path)
			listed++
		}
	}
	if listed == 0 {
		out.WriteString("No C# projects with something to document were found.\n")
	}

	return out.String()
}
