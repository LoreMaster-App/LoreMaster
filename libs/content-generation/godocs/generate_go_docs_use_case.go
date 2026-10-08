package godocs

import (
	"context"
	"fmt"
	"path"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

// DefaultTitle heads the index page when the generator is not given one.
const DefaultTitle = "Go API reference"

const indexPath = "README.md"

// Generate reads the Go packages spec selects and returns their pages: an index naming every
// package with its synopsis, and one page per package at the path of its folder, so the
// pages nest the way the directories do (a package's page is the index of its folder). The
// root folder's package, which cannot be README.md, is named after the package. A folder with
// no exported declarations and no package comment is left out; with no packages at all the
// index says so.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	folders, err := findFolders(ctx, workspaceRoot, spec.Input)
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("searching for Go packages: %w", err)
	}

	var output generatedfile.Output
	var pages []documentedPackage
	for _, folder := range folders {
		if err := ctx.Err(); err != nil {
			return generatedfile.Output{}, err
		}
		loaded, warnings, ok := loadPackage(workspaceRoot, folder)
		output.Warnings = append(output.Warnings, warnings...)
		if ok {
			pages = append(pages, documentedPackage{pkg: loaded, file: pagePath(loaded)})
		}
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}
	output.Files = append(output.Files, generatedfile.File{Path: indexPath, Body: []byte(renderIndex(title, pages))})
	for _, page := range pages {
		output.Files = append(output.Files, generatedfile.File{Path: page.file, Body: []byte(renderPackage(page.pkg))})
	}

	return output, nil
}

// documentedPackage is a package with the page it is written to.
type documentedPackage struct {
	pkg  goPackage
	file string
}

// pagePath is where a package's page goes: README.md in its folder, which makes it the index
// of that folder; the root folder's page is named for the package, since README.md there is
// the index of the whole reference.
func pagePath(p goPackage) string {
	if p.folder.Dir == "." {
		return strings.ToLower(p.docs.Name) + ".md"
	}

	return path.Join(p.folder.Dir, "README.md")
}

func renderIndex(title string, pages []documentedPackage) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", title)
	if len(pages) == 0 {
		out.WriteString("No Go packages were found.\n")

		return out.String()
	}
	fmt.Fprintf(&out, "%d packages.\n\n| Package | Synopsis |\n|---|---|\n", len(pages))
	for _, page := range pages {
		fmt.Fprintf(&out, "| [%s](%s) | %s |\n", page.pkg.folder.ImportPath, page.file, cell(page.pkg.docs.Synopsis(page.pkg.docs.Doc)))
	}

	return out.String()
}

// cell is text safe inside a table cell.
func cell(text string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ", "\r", "", "<", "&lt;").Replace(strings.TrimSpace(text))
}
