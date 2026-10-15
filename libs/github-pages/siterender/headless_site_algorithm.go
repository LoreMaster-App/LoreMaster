package siterender

import "encoding/json"

// headlessPage is one entry of site.json.
type headlessPage struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Depth    int      `json:"depth"`
	Parent   string   `json:"parent,omitempty"`
	Children []string `json:"children,omitempty"`
}

// headlessFiles is a site for a team's own front end: every document's Markdown at its own
// path (children tags expanded, diagrams drawn) and a site.json describing the tree, with
// nothing about how it should look.
func headlessFiles(siteTitle string, pages []page) ([]SiteFile, error) {
	entries := make([]headlessPage, len(pages))
	var stack []int
	for i, current := range pages {
		for len(stack) > 0 && pages[stack[len(stack)-1]].depth >= current.depth {
			stack = stack[:len(stack)-1]
		}
		entry := headlessPage{Path: current.documentPath, Title: current.title, Depth: current.depth}
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			entry.Parent = pages[parent].documentPath
			entries[parent].Children = append(entries[parent].Children, current.documentPath)
		}
		entries[i] = entry
		stack = append(stack, i)
	}
	home := ""
	if len(pages) > 0 {
		home = pages[homeIndex(pages)].documentPath
	}
	manifest, err := json.MarshalIndent(struct {
		Title string         `json:"title"`
		Home  string         `json:"home"`
		Pages []headlessPage `json:"pages"`
	}{siteTitle, home, entries}, "", "  ")
	if err != nil {
		return nil, err
	}

	files := []SiteFile{{Path: "site.json", Content: append(manifest, '\n')}}
	for _, current := range pages {
		files = append(files, SiteFile{Path: current.documentPath, Content: current.body})
	}

	return append(files, SiteFile{Path: ".nojekyll", Content: []byte{}}), nil
}
