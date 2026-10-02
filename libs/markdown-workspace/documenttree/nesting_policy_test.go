package documenttree

import (
	"reflect"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

func policyOver(paths ...string) NestingPolicy {
	documents := make([]documentdiscovery.DocumentPath, len(paths))
	for i, p := range paths {
		documents[i] = documentdiscovery.DocumentPath(p)
	}

	return NewNestingPolicy(documents)
}

type decisionRow struct {
	Parent   string
	Rule     Rule
	Warnings []string
}

func rowOf(decision ParentDecision) decisionRow {
	row := decisionRow{Rule: decision.Rule, Warnings: decision.Warnings}
	if decision.Parent != nil {
		row.Parent = string(*decision.Parent)
	}

	return row
}

func TestDecideParentExplicitAndDotted(t *testing.T) {
	policy := policyOver(
		"README.md", "readme.architecture.md", "readme.architecture.db.md", "readme.ops.runbook.alerts.md",
		"docs/Guide.md", "docs/guide.setup.md", "docs/other.md", "docs/sub/page.md",
		"orphan.child.md", "a.md", "x/a.b.md",
	)
	cases := []struct {
		name, document, explicit string
		want                     decisionRow
	}{
		{"explicit parent wins over a dotted name", "readme.architecture.db.md", "docs/other.md",
			decisionRow{Parent: "docs/other.md", Rule: RuleExplicitParent}},
		{"explicit parent relative to the document", "docs/sub/page.md", "../Guide.md",
			decisionRow{Parent: "docs/Guide.md", Rule: RuleExplicitParent}},
		{"explicit parent from the workspace root, any case", "docs/sub/page.md", "/readme.MD",
			decisionRow{Parent: "README.md", Rule: RuleExplicitParent}},
		{"missing explicit parent falls through to the dotted name", "readme.architecture.db.md", "./gone.md",
			decisionRow{Parent: "readme.architecture.md", Rule: RuleDottedName, Warnings: []string{
				`readme.architecture.db.md: parent "./gone.md" (gone.md) is not a synced document; the next nesting rule applies`,
			}}},
		{"explicit parent pointing at itself falls through to the directory index", "docs/other.md", "other.md",
			decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex, Warnings: []string{
				`docs/other.md: parent "other.md" is the document itself; the next nesting rule applies`,
			}}},
		{"dotted child of README.md, matched case-insensitively", "readme.architecture.md", "",
			decisionRow{Parent: "README.md", Rule: RuleDottedName}},
		{"dotted grandchild", "readme.architecture.db.md", "",
			decisionRow{Parent: "readme.architecture.md", Rule: RuleDottedName}},
		{"dotted, in a subdirectory, base in another case", "docs/guide.setup.md", "",
			decisionRow{Parent: "docs/Guide.md", Rule: RuleDottedName}},
		{"missing intermediates: nearest existing prefix, with a warning", "readme.ops.runbook.alerts.md", "",
			decisionRow{Parent: "README.md", Rule: RuleDottedName, Warnings: []string{
				"readme.ops.runbook.alerts.md: readme.ops.runbook.md does not exist, so it is nested under README.md instead",
			}}},
		{"dotted name whose base does not exist at all: rule 2 does not apply", "orphan.child.md", "",
			decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex}},
		{"dotted base looked up in the same directory only", "x/a.b.md", "",
			decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rowOf(policy.DecideParent(documentdiscovery.DocumentPath(tc.document), tc.explicit))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestDecideParentDirectoryIndexAndSelectedParent(t *testing.T) {
	policy := policyOver(
		"README.md", "top.md",
		"docs/index.md", "docs/a.md",
		"docs/api/Readme.md", "docs/api/INDEX.md", "docs/api/v1.md",
		"docs/api/deep/no-index-here.md",
		"docs/empty/sub/README.md",
		"guides/plain.md",
	)
	cases := []struct {
		name, document string
		want           decisionRow
	}{
		{"root index goes under the selected page", "README.md", decisionRow{Rule: RuleSelectedParent}},
		{"a root document goes under the root index", "top.md", decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex}},
		{"index.md is a directory's index", "docs/a.md", decisionRow{Parent: "docs/index.md", Rule: RuleDirectoryIndex}},
		{"a directory's index goes under the parent directory's index", "docs/index.md", decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex}},
		{"README wins over index, any case", "docs/api/v1.md", decisionRow{Parent: "docs/api/Readme.md", Rule: RuleDirectoryIndex}},
		{"the losing index nests under README, with a warning", "docs/api/INDEX.md", decisionRow{Parent: "docs/api/Readme.md", Rule: RuleDirectoryIndex, Warnings: []string{
			"docs/api/INDEX.md: docs/api/Readme.md is the page for this directory, so this index is nested under it",
		}}},
		{"a directory without an index uses the nearest ancestor's", "docs/api/deep/no-index-here.md", decisionRow{Parent: "docs/api/Readme.md", Rule: RuleDirectoryIndex}},
		{"an index skips index-less ancestors", "docs/empty/sub/README.md", decisionRow{Parent: "docs/index.md", Rule: RuleDirectoryIndex}},
		{"no index anywhere above: the root index", "guides/plain.md", decisionRow{Parent: "README.md", Rule: RuleDirectoryIndex}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rowOf(policy.DecideParent(documentdiscovery.DocumentPath(tc.document), ""))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}

	if got := rowOf(policyOver("a.md", "docs/b.md").DecideParent("docs/b.md", "")); !reflect.DeepEqual(got, decisionRow{Rule: RuleSelectedParent}) {
		t.Fatalf("without any index the selected page is the parent, got %+v", got)
	}
}
