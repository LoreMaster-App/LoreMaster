package storageformat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseRendersBackToTheGolden is the round trip that matters: every golden is real Render
// output, so parsing it and rendering again must reproduce it byte for byte. It proves the
// parser is the faithful inverse of Render for every construct, including the ones that are
// ambiguous on their own (a URL link, a code block with no language) because both halves agree
// on the same storage shape.
func TestParseRendersBackToTheGolden(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.golden.xhtml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no golden files found: %v", err)
	}
	for _, file := range files {
		file := file
		t.Run(filepath.Base(file), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			// Normalise line endings: the XML parser folds CRLF to LF (as the spec requires)
			// and Render emits LF, so compare against an LF golden regardless of how it was
			// checked out on this platform.
			want := strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
			doc, flags, err := Parse(want)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(flags) != 0 {
				t.Fatalf("a golden should parse without flags, got %v", flags)
			}
			got, err := Render(doc, Options{})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got != want {
				t.Fatalf("round trip differs:\n got %q\nwant %q", got, want)
			}
		})
	}
}

func TestParseParagraphInlines(t *testing.T) {
	doc, flags, err := Parse(`<p>a <em>em</em> <strong>b <em>c</em></strong> <span style="text-decoration: line-through;">d</span> <code>e &lt; f</code><br />g</p>`)
	if err != nil || len(flags) != 0 {
		t.Fatalf("err=%v flags=%v", err, flags)
	}
	para, ok := doc.Blocks[0].(Paragraph)
	if !ok {
		t.Fatalf("want a paragraph, got %T", doc.Blocks[0])
	}
	want := []Inline{
		Text{Value: "a "},
		Emphasis{Inlines: []Inline{Text{Value: "em"}}},
		Text{Value: " "},
		Strong{Inlines: []Inline{Text{Value: "b "}, Emphasis{Inlines: []Inline{Text{Value: "c"}}}}},
		Text{Value: " "},
		Strikethrough{Inlines: []Inline{Text{Value: "d"}}},
		Text{Value: " "},
		CodeSpan{Value: "e < f"},
		HardBreak{},
		Text{Value: "g"},
	}
	assertInlines(t, want, para.Inlines)
}

func TestParsePageAttachmentAndURLLinks(t *testing.T) {
	doc, _, err := Parse(`<p><ac:link ac:anchor="sec"><ri:page ri:content-title="ENG: Setup" /><ac:link-body>setup</ac:link-body></ac:link><ac:link><ri:attachment ri:filename="r.pdf" /><ac:link-body>the report</ac:link-body></ac:link><a href="https://x.test/?a=1&amp;b=2">x</a></p>`)
	if err != nil {
		t.Fatal(err)
	}
	inlines := doc.Blocks[0].(Paragraph).Inlines
	page, ok := inlines[0].(Link)
	if !ok || page.Target != (PageLink{Title: "ENG: Setup", Anchor: "sec"}) {
		t.Fatalf("page link mismatch: %#v", inlines[0])
	}
	att := inlines[1].(Link).Target.(*AttachmentRef)
	if att.Filename != "r.pdf" {
		t.Fatalf("attachment link mismatch: %#v", att)
	}
	url := inlines[2].(Link).Target.(*URLRef)
	if url.URL != "https://x.test/?a=1&b=2" {
		t.Fatalf("url link should be entity-decoded, got %q", url.URL)
	}
}

func TestParseImage(t *testing.T) {
	doc, _, err := Parse(`<p><ac:image ac:alt="a cat" ac:title="T" ac:width="400"><ri:attachment ri:filename="c.png" /></ac:image><ac:image><ri:url ri:value="https://x.test/a.png" /></ac:image></p>`)
	if err != nil {
		t.Fatal(err)
	}
	inlines := doc.Blocks[0].(Paragraph).Inlines
	att := inlines[0].(Image)
	if att.Alt != "a cat" || att.Title != "T" || att.Width != 400 || att.Source.(*AttachmentRef).Filename != "c.png" {
		t.Fatalf("attachment image mismatch: %#v", att)
	}
	if inlines[1].(Image).Source.(*URLRef).URL != "https://x.test/a.png" {
		t.Fatalf("url image mismatch: %#v", inlines[1])
	}
}

func TestParseCodeBlockReassemblesCDATA(t *testing.T) {
	doc, _, err := Parse(`<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[a ]]]]><![CDATA[> b]]></ac:plain-text-body></ac:structured-macro>`)
	if err != nil {
		t.Fatal(err)
	}
	code, ok := doc.Blocks[0].(CodeBlock)
	if !ok || code.Language != "go" || code.Code != "a ]]> b" {
		t.Fatalf("code block mismatch: %#v", doc.Blocks[0])
	}
}

func TestParseCollapsedMacroBecomesMermaidWithItsImage(t *testing.T) {
	doc, flags, err := Parse(`<p><ac:image ac:alt="Mermaid diagram"><ri:attachment ri:filename="mermaid-1.svg" /></ac:image></p><ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="collapse">true</ac:parameter><ac:plain-text-body><![CDATA[graph TD
]]></ac:plain-text-body></ac:structured-macro>`)
	if err != nil || len(flags) != 0 {
		t.Fatalf("err=%v flags=%v", err, flags)
	}
	if len(doc.Blocks) != 1 {
		t.Fatalf("the image paragraph should have folded into the diagram, got %d blocks", len(doc.Blocks))
	}
	diagram, ok := doc.Blocks[0].(Mermaid)
	if !ok || diagram.Source != "graph TD\n" {
		t.Fatalf("want a Mermaid diagram, got %#v", doc.Blocks[0])
	}
	if diagram.Image == nil || diagram.Image.Filename != "mermaid-1.svg" {
		t.Fatalf("the diagram should keep its rendered picture, got %#v", diagram.Image)
	}
}

func TestParseMarkedCodeMacroIsMermaidWithoutCollapse(t *testing.T) {
	// The explicit marker identifies a diagram even when it is not collapsed — a code-mode
	// diagram, or a page where a Confluence-side edit dropped the collapse flag.
	doc, flags, err := Parse(`<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="lore-master">mermaid</ac:parameter><ac:plain-text-body><![CDATA[graph TD]]></ac:plain-text-body></ac:structured-macro>`)
	if err != nil || len(flags) != 0 {
		t.Fatalf("err=%v flags=%v", err, flags)
	}
	if diagram, ok := doc.Blocks[0].(Mermaid); !ok || diagram.Source != "graph TD" {
		t.Fatalf("a marked code macro should be a Mermaid diagram, got %#v", doc.Blocks[0])
	}
}

func TestParseTightAndLooseListItems(t *testing.T) {
	doc, _, err := Parse(`<ul><li>tight</li><li><p>loose</p><p>second</p></li></ul>`)
	if err != nil {
		t.Fatal(err)
	}
	list := doc.Blocks[0].(List)
	if len(list.Items[0].Blocks) != 1 {
		t.Fatalf("a tight item should hold one paragraph, got %d", len(list.Items[0].Blocks))
	}
	if len(list.Items[1].Blocks) != 2 {
		t.Fatalf("a loose item should hold two paragraphs, got %d", len(list.Items[1].Blocks))
	}
}

func TestParseConfluenceTableWithParagraphWrappedCells(t *testing.T) {
	// Confluence's editor wraps each cell's content in <p>; that must not flag or skip the pull.
	doc, flags, err := Parse(`<table><tbody><tr><th><p>Name</p></th><th><p>Qty</p></th></tr><tr><td><p>Widget</p></td><td><p>3</p></td></tr></tbody></table>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(flags) != 0 {
		t.Fatalf("a <p>-wrapped cell should not be flagged, got %v", flags)
	}
	table, ok := doc.Blocks[0].(Table)
	if !ok || len(table.Header) != 2 || len(table.Rows) != 1 {
		t.Fatalf("table mismatch: %#v", doc.Blocks[0])
	}
	if cell := table.Rows[0][0].Inlines; len(cell) != 1 || cell[0] != (Text{Value: "Widget"}) {
		t.Fatalf("cell content not unwrapped through <p>: %#v", table.Rows[0][0])
	}
}

func TestParseFlagsUnknownMacro(t *testing.T) {
	doc, flags, err := Parse(`<ac:structured-macro ac:name="info"><ac:rich-text-body><p>hi</p></ac:rich-text-body></ac:structured-macro>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Blocks) != 0 {
		t.Fatalf("an unknown macro should be dropped, got %d blocks", len(doc.Blocks))
	}
	if len(flags) != 1 || !strings.Contains(flags[0], "info") {
		t.Fatalf("expected a flag naming the unknown macro, got %v", flags)
	}
}

func assertInlines(t *testing.T, want, got []Inline) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("inline count: want %d got %d (%#v)", len(want), len(got), got)
	}
	for i := range want {
		if !inlineEqual(want[i], got[i]) {
			t.Fatalf("inline %d: want %#v got %#v", i, want[i], got[i])
		}
	}
}

func inlineEqual(a, b Inline) bool {
	switch x := a.(type) {
	case Text:
		y, ok := b.(Text)

		return ok && x.Value == y.Value
	case CodeSpan:
		y, ok := b.(CodeSpan)

		return ok && x.Value == y.Value
	case HardBreak:
		_, ok := b.(HardBreak)

		return ok
	case Emphasis:
		y, ok := b.(Emphasis)

		return ok && inlinesEqual(x.Inlines, y.Inlines)
	case Strong:
		y, ok := b.(Strong)

		return ok && inlinesEqual(x.Inlines, y.Inlines)
	case Strikethrough:
		y, ok := b.(Strikethrough)

		return ok && inlinesEqual(x.Inlines, y.Inlines)
	}

	return false
}

func inlinesEqual(a, b []Inline) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !inlineEqual(a[i], b[i]) {
			return false
		}
	}

	return true
}
