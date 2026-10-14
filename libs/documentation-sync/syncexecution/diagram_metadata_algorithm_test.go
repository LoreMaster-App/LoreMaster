package syncexecution

import (
	"strings"
	"testing"
)

func TestInjectAndExtractDiagramSource(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><g/></svg>`)
	cases := map[string]string{"mermaid": "graph TD; A-->B\n  B-->C", "d2": "web -> api: calls\napi -> db"}
	for language, source := range cases {
		t.Run(language, func(t *testing.T) {
			injected := injectDiagramSource(svg, language, source)
			if !strings.Contains(string(injected), `<svg xmlns="http://www.w3.org/2000/svg"><!--lore-master:`+language+`=`) {
				t.Fatalf("the comment was not placed just inside <svg>: %s", injected)
			}
			gotLanguage, got, ok := extractDiagramSource(injected)
			if !ok || got != source || gotLanguage != language {
				t.Fatalf("round trip got %q %q ok %v", gotLanguage, got, ok)
			}
		})
	}
}

func TestExtractDiagramSourceIsFalseWithoutMetadata(t *testing.T) {
	if _, _, ok := extractDiagramSource([]byte("<svg></svg>")); ok {
		t.Fatal("an SVG with no metadata must extract nothing")
	}
}

func TestInjectLeavesNonSVGUnchanged(t *testing.T) {
	if got := injectDiagramSource([]byte("not an svg"), "mermaid", "x"); string(got) != "not an svg" {
		t.Fatalf("non-SVG bytes should be unchanged, got %q", got)
	}
}
