package filemoves

import (
	"context"
	"os"
	"path/filepath"

	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// PlanWorkspaceMoves reads every Markdown file under the workspace root (whatever the
// outputs leave out: a hidden file can still carry a parent line) and plans the follow-up
// changes of the moves. A file that cannot be read is skipped.
func PlanWorkspaceMoves(ctx context.Context, root string, moves []Move) (Plan, error) {
	found, err := documentdiscovery.DiscoverDocuments(ctx, documentdiscovery.Options{WorkspaceRoot: root, IncludeGitignored: true})
	if err != nil {
		return Plan{}, err
	}

	documents := make([]Document, 0, len(found.Documents))
	for _, p := range found.Documents {
		document := Document{Path: string(p)}
		if content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(string(p)))); readErr == nil {
			if field, ok := syncannotation.LocateParent(content); ok {
				document.Parent = &ParentRef{Line: field.Line, Text: field.Text, Value: field.Value}
			}
		}
		documents = append(documents, document)
	}

	return PlanMoves(documents, moves), nil
}
