package pythondocs

import (
	"context"
	"fmt"
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
const DefaultTitle = "Python API reference"

const indexPath = "README.md"

// Generate documents the Python projects spec selects with pydoc-markdown. It satisfies
// generatedfile.GenerateFunc.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	return generateWith(ctx, externaltool.Run, workspaceRoot, spec)
}

// generateWith is Generate with the tool runner supplied.
//
// The projects are those language detection finds, narrowed by spec.Input (gitignore-style
// patterns against project folders, "!" leaving one out). Each project's pages go under its own
// folder; a project with no package or module to document is skipped. pydoc-markdown not being
// installed fails the whole run with what to install.
func generateWith(ctx context.Context, runner Runner, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	projects, err := languagedetection.Detect(ctx, workspaceRoot)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("finding Python projects: %w", err)
	}
	selector := inputselection.NewSelector(spec.Input, nil)

	var output generatedfile.Output
	var pages []generatedfile.File
	for _, project := range projects {
		if project.Language != languagedetection.Python || (project.Dir != "." && !selector.Selects(project.Dir, true)) {
			continue
		}
		projectPages, warnings, err := documentProject(ctx, runner, workspaceRoot, project)
		if err != nil {
			return generatedfile.Output{}, err
		}
		output.Warnings = append(output.Warnings, warnings...)
		pages = append(pages, projectPages...)
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	pages = append(pages, generatedfile.File{Path: indexPath, Body: []byte(projectIndex(title, pages))})
	normalised, notes := markdownnormalising.NormalisePages(pages, markdownnormalising.Options{IndexTitle: title})
	output.Files, output.Warnings = normalised, append(output.Warnings, notes...)

	return output, nil
}

// documentProject runs the tool over one project and returns its pages under the project's
// folder. A project with nothing to document returns no pages and no error.
func documentProject(ctx context.Context, runner Runner, workspaceRoot string, project languagedetection.Project) ([]generatedfile.File, []string, error) {
	projectDir := filepath.Join(workspaceRoot, filepath.FromSlash(project.Dir))
	found := chooseSources(projectDir)
	if found.empty() {
		return nil, nil, nil
	}
	tool, err := locatePydocMarkdown(workspaceRoot, projectDir)
	if err != nil {
		return nil, nil, err
	}
	printed, warnings, err := runPydocMarkdown(ctx, runner, tool, projectDir, found)
	if err != nil {
		return nil, nil, fmt.Errorf("documenting %s: %w", project.Dir, err)
	}

	prefix := project.Dir
	if prefix == "." {
		prefix = ""
	}
	isPackage := func(module string) bool {
		return fileExists(filepath.Join(projectDir, found.SearchPath, filepath.FromSlash(strings.ReplaceAll(module, ".", "/")), "__init__.py"))
	}
	for i, warning := range warnings {
		warnings[i] = project.Dir + ": " + warning
	}

	return toFiles(splitModules(printed), prefix, isPackage), warnings, nil
}

// projectIndex lists every package page, indented by depth, so the reference opens on the
// package tree rather than on a flat list of modules.
func projectIndex(title string, pages []generatedfile.File) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	listed := 0
	for _, page := range pages {
		if path.Base(page.Path) != "README.md" {
			continue
		}
		folder := path.Dir(page.Path)
		fmt.Fprintf(&out, "- [%s](%s)\n", strings.ReplaceAll(folder, "/", "."), page.Path)
		listed++
	}
	if listed == 0 {
		for _, page := range pages {
			fmt.Fprintf(&out, "- [%s](%s)\n", strings.TrimSuffix(page.Path, ".md"), page.Path)
			listed++
		}
	}
	if listed == 0 {
		out.WriteString("No Python packages with something to document were found.\n")
	}

	return out.String()
}
