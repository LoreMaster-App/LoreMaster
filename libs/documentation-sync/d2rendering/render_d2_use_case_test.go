package d2rendering

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

const architecture = "web: Web app\napi: API {shape: hexagon}\ndb: Database {shape: cylinder}\nweb -> api: calls\napi -> db: reads\n"

func TestRenderDrawsAnSVGWithTheLabels(t *testing.T) {
	svg, err := New().Render(context.Background(), "d2", architecture)
	if err != nil {
		t.Fatal(err)
	}
	text := string(svg)
	if !strings.Contains(text, "<svg") {
		t.Fatalf("not an SVG: %.200s", text)
	}
	for _, label := range []string{"Web app", "Database", "calls"} {
		if !strings.Contains(text, label) {
			t.Errorf("the drawing lacks the label %q", label)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	first, err := New().Render(context.Background(), "d2", architecture)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New().Render(context.Background(), "d2", architecture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same diagram drew two different SVGs, so every sync would upload it again")
	}
}

func TestRenderExplainsAnInvalidDiagram(t *testing.T) {
	_, err := New().Render(context.Background(), "d2", "a -> {")
	if err == nil || strings.TrimSpace(err.Error()) == "" {
		t.Fatalf("want D2's own error, got %v", err)
	}
}

func TestRenderRefusesOtherLanguagesAndEmptySource(t *testing.T) {
	if _, err := New().Render(context.Background(), "mermaid", "graph TD; A-->B"); err == nil {
		t.Fatal("a Mermaid diagram must not be drawn by the D2 renderer")
	}
	if _, err := New().Render(context.Background(), "d2", "  \n"); err == nil {
		t.Fatal("an empty diagram must be refused")
	}
}

func TestRenderStopsWhenTheContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().Render(ctx, "d2", architecture); err == nil {
		t.Fatal("want an error for a cancelled context")
	}
}
