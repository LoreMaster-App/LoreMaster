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
