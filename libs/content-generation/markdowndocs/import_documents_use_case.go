package markdowndocs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/markdownnormalising"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "Documentation"

// Generate imports the Markdown spec selects. There is no sensible default input (any Markdown
// in a project would be imported), so a spec without a selecting pattern is an error. The files
// keep their layout below the deepest folder they and the images they show share.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	if !hasSelectingPattern(spec.Input) {
		return generatedfile.Output{}, fmt.Errorf("markdown-docs needs an input: the folder or files of Markdown to import, e.g. build/docs/, or leave the input as the output folder when the command writes the pages there")
	}
	paths, err := findDocuments(ctx, workspaceRoot, spec.Input, spec.Output)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("searching for Markdown: %w", err)
	}

	var output generatedfile.Output
	type document struct {
		path string
		body []byte
	}
	var documents []document
	assetPaths := map[string][]byte{}
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(relative)))
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", relative, err))

			continue
		}
		documents = append(documents, document{path: relative, body: data})
		for _, reference := range relativeImages(string(data)) {
			image := path.Join(path.Dir(relative), reference)
			if image == ".." || strings.HasPrefix(image, "../") {
				output.Warnings = append(output.Warnings, fmt.Sprintf("%s: the image %s is outside the workspace and was not copied", relative, reference))

				continue
			}
			if _, copied := assetPaths[image]; copied {
				continue
			}
			content, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(image)))
			if err != nil {
				output.Warnings = append(output.Warnings, fmt.Sprintf("%s: the image %s was not found and was not copied", relative, reference))

				continue
			}
			assetPaths[image] = content
		}
	}
	if len(documents) == 0 {
		output.Warnings = append(output.Warnings, "no Markdown file matched the input")
	}

	all := make([]string, 0, len(documents)+len(assetPaths))
	for _, doc := range documents {
		all = append(all, doc.path)
	}
	for image := range assetPaths {
		all = append(all, image)
	}
	base := commonFolder(all)

	var pages []generatedfile.File
	for _, doc := range documents {
		pages = append(pages, generatedfile.File{Path: strip(base, doc.path), Body: doc.body})
	}
	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	pages, notes := markdownnormalising.NormalisePages(pages, markdownnormalising.Options{IndexTitle: title})
	output.Warnings = append(output.Warnings, notes...)
	output.Files = pages

	images := make([]string, 0, len(assetPaths))
	for image := range assetPaths {
		images = append(images, image)
	}
	slices.Sort(images)
	for _, image := range images {
		output.Files = append(output.Files, generatedfile.File{Path: strip(base, image), Body: assetPaths[image], Asset: true})
	}

	return output, nil
}

// strip removes the base folder from a workspace-relative path.
func strip(base, p string) string {
	if base == "" {
		return p
	}

	return strings.TrimPrefix(p, base+"/")
}

// commonFolder is the deepest folder every path is under; empty when they share none.
func commonFolder(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	parts := strings.Split(path.Dir(paths[0]), "/")
	if len(parts) == 1 && parts[0] == "." {
		return ""
	}
	for _, p := range paths[1:] {
		other := strings.Split(path.Dir(p), "/")
		n := 0
		for n < len(parts) && n < len(other) && parts[n] == other[n] {
			n++
		}
		parts = parts[:n]
	}

	return strings.Join(parts, "/")
}
