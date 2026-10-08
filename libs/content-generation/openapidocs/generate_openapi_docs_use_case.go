package openapidocs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "API reference"

const indexPath = "README.md"

// Generate reads every OpenAPI 3 description spec selects and returns the pages: a README.md
// listing the APIs, and for each API a folder with its overview, a page per tag and a schemas
// page. A file that is not an OpenAPI 3 description is a warning, and the others are still
// rendered. With none at all the index says so, so old pages never linger.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	sources, err := findDescriptions(ctx, workspaceRoot, spec.Input)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("searching for OpenAPI descriptions: %w", err)
	}

	var output generatedfile.Output
	var apis []api
	folders, titles := map[string]bool{}, map[string]bool{}
	indexTitle := spec.Title
	if indexTitle == "" {
		indexTitle = DefaultTitle
	}
	titles[strings.ToLower(indexTitle)] = true

	for _, source := range sources {
		data, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(source)))
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", source, err))

			continue
		}
		desc, err := parseDescription(data)
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("%s: %v", source, err))

			continue
		}

		title := strings.TrimSpace(desc.Info.Title)
		if title == "" {
			title = strings.TrimSuffix(path.Base(source), path.Ext(source))
		}
		slug := slugOf(title)
		dir := slug
		for n := 2; folders[dir]; n++ {
			dir = fmt.Sprintf("%s-%d", slug, n)
		}
		folders[dir] = true
		unique := title
		if titles[strings.ToLower(unique)] {
			unique = fmt.Sprintf("%s (%s)", title, source)
		}
		for n := 2; titles[strings.ToLower(unique)]; n++ {
			unique = fmt.Sprintf("%s (%s) #%d", title, source, n)
		}
		titles[strings.ToLower(unique)] = true

		apis = append(apis, api{desc: desc, source: source, title: unique, dir: dir})
	}

	output.Files = append(output.Files, generatedfile.File{Path: indexPath, Body: []byte(renderIndex(indexTitle, apis))})
	for _, a := range apis {
		output.Files = append(output.Files, renderAPI(a)...)
	}

	return output, nil
}

func renderIndex(title string, apis []api) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	if len(apis) == 0 {
		out.WriteString("No OpenAPI descriptions were found.\n")

		return out.String()
	}
	out.WriteString("| API | Version | Source |\n|---|---|---|\n")
	for _, a := range apis {
		fmt.Fprintf(&out, "| [%s](%s/README.md) | %s | %s |\n", cell(a.title), a.dir, cell(a.desc.Info.Version), code(a.source))
	}

	return out.String()
}
