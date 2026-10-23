package documentexport

import (
	"errors"
	"strings"
	"testing"
)

const d2Document = "# D\n\n```d2\na -> b\n```\n\n- item\n\n  ```d2\n  a -> b\n  ```\n\n```D2 theme=dark\nbad -> {\n```\n\n```go\nfmt.Println()\n```\n"

func drawer(asked *[]string) func(string) (string, error) {
	return func(source string) (string, error) {
		*asked = append(*asked, source)
		if strings.Contains(source, "bad") {
			return "", errors.New("D2 refused it")
		}

		return "<svg id=\"drawn\"></svg>", nil
	}
}

func exportHTML(t *testing.T, options Options) string {
	t.Helper()
	file, err := Export("d.md", []byte(d2Document), "html", options)
	if err != nil {
		t.Fatal(err)
	}

	return string(file.Content)
}

func TestHTMLDrawsD2DiagramsWithoutABrowserAndDrawsEachSourceOnce(t *testing.T) {
	var asked []string
	page := exportHTML(t, Options{DrawD2: drawer(&asked)})

	if strings.Count(page, `<svg id="drawn">`) != 2 {
		t.Fatalf("both fences of the drawn source should be pictures: %s", page)
	}
	if len(asked) != 2 {
		t.Fatalf("each distinct source is drawn once, got %q", asked)
	}
}

func TestADiagramD2RefusesAndOtherCodeStayCode(t *testing.T) {
	var asked []string
	page := exportHTML(t, Options{DrawD2: drawer(&asked)})

	if !strings.Contains(page, "bad -&gt; {") || !strings.Contains(page, "fmt.Println()") {
		t.Fatalf("a refused diagram and a Go block stay as code: %s", page)
	}
}

func TestWithoutADrawerD2StaysCodeAsItAlwaysWas(t *testing.T) {
	page := exportHTML(t, Options{})

	if strings.Contains(page, "<svg") || !strings.Contains(page, "a -&gt; b") {
		t.Fatalf("no drawer, no pictures: %s", page)
	}
}

func TestAPreviewIsLeftToDrawItsOwnD2(t *testing.T) {
	var asked []string
	page := exportHTML(t, Options{DrawD2: drawer(&asked), Preview: true})

	if len(asked) != 0 || strings.Contains(page, "<svg") {
		t.Fatalf("a preview draws its own: asked %q", asked)
	}
}
