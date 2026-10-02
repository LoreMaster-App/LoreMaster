package confluenceplatform

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/platformport"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

// everyNode uses every node type the port defines, so the golden output proves each
// one reaches the page.
var everyNode = platformport.Document{Blocks: []platformport.Block{
	platformport.Heading{Level: 1, Inlines: []platformport.Inline{platformport.Text{Value: "Title"}}},
	platformport.Paragraph{Inlines: []platformport.Inline{
		platformport.Text{Value: "a "}, platformport.Emphasis{Inlines: []platformport.Inline{platformport.Text{Value: "em"}}},
		platformport.Strong{Inlines: []platformport.Inline{platformport.Text{Value: "strong"}}},
		platformport.Strikethrough{Inlines: []platformport.Inline{platformport.Text{Value: "gone"}}},
		platformport.CodeSpan{Value: "x<y"}, platformport.HardBreak{},
		platformport.Link{Target: platformport.PageLink{Title: "ENG: Setup", Anchor: "install"}, Inlines: []platformport.Inline{platformport.Text{Value: "setup"}}},
		platformport.Link{Target: &platformport.AttachmentRef{Filename: "spec.pdf"}},
		platformport.Link{Target: &platformport.URLRef{URL: "https://example.com"}, Inlines: []platformport.Inline{platformport.Text{Value: "site"}}},
		platformport.Image{Source: &platformport.AttachmentRef{Filename: "a.png"}, Alt: "A", Width: 300},
		platformport.Image{Source: &platformport.URLRef{URL: "https://example.com/b.png"}},
	}},
	platformport.Blockquote{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Text{Value: "quote"}}}}},
	platformport.List{Ordered: true, Start: 2, Items: []platformport.ListItem{{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{platformport.Text{Value: "two"}}}}}}},
	platformport.TaskList{Items: []platformport.TaskItem{{Done: true, Inlines: []platformport.Inline{platformport.Text{Value: "done"}}}}},
	platformport.Table{
		Header: []platformport.TableCell{{Inlines: []platformport.Inline{platformport.Text{Value: "H"}}}},
		Rows:   [][]platformport.TableCell{{{Inlines: []platformport.Inline{platformport.Text{Value: "c"}}}}},
		Align:  []platformport.Alignment{platformport.AlignCenter},
	},
	platformport.ThematicBreak{},
	platformport.CodeBlock{Language: "go", Code: "package main"},
	platformport.Diagram{Language: "mermaid", Source: "graph TD; A-->B", Image: &platformport.AttachmentRef{Filename: "mermaid-1.svg"}},
	platformport.Diagram{Language: "plantuml", Source: "@startuml"},
}}

func TestEveryNodeReachesStorageFormat(t *testing.T) {
	mapped, err := toStorage(everyNode)
	if err != nil {
		t.Fatal(err)
	}
	got, err := storageformat.Render(mapped, storageformat.Options{})
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "every-node.golden.xhtml")
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
		t.Fatalf("differs from %s\n got: %s\nwant: %s", golden, got, want)
	}
}

func TestNilNodesAreErrors(t *testing.T) {
	for _, doc := range []platformport.Document{
		{Blocks: []platformport.Block{nil}},
		{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{nil}}}},
	} {
		if _, err := toStorage(doc); err == nil {
			t.Errorf("expected an error for %+v", doc)
		}
	}
}
