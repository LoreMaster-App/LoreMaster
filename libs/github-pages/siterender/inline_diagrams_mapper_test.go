package siterender

import (
	"errors"
	"strings"
	"testing"
)

func fakeDraw(source string) ([]byte, error) {
	if strings.Contains(source, "broken") {
		return nil, errors.New("syntax error")
	}

	return []byte("<svg>" + source + "</svg>"), nil
}

func TestInlineDiagramsReplacesTheFencedBlockWithAnImage(t *testing.T) {
	markdown := "# Title\n\n```d2\nweb -> api\n```\n\nafter\n"
	got, notes := InlineDiagrams([]byte(markdown), "d2", fakeDraw)
	if len(notes) != 0 {
		t.Fatalf("notes %v", notes)
	}
	text := string(got)
	if !strings.Contains(text, "![D2 diagram](data:image/svg+xml;base64,") || strings.Contains(text, "web -> api") {
		t.Fatalf("the block was not replaced:\n%s", text)
	}
	if !strings.HasPrefix(text, "# Title\n\n") || !strings.HasSuffix(text, "\n\nafter\n") {
		t.Fatalf("the rest of the page changed:\n%s", text)
	}
}

func TestInlineDiagramsLeavesOtherFencesAndBrokenDiagramsAlone(t *testing.T) {
	markdown := "```go\nfmt.Println()\n```\n\n~~~d2\nbroken\n~~~\n\n```mermaid\ngraph TD; A-->B\n```\n"
	got, notes := InlineDiagrams([]byte(markdown), "d2", fakeDraw)
	if string(got) != markdown {
		t.Fatalf("nothing should change:\n%s", got)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "line 5") || !strings.Contains(notes[0], "syntax error") {
		t.Fatalf("notes %v", notes)
	}
}

func TestInlineDiagramsHandlesLongerFencesAndAnUnclosedBlock(t *testing.T) {
	got, _ := InlineDiagrams([]byte("````d2\n```\ninner\n```\n````\n"), "d2", fakeDraw)
	if !strings.Contains(string(got), "![D2 diagram](") || strings.Contains(string(got), "inner") {
		t.Fatalf("a longer fence was not matched whole:\n%s", got)
	}
	unclosed := "```d2\nweb -> api\n"
	got, notes := InlineDiagrams([]byte(unclosed), "d2", fakeDraw)
	if string(got) != unclosed || len(notes) != 0 {
		t.Fatalf("an unclosed block must be left alone: %q %v", got, notes)
	}
}
