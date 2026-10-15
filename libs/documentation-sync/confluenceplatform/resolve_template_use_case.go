package confluenceplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/documentation-sync/templatereference"
)

// Template is a Confluence output's page layout as it was read: the storage format with its
// {{content}} placeholder, and a hash that changes whenever the layout does.
type Template struct {
	Shell string
	Hash  string
}

// IsEmpty reports whether there is no template, which leaves pages as they are.
func (template Template) IsEmpty() bool {
	return template.Shell == ""
}

// ResolveTemplate reads the template an output names: nothing for the default, the body of a
// Confluence page, or a file in the workspace. It checks the template can hold a page.
func (p *Platform) ResolveTemplate(ctx context.Context, workspaceRoot string, value string) (Template, error) {
	reference, problem := templatereference.Check("confluence", value)
	if problem != "" {
		return Template{}, fmt.Errorf("the template: %s", problem)
	}

	var shell, where string
	switch reference.Kind {
	case templatereference.Builtin:
		return Template{}, nil
	case templatereference.Page:
		page, err := p.pages.GetPage(ctx, reference.PageID, true)
		if err != nil {
			return Template{}, fmt.Errorf("the template page %s cannot be read: %w", reference.PageID, portError(err))
		}
		shell, where = page.BodyStorage, "the template page "+reference.PageID
	case templatereference.File:
		content, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(reference.Path)))
		if err != nil {
			return Template{}, fmt.Errorf("the template file %q cannot be read: %v", reference.Path, err)
		}
		shell, where = string(content), reference.Path
	}
	if problems := TemplateProblems(shell); len(problems) > 0 {
		return Template{}, fmt.Errorf("%s: %s", where, strings.Join(problems, "; "))
	}
	sum := sha256.Sum256([]byte(shell))

	return Template{Shell: shell, Hash: "sha256:" + hex.EncodeToString(sum[:])}, nil
}
