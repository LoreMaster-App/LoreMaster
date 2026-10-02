package documenttree

import (
	"fmt"
	"path"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// NestingPolicy decides each document's parent among a fixed set of documents.
type NestingPolicy struct {
	documents documentdiscovery.Discovery
}

// NewNestingPolicy indexes the documents the rules may choose a parent from.
func NewNestingPolicy(documents []documentdiscovery.DocumentPath) NestingPolicy {
	return NestingPolicy{documents: documentdiscovery.NewDiscovery(documents, nil)}
}

// DecideParent applies the rules in order to one document. explicitParent is the
// annotation's "parent:" value, empty when there is none. A rule that points at a
// document that does not exist warns and gives way to the next rule; it never fails.
func (p NestingPolicy) DecideParent(document documentdiscovery.DocumentPath, explicitParent string) ParentDecision {
	var warnings []string
	if explicitParent != "" {
		parent, warning := p.explicitParent(document, explicitParent)
		if parent != nil {
			return ParentDecision{Parent: parent, Rule: RuleExplicitParent}
		}
		warnings = append(warnings, warning)
	}
	if parent, warning := p.dottedParent(document); parent != nil {
		if warning != "" {
			warnings = append(warnings, warning)
		}

		return ParentDecision{Parent: parent, Rule: RuleDottedName, Warnings: warnings}
	}

	return ParentDecision{Rule: RuleSelectedParent, Warnings: warnings}
}

// explicitParent resolves "parent:" against the document's directory ('/' meaning the
// workspace root), matching the target's case-insensitively.
func (p NestingPolicy) explicitParent(document documentdiscovery.DocumentPath, explicit string) (*documentdiscovery.DocumentPath, string) {
	var resolved string
	if strings.HasPrefix(explicit, "/") {
		resolved = path.Clean(strings.TrimLeft(explicit, "/"))
	} else {
		resolved = path.Join(path.Dir(string(document)), explicit)
	}
	found, exists := p.documents.Lookup(resolved)
	switch {
	case !exists:
		return nil, fmt.Sprintf("%s: parent %q (%s) is not a synced document; the next nesting rule applies", document, explicit, resolved)
	case found == document:
		return nil, fmt.Sprintf("%s: parent %q is the document itself; the next nesting rule applies", document, explicit)
	}

	return &found, ""
}

// dottedParent nests "a.b.c.md" under "a.b.md", in the same directory, matching case-
// insensitively so "readme.setup.md" finds "README.md". When the direct parent is
// missing, the nearest existing shorter prefix is used, with a warning; when no prefix
// exists at all, the rule does not apply.
func (p NestingPolicy) dottedParent(document documentdiscovery.DocumentPath) (*documentdiscovery.DocumentPath, string) {
	dir, file := path.Split(string(document))
	segments := strings.Split(strings.TrimSuffix(file, path.Ext(file)), ".")
	for keep := len(segments) - 1; keep >= 1; keep-- {
		candidate := dir + strings.Join(segments[:keep], ".") + ".md"
		found, exists := p.documents.Lookup(candidate)
		if !exists || found == document {
			continue
		}
		if keep == len(segments)-1 {
			return &found, ""
		}
		missing := dir + strings.Join(segments[:len(segments)-1], ".") + ".md"

		return &found, fmt.Sprintf("%s: %s does not exist, so it is nested under %s instead", document, missing, found)
	}

	return nil, ""
}
