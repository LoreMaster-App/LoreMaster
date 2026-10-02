package synccommands

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// session is the in-memory platform as a session's platform.
type session struct {
	platformport.DocumentationPlatform
	outputs *[]workspacesettings.Output
}

// ForOutput records the output it was set up for.
func (s session) ForOutput(output workspacesettings.Output) platformport.DocumentationPlatform {
	*s.outputs = append(*s.outputs, output)

	return s.DocumentationPlatform
}

// editor plays VS Code: it draws diagrams and collects progress. It is served
// synchronously, as vscode-jsonrpc handles messages in arrival order: every progress
// notification is recorded before the response to sync/execute reaches the caller.
type editor struct {
	mu       sync.Mutex
	progress []rpcprotocol.ProgressParams
}

func (e *editor) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	switch request.Method {
	case rpcprotocol.MethodHostRenderDiagram:
		_ = conn.Reply(ctx, request.ID, rpcprotocol.RenderDiagramResult{SVG: "<svg>drawn</svg>"})
	case rpcprotocol.MethodHostProgress:
		var params rpcprotocol.ProgressParams
		_ = json.Unmarshal(*request.Params, &params)
		e.mu.Lock()
		e.progress = append(e.progress, params)
		e.mu.Unlock()
	}
}

type world struct {
	t         *testing.T
	root      string
	fake      *platformport.InMemoryPlatform
	parent    string
	sessionID string
	conn      *jsonrpc2.Conn
	editor    *editor
	outputs   []workspacesettings.Output
}

const settings = `version: 1
outputs:
  - platform: confluence
    baseUrl: https://docs.example/
    space: ENG
    parentPageId: %PARENT%
    titlePrefix: ENG
    content:
      - type: markdown
        roots: ["."]
`

// newWorld is a workspace of three files, one with an image and a diagram, synced to a
// fake platform through an engine served over a pipe.
func newWorld(t *testing.T, platform func(*platformport.InMemoryPlatform) platformport.DocumentationPlatform) *world {
	t.Helper()
	fake := platformport.NewInMemoryPlatform(platformport.Space{ID: "1", Key: "ENG", Name: "Engineering"})
	w := &world{t: t, root: t.TempDir(), fake: fake, parent: fake.SeedPage("ENG", "", "Engineering Home"), editor: &editor{}}
	w.write(".lore-master.yaml", strings.ReplaceAll(settings, "%PARENT%", w.parent))
	w.write("README.md", "# Home\n\nSee [the guide](readme.guide.md).\n")
	w.write("readme.guide.md", "# Guide\n\n![flow](img/flow.png)\n\n```mermaid\ngraph TD; A-->B\n```\n")
	w.write("readme.guide.deep.md", "# Deep\n")
	w.write("img/flow.png", "PNG")

	var behind platformport.DocumentationPlatform = fake
	if platform != nil {
		behind = platform(fake)
	}
	sessions := sessionlifecycle.NewStore()
	w.sessionID = sessions.Add(sessionlifecycle.Session{BaseURL: "https://docs.example", Platform: session{DocumentationPlatform: behind, outputs: &w.outputs}}).ID
	plans := NewPlanStore()
	engineEnd, editorEnd := net.Pipe()
	go func() {
		_ = rpcserver.Serve(context.Background(), engineEnd, rpcserver.Methods{
			rpcprotocol.MethodSyncPlan:    PlanSync(sessions, plans),
			rpcprotocol.MethodSyncExecute: ExecuteSync(sessions, plans, time.Second),
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	w.conn = jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(editorEnd, jsonrpc2.VSCodeObjectCodec{}), w.editor)
	t.Cleanup(func() { _ = w.conn.Close() })

	return w
}

func (w *world) write(path string, content string) {
	w.t.Helper()
	full := filepath.Join(w.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) plan() (rpcprotocol.SyncPlanResult, error) {
	var result rpcprotocol.SyncPlanResult
	err := w.conn.Call(context.Background(), rpcprotocol.MethodSyncPlan, rpcprotocol.SyncPlanParams{SessionID: w.sessionID, WorkspaceRoot: w.root}, &result)

	return result, err
}

func (w *world) execute(planID string) rpcprotocol.SyncExecuteResult {
	w.t.Helper()
	var result rpcprotocol.SyncExecuteResult
	if err := w.conn.Call(context.Background(), rpcprotocol.MethodSyncExecute, rpcprotocol.SyncExecuteParams{PlanID: planID}, &result); err != nil {
		w.t.Fatal(err)
	}

	return result
}

func (w *world) annotation(path string) *syncannotation.Annotation {
	w.t.Helper()
	content, err := os.ReadFile(filepath.Join(w.root, path))
	if err != nil {
		w.t.Fatal(err)
	}
	read, err := syncannotation.Read(content)
	if err != nil {
		w.t.Fatal(err)
	}

	return read.Annotation
}

func code(err error) int64 {
	var wire *jsonrpc2.Error
	if errors.As(err, &wire) {
		return wire.Code
	}

	return 0
}

func TestPlanExecuteThenNothingToDo(t *testing.T) {
	w := newWorld(t, nil)
	plan, err := w.plan()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Counts["create"] != 3 || len(plan.Errors) != 0 || plan.Actions[1].ParentPath != "README.md" {
		t.Fatalf("%+v", plan)
	}
	if w.annotation("README.md") != nil {
		t.Fatal("planning writes nothing")
	}

	report := w.execute(plan.PlanID)
	if len(report.Pages) != 3 || len(report.Rewritten) != 3 {
		t.Fatalf("%+v", report)
	}
	for _, page := range report.Pages {
		if page.Outcome != "written" {
			t.Fatalf("%+v", page)
		}
	}
	guide := w.annotation("readme.guide.md")
	if guide == nil || guide.PageID != report.Pages[1].PageID || guide.Attachments["flow.png"] == "" {
		t.Fatalf("annotation %+v", guide)
	}
	body := w.fake.Page(guide.PageID).Body
	if diagram, ok := body.Blocks[1].(platformport.Diagram); !ok || diagram.Image == nil {
		t.Fatalf("the editor drew the diagram: %+v", body.Blocks)
	}
	w.editor.mu.Lock()
	progress := len(w.editor.progress)
	last := w.editor.progress[progress-1]
	w.editor.mu.Unlock()
	if progress != 3 || last.PlanID != plan.PlanID || last.Done != 3 || last.Total != 3 {
		t.Fatalf("progress %d %+v", progress, last)
	}

	if len(w.outputs) != 1 || w.outputs[0].Space != "ENG" || w.outputs[0].MermaidMode != "image" {
		t.Fatalf("pages are written the way the output asks: %+v", w.outputs)
	}

	again, err := w.plan()
	if err != nil || again.Counts["unchanged"] != 3 {
		t.Fatalf("%+v %v", again, err)
	}
	w.write("img/flow.png", "PNG, edited")
	if edited, err := w.plan(); err != nil || edited.Counts["update"] != 1 || edited.Actions[1].Kind != "update" {
		t.Fatalf("an edited image updates its page: %+v %v", edited.Counts, err)
	}
	var stale rpcprotocol.SyncExecuteResult
	if err := w.conn.Call(context.Background(), rpcprotocol.MethodSyncExecute, rpcprotocol.SyncExecuteParams{PlanID: plan.PlanID}, &stale); code(err) != rpcprotocol.CodeUnknownPlan {
		t.Fatalf("a plan runs once: %v", err)
	}
}

// held holds the guide's creation until the request is cancelled; the call then
// completes, as an HTTP request already sent would.
type held struct {
	*platformport.InMemoryPlatform
	started chan struct{}
}

func (h held) CreatePage(ctx context.Context, page platformport.NewPage) (platformport.RemotePage, error) {
	if page.Title == "ENG: Guide" {
		close(h.started)
		<-ctx.Done()
	}

	return h.InMemoryPlatform.CreatePage(context.WithoutCancel(ctx), page)
}

func TestACancelledSyncReportsAndWritesBackWhatItDid(t *testing.T) {
	started := make(chan struct{})
	w := newWorld(t, func(fake *platformport.InMemoryPlatform) platformport.DocumentationPlatform {
		return held{InMemoryPlatform: fake, started: started}
	})
	plan, err := w.plan()
	if err != nil {
		t.Fatal(err)
	}
	waiter, err := w.conn.DispatchCall(context.Background(), rpcprotocol.MethodSyncExecute, rpcprotocol.SyncExecuteParams{PlanID: plan.PlanID}, jsonrpc2.PickID(jsonrpc2.ID{Num: 77}))
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := w.conn.Notify(context.Background(), rpcprotocol.MethodCancelRequest, map[string]int{"id": 77}); err != nil {
		t.Fatal(err)
	}
	var report rpcprotocol.SyncExecuteResult
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := waiter.Wait(ctx, &report); err != nil {
		t.Fatal(err)
	}

	outcomes := []string{report.Pages[0].Outcome, report.Pages[1].Outcome, report.Pages[2].Outcome}
	if strings.Join(outcomes, " ") != "written written skipped" {
		t.Fatalf("%q", outcomes)
	}
	if w.annotation("README.md") == nil || w.annotation("readme.guide.md") == nil || w.annotation("readme.guide.deep.md") != nil {
		t.Fatal("the files agree with the pages: two annotated, one not")
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), "the sync was cancelled; 2 page(s) were written") {
		t.Fatalf("%q", report.Warnings)
	}
	resumed, err := w.plan()
	if err != nil || resumed.Counts["unchanged"] != 2 || resumed.Counts["create"] != 1 {
		t.Fatalf("the next plan picks up where it stopped: %+v %v", resumed.Counts, err)
	}
}

func TestPlanRefusals(t *testing.T) {
	w := newWorld(t, nil)

	w.write(".lore-master.yaml", "version: 1\noutputs:\n  - platform: confluence\n")
	if _, err := w.plan(); code(err) != rpcprotocol.CodeInvalidSettings || !strings.Contains(err.Error(), "run the first sync") {
		t.Fatalf("first sync: %v", err)
	}

	w.write(".lore-master.yaml", strings.ReplaceAll(strings.ReplaceAll(settings, "%PARENT%", w.parent), "docs.example", "other.example"))
	if _, err := w.plan(); code(err) != rpcprotocol.CodeInvalidParams || !strings.Contains(err.Error(), "the session is open on https://docs.example") {
		t.Fatalf("other site: %v", err)
	}

	w.write(".lore-master.yaml", strings.ReplaceAll(strings.ReplaceAll(settings, "%PARENT%", w.parent), "space: ENG", "space: NOPE"))
	if _, err := w.plan(); code(err) != rpcprotocol.CodeNotFound {
		t.Fatalf("unknown space: %v", err)
	}

	w.write(".lore-master.yaml", strings.ReplaceAll(settings, "%PARENT%", w.parent))
	w.write("other/README.md", "# Home\n")
	plan, err := w.plan()
	if err != nil || len(plan.Errors) != 1 || !strings.Contains(plan.Errors[0], `"ENG: Home"`) {
		t.Fatalf("duplicate titles are plan errors: %+v %v", plan.Errors, err)
	}
	var report rpcprotocol.SyncExecuteResult
	if err := w.conn.Call(context.Background(), rpcprotocol.MethodSyncExecute, rpcprotocol.SyncExecuteParams{PlanID: plan.PlanID}, &report); code(err) != rpcprotocol.CodePlanHasErrors {
		t.Fatalf("a plan with errors is not executed: %v", err)
	}

	w.write("other/README.md", "<!-- lore-master\nversion: seven\n-->\n# Other home\n")
	broken, err := w.plan()
	if err != nil || len(broken.Errors) != 1 || !strings.Contains(broken.Errors[0], "other/README.md") {
		t.Fatalf("a broken annotation is a plan error: %+v %v", broken.Errors, err)
	}

	var relative rpcprotocol.SyncPlanResult
	if err := w.conn.Call(context.Background(), rpcprotocol.MethodSyncPlan, rpcprotocol.SyncPlanParams{SessionID: w.sessionID, WorkspaceRoot: "docs"}, &relative); code(err) != rpcprotocol.CodeInvalidParams {
		t.Fatalf("relative root: %v", err)
	}
}

func TestFilesAreNeverReadOutsideTheWorkspace(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"../secret.txt", "..", "docs/../../secret.txt"} {
		if _, err := fileReader(root)(documentdiscovery.DocumentPath(path)); err == nil || !strings.Contains(err.Error(), "outside the workspace") {
			t.Fatalf("%s: %v", path, err)
		}
	}
}

func TestThePlanStoreForgetsTheOldest(t *testing.T) {
	store := NewPlanStore()
	first := store.put(&storedPlan{})
	for range keptPlans {
		store.put(&storedPlan{})
	}
	if _, err := store.take(first); err == nil {
		t.Fatal("the oldest plan is forgotten")
	}
	if len(store.plans) != keptPlans || len(store.order) != keptPlans {
		t.Fatalf("%d %d", len(store.plans), len(store.order))
	}
}
