package settingscommands

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
)

func connect(t *testing.T) *jsonrpc2.Conn {
	t.Helper()
	t.Setenv("LORE_MASTER_USER_SETTINGS", filepath.Join(t.TempDir(), "none.json"))
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSettingsRead:    ReadSettings(),
			rpcprotocol.MethodSettingsSave:    SaveSettings(),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	conn := jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (any, error) { return nil, nil })))
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// settingsFile writes the project's settings.json, which is where the configuration lives.
func settingsFile(t *testing.T, root string, content string) string {
	t.Helper()
	file := filepath.Join(root, ".vscode", "settings.json")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return file
}

const confluenceOutputJSON = `{ "platform": "confluence", "baseUrl": "https://acme.atlassian.net/wiki", "space": "ENG", "parentPageId": "98306", "titlePrefix": "ENG" }`

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func read(t *testing.T, conn *jsonrpc2.Conn, root string) rpcprotocol.SettingsReadResult {
	t.Helper()
	var result rpcprotocol.SettingsReadResult
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, &result); err != nil {
		t.Fatal(err)
	}

	return result
}

// answer fills in what the first-sync wizard asks.
func answer(settings rpcprotocol.Settings) rpcprotocol.Settings {
	settings.Outputs[0].BaseURL = "https://acme.atlassian.net/wiki"
	settings.Outputs[0].Space = "ENG"
	settings.Outputs[0].ParentPageID = "98306"
	settings.Outputs[0].TitlePrefix = "ENG"

	return settings
}

func TestTheFirstSyncWizardRoundTrip(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	fresh := read(t, conn, root)
	if fresh.Exists || !fresh.FirstSync || len(fresh.Settings.Outputs) != 1 || fresh.Settings.Outputs[0].MermaidMode != "image" {
		t.Fatalf("defaults: %+v", fresh)
	}

	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: answer(fresh.Settings)}, nil); err != nil {
		t.Fatal(err)
	}
	saved := read(t, conn, root)
	if !saved.Exists || saved.FirstSync || saved.Settings.Outputs[0].ParentPageID != "98306" || saved.Settings.Outputs[0].Content[0].Roots[0] != "." {
		t.Fatalf("saved: %+v", saved)
	}
	content, _ := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if !strings.Contains(string(content), `"loreMaster.outputs"`) {
		t.Fatalf("a new workspace is configured in the editor's settings:\n%s", content)
	}
	if _, err := os.Stat(filepath.Join(root, ".lore-master.yaml")); err == nil {
		t.Fatal("a new workspace got the retired yaml file")
	}
}

func TestSavingKeepsTheAuthorsComments(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	file := settingsFile(t, root, `{
  // Our handbook goes to the ENG space.
  "editor.tabSize": 2,
  "loreMaster.outputs": [
    { "platform": "confluence", "baseUrl": "https://acme.atlassian.net/wiki", "space": "ENG", /* the team space */ "parentPageId": "98306", "titlePrefix": "ENG" }
  ]
}
`)
	settings := read(t, conn, root).Settings
	settings.Outputs[0].Space = "DOCS"
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(file)
	if !strings.Contains(string(content), "// Our handbook goes to the ENG space.") || !strings.Contains(string(content), `"editor.tabSize": 2`) || !strings.Contains(string(content), `"DOCS"`) {
		t.Fatalf("comments and other settings survive:\n%s", content)
	}
}

func TestBadSettingsAreRefusedWithTheReason(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	settings := answer(read(t, conn, root).Settings)
	settings.Outputs[0].MermaidMode = "crayon"
	err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil)
	if code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), "crayon") {
		t.Fatalf("%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".vscode", "settings.json")); !os.IsNotExist(statErr) {
		t.Fatal("nothing is written when refused")
	}

	settingsFile(t, root, `{ "loreMaster.outputs": [ { "platform": "confluence", "apiToken": "abc" } ] }`)
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, nil); code(err) != rpcprotocol.CodeInvalidSettings {
		t.Fatalf("a secret in the file: %v", err)
	}

	settingsFile(t, root, `{ "loreMaster.outputs": [ { "platform": "confluence", "baseUrl": "https://acme.atlassian.net/wiki", "space": "ENG", "parentPageId": "1", "titlePrefix": "ENG", "linkMode": "rainbow" } ] }`)
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: root}, nil); code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), `linkMode "rainbow" is not one of title, id`) {
		t.Fatalf("a bad value names itself: %v", err)
	}

	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsRead, rpcprotocol.SettingsReadParams{WorkspaceRoot: "relative"}, nil); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative: %v", err)
	}
}

func TestAGitHubPagesOutputKeepsItsRepoAndBranchThroughASave(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	settings := read(t, conn, root).Settings
	settings.Outputs[0] = rpcprotocol.Output{
		Platform: "github-pages", Direction: "to-platform", Repo: "acme/handbook", Branch: "docs-site",
		Content: []rpcprotocol.Content{{Type: "markdown", Roots: []string{"docs"}, Template: "default"}},
	}
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil); err != nil {
		t.Fatal(err)
	}

	saved := read(t, conn, root).Settings.Outputs[0]
	if saved.Repo != "acme/handbook" || saved.Branch != "docs-site" {
		t.Fatalf("repo and branch were lost: %+v", saved)
	}
	content, _ := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if !strings.Contains(string(content), `"repo": "acme/handbook"`) || !strings.Contains(string(content), `"branch": "docs-site"`) {
		t.Fatalf("the file lacks them:\n%s", content)
	}
}

func TestTheDiscoveryScopeSurvivesAReadAndSave(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	settingsFile(t, root, `{
  "loreMaster.skipGitignored": false,
  "loreMaster.ignore": ["drafts/**"],
  "loreMaster.outputs": [ `+confluenceOutputJSON+` ]
}`)

	loaded := read(t, conn, root).Settings
	if loaded.SkipGitignored == nil || *loaded.SkipGitignored || len(loaded.Ignore) != 1 || loaded.Ignore[0] != "drafts/**" {
		t.Fatalf("read dropped the scope: %+v", loaded)
	}

	loaded.Ignore = append(loaded.Ignore, "NOTES.md")
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: loaded}, nil); err != nil {
		t.Fatal(err)
	}
	again := read(t, conn, root).Settings
	if again.SkipGitignored == nil || *again.SkipGitignored || len(again.Ignore) != 2 || again.Ignore[1] != "NOTES.md" {
		t.Fatalf("save dropped the scope: %+v", again)
	}
}

func TestGeneratorsSurviveAReadAndSave(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	settingsFile(t, root, `{
  "loreMaster.generators": [ { "type": "test-results", "input": ["ci/"], "output": "docs/tests", "title": "CI results" } ],
  "loreMaster.outputs": [ `+confluenceOutputJSON+` ]
}`)

	loaded := read(t, conn, root).Settings
	if len(loaded.Generators) != 1 || loaded.Generators[0].Type != "test-results" || loaded.Generators[0].Output != "docs/tests" || loaded.Generators[0].Title != "CI results" || loaded.Generators[0].Input[0] != "ci/" {
		t.Fatalf("read dropped the generators: %+v", loaded.Generators)
	}

	loaded.Generators = append(loaded.Generators, rpcprotocol.Generator{Type: "test-results", Output: "docs/unit"})
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: loaded}, nil); err != nil {
		t.Fatal(err)
	}
	again := read(t, conn, root).Settings
	if len(again.Generators) != 2 || again.Generators[1].Output != "docs/unit" {
		t.Fatalf("save dropped a generator: %+v", again.Generators)
	}
}

func TestAnOutputsIncludeAndExcludeListsSurviveReadAndSave(t *testing.T) {
	root := t.TempDir()
	file := settingsFile(t, root, `{ "loreMaster.outputs": [ { "platform": "confluence", "baseUrl": "https://acme.atlassian.net/wiki", "space": "ENG", "parentPageId": "1", "titlePrefix": "ENG", "include": ["docs/keep.md"], "exclude": ["drafts/"] } ] }`)
	conn := connect(t)

	result := read(t, conn, root)
	output := result.Settings.Outputs[0]
	if len(output.Include) != 1 || output.Include[0] != "docs/keep.md" || len(output.Exclude) != 1 || output.Exclude[0] != "drafts/" {
		t.Fatalf("read include=%v exclude=%v", output.Include, output.Exclude)
	}
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: result.Settings}, nil); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "docs/keep.md") || !strings.Contains(string(saved), "drafts/") {
		t.Fatalf("a save dropped the lists:\n%s", saved)
	}
}

func TestSettingsComeBackWithTheNamesTheEngineMakesUp(t *testing.T) {
	conn, root := connect(t), t.TempDir()
	fresh := read(t, conn, root)
	if len(fresh.DisplayNames) != 1 || fresh.DisplayNames[0] != "Confluence · no space yet" {
		t.Fatalf("a fresh project: %v", fresh.DisplayNames)
	}

	settings := answer(fresh.Settings)
	settings.Outputs = append(settings.Outputs, rpcprotocol.Output{Platform: "github-pages", Repo: "acme/docs", Content: settings.Outputs[0].Content, Direction: "to-platform"})
	if err := conn.Call(context.Background(), rpcprotocol.MethodSettingsSave, rpcprotocol.SettingsSaveParams{WorkspaceRoot: root, Settings: settings}, nil); err != nil {
		t.Fatal(err)
	}

	saved := read(t, conn, root)
	if len(saved.Summaries) != 2 || saved.Summaries[0] != "https://acme.atlassian.net/wiki" || saved.Summaries[1] != "acme/docs" {
		t.Fatalf("summaries: %v", saved.Summaries)
	}
	if len(saved.DisplayNames) != 2 || saved.DisplayNames[0] != "Confluence · ENG" || saved.DisplayNames[1] != "GitHub Pages · acme/docs" {
		t.Fatalf("names: %v", saved.DisplayNames)
	}
}
