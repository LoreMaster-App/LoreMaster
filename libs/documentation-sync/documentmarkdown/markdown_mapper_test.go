package documentmarkdown_test

import (
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/documentmarkdown"
	"lore-master/libs/documentation-sync/platformport"
)

func text(s string) platformport.Text { return platformport.Text{Value: s} }

func render(t *testing.T, doc platformport.Document) (string, []string) {
	t.Helper()
	return documentmarkdown.ToMarkdown(doc, nil, nil)
}

func renderWithLinks(t *testing.T, doc platformport.Document, links documentmarkdown.Links) (string, []string) {
	t.Helper()
	return documentmarkdown.ToMarkdown(doc, links, nil)
}

func TestHeadingsAndParagraph(t *testing.T) {
	md, flags := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Heading{Level: 1, Inlines: []platformport.Inline{text("Title")}},
		platformport.Paragraph{Inlines: []platformport.Inline{text("A plain line.")}},
		platformport.Heading{Level: 3, Inlines: []platformport.Inline{text("Sub")}},
	}})
	want := "# Title\n\nA plain line.\n\n### Sub\n"
	if md != want {
		t.Fatalf("markdown mismatch:\n got %q\nwant %q", md, want)
	}
	if len(flags) != 0 {
		t.Fatalf("unexpected flags: %v", flags)
	}
}

func TestHeadingLevelClamped(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Heading{Level: 9, Inlines: []platformport.Inline{text("deep")}},
		platformport.Heading{Level: 0, Inlines: []platformport.Inline{text("shallow")}},
	}})
	if md != "###### deep\n\n# shallow\n" {
		t.Fatalf("heading level not clamped to 1..6: %q", md)
	}
}

func TestInlineMarks(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			text("a "),
			platformport.Emphasis{Inlines: []platformport.Inline{text("em")}},
			text(" "),
			platformport.Strong{Inlines: []platformport.Inline{text("bold")}},
			text(" "),
			platformport.Strikethrough{Inlines: []platformport.Inline{text("gone")}},
			text(" "),
			platformport.CodeSpan{Value: "x<y"},
		}},
	}})
	if md != "a *em* **bold** ~~gone~~ `x<y`\n" {
		t.Fatalf("inline marks mismatch: %q", md)
	}
}

func TestHardBreak(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			text("line one"),
			platformport.HardBreak{},
			text("line two"),
		}},
	}})
	if md != "line one  \nline two\n" {
		t.Fatalf("hard break mismatch: %q", md)
	}
}

func TestTextEscaping(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{text(`a * b [c] \ d <e>`)}},
	}})
	if md != `a \* b \[c\] \\ d \<e>`+"\n" {
		t.Fatalf("escaping mismatch: %q", md)
	}
}

func TestSnakeCaseNotEscaped(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{text("call do_the_thing now")}},
	}})
	if md != "call do_the_thing now\n" {
		t.Fatalf("underscores should not be escaped: %q", md)
	}
}

func TestBulletList(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.List{Items: []platformport.ListItem{
			{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{text("one")}}}},
			{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{text("two")}}}},
		}},
	}})
	if md != "- one\n- two\n" {
		t.Fatalf("bullet list mismatch: %q", md)
	}
}

func TestOrderedListWithStart(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.List{Ordered: true, Start: 2, Items: []platformport.ListItem{
			{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{text("two")}}}},
			{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{text("three")}}}},
		}},
	}})
	if md != "2. two\n3. three\n" {
		t.Fatalf("ordered list mismatch: %q", md)
	}
}

func TestNestedList(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.List{Items: []platformport.ListItem{
			{Blocks: []platformport.Block{
				platformport.Paragraph{Inlines: []platformport.Inline{text("outer")}},
				platformport.List{Items: []platformport.ListItem{
					{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{text("inner")}}}},
				}},
			}},
		}},
	}})
	want := "- outer\n\n  - inner\n"
	if md != want {
		t.Fatalf("nested list mismatch:\n got %q\nwant %q", md, want)
	}
}

func TestTaskList(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.TaskList{Items: []platformport.TaskItem{
			{Done: false, Inlines: []platformport.Inline{text("todo")}},
			{Done: true, Inlines: []platformport.Inline{text("done")}},
		}},
	}})
	if md != "- [ ] todo\n- [x] done\n" {
		t.Fatalf("task list mismatch: %q", md)
	}
}

func TestBlockquote(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Blockquote{Blocks: []platformport.Block{
			platformport.Paragraph{Inlines: []platformport.Inline{text("first")}},
			platformport.Paragraph{Inlines: []platformport.Inline{text("second")}},
		}},
	}})
	want := "> first\n>\n> second\n"
	if md != want {
		t.Fatalf("blockquote mismatch:\n got %q\nwant %q", md, want)
	}
}

func TestThematicBreak(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{text("above")}},
		platformport.ThematicBreak{},
		platformport.Paragraph{Inlines: []platformport.Inline{text("below")}},
	}})
	if md != "above\n\n---\n\nbelow\n" {
		t.Fatalf("thematic break mismatch: %q", md)
	}
}

func TestCodeBlock(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.CodeBlock{Language: "go", Code: "func main() {}\n"},
	}})
	if md != "```go\nfunc main() {}\n```\n" {
		t.Fatalf("code block mismatch: %q", md)
	}
}

func TestCodeBlockWithBacktickRun(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.CodeBlock{Language: "", Code: "a ``` b"},
	}})
	if md != "````\na ``` b\n````\n" {
		t.Fatalf("fence not widened past inner backticks: %q", md)
	}
}

func TestDiagramToMermaidFence(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Diagram{Source: "graph TD\nA-->B"},
	}})
	if md != "```mermaid\ngraph TD\nA-->B\n```\n" {
		t.Fatalf("diagram should become a mermaid fence: %q", md)
	}
}

func TestTable(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Table{
			Header: []platformport.TableCell{
				{Inlines: []platformport.Inline{text("Name")}},
				{Inlines: []platformport.Inline{text("Qty")}},
			},
			Align: []platformport.Alignment{platformport.AlignLeft, platformport.AlignRight},
			Rows: [][]platformport.TableCell{
				{
					{Inlines: []platformport.Inline{text("Widget")}},
					{Inlines: []platformport.Inline{text("3")}},
				},
			},
		},
	}})
	want := "| Name | Qty |\n| :--- | ---: |\n| Widget | 3 |\n"
	if md != want {
		t.Fatalf("table mismatch:\n got %q\nwant %q", md, want)
	}
}

func TestTableCellEscapesPipe(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Table{
			Header: []platformport.TableCell{{Inlines: []platformport.Inline{text("a|b")}}},
			Rows:   [][]platformport.TableCell{{{Inlines: []platformport.Inline{text("c|d")}}}},
		},
	}})
	want := "| a\\|b |\n| --- |\n| c\\|d |\n"
	if md != want {
		t.Fatalf("pipe not escaped once in cell:\n got %q\nwant %q", md, want)
	}
}

func TestExternalLink(t *testing.T) {
	md, flags := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Link{Target: &platformport.URLRef{URL: "https://example.com"}, Inlines: []platformport.Inline{text("site")}},
		}},
	}})
	if md != "[site](https://example.com)\n" {
		t.Fatalf("external link mismatch: %q", md)
	}
	if len(flags) != 0 {
		t.Fatalf("external link should not flag: %v", flags)
	}
}

func TestAttachmentLink(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Link{Target: &platformport.AttachmentRef{Filename: "spec.pdf"}, Inlines: []platformport.Inline{text("the spec")}},
		}},
	}})
	if md != "[the spec](spec.pdf)\n" {
		t.Fatalf("attachment link mismatch: %q", md)
	}
}

func TestPageLinkResolvedToLocalFile(t *testing.T) {
	links := func(title string) (string, bool) {
		if title == "ENG: Setup" {
			return "setup.md", true
		}
		return "", false
	}
	md, flags := renderWithLinks(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Link{Target: platformport.PageLink{Title: "ENG: Setup", Anchor: "install"}, Inlines: []platformport.Inline{text("setup")}},
		}},
	}}, links)
	if md != "[setup](setup.md#install)\n" {
		t.Fatalf("resolved page link mismatch: %q", md)
	}
	if len(flags) != 0 {
		t.Fatalf("resolved page link should not flag: %v", flags)
	}
}

func TestPageLinkFallsBackToURLAndFlags(t *testing.T) {
	md, flags := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Link{Target: platformport.PageLink{Title: "Other", URL: "https://wiki/x"}, Inlines: []platformport.Inline{text("other")}},
		}},
	}})
	if md != "[other](https://wiki/x)\n" {
		t.Fatalf("page link URL fallback mismatch: %q", md)
	}
	if len(flags) != 1 || !strings.Contains(flags[0], "Other") {
		t.Fatalf("expected one flag naming the title, got %v", flags)
	}
}

func TestPageLinkUnresolvableKeepsTextAndFlags(t *testing.T) {
	md, flags := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Link{Target: platformport.PageLink{Title: "Gone"}, Inlines: []platformport.Inline{text("gone")}},
		}},
	}})
	if md != "gone\n" {
		t.Fatalf("unresolvable page link should keep link text: %q", md)
	}
	if len(flags) != 1 {
		t.Fatalf("expected one flag, got %v", flags)
	}
}

func TestImageAttachmentWithTitle(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Image{Source: &platformport.AttachmentRef{Filename: "a.png"}, Alt: "a cat", Title: "Whiskers"},
		}},
	}})
	if md != `![a cat](a.png "Whiskers")`+"\n" {
		t.Fatalf("image mismatch: %q", md)
	}
}

func TestImageURLNoTitle(t *testing.T) {
	md, _ := render(t, platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Image{Source: &platformport.URLRef{URL: "https://img/x.png"}, Alt: "x"},
		}},
	}})
	if md != "![x](https://img/x.png)\n" {
		t.Fatalf("image URL mismatch: %q", md)
	}
}

func TestAttachmentResolvedToLocalPath(t *testing.T) {
	resolve := func(name string) string {
		if name == "flow.png" {
			return "assets/flow.png"
		}

		return ""
	}
	md, _ := documentmarkdown.ToMarkdown(platformport.Document{Blocks: []platformport.Block{
		platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Image{Source: &platformport.AttachmentRef{Filename: "flow.png"}, Alt: "d"},
			platformport.Link{Target: &platformport.AttachmentRef{Filename: "flow.png"}, Inlines: []platformport.Inline{text("see")}},
		}},
	}}, nil, resolve)
	if md != "![d](assets/flow.png)[see](assets/flow.png)\n" {
		t.Fatalf("attachment not resolved to its local path: %q", md)
	}
}

func TestEmptyDocument(t *testing.T) {
	md, flags := render(t, platformport.Document{})
	if md != "\n" {
		t.Fatalf("empty document should be a single newline, got %q", md)
	}
	if len(flags) != 0 {
		t.Fatalf("empty document should not flag: %v", flags)
	}
}
