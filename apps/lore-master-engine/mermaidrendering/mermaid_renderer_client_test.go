package mermaidrendering

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lore-master/libs/content-generation/storybookdocs"
	"lore-master/libs/documentation-sync/platformport"
)

func TestTheEmbeddedBundleIsMermaid(t *testing.T) {
	script, err := bundle()
	if err != nil {
		t.Fatal(err)
	}
	if len(script) < 1_000_000 || !strings.Contains(script, "mermaid") {
		t.Fatalf("the embedded bundle is %d bytes and does not look like Mermaid", len(script))
	}
}

func TestWithoutABrowserTheRendererSaysItIsUnavailableOnce(t *testing.T) {
	calls := 0
	renderer := &Renderer{timeout: time.Second, find: func() (string, error) {
		calls++

		return "", errors.New("no browser here")
	}}

	for range 2 {
		_, err := renderer.Render(context.Background(), "mermaid", "graph TD; A-->B")
		if !errors.Is(err, platformport.ErrRendererUnavailable) {
			t.Fatalf("err = %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("the browser was looked for %d times", calls)
	}
}

func TestOtherLanguagesAreRefused(t *testing.T) {
	renderer := New()

	if _, err := renderer.Render(context.Background(), "d2", "a -> b"); err == nil || errors.Is(err, platformport.ErrRendererUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

// browserTest skips when this machine has no Chrome, Chromium or Edge.
func browserTest(t *testing.T) *Renderer {
	t.Helper()
	if _, err := storybookdocs.FindBrowser(); err != nil {
		t.Skip("no browser to draw in: " + err.Error())
	}
	renderer := New()
	t.Cleanup(renderer.Close)

	return renderer
}

func TestADiagramIsDrawnInTheBrowser(t *testing.T) {
	renderer := browserTest(t)

	svg, err := renderer.Render(context.Background(), "mermaid", "flowchart LR\n  A[Start] --> B{Ok?}\n  B -->|yes| C[Done]")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(svg)), "<svg") || !strings.Contains(string(svg), "Start") {
		t.Fatalf("not the drawing: %.200s", svg)
	}
}

func TestABadDiagramFailsAloneAndTheNextOneStillDraws(t *testing.T) {
	renderer := browserTest(t)

	_, err := renderer.Render(context.Background(), "mermaid", "flowchart LR\n  A -->")
	if err == nil || !strings.Contains(err.Error(), "Parse error") {
		t.Fatalf("err = %v", err)
	}
	if _, err := renderer.Render(context.Background(), "mermaid", "sequenceDiagram\n  A->>B: hi"); err != nil {
		t.Fatalf("the next diagram failed too: %v", err)
	}
}

func TestEveryDiagramTypeDrawsWithoutAnEditor(t *testing.T) {
	renderer := browserTest(t)

	for name, source := range map[string]string{
		"class":    "classDiagram\n  Animal <|-- Dog",
		"state":    "stateDiagram-v2\n  [*] --> Still",
		"er":       "erDiagram\n  CUSTOMER ||--o{ ORDER : places",
		"gantt":    "gantt\n  dateFormat YYYY-MM-DD\n  section A\n  a :a1, 2026-01-01, 30d",
		"pie":      "pie title Pets\n  \"Dogs\" : 4\n  \"Cats\" : 2",
		"mindmap":  "mindmap\n  root((r))\n    a\n    b",
		"timeline": "timeline\n  title t\n  2002 : x",
	} {
		if _, err := renderer.Render(context.Background(), "mermaid", source); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestADiagramThatRunsOutOfTimeIsReplacedByAFreshTabForTheNext(t *testing.T) {
	renderer := browserTest(t)
	if _, err := renderer.Render(context.Background(), "mermaid", "flowchart LR\n  A --> B"); err != nil {
		t.Fatal(err)
	}

	renderer.timeout = time.Millisecond
	_, err := renderer.Render(context.Background(), "mermaid", "flowchart LR\n  A --> B")
	if err == nil || !strings.Contains(err.Error(), "did not finish drawing") {
		t.Fatalf("err = %v", err)
	}

	renderer.timeout = DefaultTimeout
	if _, err := renderer.Render(context.Background(), "mermaid", "flowchart LR\n  C --> D"); err != nil {
		t.Fatalf("the next diagram was held up: %v", err)
	}
}

func TestLineOfReadsTheLineMermaidNames(t *testing.T) {
	for message, want := range map[string]int{"Parse error on line 3:\n...": 3, "Lexical error on line 12. Unrecognized text.": 12, "no line here": 0} {
		if got := lineOf(message); got != want {
			t.Errorf("%q: got %d, want %d", message, got, want)
		}
	}
}

func TestWithoutABrowserACheckSaysItIsUnavailable(t *testing.T) {
	renderer := &Renderer{timeout: time.Second, find: func() (string, error) { return "", errors.New("no browser here") }}

	if _, err := renderer.Check(context.Background(), "graph TD; A-->B"); !errors.Is(err, platformport.ErrRendererUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestACheckFindsTheErrorOfABadDiagramAndPassesAGoodOne(t *testing.T) {
	renderer := browserTest(t)

	if problems, err := renderer.Check(context.Background(), "graph TD\n  A-->B"); err != nil || len(problems) != 0 {
		t.Fatalf("a good diagram: %v %v", problems, err)
	}
	problems, err := renderer.Check(context.Background(), "graph TD\n  A-->B\n  B --- ??? C")
	if err != nil || len(problems) != 1 || problems[0].Line == 0 {
		t.Fatalf("a bad diagram should give one problem with a line: %v %v", problems, err)
	}
}
