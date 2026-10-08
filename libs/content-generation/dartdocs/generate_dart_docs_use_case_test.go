package dartdocs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// shopWorkspace is the package the captured dartdoc_json output (testdata) is of, plus files that
// must not be documented.
func shopWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "pubspec.yaml", "name: shop\ndescription: x\n")
	for _, file := range []string{"lib/shop.dart", "lib/src/cart.dart", "lib/src/more.dart", "lib/src/mods.dart", "lib/src/cart.g.dart", "lib/src/_internal.dart", "lib/generated/api.dart", "test/cart_test.dart"} {
		put(t, root, file, "x")
	}

	return root
}

// fakeDart stands in for dart: it lists dartdoc_json among the global tools and answers a parse
// with the units captured from the real tool (testdata) for the files asked about.
type fakeDart struct {
	commands []externaltool.Command
	globals  string
	failRun  bool
}

func (f *fakeDart) run(_ context.Context, command externaltool.Command) (externaltool.Result, error) {
	f.commands = append(f.commands, command)
	if slices.Equal(command.Args, []string{"pub", "global", "list"}) {
		return externaltool.Result{Stdout: f.globals}, nil
	}
	if f.failRun {
		return externaltool.Result{ExitCode: 65, Stderr: "Failed to parse lib/src/cart.dart"}, nil
	}
	asked := command.Args[slices.Index(command.Args, "-o")+2:]
	var units []unit
	for _, fixture := range []string{"cart.json", "more.json", "mods.json"} {
		content, err := os.ReadFile(filepath.Join("testdata", fixture))
		if err != nil {
			return externaltool.Result{}, err
		}
		var parsed []unit
		if err := json.Unmarshal(content, &parsed); err != nil {
			return externaltool.Result{}, err
		}
		for _, u := range parsed {
			if slices.Contains(asked, u.Source) {
				units = append(units, u)
			}
		}
	}
	encoded, _ := json.Marshal(units)
	if err := os.WriteFile(command.Args[slices.Index(command.Args, "-o")+1], encoded, 0o644); err != nil {
		return externaltool.Result{}, err
	}

	return externaltool.Result{}, nil
}

func newFake() *fakeDart { return &fakeDart{globals: "dartdoc_json 0.6.0\nflutterfire_cli 1.4.0\n"} }

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

func mustContain(t *testing.T, page string, wanted ...string) {
	t.Helper()
	for _, want := range wanted {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in:\n%s", want, page)
		}
	}
}

func TestEachLibraryWithSomethingPublicGetsAPageTitledByItsImportPath(t *testing.T) {
	output, err := generateWith(context.Background(), newFake().run, shopWorkspace(t), generatedfile.Spec{})

	if err != nil {
		t.Fatal(err)
	}
	if got, want := paths(output.Files), []string{"README.md", "src/cart.md", "src/mods.md", "src/more.md"}; !slices.Equal(got, want) {
		t.Fatalf("paths %v, want %v", got, want)
	}
	index := bodyOf(t, output.Files, "README.md")
	mustContain(t, index, "# Dart API reference", "[package:shop/src/cart.dart](src/cart.md)", "[package:shop/src/more.dart](src/more.md)")
}

func TestClassesShowTheirSignaturesAndDocComments(t *testing.T) {
	output, _ := generateWith(context.Background(), newFake().run, shopWorkspace(t), generatedfile.Spec{})

	mustContain(t, bodyOf(t, output.Files, "src/cart.md"),
		"# package:shop/src/cart.dart",
		"## class Cart", "class Cart\n", "A cart of items.",
		"### Cart (constructor)", "Cart(String this.owner)", "Creates a cart for [owner].",
		"### owner", "final String owner",
		"### add", "int add(String sku, {int quantity = 1})", "Adds an item and returns the new item count.")
}

func TestTheDeclarationKindsAreWrittenAsDartDeclaresThem(t *testing.T) {
	output, _ := generateWith(context.Background(), newFake().run, shopWorkspace(t), generatedfile.Spec{})

	more := bodyOf(t, output.Files, "src/more.md")
	mustContain(t, more,
		"abstract class Repo<T extends Object> extends Base implements Readable",
		"Repo.named(int this.id, {required String this.tag})",
		"factory Repo.make()",
		"final int id", "late String tag", "static const int max",
		"Future<T?> find(int id, [String? hint])",
		"int get size", "set size(int v)",
		"Repo<T> operator +(Repo<T> other)",
		"enum Kind", "- `one` — The first.", "String get label",
		"mixin Readable", "String read()",
		"extension Shout on String",
		"@Deprecated('use x')\nFuture<void> run<T>(List<T> items, {int retries = 3})",
		"typedef Callback",
		"const String version")
	mods := bodyOf(t, output.Files, "src/mods.md")
	mustContain(t, mods,
		"sealed class Shape", "base class B1", "final class F1", "interface class I1", "mixin class MC",
		"class W extends B1 with MC implements I1, F1", "static int make<X>(X x)", "void opt({int? a, required String b, int c = 2})",
		"extension type Meters(int v)", "mixin Logs on W implements I1", "typedef Pair<A, B>")
}

func TestPrivateAndGeneratedCodeIsLeftOut(t *testing.T) {
	double := newFake()

	output, _ := generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{})

	more := bodyOf(t, output.Files, "src/more.md")
	for _, hidden := range []string{"_hidden", "_private", "_hiddenFn", "@override"} {
		if strings.Contains(more, hidden) {
			t.Errorf("%s is documented", hidden)
		}
	}
	parse := double.commands[1]
	for _, skipped := range []string{"lib/src/cart.g.dart", "lib/src/_internal.dart", "lib/generated/api.dart", "test/cart_test.dart"} {
		if slices.Contains(parse.Args, skipped) {
			t.Errorf("%s was parsed", skipped)
		}
	}
	if !slices.Contains(parse.Args, "lib/shop.dart") || !slices.Contains(parse.Args, "lib/src/more.dart") {
		t.Fatalf("arguments %v", parse.Args)
	}
}

func TestEveryPageHasAUniqueTitleAndHeadingsInDocCommentsDoNotBreakTheOutline(t *testing.T) {
	declarations := []declaration{{Kind: "class", Name: "A", Description: "Intro.\n\n# Big heading\n\n```\n# not a heading\n```\n"}}

	page := renderLibrary("package:x/a.dart", declarations)

	mustContain(t, page, "**Big heading**", "# not a heading")
	if strings.Count(page, "\n# ")+boolToInt(strings.HasPrefix(page, "# ")) != 2 {
		// the title, and the line inside the fence
		t.Fatalf("headings:\n%s", page)
	}
	output, _ := generateWith(context.Background(), newFake().run, shopWorkspace(t), generatedfile.Spec{Title: "Shop"})
	seen := map[string]string{}
	for _, file := range output.Files {
		title, _, _ := strings.Cut(strings.TrimPrefix(string(file.Body), "# "), "\n")
		if other, clash := seen[title]; clash {
			t.Errorf("%s and %s are both titled %q", other, file.Path, title)
		}
		seen[title] = file.Path
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

func TestAMissingDartOrToolStopsTheRunSayingWhatToInstall(t *testing.T) {
	noDart := func(context.Context, externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{}, exec.ErrNotFound
	}
	_, err := generateWith(context.Background(), noDart, shopWorkspace(t), generatedfile.Spec{})
	var missing *externaltool.MissingToolError
	if !errors.As(err, &missing) || missing.Tool != "dart" || !strings.Contains(err.Error(), "install the Dart SDK") {
		t.Fatalf("no dart: %v", err)
	}

	double := newFake()
	double.globals = "flutterfire_cli 1.4.0\n"
	_, err = generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{})
	if !errors.As(err, &missing) || missing.Tool != "dartdoc_json" || !strings.Contains(err.Error(), "dart pub global activate dartdoc_json") {
		t.Fatalf("no dartdoc_json: %v", err)
	}
}

func TestAFailingParseIsAnErrorNamingTheProject(t *testing.T) {
	double := newFake()
	double.failRun = true

	_, err := generateWith(context.Background(), double.run, shopWorkspace(t), generatedfile.Spec{})

	if err == nil || !strings.Contains(err.Error(), "documenting .:") || !strings.Contains(err.Error(), "Failed to parse lib/src/cart.dart") {
		t.Fatalf("error %v", err)
	}
}

func TestABigPackageIsParsedInChunks(t *testing.T) {
	root := t.TempDir()
	put(t, root, "pubspec.yaml", "name: big\n")
	for i := 0; i < 130; i++ {
		put(t, root, fmt.Sprintf("lib/f%03d.dart", i), "x")
	}
	double := newFake()

	if _, err := generateWith(context.Background(), double.run, root, generatedfile.Spec{}); err != nil {
		t.Fatal(err)
	}

	if len(double.commands) != 1+3 {
		t.Fatalf("%d commands", len(double.commands))
	}
	for _, command := range double.commands[1:] {
		if files := len(command.Args) - 8; files < 1 || files > filesPerRun {
			t.Errorf("%d files in one run", files)
		}
	}
}

func TestInputNarrowsProjectsAndNoPackageNeedsNoTool(t *testing.T) {
	root := t.TempDir()
	for _, project := range []string{"apps/shop", "apps/legacy"} {
		put(t, root, project+"/pubspec.yaml", "name: p\n")
		put(t, root, project+"/lib/src/cart.dart", "x")
	}
	double := newFake()

	output, err := generateWith(context.Background(), double.run, root, generatedfile.Spec{Input: []string{"apps/", "!apps/legacy/"}})

	if err != nil {
		t.Fatal(err)
	}
	if got := paths(output.Files); !slices.Equal(got, []string{"README.md", "apps/shop/src/cart.md"}) {
		t.Fatalf("paths %v", got)
	}
	for _, command := range double.commands {
		if strings.Contains(command.Dir, "legacy") {
			t.Fatalf("legacy was parsed: %+v", command)
		}
	}

	empty, err := generateWith(context.Background(), func(context.Context, externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{}, exec.ErrNotFound
	}, t.TempDir(), generatedfile.Spec{})
	if err != nil || !strings.Contains(bodyOf(t, empty.Files, "README.md"), "No Dart packages") {
		t.Fatalf("err %v", err)
	}
}

func TestPackageNameComesFromThePubspec(t *testing.T) {
	root := t.TempDir()
	put(t, root, "pubspec.yaml", "# c\nname: 'my_app'\nversion: 1.0.0\n")
	if got := readPackageName(root); got != "my_app" {
		t.Fatalf("%q", got)
	}
	if got := readPackageName(t.TempDir()); got == "" {
		t.Fatal("no fallback")
	}
}
