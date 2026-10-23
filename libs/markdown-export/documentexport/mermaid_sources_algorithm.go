package documentexport

import (
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// mermaidSources lists the Mermaid diagrams of a document, each once, in the order they first appear: the code blocks written
// as mermaid, wherever they sit (in a list or a quote too).
func mermaidSources(blocks []exportmodel.Block) []string {
	return diagramSources(blocks, "mermaid")
}

// diagramSources lists the diagrams of one language, each once, in the order they first appear.
func diagramSources(blocks []exportmodel.Block, language string) []string {
	var sources []string
	seen := map[string]bool{}
	var walk func(list []exportmodel.Block)
	walk = func(list []exportmodel.Block) {
		for _, block := range list {
			if block.Kind == exportmodel.CodeBlock && strings.EqualFold(block.Lang, language) && !seen[block.Text] {
				seen[block.Text] = true
				sources = append(sources, block.Text)
			}
			walk(block.Children)
			for _, item := range block.Items {
				walk(item.Blocks)
			}
		}
	}
	walk(blocks)

	return sources
}
