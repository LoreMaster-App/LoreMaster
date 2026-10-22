package syncexecution

import (
	"slices"

	"lore-master/libs/documentation-sync/platformport"
)

// diagramSource is one diagram of a document: its language and its source.
type diagramSource struct {
	Language string
	Source   string
}

// key identifies a diagram within a document: the same source in another language is another
// diagram.
func (d diagramSource) key() string {
	return d.Language + "\x00" + d.Source
}

// diagramSources lists a document's distinct diagrams in order, nested ones (in a list or a
// quote) included.
func diagramSources(blocks []platformport.Block) []diagramSource {
	var sources []diagramSource
	seen := map[string]bool{}
	var walk func([]platformport.Block)
	walk = func(blocks []platformport.Block) {
		for _, block := range blocks {
			switch b := block.(type) {
			case platformport.Diagram:
				found := diagramSource{Language: b.Language, Source: b.Source}
				if slices.Contains(diagramLanguages, b.Language) && !seen[found.key()] {
					seen[found.key()] = true
					sources = append(sources, found)
				}
			case platformport.Blockquote:
				walk(b.Blocks)
			case platformport.List:
				for _, item := range b.Items {
					walk(item.Blocks)
				}
			}
		}
	}
	walk(blocks)

	return sources
}

// withDiagramImages returns a copy of blocks in which each diagram with a rendered picture
// (both maps are keyed by diagramSource.key) points at it: images are attachment names, pictures
// are SVG documents to show in the page itself. The input is not modified.
func withDiagramImages(blocks []platformport.Block, images map[string]string, pictures map[string]string) []platformport.Block {
	if len(images) == 0 && len(pictures) == 0 {
		return blocks
	}
	copied := make([]platformport.Block, len(blocks))
	for i, block := range blocks {
		switch b := block.(type) {
		case platformport.Diagram:
			key := diagramSource{Language: b.Language, Source: b.Source}.key()
			if name, ok := images[key]; ok {
				b.Image = &platformport.AttachmentRef{Filename: name}
			}
			if svg, ok := pictures[key]; ok {
				b.SVG = svg
			}
			copied[i] = b
		case platformport.Blockquote:
			copied[i] = platformport.Blockquote{Blocks: withDiagramImages(b.Blocks, images, pictures)}
		case platformport.List:
			list := platformport.List{Ordered: b.Ordered, Start: b.Start, Items: make([]platformport.ListItem, len(b.Items))}
			for j, item := range b.Items {
				list.Items[j] = platformport.ListItem{Blocks: withDiagramImages(item.Blocks, images, pictures)}
			}
			copied[i] = list
		default:
			copied[i] = block
		}
	}

	return copied
}
