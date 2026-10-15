package diagramrouting

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
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

type browserStub struct {
	err   error
	asked []string
}

func (b *browserStub) Render(_ context.Context, language string, _ string) ([]byte, error) {
	b.asked = append(b.asked, language)
	if b.err != nil {
		return nil, b.err
	}

	return []byte("<svg>browser</svg>"), nil
}

func TestMermaidIsDrawnInTheBrowserWhenThereIsOne(t *testing.T) {
	editor, browser := &editorStub{}, &browserStub{}

	svg, err := RouteWith(editor, browser).Render(context.Background(), "mermaid", "graph TD; A-->B")

	if err != nil || !strings.Contains(string(svg), "browser") || len(editor.asked) != 0 {
		t.Fatalf("svg %q, err %v, editor asked %v", svg, err, editor.asked)
	}
}

func TestMermaidFallsBackToTheEditorWhenTheBrowserIsUnavailable(t *testing.T) {
	editor, browser := &editorStub{}, &browserStub{err: platformport.ErrRendererUnavailable}

	svg, err := RouteWith(editor, browser).Render(context.Background(), "mermaid", "graph TD; A-->B")

	if err != nil || !strings.Contains(string(svg), "editor") || len(browser.asked) != 1 {
		t.Fatalf("svg %q, err %v", svg, err)
	}
}

func TestADiagramTheBrowserCouldNotDrawIsNotAskedOfTheEditor(t *testing.T) {
	editor, browser := &editorStub{}, &browserStub{err: errors.New("Parse error on line 1")}

	_, err := RouteWith(editor, browser).Render(context.Background(), "mermaid", "graph TD; A-->")

	if err == nil || !strings.Contains(err.Error(), "Parse error") || len(editor.asked) != 0 {
		t.Fatalf("err %v, editor asked %v", err, editor.asked)
	}
}

func TestWithNeitherABrowserNorAnEditorMermaidIsRefused(t *testing.T) {
	browser := &browserStub{err: platformport.ErrRendererUnavailable}

	if _, err := RouteWith(nil, browser).Render(context.Background(), "mermaid", "graph TD; A-->B"); err == nil {
		t.Fatal("expected an error")
	}
}
