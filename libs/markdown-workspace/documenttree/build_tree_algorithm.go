package documenttree

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// BuildTree places every document by the nesting rules and returns the tree. Siblings
// sort by their numeric file-name prefix (prefixed ones first), then by title, then by
// path. Only "parent:" annotations can form a cycle (the other rules always point to a
// shorter name or a higher directory); a cycle is an error naming every document in it,
// since no creation order could satisfy it.
func BuildTree(documents []documentparsing.MarkdownDocument) (DocumentTree, error) {
	nodes := make(map[documentdiscovery.DocumentPath]*TreeNode, len(documents))
	paths := make([]documentdiscovery.DocumentPath, 0, len(documents))
	for i := range documents {
		document := &documents[i]
		if _, duplicate := nodes[document.Path]; duplicate {
			return DocumentTree{}, fmt.Errorf("%s is listed twice", document.Path)
		}
		nodes[document.Path] = &TreeNode{Document: document}
		paths = append(paths, document.Path)
	}
	slices.Sort(paths)

	policy := NewNestingPolicy(paths)
	tree := DocumentTree{Root: &TreeNode{}}
	for _, documentPath := range paths {
		node := nodes[documentPath]
		var explicit string
		if node.Document.Annotation != nil {
			explicit = node.Document.Annotation.Parent
		}
		node.Decision = policy.DecideParent(documentPath, explicit)
		tree.Warnings = append(tree.Warnings, node.Decision.Warnings...)
	}
	if err := parentCycles(paths, nodes); err != nil {
		return DocumentTree{}, err
	}

	for _, documentPath := range paths {
		node := nodes[documentPath]
		parent := tree.Root
		if node.Decision.Parent != nil {
			parent = nodes[*node.Decision.Parent]
		}
		parent.Children = append(parent.Children, node)
	}
	sortChildren(tree.Root)

	return tree, nil
}

func parentCycles(paths []documentdiscovery.DocumentPath, nodes map[documentdiscovery.DocumentPath]*TreeNode) error {
	const (
		unvisited = iota
		onPath
		settled
	)
	state := make(map[documentdiscovery.DocumentPath]int, len(paths))
	var cycles []error
	for _, start := range paths {
		var trail []documentdiscovery.DocumentPath
		current := &start
		for current != nil && state[*current] == unvisited {
			state[*current] = onPath
			trail = append(trail, *current)
			current = nodes[*current].Decision.Parent
		}
		if current != nil && state[*current] == onPath {
			cycle := trail[slices.Index(trail, *current):]
			names := make([]string, 0, len(cycle)+1)
			for _, member := range cycle {
				names = append(names, string(member))
			}
			names = append(names, string(*current))
			cycles = append(cycles, fmt.Errorf("the parent: annotations form a cycle: %s; remove one of them", strings.Join(names, " → ")))
		}
		for _, visited := range trail {
			state[visited] = settled
		}
	}

	return errors.Join(cycles...)
}

func sortChildren(node *TreeNode) {
	slices.SortFunc(node.Children, func(a, b *TreeNode) int {
		if a.Document.Ordered != b.Document.Ordered {
			if a.Document.Ordered {
				return -1
			}

			return 1
		}

		return cmp.Or(
			cmp.Compare(a.Document.Order, b.Document.Order),
			cmp.Compare(strings.ToLower(a.Title()), strings.ToLower(b.Title())),
			cmp.Compare(a.Document.Path, b.Document.Path),
		)
	})
	for _, child := range node.Children {
		sortChildren(child)
	}
}
