package documenttree

import (
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

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

// TreeNode is one page in the tree. The root stands for the page the user selected and
// has no Document.
type TreeNode struct {
	Document *documentparsing.MarkdownDocument
	Decision ParentDecision
	Children []*TreeNode
}

// Title is the page title before the prefix: the annotation's "title:" override when
// set, otherwise the document's own title (first H1, else the file name).
func (n *TreeNode) Title() string {
	if n.Document == nil {
		return ""
	}
	if n.Document.Annotation != nil {
		if override := strings.TrimSpace(n.Document.Annotation.Title); override != "" {
			return override
		}
	}

	return n.Document.Title
}

// DocumentTree is every document placed under the selected page.
type DocumentTree struct {
	Root *TreeNode
	// Warnings are the nesting decisions' warnings, in path order.
	Warnings []string
}

// Walk visits every document node parents-first (depth 0 is directly under the
// selected page), which is the order pages must be created in.
func (t DocumentTree) Walk(visit func(node *TreeNode, depth int)) {
	var walk func(nodes []*TreeNode, depth int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, node := range nodes {
			visit(node, depth)
			walk(node.Children, depth+1)
		}
	}
	walk(t.Root.Children, 0)
}
