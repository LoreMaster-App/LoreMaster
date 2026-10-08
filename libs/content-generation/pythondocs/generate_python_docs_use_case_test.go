package pythondocs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/externaltool"
	"lore-master/libs/content-generation/generatedfile"
)

func put(t *testing.T, root string, relative string, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// shopWorkspace is the sample project the captured pydoc-markdown output (testdata) is of, with
// the tool installed in its .venv so it can be found.
func shopWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, file := range []string{"pyproject.toml", "src/shop/__init__.py", "src/shop/cart.py", "src/shop/billing/__init__.py", "src/shop/billing/invoice.py"} {
		put(t, root, file, "x")
	}
	installTool(t, root, "")

	return root
}

func installTool(t *testing.T, base string, folder string) {
	t.Helper()
	scripts, exe := "bin", "pydoc-markdown"
	if runtime.GOOS == "windows" {
		scripts, exe = "Scripts", "pydoc-markdown.exe"
	}
	put(t, base, filepath.Join(folder, ".venv", scripts, exe), "x")
}

// fakePydoc stands in for the tool: it prints what the real pydoc-markdown printed for the
// sample package.
type fakePydoc struct {
	commands []externaltool.Command
	result   func(externaltool.Command) externaltool.Result
}

func (f *fakePydoc) run(_ context.Context, command externaltool.Command) (externaltool.Result, error) {
	f.commands = append(f.commands, command)
	if f.result != nil {
		return f.result(command), nil
	}
	printed, err := os.ReadFile("testdata/pydoc-shop.md")
	if err != nil {
		return externaltool.Result{}, err
	}

	return externaltool.Result{Stdout: string(printed)}, nil
}

func paths(files []generatedfile.File) []string {
	var out []string
	for _, file := range files {
		out = append(out, file.Path)
	}
	slices.Sort(out)

	return out
}

func bodyOf(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no %s in %v", path, paths(files))

	return ""
}

func TestPackagesNestLikeFoldersAndModulesAreFilesBesideThem(t *testing.T) {
	root := shopWorkspace(t)

	output, err := generateWith(context.Background(), (&fakePydoc{}).run, root, generatedfile.Spec{})

	if err != nil {
		t.Fatal(err)
	}
	want := []string{"README.md", "shop/README.md", "shop/billing/README.md", "shop/billing/invoice.md", "shop/cart.md"}
	if got := paths(output.Files); !slices.Equal(got, want) {
		t.Fatalf("paths %v, want %v", got, want)
	}
	cart := bodyOf(t, output.Files, "shop/cart.md")
	for _, wanted := range []string{"# shop.cart", "## Cart Objects", "def add(sku: str, quantity: int = 1) -> int", "**Arguments**:"} {
		if !strings.Contains(cart, wanted) {
			t.Errorf("shop/cart.md lacks %q:\n%s", wanted, cart)
		}
	}
	if strings.Contains(cart, "<a id=") || strings.Contains(cart, "\r") {
		t.Fatalf("anchors or CRs left in:\n%q", cart)
	}
	index := bodyOf(t, output.Files, "README.md")
	if !strings.HasPrefix(index, "# Python API reference\n") || !strings.Contains(index, "[shop](shop/README.md)") || !strings.Contains(index, "[shop.billing](shop/billing/README.md)") {
		t.Fatalf("index:\n%s", index)
	}
	if len(output.Warnings) != 0 {
		t.Fatalf("warnings %v", output.Warnings)
	}
}

func TestEveryPageHasAUniqueTitle(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakePydoc{}).run, shopWorkspace(t), generatedfile.Spec{Title: "Shop API"})
	if err != nil {
		t.Fatal(err)
	}

	seen := map[string]string{}
	for _, file := range output.Files {
		title, _, _ := strings.Cut(strings.TrimPrefix(string(file.Body), "# "), "\n")
		if other, clash := seen[title]; clash {
			t.Errorf("%s and %s are both titled %q", other, file.Path, title)
		}
		seen[title] = file.Path
	}
	if !strings.HasPrefix(bodyOf(t, output.Files, "README.md"), "# Shop API\n") {
		t.Fatal("the title the user chose does not head the index")
	}
}

func TestTheToolIsRunInTheProjectOverItsSourcesInUTF8(t *testing.T) {
	double := &fakePydoc{}

	if _, err := generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{}); err != nil {
		t.Fatal(err)
	}

	if len(double.commands) != 1 {
		t.Fatalf("%d runs", len(double.commands))
	}
	command := double.commands[0]
	if got := strings.Join(command.Args, " "); got != "-I src -p shop" {
		t.Fatalf("arguments %q", got)
	}
	if !slices.Contains(command.Env, "PYTHONUTF8=1") || command.Timeout == 0 {
		t.Fatalf("command %+v", command)
	}
}

func TestAMissingToolStopsTheRunSayingWhatToInstall(t *testing.T) {
	t.Setenv("PATH", "")
	root := t.TempDir()
	put(t, root, "pyproject.toml", "x")
	put(t, root, "app/__init__.py", "x")

	_, err := generateWith(context.Background(), (&fakePydoc{}).run, root, generatedfile.Spec{})

	var missing *externaltool.MissingToolError
	if !errors.As(err, &missing) || missing.Tool != "pydoc-markdown" || !strings.Contains(err.Error(), "pip install pydoc-markdown") {
		t.Fatalf("error %v", err)
	}
}

func TestAFailingToolIsAnErrorNamingTheProject(t *testing.T) {
	double := &fakePydoc{result: func(externaltool.Command) externaltool.Result {
		return externaltool.Result{ExitCode: 1, Stderr: "SyntaxError: bad.py"}
	}}

	_, err := generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{})

	if err == nil || !strings.Contains(err.Error(), "documenting .:") || !strings.Contains(err.Error(), "SyntaxError: bad.py") {
		t.Fatalf("error %v", err)
	}
}

func TestToolWarningsAreReportedWithTheProject(t *testing.T) {
	double := &fakePydoc{result: func(externaltool.Command) externaltool.Result {
		printed, _ := os.ReadFile("testdata/pydoc-shop.md")

		return externaltool.Result{Stdout: string(printed), Stderr: "WARNING: could not parse x.py\nINFO: done\n"}
	}}

	output, err := generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{})

	if err != nil || len(output.Warnings) != 1 || output.Warnings[0] != ".: pydoc-markdown: WARNING: could not parse x.py" {
		t.Fatalf("err %v warnings %v", err, output.Warnings)
	}
}

func TestEachProjectOfAMonorepoGetsItsOwnSectionAndInputNarrowsThem(t *testing.T) {
	root := t.TempDir()
	for _, project := range []string{"services/billing", "services/legacy"} {
		put(t, root, project+"/pyproject.toml", "x")
		put(t, root, project+"/app/__init__.py", "x")
	}
	installTool(t, root, "")
	double := &fakePydoc{result: func(externaltool.Command) externaltool.Result {
		return externaltool.Result{Stdout: "# app\n\nThe app.\n"}
	}}

	output, err := generateWith(context.Background(), double.run, root, generatedfile.Spec{Input: []string{"services/", "!services/legacy/"}})

	if err != nil {
		t.Fatal(err)
	}
	if got := paths(output.Files); !slices.Equal(got, []string{"README.md", "services/billing/app/README.md"}) {
		t.Fatalf("paths %v", got)
	}
	if len(double.commands) != 1 || filepath.Base(double.commands[0].Dir) != "billing" {
		t.Fatalf("commands %+v", double.commands)
	}
}

func TestAWorkspaceWithNothingToDocumentGetsAnIndexSayingSo(t *testing.T) {
	root := t.TempDir()
	put(t, root, "pyproject.toml", "x")
	put(t, root, "tests/test_a.py", "x")

	output, err := generateWith(context.Background(), (&fakePydoc{}).run, root, generatedfile.Spec{})

	if err != nil || len(output.Files) != 1 || !strings.Contains(bodyOf(t, output.Files, "README.md"), "No Python packages") {
		t.Fatalf("err %v files %v", err, paths(output.Files))
	}
}

func TestSplitModulesLeavesCodeFencesAndMembersAlone(t *testing.T) {
	printed := "<a id=\"m\"></a>\r\n\r\n# m\r\n\r\nIntro.\r\n\r\n```python\r\n# not a module\r\nx = 1\r\n```\r\n\r\n<a id=\"m.f\"></a>\r\n\r\n#### f\r\n\r\n<a id=\"a\"></a>\r\n\r\n# a\r\n\r\nA.\r\n"

	pages := splitModules(printed)

	if len(pages) != 2 || pages[0].Name != "a" || pages[1].Name != "m" {
		t.Fatalf("pages %+v", pages)
	}
	if !strings.Contains(pages[1].Body, "# not a module") || !strings.Contains(pages[1].Body, "#### f") || strings.Contains(pages[1].Body, "<a id") {
		t.Fatalf("body:\n%s", pages[1].Body)
	}
}

func TestChooseSourcesSkipsTestsDocsAndSetupScripts(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{"mylib/__init__.py", "tests/__init__.py", "docs/__init__.py", ".venv/x/__init__.py", "scripts/run.sh", "main.py", "setup.py", "conftest.py", "test_main.py", "_private.py", "mylib.egg-info/__init__.py"} {
		put(t, root, file, "x")
	}

	got := chooseSources(root)

	if got.SearchPath != "." || !slices.Equal(got.Packages, []string{"mylib"}) || !slices.Equal(got.Modules, []string{"main"}) {
		t.Fatalf("%+v", got)
	}
}

func TestChooseSourcesPrefersSrcAndIgnoresAProjectWithNone(t *testing.T) {
	root := t.TempDir()
	put(t, root, "src/pkg/__init__.py", "x")
	put(t, root, "other/__init__.py", "x")
	if got := chooseSources(root); got.SearchPath != "src" || !slices.Equal(got.Packages, []string{"pkg"}) {
		t.Fatalf("%+v", got)
	}
	if got := chooseSources(t.TempDir()); !got.empty() {
		t.Fatalf("%+v", got)
	}
}
