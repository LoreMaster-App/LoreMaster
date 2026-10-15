package documentconversion

import (
	"reflect"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

func convert(t *testing.T, markdown string) []platformport.Block {
	t.Helper()
	document := parse(t, "docs/guide.md", markdown)
	workspace := NewWorkspace([]documentparsing.MarkdownDocument{document}, map[documentdiscovery.DocumentPath]string{"docs/guide.md": "ENG: Guide"})

	return ConvertDocument(document, workspace).Document.Blocks
}

func TestAChildrenTagBecomesAChildrenBlock(t *testing.T) {
	blocks := convert(t, "# Guide\n\nIntro.\n\n[children]\n\n[children all]\n\n[Children depth=2]\n")

	want := []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{platformport.Text{Value: "Intro."}}},
		platformport.Children{},
		platformport.Children{All: true},
		platformport.Children{Depth: 2},
	}
	if !reflect.DeepEqual(blocks, want) {
		t.Fatalf("blocks = %+v\nwant     %+v", blocks, want)
	}
}

func TestAChildrenTagInCodeOrTextStaysText(t *testing.T) {
	blocks := convert(t, "# Guide\n\nUse the [children] tag.\n\n```md\n[children]\n```\n")

	for _, block := range blocks {
		if _, isChildren := block.(platformport.Children); isChildren {
			t.Fatalf("blocks = %+v", blocks)
		}
	}
}
