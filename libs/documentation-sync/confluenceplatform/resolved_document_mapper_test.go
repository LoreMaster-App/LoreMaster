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

// TestPullReconstructsTheSameStorage is the pull's correctness bar: rendering everyNode,
// parsing it back and re-mapping through fromStorage must reproduce the identical storage. It
// proves Parse and fromStorage together invert Render and toStorage for every node type —
// including the intentionally lossy ones (a non-mermaid diagram, a link with no text), which
// are stable because they re-render the same way.
func TestPullReconstructsTheSameStorage(t *testing.T) {
	mapped, err := toStorage(everyNode)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := storageformat.Render(mapped, storageformat.Options{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, flags, err := storageformat.Parse(rendered)
	if err != nil {
		t.Fatal(err)
	}
	if len(flags) != 0 {
		t.Fatalf("every-node storage should parse without flags, got %v", flags)
	}
	remapped, err := toStorage(fromStorage(parsed))
	if err != nil {
		t.Fatal(err)
	}
	again, err := storageformat.Render(remapped, storageformat.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if again != rendered {
		t.Fatalf("pull did not reconstruct the same storage:\n got %q\nwant %q", again, rendered)
	}
}

// TestFromStorageMapsDiagramAndCode pins the node types a pull restores, catching a mapping that
// would re-render the same but mean something different (a diagram read back as a code block).
func TestFromStorageMapsDiagramAndCode(t *testing.T) {
	parsed, _, err := storageformat.Parse(
		`<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="collapse">true</ac:parameter><ac:plain-text-body><![CDATA[graph TD]]></ac:plain-text-body></ac:structured-macro>` +
			`<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[x]]></ac:plain-text-body></ac:structured-macro>`)
	if err != nil {
		t.Fatal(err)
	}
	doc := fromStorage(parsed)
	if diagram, ok := doc.Blocks[0].(platformport.Diagram); !ok || diagram.Language != "mermaid" || diagram.Source != "graph TD" {
		t.Fatalf("want a mermaid diagram, got %#v", doc.Blocks[0])
	}
	if code, ok := doc.Blocks[1].(platformport.CodeBlock); !ok || code.Language != "go" || code.Code != "x" {
		t.Fatalf("want a go code block, got %#v", doc.Blocks[1])
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
