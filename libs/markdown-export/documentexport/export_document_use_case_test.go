package documentexport

import (
	"encoding/json"
	"strings"
	"testing"
)

const sample = "---\nowner: team\n---\n\n# Guide\n\nSome **bold**, *soft*, ~~gone~~ and `code` with a [link](https://example.com/a) and a [sibling](other.md#part).\n\n## Steps\n\n- one\n- [x] done\n  - nested\n\n1. first\n2. second\n\n> quoted\n\n```go\nfmt.Println(\"hi\")\n```\n\n| Name | Count |\n|:-----|------:|\n| a | 1 |\n\n---\n"

func export(t *testing.T, format string) string {
	t.Helper()
	file, err := Export("docs/guide.md", []byte(sample), format, Options{})
	if err != nil {
		t.Fatalf("export %s: %v", format, err)
	}

	return string(file.Content)
}

func requireAll(t *testing.T, format, output string, wanted ...string) {
	t.Helper()
	for _, want := range wanted {
		if !strings.Contains(output, want) {
			t.Errorf("%s output lacks %q:\n%s", format, want, output)
		}
	}
	if strings.Contains(output, "owner: team") {
		t.Errorf("%s output leaks the front matter:\n%s", format, output)
	}
}

func TestPlainTextHasNoMarkdownSyntax(t *testing.T) {
	output := export(t, "txt")

	requireAll(t, "txt", output, "Guide\n=====", "Some bold, soft, gone and code with a link (https://example.com/a)", "- [x] done", "1. first", "fmt.Println(\"hi\")", "Name  Count")
	for _, syntax := range []string{"**", "~~", "```", "|:--"} {
		if strings.Contains(output, syntax) {
			t.Errorf("plain text keeps %q:\n%s", syntax, output)
		}
	}
}

func TestHTMLIsSelfContained(t *testing.T) {
	output := export(t, "html")

	requireAll(t, "html", output, "<title>Guide</title>", "<style>", `<h2 id="steps">Steps</h2>`, "<strong>bold</strong>", "<del>gone</del>",
		`<a href="other.html#part">sibling</a>`, `<input type="checkbox" disabled checked>`, `<code class="language-go">`, `<th style="text-align:left">Name</th>`, `<td style="text-align:right">1</td>`, "<hr>")
}

func TestJiraWiki(t *testing.T) {
	output := export(t, "jira-wiki")

	requireAll(t, "jira-wiki", output, "h1. Guide", "h2. Steps", "*bold*", "_soft_", "-gone-", "{{code}}", "[link|https://example.com/a]", "* one", "* (/) done", "** nested", "# first",
		"{quote}", "{code:go}", "||Name||Count||", "|a|1|", "----")
}

func TestJiraADFIsValidJSONWithTheExpectedNodes(t *testing.T) {
	output := export(t, "jira-adf")

	var parsed struct {
		Version int `json:"version"`
		Type    string
		Content []struct{ Type string }
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, output)
	}
	if parsed.Version != 1 || parsed.Type != "doc" {
		t.Fatalf("not an ADF document: %+v", parsed)
	}
	var types []string
	for _, node := range parsed.Content {
		types = append(types, node.Type)
	}
	if got := strings.Join(types, ","); got != "heading,paragraph,heading,bulletList,orderedList,blockquote,codeBlock,table,rule" {
		t.Errorf("top-level nodes = %s", got)
	}
	requireAll(t, "jira-adf", output, `"type": "strong"`, `"type": "link"`, `"href": "https://example.com/a"`, `"language": "go"`, `"type": "tableHeader"`)
}

func TestMediaWiki(t *testing.T) {
	output := export(t, "mediawiki")

	requireAll(t, "mediawiki", output, "= Guide =", "== Steps ==", "'''bold'''", "''soft''", "<s>gone</s>", "<code>code</code>", "[https://example.com/a link]", "[[other#part|sibling]]",
		"* one", "** nested", "# first", "<blockquote>", `<syntaxhighlight lang="go">`, `{| class="wikitable"`, "! style=\"text-align:left\" | Name", "|}", "----")
}

func TestAsciiDoc(t *testing.T) {
	output := export(t, "asciidoc")

	requireAll(t, "asciidoc", output, "= Guide", "== Steps", "**bold**", "__soft__", "[.line-through]#gone#", "`+code+`", "https://example.com/a[link]", "xref:other.adoc#part[sibling]",
		"* one", "* [x] done", "** nested", ". first", "[quote]\n____", "[source,go]\n----", `[cols="<,>",options="header"]`, "|===", "'''")
}

func TestReStructuredText(t *testing.T) {
	output := export(t, "rst")

	requireAll(t, "rst", output, "Guide\n=====", "Steps\n-----", "**bold**", "*soft*", "``code``", "`link <https://example.com/a>`__", ":doc:`sibling <other>`",
		"- one", "- [x] done", "#. first", ".. code-block:: go", ".. list-table::", ":header-rows: 1", "   * - Name", "     - Count", "----")
}

func TestSlack(t *testing.T) {
	output := export(t, "slack")

	requireAll(t, "slack", output, "*Guide*", "*Steps*", "*bold*", "_soft_", "~gone~", "`code`", "<https://example.com/a|link>", "• one", "• ☑ done", "1. first", "> quoted", "```\nfmt.Println", "Name | Count", "-----")
}

func TestOrg(t *testing.T) {
	output := export(t, "org")

	requireAll(t, "org", output, "* Guide", "** Steps", "*bold*", "/soft/", "+gone+", "~code~", "[[https://example.com/a][link]]", "- one", "- [X] done", "1. first",
		"#+begin_quote", "#+begin_src go", "| Name | Count |", "-----")
}

func TestLaTeX(t *testing.T) {
	output := export(t, "latex")

	requireAll(t, "latex", output, "\\documentclass{article}", "\\begin{document}", "\\section{Guide}", "\\subsection{Steps}", "\\textbf{bold}", "\\emph{soft}", "\\sout{gone}", "\\texttt{code}",
		"\\href{https://example.com/a}{link}", "\\begin{itemize}", "$\\boxtimes$ done", "\\begin{enumerate}", "\\begin{quote}", "\\begin{lstlisting}", "\\begin{tabular}{lr}", "\\end{document}")
}

func TestJSONOutline(t *testing.T) {
	output := export(t, "outline")

	requireAll(t, "outline", output, `"title": "Guide"`, `"text": "Steps"`, `"level": 2`, `"url": "https://example.com/a"`, `"url": "other.md#part"`, `"text": "done"`, `"done": true`)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("outline is not JSON: %v", err)
	}
}

func TestManPage(t *testing.T) {
	output := export(t, "man")

	requireAll(t, "man", output, `.TH "GUIDE" "7"`, `.SH "STEPS"`, ".PP", `\fBbold\fR`, `\fIsoft\fR`, `link (\fIhttps://example.com/a\fR)`, `.IP \(bu 2`, ".IP 1. 4", ".nf", ".TS", "allbox;", ".TE")
}

func TestRichTextFormat(t *testing.T) {
	output := export(t, "rtf")

	requireAll(t, "rtf", output, `{\rtf1\ansi`, `\b\fs40 Guide`, `{\b bold}`, `{\i soft}`, `{\strike gone}`, `{\f1 code}`, `HYPERLINK "https://example.com/a"`, `\bullet\tab one`, `1.\tab first`, `\trowd`, `\cell`, `\row`)
	if strings.Count(output, "{") != strings.Count(output, "}")+strings.Count(output, `\{`)-strings.Count(output, `\}`) {
		t.Errorf("rtf braces do not balance:\n%s", output)
	}
}

func TestMDX(t *testing.T) {
	output := export(t, "mdx")

	requireAll(t, "mdx", output, "---\ntitle: \"Guide\"\n---", "# Guide", "## Steps", "**bold**", "*soft*", "~~gone~~", "`code`", "[link](https://example.com/a)", "- one", "- [x] done", "1. first", "> quoted", "```go", "| Name | Count |", "| :-- | --: |")
}

func TestUnknownFormatIsRefused(t *testing.T) {
	if _, err := Export("a.md", []byte("# A\n"), "pdf-from-the-future", Options{}); err == nil {
		t.Fatal("an unknown format was accepted")
	}
}

func TestResultIsNamedAfterTheDocument(t *testing.T) {
	file, err := Export("docs/getting-started.md", []byte("# A\n"), "jira-adf", Options{})
	if err != nil || file.Name != "getting-started.adf.json" {
		t.Fatalf("name = %q, err = %v", file.Name, err)
	}
}

func TestPreviewKeepsLinksToMarkdownFilesAsTheyAre(t *testing.T) {
	source := []byte("# A\n\n[other](other.md#part)\n")

	exported, err := Export("a.md", source, "html", Options{})
	if err != nil || !strings.Contains(string(exported.Content), `href="other.html#part"`) {
		t.Fatalf("export: %v %s", err, exported.Content)
	}
	preview, err := Export("a.md", source, "html", Options{Preview: true})
	if err != nil || !strings.Contains(string(preview.Content), `href="other.md#part"`) {
		t.Fatalf("preview: %v %s", err, preview.Content)
	}
}
