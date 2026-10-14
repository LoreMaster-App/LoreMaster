package storageformat

import (
	"strings"
	"testing"
)

func TestD2DiagramIsAPictureAboveItsMarkedSourceAndReadsBack(t *testing.T) {
	diagram := Mermaid{Language: "d2", Source: "web -> api\n", Image: &AttachmentRef{Filename: "d2-abc.svg"}}
	for _, mode := range []MermaidMode{MermaidImage, MermaidCode, MermaidHTMLMacro} {
		storage, err := Render(Document{Blocks: []Block{diagram}}, Options{MermaidMode: mode})
		if err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		for _, want := range []string{`ac:alt="D2 diagram"`, `ri:filename="d2-abc.svg"`, `<ac:parameter ac:name="lore-master">d2</ac:parameter>`} {
			if !strings.Contains(storage, want) {
				t.Fatalf("mode %d: storage lacks %s:\n%s", mode, want, storage)
			}
		}
		parsed, flags, err := Parse(storage)
		if err != nil || len(flags) != 0 {
			t.Fatalf("mode %d: %v %v", mode, err, flags)
		}
		if len(parsed.Blocks) != 1 {
			t.Fatalf("mode %d: blocks %+v", mode, parsed.Blocks)
		}
		got, ok := parsed.Blocks[0].(Mermaid)
		if !ok || got.Language != "d2" || got.Source != "web -> api\n" || got.Image == nil || got.Image.Filename != "d2-abc.svg" {
			t.Fatalf("mode %d: read back %+v", mode, parsed.Blocks[0])
		}
	}
}

func TestD2DiagramWithoutAPictureIsJustItsMarkedSource(t *testing.T) {
	storage, err := Render(Document{Blocks: []Block{Mermaid{Language: "d2", Source: "a -> b"}}}, Options{MermaidMode: MermaidImage})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storage, "<ac:image") || !strings.Contains(storage, ">d2</ac:parameter>") {
		t.Fatalf("storage = %s", storage)
	}
	parsed, _, err := Parse(storage)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := parsed.Blocks[0].(Mermaid); !ok || got.Language != "d2" || got.Image != nil {
		t.Fatalf("read back %+v", parsed.Blocks[0])
	}
}

func TestAMermaidDiagramKeepsItsMarkerAndAltText(t *testing.T) {
	storage, err := Render(Document{Blocks: []Block{Mermaid{Source: "graph TD", Image: &AttachmentRef{Filename: "m.svg"}}}}, Options{MermaidMode: MermaidImage})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(storage, `ac:alt="Mermaid diagram"`) || !strings.Contains(storage, `>mermaid</ac:parameter>`) {
		t.Fatalf("storage = %s", storage)
	}
}
