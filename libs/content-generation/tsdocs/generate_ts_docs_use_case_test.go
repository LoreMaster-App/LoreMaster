package tsdocs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/externaltool"
	"lore-master/libs/content-generation/generatedfile"
)

// workspaceWith writes the files, each with the text "x", and TypeDoc and its plugin into the
// node_modules of the folder (the workspace root by default) so the tool can be found.
func workspaceWith(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, file := range append(files, "node_modules/typedoc/bin/typedoc", "node_modules/typedoc-plugin-markdown/package.json") {
		put(t, root, file, "x")
	}

	return root
}

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

// fakeTypeDoc stands in for the tool: it writes what the real TypeDoc wrote for the sample
// package (captured in testdata), the expanded version when asked to expand modules.
type fakeTypeDoc struct {
	commands []externaltool.Command
	result   func(command externaltool.Command) (externaltool.Result, error)
}

func (f *fakeTypeDoc) run(_ context.Context, command externaltool.Command) (externaltool.Result, error) {
	f.commands = append(f.commands, command)
	if f.result != nil {
		if result, err := f.result(command); err != nil || result.ExitCode != 0 || result.Stdout != "" || result.Stderr != "" {
			return result, err
		}
	}
	out := argAfter(command.Args, "--out")
	fixture := "testdata/typedoc-entry"
	if slices.Contains(command.Args, "expand") {
		fixture = "testdata/typedoc-expand"
	}
	err := filepath.WalkDir(fixture, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, _ := filepath.Rel(fixture, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(out, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, content, 0o644)
	})

	return externaltool.Result{}, err
}

func argAfter(args []string, flag string) string {
	if at := slices.Index(args, flag); at >= 0 && at+1 < len(args) {
		return args[at+1]
	}

	return ""
}

func paths(output generatedfile.Output) []string {
	out := make([]string, len(output.Files))
	for i, file := range output.Files {
		out[i] = file.Path
	}

	return out
}

func bodyOf(t *testing.T, output generatedfile.Output, path string) string {
	t.Helper()
	for _, file := range output.Files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no page %s in %v", path, paths(output))

	return ""
}

func TestGenerateDocumentsALibraryThroughItsIndex(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "src/index.ts", "package.json")
	tool := &fakeTypeDoc{}

	output, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(tool.commands) != 1 {
		t.Fatalf("ran %d times", len(tool.commands))
	}
	command := tool.commands[0]
	if command.Name != "node" || !strings.HasSuffix(filepath.ToSlash(command.Args[0]), "node_modules/typedoc/bin/typedoc") || !strings.EqualFold(command.Dir, root) {
		t.Fatalf("command %+v", command)
	}
	for _, want := range []string{"--plugin", "typedoc-plugin-markdown", "--hidePageHeader", "--hideBreadcrumbs", "--hideGenerator", "--disableSources", "--skipErrorChecking"} {
		if !slices.Contains(command.Args, want) {
			t.Errorf("the command lacks %s: %v", want, command.Args)
		}
	}
	if argAfter(command.Args, "--tsconfig") != "tsconfig.json" || argAfter(command.Args, "--entryPoints") != "src/index.ts" || slices.Contains(command.Args, "expand") {
		t.Fatalf("arguments %v", command.Args)
	}

	want := []string{"README.md", "classes/Store.md", "functions/slugify.md", "interfaces/SlugOptions.md", "interfaces/StoreOptions.md", "variables/VERSION.md"}
	if got := paths(output); !slices.Equal(got, want) {
		t.Fatalf("pages %v, want %v", got, want)
	}
	if !strings.HasPrefix(bodyOf(t, output, "classes/Store.md"), "# Class: Store\\<T\\>\n") {
		t.Fatalf("page %s", bodyOf(t, output, "classes/Store.md"))
	}
	if len(output.Warnings) != 0 {
		t.Fatalf("warnings %q", output.Warnings)
	}
}

func TestGenerateDocumentsEveryModuleOfAnAppThatHasNoIndex(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "src/main.ts")
	tool := &fakeTypeDoc{}

	output, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}

	args := tool.commands[0].Args
	if argAfter(args, "--entryPoints") != "src" || argAfter(args, "--entryPointStrategy") != "expand" || !slices.Contains(args, "**/*.spec.ts") {
		t.Fatalf("arguments %v", args)
	}
	if got := paths(output); !slices.Contains(got, "store/store/classes/Store.md") || !slices.Contains(got, "README.md") {
		t.Fatalf("pages %v", got)
	}
}

func TestGenerateGivesAMonorepoOneSectionPerProjectAndAnIndexOfThem(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "apps/web/tsconfig.json", "apps/web/src/index.ts", "libs/core/tsconfig.json", "libs/core/src/index.ts")
	tool := &fakeTypeDoc{}

	output, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs", Title: "Our APIs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(tool.commands) != 2 || !strings.EqualFold(tool.commands[0].Dir, filepath.Join(root, "apps", "web")) {
		t.Fatalf("commands %+v", tool.commands)
	}
	got := paths(output)
	for _, want := range []string{"README.md", "apps/web/README.md", "apps/web/classes/Store.md", "libs/core/README.md", "libs/core/classes/Store.md"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	index := bodyOf(t, output, "README.md")
	if index != "# Our APIs\n\n- [apps/web](apps/web/README.md)\n- [libs/core](libs/core/README.md)\n" {
		t.Fatalf("index %q", index)
	}

	titles := map[string]string{}
	for _, file := range output.Files {
		heading := strings.ToLower(strings.SplitN(string(file.Body), "\n", 2)[0])
		if other, taken := titles[heading]; taken {
			t.Fatalf("%s and %s share the title %q", file.Path, other, heading)
		}
		titles[heading] = file.Path
	}
	if !strings.HasPrefix(bodyOf(t, output, "libs/core/classes/Store.md"), "# Class: Store\\<T\\> (libs/core/classes)\n") {
		t.Fatalf("a title repeated across projects is not told apart:\n%s", bodyOf(t, output, "libs/core/classes/Store.md"))
	}
	if len(output.Warnings) < 3 {
		t.Fatalf("the repaired titles are reported: %q", output.Warnings)
	}
}

func TestGenerateAProjectAtTheRootIsTheIndexItself(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "src/index.ts")

	output, err := generateWith(context.Background(), (&fakeTypeDoc{}).run, root, generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(bodyOf(t, output, "README.md"), "# sample-lib\n") {
		t.Fatalf("index %s", bodyOf(t, output, "README.md"))
	}

	titled, err := generateWith(context.Background(), (&fakeTypeDoc{}).run, root, generatedfile.Spec{Type: "ts-docs", Title: "Sample API"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(bodyOf(t, titled, "README.md"), "# Sample API\n\nA small library") {
		t.Fatalf("a title the user chose heads the index:\n%s", bodyOf(t, titled, "README.md"))
	}
}

func TestGenerateSkipsProjectsWithNothingToDocument(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "tsconfig.base.json", "apps/solution/tsconfig.json", "libs/real/tsconfig.json", "libs/real/src/index.ts")
	tool := &fakeTypeDoc{}

	output, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(tool.commands) != 1 || !strings.EqualFold(tool.commands[0].Dir, filepath.Join(root, "libs", "real")) {
		t.Fatalf("commands %+v", tool.commands)
	}
	if len(output.Warnings) != 0 {
		t.Fatalf("warnings %q", output.Warnings)
	}
}

func TestGenerateLeavesTypeDocToAProjectsOwnConfig(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "typedoc.json", "src/index.ts")
	tool := &fakeTypeDoc{}

	if _, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"}); err != nil {
		t.Fatal(err)
	}

	if slices.Contains(tool.commands[0].Args, "--entryPoints") {
		t.Fatalf("a project that configured TypeDoc was given entry points: %v", tool.commands[0].Args)
	}
}

func TestGeneratePrefersTheTsconfigNxWritesForALibraryOrApp(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "tsconfig.app.json", "src/index.ts")
	tool := &fakeTypeDoc{}

	if _, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"}); err != nil {
		t.Fatal(err)
	}

	if got := argAfter(tool.commands[0].Args, "--tsconfig"); got != "tsconfig.app.json" {
		t.Fatalf("tsconfig %q", got)
	}
}

func TestGenerateFollowsTheInputPatterns(t *testing.T) {
	root := workspaceWith(t, "apps/web/tsconfig.json", "apps/web/src/index.ts", "libs/core/tsconfig.json", "libs/core/src/index.ts", "libs/legacy/tsconfig.json", "libs/legacy/src/index.ts")
	tool := &fakeTypeDoc{}

	if _, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs", Input: []string{"libs/", "!libs/legacy/"}}); err != nil {
		t.Fatal(err)
	}

	if len(tool.commands) != 1 || !strings.EqualFold(tool.commands[0].Dir, filepath.Join(root, "libs", "core")) {
		t.Fatalf("commands %+v", tool.commands)
	}
}

func TestGenerateFindsTypeDocHoistedToTheWorkspaceRoot(t *testing.T) {
	root := workspaceWith(t, "packages/a/tsconfig.json", "packages/a/src/index.ts")

	if _, err := generateWith(context.Background(), (&fakeTypeDoc{}).run, root, generatedfile.Spec{Type: "ts-docs"}); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateSaysWhatToInstallWhenTypeDocOrItsPluginIsMissing(t *testing.T) {
	for name, missing := range map[string]string{"TypeDoc": "node_modules/typedoc/bin/typedoc", "typedoc-plugin-markdown": "node_modules/typedoc-plugin-markdown/package.json"} {
		root := workspaceWith(t, "tsconfig.json", "src/index.ts")
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(missing))); err != nil {
			t.Fatal(err)
		}

		_, err := generateWith(context.Background(), (&fakeTypeDoc{}).run, root, generatedfile.Spec{Type: "ts-docs"})

		var missingTool *externaltool.MissingToolError
		if !errors.As(err, &missingTool) || missingTool.Tool != name || !strings.Contains(err.Error(), "npm i -D typedoc typedoc-plugin-markdown") {
			t.Fatalf("%s: got %v", name, err)
		}
	}
}

func TestGenerateSaysWhenNodeIsMissing(t *testing.T) {
	root := workspaceWith(t, "tsconfig.json", "src/index.ts")
	tool := &fakeTypeDoc{result: func(externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{}, fmt.Errorf("cannot run node: %w", exec.ErrNotFound)
	}}

	_, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})

	var missingTool *externaltool.MissingToolError
	if !errors.As(err, &missingTool) || missingTool.Tool != "node" {
		t.Fatalf("got %v", err)
	}
}

func TestGenerateReportsATypeDocFailureWithTheProjectAndWhatItSaid(t *testing.T) {
	root := workspaceWith(t, "apps/web/tsconfig.json", "apps/web/src/index.ts")
	tool := &fakeTypeDoc{result: func(externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{ExitCode: 2, Stderr: "\x1b[91m[error]\x1b[0m Unable to find any entry points"}, nil
	}}

	_, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})

	if err == nil || err.Error() != "documenting apps/web: typedoc exited with status 2: [error] Unable to find any entry points" {
		t.Fatalf("got %v", err)
	}
}

func TestGeneratePassesOnTypeDocWarningsPerProjectAndBounded(t *testing.T) {
	root := workspaceWith(t, "libs/core/tsconfig.json", "libs/core/src/index.ts")
	var noise strings.Builder
	for i := range 15 {
		fmt.Fprintf(&noise, "\x1b[93m[warning]\x1b[0m {@link Missing%d} could not be resolved\n", i)
	}
	tool := &fakeTypeDoc{result: func(externaltool.Command) (externaltool.Result, error) {
		return externaltool.Result{Stdout: "[info] fine\n" + noise.String()}, nil
	}}

	output, err := generateWith(context.Background(), tool.run, root, generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}

	var typedoc []string
	for _, warning := range output.Warnings {
		if strings.Contains(warning, "typedoc:") {
			typedoc = append(typedoc, warning)
		}
	}
	if len(typedoc) != maxWarnings+1 || typedoc[0] != "libs/core: typedoc: {@link Missing0} could not be resolved" || !strings.HasSuffix(typedoc[maxWarnings], "more warnings were left out") {
		t.Fatalf("warnings %q", typedoc)
	}
}

func TestGenerateWithNoTypeScriptProjectsStillWritesTheIndexSoStalePagesGoAway(t *testing.T) {
	output, err := generateWith(context.Background(), (&fakeTypeDoc{}).run, workspaceWith(t, "service/go.mod"), generatedfile.Spec{Type: "ts-docs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Files) != 1 || !strings.Contains(string(output.Files[0].Body), "No TypeScript projects with something to document were found.") {
		t.Fatalf("pages %v", paths(output))
	}
}

func TestGenerateStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := generateWith(ctx, (&fakeTypeDoc{}).run, workspaceWith(t, "tsconfig.json", "src/index.ts"), generatedfile.Spec{Type: "ts-docs"}); err == nil {
		t.Fatal("expected the cancellation")
	}
}

func TestEntryPointsPreferTheLibraryIndexThenSrcThenTheRootIndex(t *testing.T) {
	has := func(files ...string) func(string) bool {
		return func(relative string) bool { return slices.Contains(files, relative) }
	}
	cases := []struct {
		name  string
		files []string
		want  entryChoice
	}{
		{"own config", []string{"typedoc.json", "src/index.ts"}, entryChoice{Found: true, UseOwnConfig: true}},
		{"library index", []string{"src/index.ts", "src"}, entryChoice{Found: true, Paths: []string{"src/index.ts"}}},
		{"jsx library index", []string{"src/index.tsx"}, entryChoice{Found: true, Paths: []string{"src/index.tsx"}}},
		{"src without an index", []string{"src"}, entryChoice{Found: true, Paths: []string{"src"}, Expand: true}},
		{"root index", []string{"index.ts"}, entryChoice{Found: true, Paths: []string{"index.ts"}}},
		{"nothing", []string{"tsconfig.json"}, entryChoice{}},
	}
	for _, tc := range cases {
		got := chooseEntryPoints(has(tc.files...))
		if got.Found != tc.want.Found || got.UseOwnConfig != tc.want.UseOwnConfig || got.Expand != tc.want.Expand || !slices.Equal(got.Paths, tc.want.Paths) {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
