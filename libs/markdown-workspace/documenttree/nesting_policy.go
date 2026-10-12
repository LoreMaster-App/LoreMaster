package documenttree

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// NestingPolicy decides each document's parent among a fixed set of documents.
type NestingPolicy struct {
	documents documentdiscovery.Discovery
	// indexes maps a directory to its index document (README.md, else index.md).
	indexes map[string]documentdiscovery.DocumentPath
	// demoted maps an index.md that lost to a README.md in the same directory to it.
	demoted map[documentdiscovery.DocumentPath]documentdiscovery.DocumentPath
}

// NewNestingPolicy indexes the documents the rules may choose a parent from.
func NewNestingPolicy(documents []documentdiscovery.DocumentPath) NestingPolicy {
	policy := NestingPolicy{
		documents: documentdiscovery.NewDiscovery(documents, nil),
		indexes:   map[string]documentdiscovery.DocumentPath{},
		demoted:   map[documentdiscovery.DocumentPath]documentdiscovery.DocumentPath{},
	}
	sorted := slices.Clone(documents)
	slices.Sort(sorted)
	var indexMd []documentdiscovery.DocumentPath
	for _, document := range sorted {
		switch strings.ToLower(path.Base(string(document))) {
		case "readme.md":
			if _, taken := policy.indexes[path.Dir(string(document))]; !taken {
				policy.indexes[path.Dir(string(document))] = document
			}
		case "index.md":
			indexMd = append(indexMd, document)
		}
	}
	for _, document := range indexMd {
		dir := path.Dir(string(document))
		if readme, taken := policy.indexes[dir]; taken {
			policy.demoted[document] = readme
		} else {
			policy.indexes[dir] = document
		}
	}

	return policy
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
	if readme, demoted := p.demoted[document]; demoted {
		warnings = append(warnings, fmt.Sprintf("%s: %s is the page for this directory, so this index is nested under it", document, readme))
	}
	if parent := p.indexParent(document); parent != nil {
		return ParentDecision{Parent: parent, Rule: RuleDirectoryIndex, Warnings: warnings}
	}

	return ParentDecision{Rule: RuleSelectedParent, Warnings: warnings}
}

// indexParent is rule 3. A directory's index (README.md, else index.md, any case) is
// the parent of the directory's other documents and of its subdirectories' indexes; a
// directory without one hands its documents to the nearest ancestor that has one.
func (p NestingPolicy) indexParent(document documentdiscovery.DocumentPath) *documentdiscovery.DocumentPath {
	dir := path.Dir(string(document))
	if p.indexes[dir] == document {
		if dir == "." {
			return nil
		}
		dir = path.Dir(dir)
	}
	for {
		if index, exists := p.indexes[dir]; exists && index != document {
			return &index
		}
		if dir == "." {
			return nil
		}
		dir = path.Dir(dir)
	}
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

// dottedParent nests "a.b.c.md" under "a.b.md", matching case-insensitively so
// "readme.setup.md" finds "README.md". The longest shorter name wins; for each name the
// document's own directory is tried first, then its ancestors (nearest first) when the
// name has two or more segments, so a topic page and its children may sit in different
// folders. A single segment such as "README" matches the same directory only: that is
// the directory-index rule's job. When the direct parent (one segment shorter) is
// missing, the nearest existing shorter name is used, with a warning; when none exists
// at all, the rule does not apply.
func (p NestingPolicy) dottedParent(document documentdiscovery.DocumentPath) (*documentdiscovery.DocumentPath, string) {
	dir, file := path.Split(string(document))
	segments := strings.Split(strings.TrimSuffix(file, path.Ext(file)), ".")
	for keep := len(segments) - 1; keep >= 1; keep-- {
		name := strings.Join(segments[:keep], ".") + ".md"
		for _, where := range p.searchDirs(dir, keep) {
			found, exists := p.documents.Lookup(where + name)
			if !exists || found == document {
				continue
			}
			if keep == len(segments)-1 {
				return &found, ""
			}
			missing := dir + strings.Join(segments[:len(segments)-1], ".") + ".md"

			return &found, fmt.Sprintf("%s: %s does not exist, so it is nested under %s instead", document, missing, found)
		}
	}

	return nil, ""
}

// searchDirs lists, as prefixes ending in "/" (or empty for the root), the directories a
// dotted name of the given segment count is looked for in: the document's own, then its
// ancestors when the name has two or more segments.
func (p NestingPolicy) searchDirs(dir string, segments int) []string {
	dirs := []string{dir}
	if segments < 2 {
		return dirs
	}
	for current := strings.TrimSuffix(dir, "/"); current != ""; {
		current = path.Dir(current)
		if current == "." {
			dirs = append(dirs, "")

			break
		}
		dirs = append(dirs, current+"/")
	}

	return dirs
}
