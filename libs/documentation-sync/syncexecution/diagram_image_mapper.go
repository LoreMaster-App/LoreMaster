package syncexecution

import "lore-master/libs/documentation-sync/platformport"

// mermaidSources lists a document's distinct Mermaid sources in order, nested ones
// (in a list or a quote) included.
func mermaidSources(blocks []platformport.Block) []string {
	var sources []string
	seen := map[string]bool{}
	var walk func([]platformport.Block)
	walk = func(blocks []platformport.Block) {
		for _, block := range blocks {
			switch b := block.(type) {
			case platformport.Diagram:
				if b.Language == "mermaid" && !seen[b.Source] {
					seen[b.Source] = true
					sources = append(sources, b.Source)
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

// withDiagramImages returns a copy of blocks in which each Mermaid diagram with a
// rendered picture points at it. The input is not modified.
func withDiagramImages(blocks []platformport.Block, images map[string]string) []platformport.Block {
	if len(images) == 0 {
		return blocks
	}
	copied := make([]platformport.Block, len(blocks))
	for i, block := range blocks {
		switch b := block.(type) {
		case platformport.Diagram:
			if name, ok := images[b.Source]; ok && b.Language == "mermaid" {
				b.Image = &platformport.AttachmentRef{Filename: name}
			}
			copied[i] = b
		case platformport.Blockquote:
			copied[i] = platformport.Blockquote{Blocks: withDiagramImages(b.Blocks, images)}
		case platformport.List:
			list := platformport.List{Ordered: b.Ordered, Start: b.Start, Items: make([]platformport.ListItem, len(b.Items))}
			for j, item := range b.Items {
				list.Items[j] = platformport.ListItem{Blocks: withDiagramImages(item.Blocks, images)}
			}
			copied[i] = list
		default:
			copied[i] = block
		}
	}

	return copied
}
