package dartdocs

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
const DefaultTitle = "Dart API reference"

const indexPath = "README.md"

// Generate documents the Dart and Flutter packages spec selects. It satisfies
// generatedfile.GenerateFunc.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	return generateWith(ctx, externaltool.Run, workspaceRoot, spec)
}

// generateWith is Generate with the tool runner supplied.
//
// The projects are those language detection finds (a folder with a pubspec.yaml), narrowed by
// spec.Input (gitignore-style patterns against project folders, "!" leaving one out). Each
// project's libraries become pages under its own folder; a project with no library to document
// is skipped. Dart or dartdoc_json not being installed fails the whole run with what to install.
func generateWith(ctx context.Context, runner Runner, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	projects, err := languagedetection.Detect(ctx, workspaceRoot)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("finding Dart projects: %w", err)
	}
	selector := inputselection.NewSelector(spec.Input, nil)

	var chosen []languagedetection.Project
	for _, project := range projects {
		if project.Language == languagedetection.Dart && (project.Dir == "." || selector.Selects(project.Dir, true)) {
			chosen = append(chosen, project)
		}
	}
	if len(chosen) > 0 {
		if err := checkTool(ctx, runner, workspaceRoot); err != nil {
			return generatedfile.Output{}, err
		}
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

// documentProject parses one package's library files and renders a page for each that has
// something public in it.
func documentProject(ctx context.Context, runner Runner, workspaceRoot string, project languagedetection.Project) ([]generatedfile.File, error) {
	projectDir := filepath.Join(workspaceRoot, filepath.FromSlash(project.Dir))
	files := libraryFiles(projectDir)
	if len(files) == 0 {
		return nil, nil
	}
	units, err := parseFiles(ctx, runner, projectDir, files)
	if err != nil {
		return nil, fmt.Errorf("documenting %s: %w", project.Dir, err)
	}

	prefix := project.Dir
	if prefix == "." {
		prefix = ""
	}
	name := readPackageName(projectDir)
	var pages []generatedfile.File
	for _, parsed := range units {
		body := renderLibrary(libraryURI(name, parsed.Source), parsed.Declarations)
		if body != "" {
			pages = append(pages, generatedfile.File{Path: pagePath(prefix, filepath.ToSlash(parsed.Source)), Body: []byte(body)})
		}
	}

	return pages, nil
}

// projectIndex lists the libraries, each by the import path its page is titled with.
func projectIndex(title string, pages []generatedfile.File) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	if len(pages) == 0 {
		out.WriteString("No Dart packages with something to document were found.\n")

		return out.String()
	}
	for _, page := range pages {
		heading, _, _ := strings.Cut(strings.TrimPrefix(string(page.Body), "# "), "\n")
		fmt.Fprintf(&out, "- [%s](%s)\n", heading, path.Clean(page.Path))
	}

	return out.String()
}
