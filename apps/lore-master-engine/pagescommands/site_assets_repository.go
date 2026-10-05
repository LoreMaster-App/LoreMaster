package pagescommands

import (
	"fmt"
	"os"
	"path/filepath"

	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// collectSiteAssets gathers the local files the documents reference — images (Markdown and
// raw-HTML <img>) and linked non-Markdown files — and reads them as site files to publish
// at their workspace-relative paths, which is where the rendered relative src/href resolve.
// A referenced file that cannot be read is a warning, not a failure; remote references are
// ignored, and the inventory never resolves a path outside the workspace. Each asset is
// published once however many documents reference it.
func collectSiteAssets(root string, documents []documentparsing.MarkdownDocument) ([]siterender.SiteFile, []string) {
	var files []siterender.SiteFile
	var warnings []string
	seen := map[documentdiscovery.DocumentPath]bool{}

	for _, document := range documents {
		inventory := documentparsing.InventoryLinks(document)
		warnings = append(warnings, inventory.Warnings...)

		references := make([]documentdiscovery.DocumentPath, 0, len(inventory.Images)+len(inventory.LinkedFiles))
		for _, image := range inventory.Images {
			if image.Path != "" { // a remote image carries a URL, not a path
				references = append(references, image.Path)
			}
		}
		for _, linked := range inventory.LinkedFiles {
			references = append(references, linked.Path)
		}

		for _, reference := range references {
			if seen[reference] {
				continue
			}
			seen[reference] = true
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(string(reference))))
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s is referenced but could not be read, so it is not published: %v", reference, err))

				continue
			}
			files = append(files, siterender.SiteFile{Path: string(reference), Content: content})
		}
	}

	return files, warnings
}
