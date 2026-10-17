package diagramcommands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func render(t *testing.T, params rpcprotocol.DiagramRenderParams) (any, error) {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	return RenderDiagram()(context.Background(), rpcserver.Call{Params: raw})
}

func TestRenderDiagramDrawsD2(t *testing.T) {
	result, err := render(t, rpcprotocol.DiagramRenderParams{Language: "d2", Source: "a -> b"})
	if err != nil {
		t.Fatal(err)
	}
	drawn, ok := result.(rpcprotocol.DiagramRenderResult)
	if !ok || !strings.Contains(drawn.SVG, "<svg") {
		t.Fatalf("got %#v", result)
	}
}

func TestRenderDiagramRefusesOtherLanguages(t *testing.T) {
	if _, err := render(t, rpcprotocol.DiagramRenderParams{Language: "mermaid", Source: "graph TD; a-->b"}); err == nil {
		t.Fatal("mermaid is drawn by the editor, not here")
	}
}

func TestRenderDiagramReportsABrokenDiagram(t *testing.T) {
	if _, err := render(t, rpcprotocol.DiagramRenderParams{Language: "d2", Source: "a -> {"}); err == nil {
		t.Fatal("a broken diagram must be an error")
	}
}
