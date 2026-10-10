package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
)

const previewTreeToolName = "preview_tree"

const previewTreeToolDescription = "Shows the current workspace's documentation pages as the tree the sync would build: each page's file, title, parent and the nesting rule that placed it."

// treeNodeView is one page in the previewed tree.
type treeNodeView struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Parent   string   `json:"parent,omitempty"` // empty means directly under the selected parent page
	Rule     string   `json:"rule"`
	Depth    int      `json:"depth"`
	Warnings []string `json:"warnings,omitempty"`
}

type previewTreeView struct {
	Nodes    []treeNodeView `json:"nodes"`
	Warnings []string       `json:"warnings,omitempty"`
}

func previewTreeTool() toolDescriptor {
	return toolDescriptor{
		Name:        previewTreeToolName,
		Description: previewTreeToolDescription,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func previewTreeResult(ctx context.Context, workspaceRoot string) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}
	documents, notes, err := loadWorkspaceDocuments(ctx, workspaceRoot)
	if err != nil {
		return errorResult("could not read the workspace: %v", err)
	}
	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		return errorResult("could not build the page tree: %v", err)
	}

	view := previewTreeView{Warnings: append(notes, tree.Warnings...)}
	var text strings.Builder
	text.WriteString("Documentation tree the sync would build (indentation is nesting):\n")
	tree.Walk(func(node *documenttree.TreeNode, depth int) {
		parent := ""
		if node.Decision.Parent != nil {
			parent = string(*node.Decision.Parent)
		}
		view.Nodes = append(view.Nodes, treeNodeView{
			Path:     string(node.Document.Path),
			Title:    node.Title(),
			Parent:   parent,
			Rule:     string(node.Decision.Rule),
			Depth:    depth,
			Warnings: node.Decision.Warnings,
		})
		fmt.Fprintf(&text, "%s- %s  (%s)\n", strings.Repeat("  ", depth), node.Title(), node.Document.Path)
	})
	if len(view.Nodes) == 0 {
		text.WriteString("(no Markdown documents found)\n")
	}
	for _, warning := range view.Warnings {
		fmt.Fprintf(&text, "\n! %s", warning)
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: strings.TrimRight(text.String(), "\n")}},
		StructuredContent: view,
	}
}

// loadWorkspaceDocuments discovers and parses every Markdown document the outputs would
// sync — each "markdown" content entry's roots and excludes, merged without repeats —
// defaulting to the whole workspace when there is no settings file. Notes are discovery
// and parse warnings.
func loadWorkspaceDocuments(ctx context.Context, root string) ([]documentparsing.MarkdownDocument, []string, error) {
	loaded, err := workspacesettings.LoadSettings(root)
	if err != nil {
		return nil, nil, err
	}

	scope := loaded.Settings.DiscoveryScope()
	var paths []documentdiscovery.DocumentPath
	var notes []string
	for _, output := range loaded.Settings.Outputs {
		for _, content := range output.Content {
			if content.Type != "markdown" {
				continue
			}
			excludes, includes := scope.ScanFor(output, content)
			found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root, Roots: content.Roots, Excludes: excludes, Includes: includes, IncludeGitignored: !scope.SkipGitignored})
			if err != nil {
				return nil, nil, err
			}
			notes = append(notes, found.Warnings...)
			for _, path := range found.Documents {
				if !slices.Contains(paths, path) {
					paths = append(paths, path)
				}
			}
		}
	}

	documents := make([]documentparsing.MarkdownDocument, 0, len(paths))
	for _, path := range paths {
		document, parseErr := readAndParse(root, path)
		if parseErr != nil {
			notes = append(notes, fmt.Sprintf("%s: %v", path, parseErr))

			continue
		}
		for _, warning := range document.Warnings {
			notes = append(notes, fmt.Sprintf("%s: %s", path, warning))
		}
		documents = append(documents, document)
	}

	return documents, notes, nil
}

// readAndParse reads and parses one workspace document, never reaching outside the
// workspace whatever the path says.
func readAndParse(root string, path documentdiscovery.DocumentPath) (documentparsing.MarkdownDocument, error) {
	full, ok := resolveInWorkspace(root, string(path))
	if !ok {
		return documentparsing.MarkdownDocument{}, fmt.Errorf("%s is outside the workspace", path)
	}
	content, err := os.ReadFile(full)
	if err != nil {
		return documentparsing.MarkdownDocument{}, err
	}

	return documentparsing.ParseDocument(path, content)
}

// resolveInWorkspace turns a workspace-relative path into an OS path, returning false when
// it would escape the workspace root.
func resolveInWorkspace(root string, path string) (string, bool) {
	full := filepath.Join(root, filepath.FromSlash(path))
	relative, err := filepath.Rel(root, full)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}

	return full, true
}

// noWorkspaceResult is returned by a workspace-aware tool when no workspace is configured.
func noWorkspaceResult() toolCallResult {
	return errorResult("no workspace is open for this MCP server. Start the engine with --workspace <dir>, or use the connection the LoreMaster extension registers.")
}
