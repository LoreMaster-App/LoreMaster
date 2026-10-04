package syncexecution

import (
	"strings"
	"testing"
)

func TestInjectAndExtractMermaidSource(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`)
	source := "graph TD; A-->B\n  B-->C"

	injected := injectMermaidSource(svg, source)
	if !strings.Contains(string(injected), `<svg xmlns="http://www.w3.org/2000/svg"><!--lore-master:mermaid=`) {
		t.Fatalf("the comment was not placed just inside <svg>: %s", injected)
	}
	got, ok := extractMermaidSource(injected)
	if !ok || got != source {
		t.Fatalf("round trip got %q ok %v", got, ok)
	}
}

func TestExtractMermaidSourceIsFalseWithoutMetadata(t *testing.T) {
	if _, ok := extractMermaidSource([]byte("<svg></svg>")); ok {
		t.Fatal("an SVG with no metadata must extract nothing")
	}
}

func TestInjectLeavesNonSVGUnchanged(t *testing.T) {
	if got := injectMermaidSource([]byte("not an svg"), "x"); string(got) != "not an svg" {
		t.Fatalf("non-SVG bytes should be unchanged, got %q", got)
	}
}
