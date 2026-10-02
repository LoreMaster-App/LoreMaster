package documenttree

import "lore-master/libs/markdown-workspace/documentdiscovery"

// Rule names the nesting rule that placed a document.
type Rule string

// The nesting rules, in order of precedence.
const (
	RuleExplicitParent Rule = "explicit-parent"
	RuleDottedName     Rule = "dotted-name"
	RuleDirectoryIndex Rule = "directory-index"
	RuleSelectedParent Rule = "selected-parent"
)

// ParentDecision is where one document goes: under Parent, or, when Parent is nil,
// directly under the page the user selected.
type ParentDecision struct {
	Parent   *documentdiscovery.DocumentPath
	Rule     Rule
	Warnings []string
}
