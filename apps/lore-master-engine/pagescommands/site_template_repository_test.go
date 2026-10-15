package pagescommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/workspacesettings"
)

func writeTemplateFile(t *testing.T, root string, name string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestACustomSiteTemplateFolderBecomesShellAndAssets(t *testing.T) {
	root := t.TempDir()
	writeTemplateFile(t, root, "brand/site/template.html", "<main>{{content}}</main>")
	writeTemplateFile(t, root, "brand/site/css/theme.css", "body{}")

	chosen, err := loadSiteTemplate(root, workspacesettings.Output{Platform: "github-pages", Template: "brand/site"})
	if err != nil {
		t.Fatal(err)
	}

	if chosen.Shell != "<main>{{content}}</main>" || len(chosen.Assets) != 1 || chosen.Assets[0].Path != "assets/template/css/theme.css" {
		t.Fatalf("template: %+v", chosen)
	}
}

func TestASiteTemplateWithoutContentIsRefusedWithTheFile(t *testing.T) {
	root := t.TempDir()
	writeTemplateFile(t, root, "brand/template.html", "<main></main>")

	_, err := loadSiteTemplate(root, workspacesettings.Output{Platform: "github-pages", Template: "./brand"})

	if err == nil || !strings.Contains(err.Error(), "brand/template.html") || !strings.Contains(err.Error(), "{{content}}") {
		t.Fatalf("error: %v", err)
	}
}

func TestBuiltinSiteTemplatesNeedNoFiles(t *testing.T) {
	for _, name := range []string{"", "default", "minimal", "handbook", "headless"} {
		if _, err := loadSiteTemplate(t.TempDir(), workspacesettings.Output{Platform: "github-pages", Template: name}); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	if _, err := loadSiteTemplate(t.TempDir(), workspacesettings.Output{Platform: "github-pages", Template: "missing/folder"}); err == nil {
		t.Error("a folder that does not exist was accepted")
	}
}

func TestAWikiTemplateIsAMarkdownFile(t *testing.T) {
	root := t.TempDir()
	writeTemplateFile(t, root, "wiki.md", "# {{title}}\n\n{{content}}\n")

	shell, err := loadWikiTemplate(root, workspacesettings.Output{Platform: "github-wiki", Template: "wiki.md"})
	if err != nil || !strings.Contains(shell, "{{content}}") {
		t.Fatalf("%q, %v", shell, err)
	}
	if shell, err := loadWikiTemplate(root, workspacesettings.Output{Platform: "github-wiki"}); err != nil || shell != "" {
		t.Fatalf("default: %q, %v", shell, err)
	}
}
