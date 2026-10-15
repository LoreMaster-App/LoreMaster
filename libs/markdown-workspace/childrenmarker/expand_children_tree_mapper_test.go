package childrenmarker

import (
	"strings"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
)

func node(path string, title string, body string, children ...*documenttree.TreeNode) *documenttree.TreeNode {
	return &documenttree.TreeNode{
		Document: &documentparsing.MarkdownDocument{Path: documentdiscovery.DocumentPath(path), Title: title, Body: []byte(body)},
		Children: children,
	}
}

func treeOf(nodes ...*documenttree.TreeNode) documenttree.DocumentTree {
	return documenttree.DocumentTree{Root: &documenttree.TreeNode{Children: nodes}}
}

func TestTheMarkerBecomesALinkedListOfTheChildren(t *testing.T) {
	deep := node("docs/api/deep.md", "Deep", "# Deep\n")
	api := node("docs/api.md", "API [v2]", "# API\n", deep)
	guide := node("docs/guide.md", "Guide", "# Guide\n")
	index := node("docs/index.md", "Docs", "# Docs\n\nPages:\n\n[children]\n\nEnd.\n", api, guide)

	expanded := ExpandTree(treeOf(index))

	body := string(expanded["docs/index.md"])
	want := "Pages:\n\n- [API \\[v2\\]](api.md)\n- [Guide](guide.md)\n\nEnd."
	if !strings.Contains(body, want) {
		t.Fatalf("body:\n%s", body)
	}
	if _, touched := expanded["docs/guide.md"]; touched {
		t.Fatal("a document without a marker was returned")
	}
}

func TestDepthAndAllReachFurtherDown(t *testing.T) {
	deeper := node("a/b/c/d.md", "D", "# D\n")
	c := node("a/b/c.md", "C", "# C\n", deeper)
	b := node("a/b.md", "B", "# B\n", c)
	top := node("a.md", "A", "[children depth=2]\n\n[children all]\n", b)

	body := string(ExpandTree(treeOf(top))["a.md"])

	lines := strings.Split(body, "\n")
	if lines[0] != "- [B](a/b.md)" || lines[1] != "  - [C](a/b/c.md)" || lines[2] != "" {
		t.Fatalf("depth=2:\n%s", body)
	}
	if !strings.Contains(body, "    - [D](a/b/c/d.md)") {
		t.Fatalf("all:\n%s", body)
	}
}

func TestLinksAreRelativeToTheChildrenLocation(t *testing.T) {
	child := node("other/place/child page.md", "Child", "# Child\n")
	parent := node("docs/readme.md", "Readme", "[children]\n", child)

	body := string(ExpandTree(treeOf(parent))["docs/readme.md"])

	if !strings.Contains(body, "[Child](../other/place/child%20page.md)") {
		t.Fatalf("body:\n%s", body)
	}
}

func TestAMarkerInCodeOrInTextIsLeftAlone(t *testing.T) {
	body := "Use `[children]` or:\n\n```md\n[children]\n```\n\n~~~\n[children]\n~~~\n\nSee [children] here.\n"
	page := node("a.md", "A", body, node("a.child.md", "Child", "# Child\n"))

	if _, touched := ExpandTree(treeOf(page))["a.md"]; touched {
		t.Fatal("a marker that is not a paragraph of its own was expanded")
	}
}

func TestAPageWithNoChildrenGetsAnEmptyList(t *testing.T) {
	page := node("a.md", "A", "# A\n\n[children]\n\nEnd\n")

	body := string(ExpandTree(treeOf(page))["a.md"])

	if strings.Contains(body, "[children]") || !strings.Contains(body, "End") {
		t.Fatalf("body:\n%s", body)
	}
}
