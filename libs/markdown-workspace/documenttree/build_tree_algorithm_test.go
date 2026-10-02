package documenttree

import (
	"fmt"
	"strings"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

func parsed(t *testing.T, files map[string]string) []documentparsing.MarkdownDocument {
	t.Helper()
	var documents []documentparsing.MarkdownDocument
	for documentPath, content := range files {
		document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(documentPath), []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, document)
	}

	return documents
}

func outline(tree DocumentTree) string {
	var lines []string
	tree.Walk(func(node *TreeNode, depth int) {
		lines = append(lines, fmt.Sprintf("%s%s  %q [%s]", strings.Repeat("  ", depth), node.Document.Path, node.Title(), node.Decision.Rule))
	})

	return strings.Join(lines, "\n")
}

func TestBuildTreeMixesEveryRule(t *testing.T) {
	documents := parsed(t, map[string]string{
		"README.md":                 "# Home\n",
		"readme.architecture.md":    "# Architecture\n",
		"readme.architecture.db.md": "# Database\n",
		"02-usage.md":               "# Usage\n",
		"01-install.md":             "# Install\n",
		"zeta.md":                   "# Alpha\n",
		"changelog.md":              "<!-- lore-master\ntitle: Release notes\n-->\n# Changelog\n",
		"docs/index.md":             "# Docs\n",
		"docs/guide.md":             "# Guide\n",
		"docs/faq.md":               "<!-- lore-master\nparent: ../readme.architecture.md\n-->\n# FAQ\n",
		"docs/broken.md":            "<!-- lore-master\nparent: ./missing.md\n-->\n# Broken parent\n",
	})
	tree, err := BuildTree(documents)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`README.md  "Home" [selected-parent]`,
		`  01-install.md  "Install" [directory-index]`,
		`  02-usage.md  "Usage" [directory-index]`,
		`  zeta.md  "Alpha" [directory-index]`,
		`  readme.architecture.md  "Architecture" [dotted-name]`,
		`    readme.architecture.db.md  "Database" [dotted-name]`,
		`    docs/faq.md  "FAQ" [explicit-parent]`,
		`  docs/index.md  "Docs" [directory-index]`,
		`    docs/broken.md  "Broken parent" [directory-index]`,
		`    docs/guide.md  "Guide" [directory-index]`,
		`  changelog.md  "Release notes" [directory-index]`,
	}, "\n")
	if got := outline(tree); got != want {
		t.Fatalf("tree\n got:\n%s\nwant:\n%s", got, want)
	}
	if want := `docs/broken.md: parent "./missing.md" (docs/missing.md) is not a synced document; the next nesting rule applies`; len(tree.Warnings) != 1 || tree.Warnings[0] != want {
		t.Fatalf("warnings %q", tree.Warnings)
	}
}

func TestBuildTreeRejectsParentCycles(t *testing.T) {
	documents := parsed(t, map[string]string{
		"a.md":        "<!-- lore-master\nparent: b.md\n-->\n# A\n",
		"b.md":        "<!-- lore-master\nparent: a.md\n-->\n# B\n",
		"c.md":        "<!-- lore-master\nparent: a.md\n-->\n# Leads into the cycle\n",
		"x/README.md": "<!-- lore-master\nparent: y/deep.md\n-->\n# X\n",
		"x/y/deep.md": "# Deep, under x/README.md by rule 3\n",
		"fine.md":     "# Fine\n",
	})
	_, err := BuildTree(documents)
	want := "the parent: annotations form a cycle: a.md → b.md → a.md; remove one of them\n" +
		"the parent: annotations form a cycle: x/README.md → x/y/deep.md → x/README.md; remove one of them"
	if err == nil || err.Error() != want {
		t.Fatalf("error\n got: %v\nwant: %s", err, want)
	}
}

func TestBuildTreeRejectsADuplicatePath(t *testing.T) {
	documents := parsed(t, map[string]string{"a.md": "# A\n"})
	documents = append(documents, documents[0])
	if _, err := BuildTree(documents); err == nil || err.Error() != "a.md is listed twice" {
		t.Fatalf("error %v", err)
	}
}
