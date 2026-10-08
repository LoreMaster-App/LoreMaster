package tsdocs

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
const DefaultTitle = "TypeScript API reference"

const indexPath = "README.md"

// Generate documents the TypeScript and JavaScript projects spec selects with TypeDoc. It
// satisfies generatedfile.GenerateFunc.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	return generateWith(ctx, externaltool.Run, workspaceRoot, spec)
}

// generateWith is Generate with the tool runner supplied.
//
// The projects are those language detection finds, narrowed by spec.Input (gitignore-style
// patterns against project folders, "!" leaving one out). Each project's pages go under its
// own folder, so a monorepo gets one section per package; a project with nothing to document
// (no src folder or index, such as a workspace root that only holds references) is skipped.
// TypeDoc not being installed fails the whole run with what to install, since an index of
// nothing would be worse than the message.
func generateWith(ctx context.Context, runner Runner, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	projects, err := languagedetection.Detect(ctx, workspaceRoot)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("finding TypeScript projects: %w", err)
	}
	selector := inputselection.NewSelector(spec.Input, nil)

	var output generatedfile.Output
	var pages []generatedfile.File
	var documented []string
	for _, project := range projects {
		if project.Language != languagedetection.TypeScript || (project.Dir != "." && !selector.Selects(project.Dir, true)) {
			continue
		}
		projectPages, warnings, err := documentProject(ctx, runner, workspaceRoot, project)
		if err != nil {
			return generatedfile.Output{}, err
		}
		output.Warnings = append(output.Warnings, warnings...)
		if len(projectPages) > 0 {
			pages = append(pages, projectPages...)
			documented = append(documented, project.Dir)
		}
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	if !hasRootIndex(pages) {
		pages = append(pages, generatedfile.File{Path: indexPath, Body: []byte(projectIndex(title, documented))})
	} else if spec.Title != "" {
		// A title the user chose heads the index even when TypeDoc wrote the index page.
		retitleIndex(pages, spec.Title)
	}
	normalised, notes := markdownnormalising.NormalisePages(pages, markdownnormalising.Options{IndexTitle: title})
	output.Files, output.Warnings = normalised, append(output.Warnings, notes...)

	return output, nil
}

// documentProject runs TypeDoc over one project and returns its pages under the project's
// folder. A project with nothing to document returns no pages and no error.
func documentProject(ctx context.Context, runner Runner, workspaceRoot string, project languagedetection.Project) ([]generatedfile.File, []string, error) {
	projectDir := filepath.Join(workspaceRoot, filepath.FromSlash(project.Dir))
	exists := func(relative string) bool {
		_, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(relative)))

		return err == nil
	}
	choice := chooseEntryPoints(exists)
	if !choice.Found {
		return nil, nil, nil
	}
	script, err := locateTypeDoc(workspaceRoot, projectDir)
	if err != nil {
		return nil, nil, err
	}

	outDir, err := os.MkdirTemp("", "lore-master-typedoc-*")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	warnings, err := runTypeDoc(ctx, runner, script, projectDir, outDir, projectTSConfig(exists), choice)
	if err != nil {
		return nil, nil, fmt.Errorf("documenting %s: %w", project.Dir, err)
	}
	prefix := project.Dir
	if prefix == "." {
		prefix = ""
	}
	pages, err := readPages(outDir, prefix)
	if err != nil {
		return nil, nil, err
	}
	for i, warning := range warnings {
		warnings[i] = project.Dir + ": " + warning
	}

	return pages, warnings, nil
}

// hasRootIndex reports whether a page is already the output's index.
func hasRootIndex(pages []generatedfile.File) bool {
	for _, page := range pages {
		if page.Path == indexPath {
			return true
		}
	}

	return false
}

// projectIndex is the index when no project sits at the workspace root: it names each
// project's own index page, so the reference opens on a short list rather than on every page.
func projectIndex(title string, folders []string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	if len(folders) == 0 {
		out.WriteString("No TypeScript projects with something to document were found.\n")

		return out.String()
	}
	for _, folder := range folders {
		fmt.Fprintf(&out, "- [%s](%s)\n", folder, path.Join(folder, "README.md"))
	}

	return out.String()
}

// retitleIndex gives the index page (README.md at the output root) the title the user asked for.
func retitleIndex(pages []generatedfile.File, title string) {
	for i, page := range pages {
		if page.Path == indexPath {
			pages[i].Body = []byte(withTopHeading(string(page.Body), title))
		}
	}
}

// withTopHeading replaces the text of the first "# " heading of a page.
func withTopHeading(body string, title string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			lines[i] = "# " + title

			break
		}
	}

	return strings.Join(lines, "\n")
}
