package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documenttree"
)

const validateDocumentToolName = "validate_document"

const validateDocumentToolDescription = "Checks one Markdown file against LoreMaster's rules: where it will nest and why, whether its title clashes with another page, whether it is missing an H1, and whether the sync includes it at all."

// validationView is the outcome of validating one document.
type validationView struct {
	Path     string   `json:"path"`
	Title    string   `json:"title,omitempty"`
	Parent   string   `json:"parent,omitempty"` // empty means directly under the selected parent page
	Rule     string   `json:"rule,omitempty"`
	Synced   bool     `json:"synced"` // whether the sync's discovery includes this file
	Warnings []string `json:"warnings,omitempty"`
}

func validateDocumentTool() toolDescriptor {
	return toolDescriptor{
		Name:        validateDocumentToolName,
		Description: validateDocumentToolDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to the Markdown file to validate, e.g. docs/architecture/engine.md.",
				},
			},
			"required": []any{"path"},
		},
	}
}

func validateDocumentResult(ctx context.Context, workspaceRoot string, arguments json.RawMessage) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}
	var args struct {
		Path string `json:"path"`
	}
	if len(arguments) > 0 {
		_ = json.Unmarshal(arguments, &args)
	}
	if strings.TrimSpace(args.Path) == "" {
		return errorResult("validate_document needs a \"path\" to a Markdown file.")
	}

	rel, full, ok := workspaceRelPath(workspaceRoot, args.Path)
	if !ok {
		return errorResult("%q is outside the workspace.", args.Path)
	}
	if info, statErr := os.Stat(full); statErr != nil || info.IsDir() {
		return errorResult("no file at %s. To see where a new page would go, use %s.", rel, previewTreeToolName)
	}

	documents, _, err := loadWorkspaceDocuments(ctx, workspaceRoot)
	if err != nil {
		return errorResult("could not read the workspace: %v", err)
	}

	synced := false
	hasTarget := false
	for i := range documents {
		if documents[i].Path == rel {
			synced = true
			hasTarget = true

			break
		}
	}
	if !hasTarget {
		// Excluded from the sync, but still worth showing where it would nest.
		target, parseErr := readAndParse(workspaceRoot, rel)
		if parseErr != nil {
			return errorResult("could not parse %s: %v", rel, parseErr)
		}
		documents = append(documents, target)
	}

	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		return errorResult("could not build the page tree: %v", err)
	}

	var target *documenttree.TreeNode
	foldedTitles := map[string][]documentdiscovery.DocumentPath{}
	tree.Walk(func(node *documenttree.TreeNode, _ int) {
		foldedTitles[strings.ToLower(strings.TrimSpace(node.Title()))] = append(
			foldedTitles[strings.ToLower(strings.TrimSpace(node.Title()))], node.Document.Path)
		if node.Document.Path == rel {
			target = node
		}
	})
	if target == nil {
		return errorResult("could not place %s in the tree.", rel)
	}

	view := validationView{Path: string(rel), Title: target.Title(), Rule: string(target.Decision.Rule), Synced: synced}
	if target.Decision.Parent != nil {
		view.Parent = string(*target.Decision.Parent)
	}
	view.Warnings = append(view.Warnings, target.Decision.Warnings...)
	if !target.Document.TitleFromHeading {
		view.Warnings = append(view.Warnings, fmt.Sprintf("no H1 heading, so the file name's title %q is used; add a `# Heading` to control the page title", target.Title()))
	}
	if clash := titleClash(rel, target.Title(), foldedTitles); clash != "" {
		view.Warnings = append(view.Warnings, clash)
	}
	if !synced {
		view.Warnings = append(view.Warnings, "this file is not matched by any output's roots/excludes in .lore-master.yaml, so the sync will not include it")
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: renderValidation(view)}},
		StructuredContent: view,
	}
}

// titleClash reports, when another document would get the same (case-insensitive) title,
// a message naming it; Confluence titles are unique per space. Empty when there is no
// clash.
func titleClash(path documentdiscovery.DocumentPath, title string, foldedTitles map[string][]documentdiscovery.DocumentPath) string {
	var others []string
	for _, other := range foldedTitles[strings.ToLower(strings.TrimSpace(title))] {
		if other != path {
			others = append(others, string(other))
		}
	}
	if len(others) == 0 {
		return ""
	}

	return fmt.Sprintf("the title %q is also used by %s; Confluence allows a title only once per space, so give one a `title:` annotation or a different H1", title, strings.Join(others, ", "))
}

func renderValidation(view validationView) string {
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n", view.Path)
	fmt.Fprintf(&text, "- title: %s\n", view.Title)
	parent := view.Parent
	if parent == "" {
		parent = "(the selected parent page)"
	}
	fmt.Fprintf(&text, "- nests under: %s  [rule: %s]\n", parent, view.Rule)
	fmt.Fprintf(&text, "- included by the sync: %t", view.Synced)
	if len(view.Warnings) == 0 {
		text.WriteString("\n\nNo problems found.")

		return text.String()
	}
	text.WriteString("\n\nWarnings:")
	for _, warning := range view.Warnings {
		fmt.Fprintf(&text, "\n- %s", warning)
	}

	return text.String()
}

// workspaceRelPath normalises an input path to a workspace-relative DocumentPath and its
// OS path, returning false when it would escape the workspace root.
func workspaceRelPath(root string, input string) (documentdiscovery.DocumentPath, string, bool) {
	full, ok := resolveInWorkspace(root, input)
	if !ok {
		return "", "", false
	}
	relative, err := filepath.Rel(root, full)
	if err != nil {
		return "", "", false
	}

	return documentdiscovery.DocumentPath(filepath.ToSlash(relative)), full, true
}
