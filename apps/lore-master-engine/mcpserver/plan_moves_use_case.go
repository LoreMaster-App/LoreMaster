package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/filemoves"
)

const planMovesToolName = "plan_moves"

const planMovesToolDescription = "Plans moving or renaming Markdown files or folders, before anything is moved: returns every file that must move with them (a page's dotted children), every line to rewrite (parent: lines that would otherwise break) and, for each moved page, where it nests now and where it will nest afterwards, so the agent can reorganise the docs until each page lands where it belongs on the storage. Writes nothing; the agent moves the files and applies the edits it returns, then checks each with validate_document."

func planMovesTool() toolDescriptor {
	return toolDescriptor{
		Name:        planMovesToolName,
		Description: planMovesToolDescription,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"moves": map[string]any{
					"type":        "array",
					"description": "The moves to plan. A folder moves with everything in it.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"from": map[string]any{"type": "string", "description": "Workspace-relative path now, e.g. docs/old/setup.md."},
							"to":   map[string]any{"type": "string", "description": "Workspace-relative path it should have, e.g. docs/guides/setup.md."},
						},
						"required": []any{"from", "to"},
					},
				},
			},
			"required": []any{"moves"},
		},
	}
}

// placement is where one page nests.
type placement struct {
	Path   string `json:"path"`
	Parent string `json:"parent,omitempty"` // empty means directly under the selected parent page
	Rule   string `json:"rule,omitempty"`
}

// movedPage is one moved page, before and after.
type movedPage struct {
	Title  string    `json:"title"`
	Before placement `json:"before"`
	After  placement `json:"after"`
}

type planMovesView struct {
	Renames  []filemoves.Rename   `json:"renames"`
	Edits    []filemoves.LineEdit `json:"edits"`
	Pages    []movedPage          `json:"pages"`
	Warnings []string             `json:"warnings,omitempty"`
}

func planMovesResult(ctx context.Context, workspaceRoot string, arguments json.RawMessage) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}
	var args struct {
		Moves []filemoves.Move `json:"moves"`
	}
	if len(arguments) > 0 {
		_ = json.Unmarshal(arguments, &args)
	}
	if len(args.Moves) == 0 {
		return errorResult("plan_moves needs \"moves\": a list of {from, to} workspace-relative paths.")
	}
	moves := make([]filemoves.Move, 0, len(args.Moves))
	for _, move := range args.Moves {
		from, _, fromOK := workspaceRelPath(workspaceRoot, move.From)
		to, _, toOK := workspaceRelPath(workspaceRoot, move.To)
		if !fromOK || !toOK {
			return errorResult("%q to %q is outside the workspace.", move.From, move.To)
		}
		moves = append(moves, filemoves.Move{From: string(from), To: string(to)})
	}

	documents, _, err := loadWorkspaceDocuments(ctx, workspaceRoot)
	if err != nil {
		return errorResult("could not read the workspace: %v", err)
	}
	plan, err := filemoves.PlanWorkspaceMoves(ctx, workspaceRoot, moves)
	if err != nil {
		return errorResult("could not plan the moves: %v", err)
	}

	before, err := documenttree.BuildTree(documents)
	if err != nil {
		return errorResult("could not build the current page tree: %v", err)
	}
	final := finalPaths(moves, plan.Renames)
	after, err := documenttree.BuildTree(movedDocuments(documents, final))
	if err != nil {
		return errorResult("after these moves the page tree would not build: %v", err)
	}

	view := planMovesView{Renames: plan.Renames, Edits: plan.Edits, Warnings: plan.Warnings, Pages: []movedPage{}}
	placedBefore := placements(before)
	placedAfter := placements(after)
	for _, document := range documents {
		target := final(string(document.Path))
		if target == string(document.Path) {
			continue
		}
		view.Pages = append(view.Pages, movedPage{
			Title:  document.Title,
			Before: placedBefore[string(document.Path)],
			After:  placedAfter[target],
		})
	}

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: describeMoves(view)}},
		StructuredContent: view,
	}
}

// finalPaths answers where a file ends up: the user's own moves (a folder carries what is in
// it), then the follow-up renames of dotted children.
func finalPaths(moves []filemoves.Move, renames []filemoves.Rename) func(string) string {
	renamed := map[string]string{}
	for _, rename := range renames {
		renamed[rename.From] = rename.To
	}

	return func(p string) string {
		for _, move := range moves {
			switch {
			case p == move.From:
				return move.To
			case strings.HasPrefix(p, move.From+"/"):
				return move.To + p[len(move.From):]
			}
		}
		if to, ok := renamed[p]; ok {
			return to
		}

		return p
	}
}

// movedDocuments are the documents as they will be: at their new paths, with an explicit
// parent still naming the same document relative to the new folder, which is the edit the
// plan returns.
func movedDocuments(documents []documentparsing.MarkdownDocument, final func(string) string) []documentparsing.MarkdownDocument {
	moved := make([]documentparsing.MarkdownDocument, len(documents))
	for i, document := range documents {
		moved[i] = document
		oldPath := string(document.Path)
		moved[i].Path = documentdiscovery.DocumentPath(final(oldPath))
		if document.Annotation == nil || document.Annotation.Parent == "" {
			continue
		}
		copied := *document.Annotation
		target := path.Join(path.Dir(oldPath), copied.Parent)
		copied.Parent = relativeFrom(path.Dir(string(moved[i].Path)), final(target))
		moved[i].Annotation = &copied
	}

	return moved
}

// relativeFrom writes target relative to dir, both workspace-relative and '/'-separated.
func relativeFrom(dir, target string) string {
	dirParts := splitPath(dir)
	targetParts := splitPath(target)
	common := 0
	for common < len(dirParts) && common < len(targetParts)-1 && dirParts[common] == targetParts[common] {
		common++
	}
	parts := make([]string, 0, len(dirParts)+len(targetParts))
	for range dirParts[common:] {
		parts = append(parts, "..")
	}
	parts = append(parts, targetParts[common:]...)

	return strings.Join(parts, "/")
}

func splitPath(p string) []string {
	if p == "." || p == "" {
		return nil
	}

	return strings.Split(p, "/")
}

func placements(tree documenttree.DocumentTree) map[string]placement {
	found := map[string]placement{}
	tree.Walk(func(node *documenttree.TreeNode, _ int) {
		view := placement{Path: string(node.Document.Path), Rule: string(node.Decision.Rule)}
		if node.Decision.Parent != nil {
			view.Parent = string(*node.Decision.Parent)
		}
		found[view.Path] = view
	})

	return found
}

func describeMoves(view planMovesView) string {
	var text strings.Builder

	fmt.Fprintf(&text, "%d page(s) move.\n", len(view.Pages))
	for _, page := range view.Pages {
		fmt.Fprintf(&text, "\n- %s (%s)\n    now:   %s\n    after: %s (at %s)", page.Before.Path, page.Title,
			describePlacement(page.Before), describePlacement(page.After), page.After.Path)
	}
	if len(view.Renames) > 0 {
		text.WriteString("\n\nAlso rename, so these pages keep nesting under the page that moved:")
		for _, rename := range view.Renames {
			fmt.Fprintf(&text, "\n- %s -> %s", rename.From, rename.To)
		}
	}
	if len(view.Edits) > 0 {
		text.WriteString("\n\nRewrite these lines (line numbers are zero-based, paths are before any move):")
		for _, edit := range view.Edits {
			fmt.Fprintf(&text, "\n- %s line %d: %s", edit.Path, edit.Line, edit.Text)
		}
	}
	for _, warning := range view.Warnings {
		fmt.Fprintf(&text, "\nWarning: %s", warning)
	}
	text.WriteString("\n\nNothing was moved. Move the files, apply the edits, then call validate_document on each moved file.")

	return text.String()
}

func describePlacement(view placement) string {
	if view.Rule == "" {
		return "outside the page tree"
	}
	parent := view.Parent
	if parent == "" {
		parent = "the selected parent page"
	}

	return fmt.Sprintf("under %s [rule: %s]", parent, view.Rule)
}
