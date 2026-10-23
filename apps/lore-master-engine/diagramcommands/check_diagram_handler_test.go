package diagramcommands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/platformport"
)

type fakeMermaid struct {
	problems []platformport.DiagramProblem
	err      error
	asked    *int
	closed   *bool
}

func (f fakeMermaid) Check(context.Context, string) ([]platformport.DiagramProblem, error) {
	*f.asked++

	return f.problems, f.err
}

func (f fakeMermaid) Close() { *f.closed = true }

func check(t *testing.T, mermaid fakeMermaid, diagrams ...rpcprotocol.DiagramCheckItem) []rpcprotocol.DiagramCheckOutcome {
	t.Helper()
	raw, err := json.Marshal(rpcprotocol.DiagramCheckParams{Diagrams: diagrams})
	if err != nil {
		t.Fatal(err)
	}
	result, err := CheckDiagramWith(func() MermaidChecker { return mermaid })(context.Background(), rpcserver.Call{Params: raw})
	if err != nil {
		t.Fatal(err)
	}

	return result.(rpcprotocol.DiagramCheckResult).Results
}

func TestCheckDiagramAnswersEachDiagramInOrder(t *testing.T) {
	asked, closed := 0, false
	mermaid := fakeMermaid{problems: []platformport.DiagramProblem{{Line: 2, Message: "Parse error on line 2"}}, asked: &asked, closed: &closed}

	results := check(t, mermaid,
		rpcprotocol.DiagramCheckItem{Language: "d2", Source: "a -> b"},
		rpcprotocol.DiagramCheckItem{Language: "d2", Source: "a -> {"},
		rpcprotocol.DiagramCheckItem{Language: "mermaid", Source: "graph TD\n  ???"},
		rpcprotocol.DiagramCheckItem{Language: "plantuml", Source: "x"},
	)

	if len(results) != 4 || len(results[0].Problems) != 0 || len(results[1].Problems) == 0 || results[2].Problems[0].Line != 2 || len(results[3].Problems) != 0 {
		t.Fatalf("results = %+v", results)
	}
	if asked != 1 || !closed {
		t.Fatalf("the Mermaid checker should be asked once and stopped: asked %d, closed %v", asked, closed)
	}
}

func TestCheckDiagramDoesNotStartMermaidForD2Alone(t *testing.T) {
	asked, closed := 0, false
	check(t, fakeMermaid{asked: &asked, closed: &closed}, rpcprotocol.DiagramCheckItem{Language: "d2", Source: "a -> b"})

	if asked != 0 || closed {
		t.Fatalf("no browser is needed for D2: asked %d, closed %v", asked, closed)
	}
}

func TestCheckDiagramSaysWhenMermaidCannotBeChecked(t *testing.T) {
	asked, closed := 0, false
	results := check(t, fakeMermaid{err: errors.New("no browser here"), asked: &asked, closed: &closed}, rpcprotocol.DiagramCheckItem{Language: "mermaid", Source: "graph TD; A-->B"})

	if results[0].Unavailable == "" || len(results[0].Problems) != 0 {
		t.Fatalf("results = %+v", results)
	}
}
