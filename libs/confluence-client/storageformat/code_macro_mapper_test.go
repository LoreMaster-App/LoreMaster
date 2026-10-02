package storageformat

import (
	"encoding/xml"
	"strings"
	"testing"
)

type macroCase struct {
	doc     Document
	options Options
}

var macroCases = map[string]macroCase{
	"code-macro": {doc: Document{Blocks: []Block{
		CodeBlock{Language: "ts", Code: "const a: number = 1 < 2 && true\n"},
		CodeBlock{Language: "go title=main.go", Code: "package main\n"},
		CodeBlock{Language: "brainfuck", Code: "+[----->+++<]>+."},
		CodeBlock{Code: "no language, and a CDATA end ]]> inside, and a nul \x00 too"},
	}}},
	"image-macro": {doc: Document{Blocks: []Block{
		Paragraph{Inlines: []Inline{
			Image{Source: &AttachmentRef{Filename: "diagram 1.svg"}, Alt: `A "quoted" alt`, Title: "Title", Width: 400},
			Text{Value: " and "},
			Image{Source: &URLRef{URL: "https://example.com/a.png?x=1&y=2"}},
		}},
	}}},
	"page-link-by-title": {doc: Document{Blocks: []Block{
		Paragraph{Inlines: []Inline{
			Link{Target: PageLink{Title: "ENG: Setup & Install", Anchor: "ENG:Setup&Install-Prerequisites"}, Inlines: []Inline{Text{Value: "see "}, Strong{Inlines: []Inline{Text{Value: "setup"}}}}},
			Text{Value: ", "},
			Link{Target: PageLink{Title: "ENG: Architecture"}},
			Text{Value: ", "},
			Link{Target: &AttachmentRef{Filename: "report.pdf"}, Inlines: []Inline{Text{Value: "the report"}}},
			Text{Value: ", "},
			Link{Target: &URLRef{URL: "https://example.com/?a=1&b=2"}, Inlines: []Inline{Text{Value: "example"}}},
		}},
	}}},
	"page-link-by-url": {options: Options{LinkMode: LinkByURL}, doc: Document{Blocks: []Block{
		Paragraph{Inlines: []Inline{
			Link{Target: PageLink{Title: "ENG: Setup", Anchor: "install", URL: "https://acme.atlassian.net/wiki/spaces/ENG/pages/42"}, Inlines: []Inline{Text{Value: "setup"}}},
			Text{Value: " / "},
			Link{Target: PageLink{Title: "ENG: Not created yet"}},
		}},
	}}},
	"mermaid-image": {doc: Document{Blocks: []Block{
		Mermaid{Source: "graph TD\n  A-->B\n", Image: &AttachmentRef{Filename: "mermaid-1.svg"}},
		Mermaid{Source: "graph LR\n  C-->D\n"},
	}}},
	"mermaid-code": {options: Options{MermaidMode: MermaidCode}, doc: Document{Blocks: []Block{
		Mermaid{Source: "graph TD\n  A-->B\n", Image: &AttachmentRef{Filename: "mermaid-1.svg"}},
	}}},
}

func TestMacroGolden(t *testing.T) {
	for name, tc := range macroCases {
		t.Run(name, func(t *testing.T) {
			got, err := Render(tc.doc, tc.options)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, name, got)
		})
	}
}

// TestCodeSurvivesCDATA reads the code macro back with an XML parser: whatever the
// code contains, the plain-text body is the code (minus characters XML cannot carry).
func TestCodeSurvivesCDATA(t *testing.T) {
	for _, code := range []string{"]]>", "a]]>b]]>c", "]]]]>", "<![CDATA[ nested ]]>", "x]]", "]>"} {
		got, err := Render(Document{Blocks: []Block{CodeBlock{Code: code}}}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(strings.NewReader(`<r xmlns:ac="a">` + got + `</r>`))
		var body strings.Builder
		inBody := false
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			switch tok := token.(type) {
			case xml.StartElement:
				inBody = tok.Name.Local == "plain-text-body"
			case xml.EndElement:
				inBody = false
			case xml.CharData:
				if inBody {
					body.Write(tok)
				}
			}
		}
		if body.String() != code {
			t.Errorf("code %q read back as %q from %s", code, body.String(), got)
		}
	}
}

func TestMacroInputsAreValidated(t *testing.T) {
	cases := map[string]macroCase{
		"page link without title":    {doc: Document{Blocks: []Block{Paragraph{Inlines: []Inline{Link{Target: PageLink{}}}}}}},
		"image without source":       {doc: Document{Blocks: []Block{Paragraph{Inlines: []Inline{Image{}}}}}},
		"attachment without name":    {doc: Document{Blocks: []Block{Paragraph{Inlines: []Inline{Image{Source: &AttachmentRef{}}}}}}},
		"link without target":        {doc: Document{Blocks: []Block{Paragraph{Inlines: []Inline{Link{}}}}}},
		"mermaid html macro not yet": {doc: Document{Blocks: []Block{Mermaid{Source: "graph TD"}}}, options: Options{MermaidMode: MermaidHTMLMacro}},
	}
	for name, tc := range cases {
		if _, err := Render(tc.doc, tc.options); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
