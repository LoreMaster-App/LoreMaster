package documenttree

import (
	"strings"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// The nesting conventions must stay aligned with the Rule constants the policy uses: the
// first four entries are those rules, in precedence order. A new or renamed rule that is
// not reflected here is the drift this test exists to catch.
func TestNestingConventionsCoverTheRulesInOrder(t *testing.T) {
	conventions := NestingConventions()
	wantNesting := []Rule{RuleExplicitParent, RuleDottedName, RuleDirectoryIndex, RuleSelectedParent}
	if len(conventions) < len(wantNesting) {
		t.Fatalf("expected at least %d conventions, got %d", len(wantNesting), len(conventions))
	}
	for i, rule := range wantNesting {
		if conventions[i].Key != string(rule) {
			t.Errorf("convention %d: key %q, want %q", i, conventions[i].Key, rule)
		}
		if conventions[i].Summary == "" {
			t.Errorf("convention %q has no summary", conventions[i].Key)
		}
	}
	// The title rules follow and must be present.
	for _, key := range []string{"title", "title-prefix", "title-unique"} {
		found := false
		for _, c := range conventions {
			if c.Key == key {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("missing convention %q", key)
		}
	}
}

// An author following the conventions must end up with the nesting they asked for: the rules
// name the annotation block, and the code reads parent: and title: from it, not from YAML
// front matter (which is skipped).
func TestTheRulesNameTheAnnotationAndTheTreeBuilderAgrees(t *testing.T) {
	for _, convention := range NestingConventions() {
		if convention.Key != string(RuleExplicitParent) && convention.Key != "title" {
			continue
		}
		if !strings.Contains(convention.Summary, "<!-- lore-master") || !strings.Contains(convention.Summary, "not YAML front matter") {
			t.Errorf("%s does not say the annotation, not front matter, carries it: %s", convention.Key, convention.Summary)
		}
	}

	parse := func(path string, content string) documentparsing.MarkdownDocument {
		document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(path), []byte(content))
		if err != nil {
			t.Fatal(err)
		}

		return document
	}
	tree, err := BuildTree([]documentparsing.MarkdownDocument{
		parse("parent.md", "# Parent\n"),
		parse("a/by-annotation.md", "<!-- lore-master\nparent: /parent.md\ntitle: Renamed\n-->\n# Original\n"),
		parse("a/by-front-matter.md", "---\nparent: /parent.md\ntitle: Ignored\n---\n# Heading\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	rules := map[string]Rule{}
	titles := map[string]string{}
	tree.Walk(func(node *TreeNode, _ int) {
		rules[string(node.Document.Path)] = node.Decision.Rule
		titles[string(node.Document.Path)] = node.Title()
	})
	if rules["a/by-annotation.md"] != RuleExplicitParent || titles["a/by-annotation.md"] != "Renamed" {
		t.Fatalf("the annotation was not honoured: rule %s, title %q", rules["a/by-annotation.md"], titles["a/by-annotation.md"])
	}
	if rules["a/by-front-matter.md"] == RuleExplicitParent || titles["a/by-front-matter.md"] != "Heading" {
		t.Fatalf("front matter was honoured: rule %s, title %q", rules["a/by-front-matter.md"], titles["a/by-front-matter.md"])
	}
}
