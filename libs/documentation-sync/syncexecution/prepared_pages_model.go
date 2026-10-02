package syncexecution

import (
	"lore-master/libs/documentation-sync/documentconversion"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// FileReader reads a workspace file by its workspace-relative path.
type FileReader func(path documentdiscovery.DocumentPath) ([]byte, error)

// Prepared is every document converted and its attachments hashed: what the planner
// compares and what the executor writes.
type Prepared struct {
	Pages    map[documentdiscovery.DocumentPath]PreparedPage
	Warnings []string
}

// PreparedPage is one document ready to write.
type PreparedPage struct {
	Converted documentconversion.Converted
	// Files are the attachments that exist on disk, with their hashes. One that could
	// not be read is left out with a warning, and the page shows it broken.
	Files []PreparedFile
	// Annotation is the file's annotation before this sync, or nil.
	Annotation *syncannotation.Annotation
}

// PreparedFile is an attachment, read again when it is uploaded so a large workspace
// is never held in memory.
type PreparedFile struct {
	Name string
	Path documentdiscovery.DocumentPath
	Hash string
}

// Rendered is syncplanning.Input.Rendered: per document, the hash of the page body as
// converted and its attachments' hashes, the same values the write-back records.
func (p Prepared) Rendered() map[documentdiscovery.DocumentPath]syncplanning.RenderedPage {
	rendered := make(map[documentdiscovery.DocumentPath]syncplanning.RenderedPage, len(p.Pages))
	for path, page := range p.Pages {
		rendered[path] = syncplanning.RenderedPage{
			RenderHash: page.Converted.RenderHash, Attachments: page.hashes(), LinkedPages: page.Converted.LinkedPages,
		}
	}

	return rendered
}

func (p PreparedPage) hashes() map[string]string {
	if len(p.Files) == 0 {
		return nil
	}
	hashes := make(map[string]string, len(p.Files))
	for _, file := range p.Files {
		hashes[file.Name] = file.Hash
	}

	return hashes
}
