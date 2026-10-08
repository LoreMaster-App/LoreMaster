package generatedfile

import "context"

// Spec is one generator as configured in .lore-master.yaml.
type Spec struct {
	// Type names the generator, for example "test-results".
	Type string
	// Input is gitignore-style patterns, matched against workspace-relative paths, that
	// select what the generator reads; empty means the generator's own default.
	Input []string
	// Output is the workspace-relative folder ('/'-separated) the pages are written to.
	Output string
	// Title heads the generated index page; empty means the generator's own default.
	Title string
}

// File is one generated page.
type File struct {
	// Path is relative to the Spec's Output folder, '/'-separated.
	Path string
	// Body is the page's Markdown, without an annotation.
	Body []byte
}

// Output is what a generator produced.
type Output struct {
	Files []File
	// Warnings are notes about inputs that were skipped or could not be read; they do not
	// stop the other pages from being produced.
	Warnings []string
}

// GenerateFunc reads the inputs under workspaceRoot and returns the pages for spec. It must
// be deterministic: the same inputs produce the same files byte for byte, so an unchanged
// input rewrites nothing.
type GenerateFunc func(ctx context.Context, workspaceRoot string, spec Spec) (Output, error)
