package pagescommands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/documentation-sync/templatereference"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/github-pages/siterender"
	"lore-master/libs/github-pages/wikirender"
)

const (
	// siteTemplateFile is the page shell inside a custom template's folder.
	siteTemplateFile = "template.html"
	// siteTemplateAssets is where the rest of the folder lands on the site; a shell refers to
	// it as {{root}}assets/template/<file>.
	siteTemplateAssets = "assets/template/"
	// maxTemplateBytes bounds what a template folder may ship, so a mistaken path (a whole
	// workspace) fails instead of being published.
	maxTemplateBytes = 20 << 20
)

// loadSiteTemplate is the template a github-pages output asks for: a built-in, or the folder
// it names.
func loadSiteTemplate(workspaceRoot string, output workspacesettings.Output) (siterender.Template, error) {
	reference, problem := templatereference.Check("github-pages", output.Template)
	if problem != "" {
		return siterender.Template{}, fmt.Errorf("outputs template: %s", problem)
	}
	if reference.Kind == templatereference.Builtin {
		chosen, _ := siterender.BuiltinTemplate(reference.Name)

		return chosen, nil
	}

	folder := filepath.Join(workspaceRoot, filepath.FromSlash(reference.Path))
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		return siterender.Template{}, fmt.Errorf("the site template %q is not a folder in the workspace; it holds %s and any files that go with it", reference.Path, siteTemplateFile)
	}
	shell, err := os.ReadFile(filepath.Join(folder, siteTemplateFile))
	if err != nil {
		return siterender.Template{}, fmt.Errorf("the site template folder %q has no %s", reference.Path, siteTemplateFile)
	}
	if problems := siterender.ShellProblems(string(shell)); len(problems) > 0 {
		return siterender.Template{}, fmt.Errorf("%s/%s: %s", reference.Path, siteTemplateFile, strings.Join(problems, "; "))
	}

	chosen := siterender.Template{Name: reference.Path, Shell: string(shell)}
	total := int64(len(shell))
	err = filepath.WalkDir(folder, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(folder, path)
		if err != nil || filepath.ToSlash(relative) == siteTemplateFile {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		total += int64(len(content))
		if total > maxTemplateBytes {
			return fmt.Errorf("the site template folder %q is larger than %d MB; keep only the files the template uses", reference.Path, maxTemplateBytes>>20)
		}
		chosen.Assets = append(chosen.Assets, siterender.SiteFile{Path: siteTemplateAssets + filepath.ToSlash(relative), Content: content})

		return nil
	})
	if err != nil {
		return siterender.Template{}, err
	}

	return chosen, nil
}

// loadWikiTemplate is the Markdown shell a github-wiki output asks for, "" for the default.
func loadWikiTemplate(workspaceRoot string, output workspacesettings.Output) (string, error) {
	reference, problem := templatereference.Check("github-wiki", output.Template)
	if problem != "" {
		return "", fmt.Errorf("outputs template: %s", problem)
	}
	if reference.Kind == templatereference.Builtin {
		return "", nil
	}
	shell, err := os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(reference.Path)))
	if err != nil {
		return "", fmt.Errorf("the wiki template %q cannot be read: %v", reference.Path, err)
	}
	if problems := wikirender.ShellProblems(string(shell)); len(problems) > 0 {
		return "", fmt.Errorf("%s: %s", reference.Path, strings.Join(problems, "; "))
	}

	return string(shell), nil
}
