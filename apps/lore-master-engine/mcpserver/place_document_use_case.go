package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/syncannotation"
)

const placeDocumentToolName = "place_document"

const placeDocumentToolDescription = "Suggests where to create a new page: given its H1 and an optional parent (a page title or file path), returns the file path and any annotation that nests it there under LoreMaster's rules. Writes nothing."

// placementView is the suggested placement for a new page.
type placementView struct {
	Path           string   `json:"path"`
	Annotation     string   `json:"annotation,omitempty"` // lore-master comment block to put at the top, or empty
	Mechanism      string   `json:"mechanism"`            // the nesting rule the suggestion relies on
	Parent         string   `json:"parent,omitempty"`     // resolved parent file, empty for a top-level page
	Title          string   `json:"title"`
	PublishedTitle string   `json:"publishedTitle"`
	Warnings       []string `json:"warnings,omitempty"`
}

func placeDocumentTool() toolDescriptor {
	return toolDescriptor{
		Name:        placeDocumentToolName,
		Description: placeDocumentToolDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"h1": map[string]any{
					"type":        "string",
					"description": "The new page's H1 / title.",
				},
				"parent": map[string]any{
					"type":        "string",
					"description": "Optional parent to nest under: an existing page's title or its workspace-relative file path. Omit for a top-level page.",
				},
				"directory": map[string]any{
					"type":        "string",
					"description": "Optional workspace-relative directory for the new file. Defaults to the parent's directory, else the workspace root.",
				},
			},
			"required": []any{"h1"},
		},
	}
}

func placeDocumentResult(ctx context.Context, workspaceRoot string, arguments json.RawMessage) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}
	var args struct {
		H1        string `json:"h1"`
		Parent    string `json:"parent"`
		Directory string `json:"directory"`
	}
	if len(arguments) > 0 {
		_ = json.Unmarshal(arguments, &args)
	}
	h1 := strings.TrimSpace(args.H1)
	if h1 == "" {
		return errorResult("place_document needs the new page's \"h1\".")
	}
	slug := slugify(h1)
	if slug == "" {
		return errorResult("could not derive a file name from the H1 %q; give an H1 with letters or digits.", h1)
	}

	documents, _, err := loadWorkspaceDocuments(ctx, workspaceRoot)
	if err != nil {
		return errorResult("could not read the workspace: %v", err)
	}

	var warnings []string
	var parentDoc *documentparsing.MarkdownDocument
	if strings.TrimSpace(args.Parent) != "" {
		parentDoc, err = resolveParent(workspaceRoot, args.Parent, documents, &warnings)
		if err != nil {
			return errorResult("%v", err)
		}
	}

	targetDir, ok := targetDirectory(workspaceRoot, args.Directory, parentDoc)
	if !ok {
		return errorResult("the directory %q is outside the workspace.", args.Directory)
	}

	candidate, explicitParent := proposePlacement(parentDoc, targetDir, slug)

	// Verify against the real tree builder, falling back to an explicit parent: annotation
	// when the idiomatic placement would not resolve to the requested parent.
	rule, resolvedParent := verifyPlacement(documents, candidate, explicitParent)
	if parentDoc != nil && resolvedParent != string(parentDoc.Path) {
		explicitParent = "/" + string(parentDoc.Path)
		candidate = joinDir(targetDir, slug+".md")
		rule, resolvedParent = verifyPlacement(documents, candidate, explicitParent)
		if resolvedParent != string(parentDoc.Path) {
			warnings = append(warnings, fmt.Sprintf("could not find a placement that nests under %s; the tree builder resolved the parent to %q", parentDoc.Path, resolvedParent))
		}
	}

	if existing := documentAt(documents, candidate); existing {
		warnings = append(warnings, fmt.Sprintf("a document already exists at %s; choose a different H1 or directory", candidate))
	}

	prefix := firstTitlePrefix(workspaceRoot)
	published := h1
	if prefix != "" {
		published = prefix + ": " + h1
	}
	if clash := titleClashAmong(documents, candidate, h1); clash != "" {
		warnings = append(warnings, clash)
	}

	view := placementView{
		Path:           candidate,
		Annotation:     annotationFor(explicitParent),
		Mechanism:      rule,
		Parent:         resolvedParent,
		Title:          h1,
		PublishedTitle: published,
		Warnings:       warnings,
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: renderPlacement(view)}},
		StructuredContent: view,
	}
}

// proposePlacement picks the idiomatic file path and, when it cannot use a dotted name,
// the explicit parent path to annotate.
func proposePlacement(parentDoc *documentparsing.MarkdownDocument, targetDir string, slug string) (candidate string, explicitParent string) {
	if parentDoc == nil {
		return joinDir(targetDir, slug+".md"), ""
	}
	parentDir := path.Dir(string(parentDoc.Path))
	if targetDir == parentDir {
		stem := strings.TrimSuffix(path.Base(string(parentDoc.Path)), path.Ext(string(parentDoc.Path)))

		return joinDir(targetDir, strings.ToLower(stem)+"."+slug+".md"), ""
	}

	return joinDir(targetDir, slug+".md"), "/" + string(parentDoc.Path)
}

// verifyPlacement builds the tree with the proposed file added and reports the nesting rule
// and parent it actually resolves to, so the suggestion is checked against the real policy.
func verifyPlacement(documents []documentparsing.MarkdownDocument, candidate string, explicitParent string) (rule string, parent string) {
	candidatePath := documentdiscovery.DocumentPath(candidate)
	docs := make([]documentparsing.MarkdownDocument, 0, len(documents)+1)
	for i := range documents {
		if documents[i].Path != candidatePath {
			docs = append(docs, documents[i])
		}
	}
	synthetic := documentparsing.MarkdownDocument{Path: candidatePath, Title: candidate, TitleFromHeading: true}
	if explicitParent != "" {
		synthetic.Annotation = &syncannotation.Annotation{Parent: explicitParent}
	}
	docs = append(docs, synthetic)

	tree, err := documenttree.BuildTree(docs)
	if err != nil {
		return string(documenttree.RuleSelectedParent), ""
	}
	var found *documenttree.TreeNode
	tree.Walk(func(node *documenttree.TreeNode, _ int) {
		if node.Document.Path == candidatePath {
			found = node
		}
	})
	if found == nil {
		return string(documenttree.RuleSelectedParent), ""
	}
	if found.Decision.Parent != nil {
		return string(found.Decision.Rule), string(*found.Decision.Parent)
	}

	return string(found.Decision.Rule), ""
}

// resolveParent finds the parent document by workspace-relative path first, then by title
// (case-insensitive). An unknown parent is an error; an ambiguous title warns and takes
// the first.
func resolveParent(root, parent string, documents []documentparsing.MarkdownDocument, warnings *[]string) (*documentparsing.MarkdownDocument, error) {
	if rel, _, ok := workspaceRelPath(root, parent); ok {
		for i := range documents {
			if documents[i].Path == rel {
				return &documents[i], nil
			}
		}
	}
	var matches []*documentparsing.MarkdownDocument
	for i := range documents {
		if strings.EqualFold(strings.TrimSpace(documents[i].Title), strings.TrimSpace(parent)) {
			matches = append(matches, &documents[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no synced page matches parent %q (by path or title); run %s to see the pages", parent, previewTreeToolName)
	case 1:
		return matches[0], nil
	default:
		paths := make([]string, len(matches))
		for i, match := range matches {
			paths[i] = string(match.Path)
		}
		*warnings = append(*warnings, fmt.Sprintf("parent %q matches several pages (%s); using the first", parent, strings.Join(paths, ", ")))

		return matches[0], nil
	}
}

// targetDirectory resolves the directory for the new file, returning false when it escapes
// the workspace.
func targetDirectory(root, directory string, parentDoc *documentparsing.MarkdownDocument) (string, bool) {
	if strings.TrimSpace(directory) != "" {
		rel, _, ok := workspaceRelPath(root, directory)
		if !ok {
			return "", false
		}

		return path.Clean(string(rel)), true
	}
	if parentDoc != nil {
		return path.Dir(string(parentDoc.Path)), true
	}

	return ".", true
}

func documentAt(documents []documentparsing.MarkdownDocument, candidate string) bool {
	target := documentdiscovery.DocumentPath(candidate)
	for i := range documents {
		if documents[i].Path == target {
			return true
		}
	}

	return false
}

// titleClashAmong reports a case-insensitive title clash between the new page and an
// existing one.
func titleClashAmong(documents []documentparsing.MarkdownDocument, candidate string, title string) string {
	folded := strings.ToLower(strings.TrimSpace(title))
	var others []string
	for i := range documents {
		if string(documents[i].Path) == candidate {
			continue
		}
		if strings.ToLower(strings.TrimSpace(documents[i].Title)) == folded {
			others = append(others, string(documents[i].Path))
		}
	}
	if len(others) == 0 {
		return ""
	}

	return fmt.Sprintf("the title %q is already used by %s; give this page a different H1 or a title: annotation", title, strings.Join(others, ", "))
}

func firstTitlePrefix(root string) string {
	loaded, err := workspacesettings.LoadSettings(root)
	if err != nil {
		return ""
	}
	for _, output := range loaded.Settings.Outputs {
		if prefix := strings.TrimSpace(output.TitlePrefix); prefix != "" {
			return prefix
		}
	}

	return ""
}

// annotationFor renders the lore-master comment block that sets an explicit parent, or
// empty when no annotation is needed.
func annotationFor(explicitParent string) string {
	if explicitParent == "" {
		return ""
	}

	return syncannotation.OpeningLine + "\n" + syncannotation.KeyParent + ": " + explicitParent + "\n" + syncannotation.ClosingLine
}

func renderPlacement(view placementView) string {
	var text strings.Builder
	fmt.Fprintf(&text, "Create: %s\n", view.Path)
	fmt.Fprintf(&text, "- H1: # %s\n", view.Title)
	fmt.Fprintf(&text, "- published as: %s\n", view.PublishedTitle)
	parent := view.Parent
	if parent == "" {
		parent = "(the selected parent page)"
	}
	fmt.Fprintf(&text, "- nests under: %s  [rule: %s]\n", parent, view.Mechanism)
	if view.Annotation != "" {
		fmt.Fprintf(&text, "- put this at the top of the file:\n%s\n", view.Annotation)
	} else {
		text.WriteString("- no annotation needed\n")
	}
	if len(view.Warnings) > 0 {
		text.WriteString("\nWarnings:")
		for _, warning := range view.Warnings {
			fmt.Fprintf(&text, "\n- %s", warning)
		}
	}

	return strings.TrimRight(text.String(), "\n")
}

// joinDir joins a workspace-relative directory and a file name, keeping the root case
// ("." + name -> name).
func joinDir(dir, name string) string {
	if dir == "." || dir == "" {
		return name
	}

	return path.Join(dir, name)
}

// slugify turns a title into a kebab-case file-name segment: letters and digits kept and
// lower-cased, every other run collapsed to a single hyphen, ends trimmed.
func slugify(title string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			prevHyphen = false

			continue
		}
		if b.Len() > 0 && !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}

	return strings.Trim(b.String(), "-")
}
