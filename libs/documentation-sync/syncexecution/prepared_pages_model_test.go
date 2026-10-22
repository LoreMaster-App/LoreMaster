package syncexecution

import (
	"testing"

	"lore-master/libs/documentation-sync/documentconversion"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
)

func preparedWith(blocks ...platformport.Block) Prepared {
	return Prepared{Pages: map[documentdiscovery.DocumentPath]PreparedPage{
		"a.md": {Converted: documentconversion.Converted{Document: platformport.Document{Blocks: blocks}, RenderHash: "sha256:a"}},
	}}
}

func TestWithDiagramModeChangesTheHashOfAPageWithADiagram(t *testing.T) {
	prepared := preparedWith(platformport.Diagram{Language: "mermaid", Source: "graph TD; A-->B"})
	asImage, asCode := prepared.WithDiagramMode("image"), prepared.WithDiagramMode("code")

	image, code := asImage.Pages["a.md"].Converted.RenderHash, asCode.Pages["a.md"].Converted.RenderHash
	if image == "sha256:a" || image == code {
		t.Fatalf("a page with a diagram must change with the mode: image %s, code %s", image, code)
	}
	if again := prepared.WithDiagramMode("image").Pages["a.md"].Converted.RenderHash; again != image {
		t.Fatalf("the same mode must give the same hash: %s, %s", image, again)
	}
}

func TestWithDiagramModeLeavesAPageWithoutADiagramAlone(t *testing.T) {
	prepared := preparedWith(platformport.Paragraph{})

	if hash := prepared.WithDiagramMode("image").Pages["a.md"].Converted.RenderHash; hash != "sha256:a" {
		t.Fatalf("a page without a diagram must keep its hash, got %s", hash)
	}
}
