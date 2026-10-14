package diagramrouting

import (
	"context"
	"strings"
	"testing"
)

type editorStub struct{ asked []string }

func (e *editorStub) Render(_ context.Context, language string, _ string) ([]byte, error) {
	e.asked = append(e.asked, language)

	return []byte("<svg>editor</svg>"), nil
}

func TestRouteDrawsD2InTheEngineAndTheRestInTheEditor(t *testing.T) {
	editor := &editorStub{}
	renderer := Route(editor)

	if svg, err := renderer.Render(context.Background(), "d2", "a -> b"); err != nil || !strings.Contains(string(svg), "<svg") || strings.Contains(string(svg), "editor") {
		t.Fatalf("D2 was not drawn by the engine: %v", err)
	}
	if _, err := renderer.Render(context.Background(), "mermaid", "graph TD; A-->B"); err != nil {
		t.Fatal(err)
	}
	if len(editor.asked) != 1 || editor.asked[0] != "mermaid" {
		t.Fatalf("the editor was asked %v, want only mermaid", editor.asked)
	}
}

func TestRouteWithoutAnEditorStillDrawsD2AndRefusesTheRest(t *testing.T) {
	renderer := Route(nil)

	if _, err := renderer.Render(context.Background(), "d2", "a -> b"); err != nil {
		t.Fatalf("D2 needs no editor: %v", err)
	}
	if _, err := renderer.Render(context.Background(), "mermaid", "graph TD; A-->B"); err == nil {
		t.Fatal("Mermaid with no editor must be refused")
	}
}
