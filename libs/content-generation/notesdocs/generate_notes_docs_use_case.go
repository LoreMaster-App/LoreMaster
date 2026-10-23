// Package notesdocs is the notes generator: it reads the tagged comments of the source files (TODO, FIXME, HACK, NOTE...) and
// writes them as one Markdown page, so the unfinished work and the known workarounds sync to Confluence or GitHub Pages like any
// other lore. It needs nothing installed.
package notesdocs

import (
	"context"
	"fmt"

	"lore-master/libs/code-notes/notescanning"
	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/inputselection"
)

// DefaultTitle heads the page when the generator is not given one.
const DefaultTitle = "Notes"

const pagePath = "README.md"

// Generate reads the notes under workspaceRoot, those in the files spec.Input selects (every file when it names none), and returns
// one page: the count per tag, then a table per tag with each note's text, where it is, and its owner, issue and date when any
// has them. The page is the same for the same notes, so an unchanged codebase rewrites nothing. A file's path is shown, not linked:
// source files are not pages.
func Generate(ctx context.Context, workspaceRoot string, spec generatedfile.Spec) (generatedfile.Output, error) {
	found, err := notescanning.Scan(ctx, notescanning.Options{WorkspaceRoot: workspaceRoot})
	if err != nil {
		return generatedfile.Output{}, fmt.Errorf("reading the notes: %w", err)
	}
	selector := inputselection.NewSelector(spec.Input, nil)
	notes := found.Notes[:0:0]
	files := map[string]bool{}
	for _, note := range found.Notes {
		if selector.Selects(note.Path, false) {
			notes = append(notes, note)
			files[note.Path] = true
		}
	}

	title := spec.Title
	if title == "" {
		title = DefaultTitle
	}

	return generatedfile.Output{Files: []generatedfile.File{{Path: pagePath, Body: []byte(renderPage(title, notes, len(files)))}}}, nil
}
