package wikirender

import (
	"strings"
	"testing"

	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
)

func tree(t *testing.T, files map[string]string) documenttree.DocumentTree {
	t.Helper()
	var documents []documentparsing.MarkdownDocument
	for name, body := range files {
		document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(name), []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, document)
	}
	built, err := documenttree.BuildTree(documents)
	if err != nil {
		t.Fatal(err)
	}

	return built
}

func contents(files []siterender.SiteFile) map[string]string {
	byPath := map[string]string{}
	for _, file := range files {
		byPath[file.Path] = string(file.Content)
	}

	return byPath
}

func TestGenerateWikiNamesPagesAfterTitlesAndMakesTheReadmeTheHome(t *testing.T) {
	got := contents(GenerateWiki(tree(t, map[string]string{
		"README.md":          "# Project\n\nSee [the guide](docs/user-guide.md#install) and ![shot](docs/img/a.png).\n",
		"docs/user-guide.md": "# User guide\n\nBack to [home](../README.md).\n\n```\n[not](a-link.md)\n```\n",
	})))

	if !strings.Contains(got["Home.md"], "[the guide](User-guide#install)") || !strings.Contains(got["Home.md"], "![shot](docs/img/a.png)") {
		t.Errorf("home links not rewritten:\n%s", got["Home.md"])
	}
	if !strings.Contains(got["User-guide.md"], "[home](Home)") || !strings.Contains(got["User-guide.md"], "[not](a-link.md)") {
		t.Errorf("guide links wrong (fenced code must stay):\n%s", got["User-guide.md"])
	}
	if !strings.Contains(got["_Sidebar.md"], "- [Project](Home)") {
		t.Errorf("sidebar missing the home:\n%s", got["_Sidebar.md"])
	}
}

func TestGenerateWikiKeepsPageNamesUnique(t *testing.T) {
	got := contents(GenerateWiki(tree(t, map[string]string{
		"README.md":  "# Project\n",
		"a/setup.md": "# Setup\n",
		"b/setup.md": "# Setup\n",
	})))

	if _, ok := got["Setup.md"]; !ok {
		t.Fatalf("missing Setup.md in %v", got)
	}
	if _, ok := got["Setup-2.md"]; !ok {
		t.Fatalf("a clashing title should get a numbered name: %v", got)
	}
}

func TestPageNameReadsAsAnAddress(t *testing.T) {
	cases := map[string]string{
		"ADR: One Go engine, thin editor shells": "ADR-One-Go-engine-thin-editor-shells",
		"Generators: pages from your project's":  "Generators-pages-from-your-projects",
		"  ":                                     "Page",
	}
	for title, want := range cases {
		if got := pageName(title); got != want {
			t.Errorf("pageName(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestGenerateWikiTurnsALinkToALeftOutDocumentIntoPlainText(t *testing.T) {
	got := contents(GenerateWiki(tree(t, map[string]string{
		"README.md": "# Project\n\nSee [the plan](internal/plan.md#risks), [the guide](guide.md) and ![a](internal/plan.md).\n",
		"guide.md":  "# Guide\n",
	})))

	home := got["Home.md"]
	if !strings.Contains(home, "See the plan, [the guide](Guide) and ![a](internal/plan.md).") {
		t.Errorf("a left-out document link should be plain text and the rest kept:\n%s", home)
	}
}

func TestLeftOutLinksNamesEachLinkToADocumentTheOutputDoesNotPublish(t *testing.T) {
	warnings := LeftOutLinks(tree(t, map[string]string{
		"README.md": "# Project\n\n[plan](internal/plan.md), [guide](guide.md), ![shot](internal/a.png)\n\n```\n[x](other.md)\n```\n",
		"guide.md":  "# Guide\n\nBack: [home](README.md) and [adr](docs/adr.md#top).\n",
	}))

	want := []string{
		"README.md: links to internal/plan.md, which this output leaves out",
		"guide.md: links to docs/adr.md#top, which this output leaves out",
	}
	if len(warnings) != len(want) || warnings[0] != want[0] || warnings[1] != want[1] {
		t.Fatalf("warnings %q, want %q", warnings, want)
	}
}
