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

func TestDecideParentRulesOneAndTwo(t *testing.T) {
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
		{"explicit parent pointing at itself falls through", "docs/other.md", "other.md",
			decisionRow{Rule: RuleSelectedParent, Warnings: []string{
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
		{"dotted name whose base does not exist at all: rule does not apply", "orphan.child.md", "",
			decisionRow{Rule: RuleSelectedParent}},
		{"dotted base looked up in the same directory only", "x/a.b.md", "",
			decisionRow{Rule: RuleSelectedParent}},
		{"plain name", "docs/other.md", "",
			decisionRow{Rule: RuleSelectedParent}},
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
