package storageformat

import (
	"encoding/xml"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

func text(value string) Inline { return Text{Value: value} }

func paragraph(inlines ...Inline) Paragraph { return Paragraph{Inlines: inlines} }

// goldenCases is one document per construct; each renders to testdata/<name>.golden.xhtml.
var goldenCases = map[string]Document{
	"paragraph-and-inlines": {Blocks: []Block{
		paragraph(text("Plain, "), Emphasis{Inlines: []Inline{text("em")}}, text(", "), Strong{Inlines: []Inline{text("strong "), Emphasis{Inlines: []Inline{text("nested")}}}},
			text(", "), Strikethrough{Inlines: []Inline{text("gone")}}, text(", "), CodeSpan{Value: "a < b && c"}, HardBreak{}, text("next line")),
	}},
	"headings": {Blocks: []Block{
		Heading{Level: 1, Inlines: []Inline{text("One")}}, Heading{Level: 2, Inlines: []Inline{text("Two "), CodeSpan{Value: "code"}}},
		Heading{Level: 6, Inlines: []Inline{text("Six")}},
	}},
	"lists": {Blocks: []Block{
		List{Items: []ListItem{
			{Blocks: []Block{paragraph(text("tight"))}},
			{Blocks: []Block{paragraph(text("loose")), paragraph(text("second paragraph"))}},
			{Blocks: []Block{paragraph(text("parent")), List{Ordered: true, Items: []ListItem{{Blocks: []Block{paragraph(text("nested one"))}}}}}},
		}},
		List{Ordered: true, Start: 3, Items: []ListItem{{Blocks: []Block{paragraph(text("three"))}}, {Blocks: []Block{paragraph(text("four"))}}}},
		List{Ordered: true, Start: 1, Items: []ListItem{{Blocks: []Block{paragraph(text("from one"))}}}},
	}},
	"task-list": {Blocks: []Block{
		TaskList{Items: []TaskItem{{Done: true, Inlines: []Inline{text("Configure")}}, {Inlines: []Inline{text("Sync "), Strong{Inlines: []Inline{text("now")}}}}}},
		TaskList{Items: []TaskItem{{Inlines: []Inline{text("Second list continues the ids")}}}},
	}},
	"table": {Blocks: []Block{
		Table{
			Header: []TableCell{{Inlines: []Inline{text("Name")}}, {Inlines: []Inline{text("Count")}}, {Inlines: []Inline{text("Note")}}},
			Rows: [][]TableCell{
				{{Inlines: []Inline{text("a")}}, {Inlines: []Inline{text("1")}}, {Inlines: []Inline{Emphasis{Inlines: []Inline{text("first")}}}}},
				{{Inlines: []Inline{text("b & c")}}, {Inlines: []Inline{text("22")}}, {}},
			},
			Align: []Alignment{AlignLeft, AlignRight},
		},
	}},
	"blockquote-and-rule": {Blocks: []Block{
		Blockquote{Blocks: []Block{paragraph(text("Quoted")), Blockquote{Blocks: []Block{paragraph(text("deeper"))}}}},
		ThematicBreak{},
		paragraph(text("after the rule")),
	}},
	"escaping": {Blocks: []Block{
		paragraph(text(`<script>alert("x")</script> & 'quotes' -- no comment <!-- here -->` + "\x00\x0b￾")),
	}},
}

func TestGolden(t *testing.T) {
	for name, doc := range goldenCases {
		t.Run(name, func(t *testing.T) {
			got, err := Render(doc)
			if err != nil {
				t.Fatal(err)
			}
			assertWellFormed(t, got)
			if strings.Contains(got, "<!--") {
				t.Fatalf("the output contains an XML comment: %s", got)
			}
			golden := filepath.Join("testdata", name+".golden.xhtml")
			if *update {
				if err := os.WriteFile(golden, []byte(got+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if got+"\n" != string(want) {
				t.Fatalf("%s differs\n got: %s\nwant: %s", golden, got, want)
			}
		})
	}
}

// assertWellFormed parses the fragment inside a root that declares Confluence's ac:
// and ri: namespaces, as Confluence's own parser would see it.
func assertWellFormed(t *testing.T, fragment string) {
	t.Helper()
	document := `<root xmlns:ac="http://atlassian.com/content" xmlns:ri="http://atlassian.com/resource/identifier">` + fragment + `</root>`
	decoder := xml.NewDecoder(strings.NewReader(document))
	decoder.Strict = true
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("not well-formed XML: %v\n%s", err, fragment)
		}
	}
}

func TestRenderRejectsWhatItCannotRender(t *testing.T) {
	cases := map[string]Document{
		"heading level 7": {Blocks: []Block{Heading{Level: 7}}},
		"heading level 0": {Blocks: []Block{Heading{Level: 0}}},
		"nil block":       {Blocks: []Block{nil}},
		"nil inline":      {Blocks: []Block{Paragraph{Inlines: []Inline{nil}}}},
		"nil inline deep": {Blocks: []Block{Table{Rows: [][]TableCell{{{Inlines: []Inline{Strong{Inlines: []Inline{nil}}}}}}}}},
	}
	for name, doc := range cases {
		if _, err := Render(doc); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
