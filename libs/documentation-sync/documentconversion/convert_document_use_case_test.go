package documentconversion

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

func parse(t *testing.T, path documentdiscovery.DocumentPath, content string) documentparsing.MarkdownDocument {
	t.Helper()
	document, err := documentparsing.ParseDocument(path, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	return document
}

func TestConvertDocumentGolden(t *testing.T) {
	source, err := os.ReadFile("testdata/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	guide := parse(t, "docs/guide.md", string(source))
	documents := []documentparsing.MarkdownDocument{
		guide,
		parse(t, "README.md", "# Home\n\n## Install\n"),
		parse(t, "docs/api.md", "# API\n"),
	}
	workspace := NewWorkspace(documents, map[documentdiscovery.DocumentPath]string{
		"docs/guide.md": "ENG: Guide", "README.md": "ENG: Home", "docs/api.md": "ENG: API",
	})

	converted := ConvertDocument(guide, workspace)
	var got strings.Builder
	dumpBlocks(&got, converted.Document.Blocks, "")
	fmt.Fprintf(&got, "\nlinked pages: %v\n", converted.LinkedPages)
	got.WriteString("\nattachments:\n")
	for _, attachment := range converted.Attachments {
		fmt.Fprintf(&got, "  %s <- %s\n", attachment.Filename, attachment.Path)
	}
	got.WriteString("\nwarnings:\n")
	for _, warning := range converted.Warnings {
		fmt.Fprintf(&got, "  %s\n", warning)
	}

	golden := "testdata/guide.golden.txt"
	if *update {
		if err := os.WriteFile(golden, []byte(got.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != string(want) {
		t.Fatalf("conversion differs from %s (go test -update to accept):\n%s", golden, got.String())
	}
}

func TestConversionIsDeterministic(t *testing.T) {
	source, _ := os.ReadFile("testdata/guide.md")
	guide := parse(t, "docs/guide.md", string(source))
	workspace := NewWorkspace([]documentparsing.MarkdownDocument{guide}, nil)
	var first, second strings.Builder
	dumpBlocks(&first, ConvertDocument(guide, workspace).Document.Blocks, "")
	dumpBlocks(&second, ConvertDocument(guide, workspace).Document.Blocks, "")
	if first.String() != second.String() {
		t.Fatal("the same document converted twice differs")
	}
}

func TestAFileWithoutAnH1KeepsEveryHeading(t *testing.T) {
	document := parse(t, "notes.md", "## Only an H2\n\ntext\n")
	converted := ConvertDocument(document, NewWorkspace([]documentparsing.MarkdownDocument{document}, nil))
	if len(converted.Document.Blocks) != 2 {
		t.Fatalf("blocks %#v", converted.Document.Blocks)
	}
}

func TestAnImageAndALinkedFileSharingABaseNameDoNotCollide(t *testing.T) {
	document := parse(t, "doc.md", "![x](sub/diagram.png)\n\n[d](other/diagram.png)\n")
	converted := ConvertDocument(document, NewWorkspace([]documentparsing.MarkdownDocument{document}, nil))

	names := map[string]string{}
	for _, attachment := range converted.Attachments {
		names[string(attachment.Path)] = attachment.Filename
	}
	if len(converted.Attachments) != 2 {
		t.Fatalf("want the image and the linked file both attached, got %#v", converted.Attachments)
	}
	if names["sub/diagram.png"] == names["other/diagram.png"] {
		t.Fatalf("the image and the linked file share a name: %v", names)
	}
}

func dumpBlocks(out *strings.Builder, blocks []platformport.Block, indent string) {
	for _, block := range blocks {
		switch b := block.(type) {
		case platformport.Paragraph:
			fmt.Fprintf(out, "%sparagraph %s\n", indent, inlineText(b.Inlines))
		case platformport.Heading:
			fmt.Fprintf(out, "%sh%d %s\n", indent, b.Level, inlineText(b.Inlines))
		case platformport.Blockquote:
			fmt.Fprintf(out, "%squote\n", indent)
			dumpBlocks(out, b.Blocks, indent+"  ")
		case platformport.List:
			fmt.Fprintf(out, "%slist ordered=%v start=%d\n", indent, b.Ordered, b.Start)
			for _, item := range b.Items {
				fmt.Fprintf(out, "%s  item\n", indent)
				dumpBlocks(out, item.Blocks, indent+"    ")
			}
		case platformport.TaskList:
			fmt.Fprintf(out, "%stasks\n", indent)
			for _, item := range b.Items {
				fmt.Fprintf(out, "%s  done=%v %s\n", indent, item.Done, inlineText(item.Inlines))
			}
		case platformport.Table:
			fmt.Fprintf(out, "%stable align=%v\n", indent, b.Align)
			fmt.Fprintf(out, "%s  header %s\n", indent, cellsText(b.Header))
			for _, row := range b.Rows {
				fmt.Fprintf(out, "%s  row %s\n", indent, cellsText(row))
			}
		case platformport.ThematicBreak:
			fmt.Fprintf(out, "%srule\n", indent)
		case platformport.CodeBlock:
			fmt.Fprintf(out, "%scode %q %q\n", indent, b.Language, b.Code)
		case platformport.Diagram:
			fmt.Fprintf(out, "%sdiagram %q %q image=%v\n", indent, b.Language, b.Source, b.Image)
		default:
			fmt.Fprintf(out, "%s%T\n", indent, b)
		}
	}
}

func cellsText(cells []platformport.TableCell) string {
	var parts []string
	for _, cell := range cells {
		parts = append(parts, inlineText(cell.Inlines))
	}

	return strings.Join(parts, " | ")
}

func inlineText(inlines []platformport.Inline) string {
	var out strings.Builder
	for _, inline := range inlines {
		switch i := inline.(type) {
		case platformport.Text:
			fmt.Fprintf(&out, "%q", i.Value)
		case platformport.Emphasis:
			fmt.Fprintf(&out, "em(%s)", inlineText(i.Inlines))
		case platformport.Strong:
			fmt.Fprintf(&out, "strong(%s)", inlineText(i.Inlines))
		case platformport.Strikethrough:
			fmt.Fprintf(&out, "del(%s)", inlineText(i.Inlines))
		case platformport.CodeSpan:
			fmt.Fprintf(&out, "code(%q)", i.Value)
		case platformport.HardBreak:
			out.WriteString("<br>")
		case platformport.Image:
			fmt.Fprintf(&out, "img(%s alt=%q title=%q width=%d)", describeTarget(i.Source), i.Alt, i.Title, i.Width)
		case platformport.Link:
			fmt.Fprintf(&out, "link(%s %s)", describeTarget(i.Target), inlineText(i.Inlines))
		default:
			fmt.Fprintf(&out, "%T", i)
		}
	}

	return out.String()
}

func describeTarget(value any) string {
	switch t := value.(type) {
	case platformport.PageLink:
		return fmt.Sprintf("page=%q anchor=%q", t.Title, t.Anchor)
	case *platformport.AttachmentRef:
		return "attachment=" + t.Filename
	case *platformport.URLRef:
		return "url=" + t.URL
	}

	return fmt.Sprintf("%T", value)
}
