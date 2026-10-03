package remotesync

import (
	"context"
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"lore-master/libs/confluence-client/authentication"
	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/confluenceplatform"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/syncannotation"
)

var syncedAt = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

// svgRenderer draws every Mermaid diagram as a tiny SVG, so the image mode uploads an
// attachment the same way the editor's webview would.
type svgRenderer struct{}

func (svgRenderer) Render(_ context.Context, language string, _ string) ([]byte, error) {
	return []byte("<svg>" + language + "</svg>"), nil
}

// remoteWorkspace is a set of files on a fake disk, synced to the fake Confluence through
// the real adapter and use cases. It mirrors the in-memory harness in syncexecution, but
// every call crosses HTTP, so it exercises the confluence-client serialisation too.
type remoteWorkspace struct {
	t        *testing.T
	platform *confluenceplatform.Platform
	fake     *fakeConfluence
	output   workspacesettings.Output
	files    map[string][]byte
	renderer platformport.DiagramRenderer
	options  syncexecution.Options
	rootID   string
}

func newRemoteWorkspace(t *testing.T, files map[string]string) *remoteWorkspace {
	t.Helper()
	fake := newFakeConfluence()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	fake.base = server.URL + "/confluence"
	rootID := fake.seedPage("ENG", "", "Engineering Home")

	platform, err := confluenceplatform.New(confluenceplatform.Options{
		Connection: connection.Connection{BaseURL: fake.base, Edition: connection.DataCenter, Version: connection.Version{Major: 8, Minor: 5}},
		Credential: authentication.PAT{Token: "t"},
		Render:     storageformat.Options{LinkMode: storageformat.LinkByTitle, MermaidMode: storageformat.MermaidImage},
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &remoteWorkspace{
		t: t, platform: platform, fake: fake, files: map[string][]byte{}, renderer: svgRenderer{}, rootID: rootID,
		output: workspacesettings.Output{
			Platform: "confluence", BaseURL: fake.base, Space: "ENG", ParentPageID: rootID, TitlePrefix: "ENG",
			Direction: "to-platform", TitleCollision: "fail", MermaidMode: "image", LinkMode: "title",
		},
	}
	for path, content := range files {
		w.files[path] = []byte(content)
	}

	return w
}

func (w *remoteWorkspace) read(path documentdiscovery.DocumentPath) ([]byte, error) {
	content, ok := w.files[string(path)]
	if !ok {
		return nil, fmt.Errorf("%s: no such file", path)
	}

	return content, nil
}

// sync plans, executes and writes the annotations back, as the editor shell does.
func (w *remoteWorkspace) sync() (syncplanning.SyncPlan, syncexecution.SyncReport) {
	w.t.Helper()
	var documents []documentparsing.MarkdownDocument
	for path, content := range w.files {
		if strings.HasSuffix(path, ".md") {
			document, err := documentparsing.ParseDocument(documentdiscovery.DocumentPath(path), content)
			if err != nil {
				w.t.Fatal(err)
			}
			documents = append(documents, document)
		}
	}
	tree, err := documenttree.BuildTree(documents)
	if err != nil {
		w.t.Fatal(err)
	}
	titles, err := documenttree.PageTitles(tree, w.output.TitlePrefix)
	if err != nil {
		w.t.Fatal(err)
	}
	prepared := syncexecution.PreparePages(documents, titles, w.read)
	plan, err := syncplanning.PlanSync(context.Background(), w.platform, syncplanning.Input{
		Tree: tree, Output: w.output, SpaceID: "1", Rendered: prepared.Rendered(),
	})
	if err != nil {
		w.t.Fatal(err)
	}
	if len(plan.Errors) > 0 {
		w.t.Fatalf("plan errors %q", plan.Errors)
	}
	report := syncexecution.ExecutePlan(context.Background(), w.platform, syncexecution.ExecuteInput{
		Plan: plan, Prepared: prepared, Output: w.output, Space: platformport.SpaceRef{ID: "1", Key: "ENG"},
		Read: w.read, Renderer: w.renderer, Options: w.options,
		Now: func() time.Time { return syncedAt },
	})
	for _, page := range report.Pages {
		if page.Annotation != nil {
			rendered, err := syncannotation.Render(w.files[string(page.Path)], *page.Annotation)
			if err != nil {
				w.t.Fatal(err)
			}
			w.files[string(page.Path)] = rendered
		}
	}

	return plan, report
}

func outcomes(report syncexecution.SyncReport) []string {
	var lines []string
	for _, page := range report.Pages {
		line := fmt.Sprintf("%s %s %s", page.Outcome, page.Planned, page.Path)
		if page.Error != "" {
			line += ": " + page.Error
		}
		lines = append(lines, line)
	}

	return lines
}

func expectOutcomes(t *testing.T, report syncexecution.SyncReport, want ...string) {
	t.Helper()
	if got := outcomes(report); !slices.Equal(got, want) {
		t.Fatalf("outcomes\n got: %q\nwant: %q\nwarnings: %q", got, want, report.Warnings)
	}
}

func threeLevels() map[string]string {
	return map[string]string{
		"README.md":            "# Home\n\nSee [the guide](readme.guide.md).\n",
		"readme.guide.md":      "# Guide\n\n![diagram](img/flow.png)\n\n```mermaid\ngraph TD; A-->B\n```\n",
		"readme.guide.deep.md": "# Deep\n\nBack to [home](README.md#home).\n",
		"img/flow.png":         "PNG-1",
	}
}

func TestASyncCreatesTheTreeOverHTTPAndIsIdempotent(t *testing.T) {
	w := newRemoteWorkspace(t, threeLevels())

	_, first := w.sync()
	expectOutcomes(t, first,
		"written create README.md", "written create readme.guide.md", "written create readme.guide.deep.md")

	home, guide, deep := first.Pages[0], first.Pages[1], first.Pages[2]
	if w.fake.page(home.PageID).parentID != w.rootID {
		t.Fatalf("home goes under the configured parent: %+v", w.fake.page(home.PageID))
	}
	if w.fake.page(guide.PageID).parentID != home.PageID || w.fake.page(deep.PageID).parentID != guide.PageID {
		t.Fatal("each page goes under its parent's new id, resolved over HTTP")
	}
	if !w.fake.page(deep.PageID).marked {
		t.Fatal("a created page is marked with the lore-master label")
	}
	annotation := guide.Annotation
	if annotation.PageID != guide.PageID || annotation.ParentID != home.PageID || annotation.Version != 1 ||
		annotation.Space != "ENG" || annotation.BaseURL != w.output.BaseURL || !annotation.SyncedAt.Equal(syncedAt) {
		t.Fatalf("annotation written back from the remote answer: %+v", annotation)
	}
	if names := w.fake.attachmentNames(guide.PageID); len(names) != 2 || !slices.Contains(names, "flow.png") {
		t.Fatalf("the image and the rendered diagram are uploaded: %q", names)
	}

	// The whole point the dogfood could never show: a re-run on the same workspace is
	// unchanged and writes nothing, because the annotations are now in the files.
	mutations := w.fake.mutationCount()
	plan, second := w.sync()
	if counts := plan.Counts(); counts[syncplanning.Unchanged] != 3 {
		t.Fatalf("second plan is all unchanged: %v", counts)
	}
	expectOutcomes(t, second,
		"unchanged unchanged README.md", "unchanged unchanged readme.guide.md", "unchanged unchanged readme.guide.deep.md")
	if extra := w.fake.mutationCount() - mutations; extra != 0 {
		t.Fatalf("a re-run with nothing to do made %d mutating requests", extra)
	}
}

func TestEditingAFileUpdatesOnlyItsPageOverHTTP(t *testing.T) {
	w := newRemoteWorkspace(t, threeLevels())
	w.sync()

	w.files["README.md"] = append(w.files["README.md"], "\nMore words.\n"...)
	_, report := w.sync()
	expectOutcomes(t, report,
		"written update README.md", "unchanged unchanged readme.guide.md", "unchanged unchanged readme.guide.deep.md")
	if w.fake.page(report.Pages[0].PageID).version != 2 || report.Pages[0].Annotation.Version != 2 {
		t.Fatalf("the edited page is at version 2: %+v", report.Pages[0])
	}

	if plan, _ := w.sync(); plan.Counts()[syncplanning.Unchanged] != 3 {
		t.Fatalf("then steady again: %v", plan.Counts())
	}
}

func TestAdoptTakesOverAnExistingPageOverHTTP(t *testing.T) {
	w := newRemoteWorkspace(t, map[string]string{"README.md": "# Home\n"})
	existing := w.fake.seedPage("ENG", w.rootID, "ENG: Home")
	w.output.TitleCollision = "adopt"

	_, report := w.sync()
	expectOutcomes(t, report, "written adopt README.md")

	page := w.fake.page(existing)
	if report.Pages[0].PageID != existing || page.version != 2 || !page.marked || report.Pages[0].Annotation.PageID != existing {
		t.Fatalf("the untracked page is taken over, bumped and marked: %+v %+v", report.Pages[0], page)
	}
	if plan, _ := w.sync(); plan.Counts()[syncplanning.Unchanged] != 1 {
		t.Fatalf("after adoption the workspace is steady: %v", plan.Counts())
	}
}
