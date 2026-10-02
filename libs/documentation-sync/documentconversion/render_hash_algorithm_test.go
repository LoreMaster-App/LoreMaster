package documentconversion

import (
	"testing"

	"lore-master/libs/documentation-sync/platformport"
)

func TestRenderHashTellsNodesApart(t *testing.T) {
	text := []platformport.Inline{platformport.Text{Value: "x"}}
	documents := map[string]platformport.Document{
		"emphasis":     {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Emphasis{Inlines: text}}}}},
		"strong":       {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Strong{Inlines: text}}}}},
		"link Setup":   {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Link{Target: platformport.PageLink{Title: "Setup"}, Inlines: text}}}}},
		"link Install": {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Link{Target: platformport.PageLink{Title: "Install"}, Inlines: text}}}}},
		"anchor":       {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Link{Target: platformport.PageLink{Title: "Setup", Anchor: "A"}, Inlines: text}}}}},
		"image a":      {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Image{Source: &platformport.AttachmentRef{Filename: "a.png"}}}}}},
		"image b":      {Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Image{Source: &platformport.AttachmentRef{Filename: "b.png"}}}}}},
		"list 1":       {Blocks: []platformport.Block{platformport.List{Ordered: true, Start: 1}}},
		"list 2":       {Blocks: []platformport.Block{platformport.List{Ordered: true, Start: 2}}},
		"empty":        {},
	}
	seen := map[string]string{}
	for name, document := range documents {
		hash := renderHash(document)
		if other, clash := seen[hash]; clash {
			t.Errorf("%s and %s hash the same", name, other)
		}
		seen[hash] = name
		if renderHash(document) != hash {
			t.Errorf("%s: two runs disagree", name)
		}
	}
}
