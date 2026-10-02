package syncexecution

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// workspace is files on a fake disk synced to a fake platform, with the annotation
// write-back done by hand until #36.
type workspace struct {
	t        *testing.T
	platform platformport.DocumentationPlatform
	fake     *platformport.InMemoryPlatform
	output   workspacesettings.Output
	files    map[string][]byte
	renderer platformport.DiagramRenderer
	options  Options
	progress []string
}

type renderer struct{ fail bool }

func (r renderer) Render(_ context.Context, language string, source string) ([]byte, error) {
	if r.fail {
		return nil, errors.New("syntax error")
	}

	return []byte("<svg>" + language + ":" + source + "</svg>"), nil
}

func newWorkspace(t *testing.T, files map[string]string) *workspace {
	t.Helper()
	fake := platformport.NewInMemoryPlatform(platformport.Space{ID: "1", Key: "ENG", Name: "Engineering"})
	root := fake.SeedPage("ENG", "", "Engineering Home")
	w := &workspace{t: t, platform: fake, fake: fake, files: map[string][]byte{}, renderer: renderer{}, output: workspacesettings.Output{
		Platform: "confluence", BaseURL: "https://docs.example", Space: "ENG", ParentPageID: root, TitlePrefix: "ENG",
		Direction: "to-platform", TitleCollision: "fail", MermaidMode: "image", LinkMode: "title",
	}}
	for path, content := range files {
		w.files[path] = []byte(content)
	}

	return w
}

func (w *workspace) read(path documentdiscovery.DocumentPath) ([]byte, error) {
	content, ok := w.files[string(path)]
	if !ok {
		return nil, fmt.Errorf("%s: no such file", path)
	}

	return content, nil
}

// sync plans and executes, then writes the annotations back.
func (w *workspace) sync() (syncplanning.SyncPlan, SyncReport) {
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
	prepared := PreparePages(documents, titles, w.read)
	plan, err := syncplanning.PlanSync(context.Background(), w.platform, syncplanning.Input{
		Tree: tree, Output: w.output, SpaceID: "1", Attachments: prepared.AttachmentHashes(),
	})
	if err != nil {
		w.t.Fatal(err)
	}
	if len(plan.Errors) > 0 {
		w.t.Fatalf("plan errors %q", plan.Errors)
	}
	w.progress = nil
	report := ExecutePlan(context.Background(), w.platform, ExecuteInput{
		Plan: plan, Prepared: prepared, Output: w.output, Space: platformport.SpaceRef{ID: "1", Key: "ENG"},
		Read: w.read, Renderer: w.renderer, Options: w.options,
		Progress: func(done int, total int, message string) {
			w.progress = append(w.progress, fmt.Sprintf("%d/%d %s", done, total, message))
		},
		Now: func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) },
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

func outcomes(report SyncReport) []string {
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

func expectOutcomes(t *testing.T, report SyncReport, want ...string) {
	t.Helper()
	if got := outcomes(report); !slices.Equal(got, want) {
		t.Fatalf("outcomes\n got: %q\nwant: %q\nwarnings: %q", got, want, report.Warnings)
	}
}

func writes(fake *platformport.InMemoryPlatform, since int) []string {
	var calls []string
	for _, call := range fake.Calls()[since:] {
		if !strings.HasPrefix(call, "Get") && !strings.HasPrefix(call, "List") && !strings.HasPrefix(call, "Find") {
			calls = append(calls, call)
		}
	}

	return calls
}

func threeLevels() map[string]string {
	return map[string]string{
		"README.md":            "# Home\n\nSee [the guide](readme.guide.md).\n",
		"readme.guide.md":      "# Guide\n\n![diagram](img/flow.png)\n\n```mermaid\ngraph TD; A-->B\n```\n",
		"readme.guide.deep.md": "# Deep\n\nBack to [home](README.md#home).\n",
		"img/flow.png":         "PNG-1",
	}
}

func TestFirstSyncThenNothingToDo(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	expectOutcomes(t, first,
		"written create README.md", "written create readme.guide.md", "written create readme.guide.deep.md")

	home, guide, deep := first.Pages[0], first.Pages[1], first.Pages[2]
	if w.fake.Page(guide.PageID).ParentID != home.PageID || w.fake.Page(deep.PageID).ParentID != guide.PageID {
		t.Fatal("each page goes under its parent's new id")
	}
	if !w.fake.Page(deep.PageID).Marked || w.fake.Page(deep.PageID).SourcePath != "readme.guide.deep.md" {
		t.Fatal("a created page is marked with its source")
	}
	annotation := guide.Annotation
	if annotation.PageID != guide.PageID || annotation.ParentID != home.PageID || annotation.Version != 1 ||
		annotation.Space != "ENG" || annotation.BaseURL != "https://docs.example" || !annotation.SyncedAt.Equal(time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)) ||
		annotation.Attachments["flow.png"] != attachmentHash([]byte("PNG-1")) || len(annotation.Attachments) != 1 {
		t.Fatalf("annotation %+v", annotation)
	}
	diagram := w.fake.Page(guide.PageID).Body.Blocks[1].(platformport.Diagram)
	if diagram.Image == nil || !strings.HasPrefix(diagram.Image.Filename, "mermaid-") {
		t.Fatalf("the diagram points at its picture: %+v", diagram)
	}
	uploads := slices.DeleteFunc(writes(w.fake, 0), func(call string) bool { return !strings.HasPrefix(call, "UploadFile") })
	if len(uploads) != 2 || !strings.HasSuffix(uploads[0], " flow.png") || !strings.HasSuffix(uploads[1], diagram.Image.Filename) {
		t.Fatalf("uploads %q", uploads)
	}
	if len(w.progress) != 3 || w.progress[2] != "3/3 written: ENG: Deep" {
		t.Fatalf("progress %q", w.progress)
	}

	calls := len(w.fake.Calls())
	plan, second := w.sync()
	if counts := plan.Counts(); counts[syncplanning.Unchanged] != 3 {
		t.Fatalf("second plan %v", counts)
	}
	expectOutcomes(t, second,
		"unchanged unchanged README.md", "unchanged unchanged readme.guide.md", "unchanged unchanged readme.guide.deep.md")
	if extra := writes(w.fake, calls); len(extra) != 0 {
		t.Fatalf("a sync with nothing to do wrote %q", extra)
	}
}

func TestAnEditedImageIsTheOnlyUpload(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	w.files["readme.guide.md"] = []byte("# Guide\n\n![diagram](img/flow.png) ![other](img/other.png)\n")
	w.files["img/other.png"] = []byte("OTHER")
	w.sync()
	w.files["img/flow.png"] = []byte("PNG-2")
	calls := len(w.fake.Calls())
	_, report := w.sync()
	expectOutcomes(t, report,
		"unchanged unchanged README.md", "written update readme.guide.md", "unchanged unchanged readme.guide.deep.md")
	got := writes(w.fake, calls)
	if len(got) != 2 || !strings.HasPrefix(got[0], "UpdatePage") || !strings.HasSuffix(got[1], " flow.png") {
		t.Fatalf("writes %q", got)
	}
	if report.Pages[1].Annotation.Attachments["flow.png"] != attachmentHash([]byte("PNG-2")) || report.Pages[1].Annotation.Attachments["other.png"] != attachmentHash([]byte("OTHER")) {
		t.Fatalf("attachments %v", report.Pages[1].Annotation.Attachments)
	}
}

func TestAConflictIsSkippedUnlessForced(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	guide := first.Pages[1].PageID
	w.fake.EditRemotely(guide)
	w.files["readme.guide.md"] = append(w.files["readme.guide.md"], "Local edit.\n"...)

	_, skipped := w.sync()
	if skipped.Pages[1].Outcome != Skipped || skipped.Pages[1].Annotation != nil || !strings.Contains(skipped.Pages[1].Error, "edited on the platform") {
		t.Fatalf("not forced: %+v", skipped.Pages[1])
	}
	if w.fake.Page(guide).Version != 2 {
		t.Fatal("a skipped conflict leaves the page alone")
	}

	w.options.Force = true
	_, forced := w.sync()
	if forced.Pages[1].Outcome != Written || forced.Pages[1].Version != 3 || forced.Pages[1].Annotation.Version != 3 {
		t.Fatalf("forced: %+v", forced.Pages[1])
	}
}

func TestOrphansAreReportedOrPruned(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	deep := first.Pages[2].PageID
	delete(w.files, "readme.guide.deep.md")

	_, reported := w.sync()
	expectOutcomes(t, reported,
		"unchanged unchanged README.md", "unchanged unchanged readme.guide.md", "reported orphan ")
	if w.fake.Page(deep) == nil {
		t.Fatal("an orphan is not touched without prune")
	}

	w.options.Prune = true
	_, pruned := w.sync()
	if pruned.Pages[2].Outcome != Trashed || w.fake.Page(deep) != nil {
		t.Fatalf("pruned: %+v", pruned.Pages[2])
	}
}

// racing takes a title on the platform between planning and executing.
type racing struct {
	*platformport.InMemoryPlatform
	title string
}

func (r racing) CreatePage(ctx context.Context, page platformport.NewPage) (platformport.RemotePage, error) {
	if page.Title == r.title {
		r.SeedPage("ENG", "", r.title)
	}

	return r.InMemoryPlatform.CreatePage(ctx, page)
}

func TestAFailedPageSkipsOnlyItsChildren(t *testing.T) {
	files := threeLevels()
	files["readme.notes.md"] = "# Notes\n"
	w := newWorkspace(t, files)
	w.platform = racing{InMemoryPlatform: w.fake, title: "ENG: Guide"}
	_, report := w.sync()
	expectOutcomes(t, report,
		"written create README.md",
		`failed create readme.guide.md: another page is already titled "ENG: Guide"`,
		"skipped create readme.guide.deep.md: its parent page (readme.guide.md) was not written",
		"written create readme.notes.md",
	)
	if report.Pages[1].Annotation != nil || report.Pages[2].Annotation != nil || report.Count(Written) != 2 {
		t.Fatal("no annotation for a page that was not written")
	}
}

// flaky refuses every upload.
type flaky struct{ *platformport.InMemoryPlatform }

func (flaky) UploadFile(context.Context, string, platformport.File) (platformport.UploadedFile, error) {
	return platformport.UploadedFile{}, errors.New("413 too large")
}

func TestAFailedUploadIsRetriedNextTime(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	w.platform = flaky{w.fake}
	_, first := w.sync()
	if first.Pages[1].Outcome != Written || first.Pages[1].Annotation.Attachments != nil {
		t.Fatalf("the page is written, the image not recorded: %+v", first.Pages[1].Annotation)
	}
	if !slices.ContainsFunc(first.Warnings, func(warning string) bool {
		return strings.Contains(warning, "readme.guide.md: the image img/flow.png could not be uploaded (413 too large)")
	}) {
		t.Fatalf("warnings %q", first.Warnings)
	}

	w.platform = w.fake
	plan, second := w.sync()
	if plan.Actions[1].Kind != syncplanning.Update || second.Pages[1].Annotation.Attachments["flow.png"] == "" {
		t.Fatalf("retry: %s %+v", plan.Actions[1].Kind, second.Pages[1].Annotation)
	}
}

func TestADiagramThatCannotBeDrawnStaysCode(t *testing.T) {
	for name, r := range map[string]platformport.DiagramRenderer{"failing": renderer{fail: true}, "absent": nil} {
		t.Run(name, func(t *testing.T) {
			files := threeLevels()
			files["readme.second.md"] = "# Second\n\n```mermaid\ngraph LR; X-->Y\n```\n"
			w := newWorkspace(t, files)
			w.renderer = r
			_, report := w.sync()
			guide := report.Pages[1]
			if guide.Outcome != Written || w.fake.Page(guide.PageID).Body.Blocks[1].(platformport.Diagram).Image != nil {
				t.Fatalf("%+v", guide)
			}
			want := "no diagram renderer is available, so Mermaid diagrams are shown as code"
			if r != nil {
				want = "readme.guide.md: a Mermaid diagram could not be drawn (syntax error); it is shown as code"
			}
			if count := len(slices.DeleteFunc(slices.Clone(report.Warnings), func(warning string) bool { return !strings.Contains(warning, "shown as code") })); count != map[bool]int{true: 2, false: 1}[r != nil] {
				t.Fatalf("one warning per diagram, or one in all without a renderer: %q", report.Warnings)
			}
			if !slices.Contains(report.Warnings, want) {
				t.Fatalf("warnings %q", report.Warnings)
			}
		})
	}
}

func TestCodeModeNeverRenders(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	w.output.MermaidMode = "code"
	w.renderer = renderer{fail: true}
	_, report := w.sync()
	if len(report.Warnings) != 0 {
		t.Fatalf("warnings %q", report.Warnings)
	}
}

func TestAMissingImageIsAWarningNotAFailure(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	delete(w.files, "img/flow.png")
	_, report := w.sync()
	if report.Count(Written) != 3 || !slices.ContainsFunc(report.Warnings, func(warning string) bool {
		return strings.HasPrefix(warning, "readme.guide.md: the image img/flow.png cannot be read")
	}) {
		t.Fatalf("%q %q", outcomes(report), report.Warnings)
	}
}

func TestACancelledSyncWritesNothingMore(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := ExecutePlan(ctx, w.platform, ExecuteInput{
		Plan:     syncplanning.SyncPlan{Actions: []syncplanning.Action{{Kind: syncplanning.Create, Path: "README.md", Title: "ENG: Home"}}},
		Prepared: Prepared{}, Output: w.output, Read: w.read,
	})
	if report.Pages[0].Outcome != Skipped || len(writes(w.fake, 0)) != 0 {
		t.Fatalf("%+v", report.Pages[0])
	}
}

func TestAPageRecreatedAfterARemoteDeleteGetsItsImagesAgain(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	_ = w.fake.TrashPage(context.Background(), first.Pages[1].PageID)
	calls := len(w.fake.Calls())
	_, second := w.sync()
	if second.Pages[1].Planned != syncplanning.Create || second.Pages[1].Outcome != Written {
		t.Fatalf("%+v", second.Pages[1])
	}
	if !slices.ContainsFunc(writes(w.fake, calls), func(call string) bool { return strings.HasSuffix(call, " flow.png") }) {
		t.Fatalf("the new page needs its image even though the hash did not change: %q", writes(w.fake, calls))
	}
}

func TestWriteBackKeepsWhatTheAuthorSet(t *testing.T) {
	w := newWorkspace(t, map[string]string{
		"README.md": "<!-- lore-master\ntitle: Start here\nowner: docs-team\n-->\n# Home\n",
	})
	_, report := w.sync()
	annotation := report.Pages[0].Annotation
	if report.Pages[0].Title != "ENG: Start here" || annotation.Title != "Start here" ||
		len(annotation.Unknown) != 1 || annotation.Unknown[0] != (syncannotation.Field{Key: "owner", Value: "docs-team"}) {
		t.Fatalf("%q %+v", report.Pages[0].Title, annotation)
	}
	if !strings.Contains(string(w.files["README.md"]), "owner: docs-team") {
		t.Fatalf("file %s", w.files["README.md"])
	}
}

func TestAnAdoptedPageIsTakenOverAndMarked(t *testing.T) {
	w := newWorkspace(t, map[string]string{"README.md": "# Home\n"})
	existing := w.fake.SeedPage("ENG", w.output.ParentPageID, "ENG: Home")
	w.output.TitleCollision = "adopt"
	_, report := w.sync()
	expectOutcomes(t, report, "written adopt README.md")
	page := w.fake.Page(existing)
	if report.Pages[0].PageID != existing || page.Version != 2 || !page.Marked || report.Pages[0].Annotation.PageID != existing {
		t.Fatalf("%+v %+v", report.Pages[0], page)
	}
}

func TestAMovedFileMovesItsPage(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	w.files["readme.deep.md"] = w.files["readme.guide.deep.md"]
	delete(w.files, "readme.guide.deep.md")
	_, report := w.sync()
	expectOutcomes(t, report,
		"unchanged unchanged README.md", "written move readme.deep.md", "unchanged unchanged readme.guide.md")
	if w.fake.Page(first.Pages[2].PageID).ParentID != first.Pages[0].PageID || report.Pages[1].Annotation.ParentID != first.Pages[0].PageID {
		t.Fatal("the page now sits under README's page")
	}
}

func TestPruneTrashesOnlyWhatTheSyncMade(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	handMade := w.fake.SeedPage("ENG", first.Pages[0].PageID, "Meeting notes")
	delete(w.files, "readme.guide.deep.md")
	w.options.Prune = true
	_, report := w.sync()
	expectOutcomes(t, report,
		"unchanged unchanged README.md", "unchanged unchanged readme.guide.md", "trashed orphan ")
	if w.fake.Page(first.Pages[2].PageID) != nil || w.fake.Page(handMade) == nil {
		t.Fatal("the marked orphan is trashed, the hand-made page is not")
	}
}

func TestALostAnnotationIsTakenBackAndWrittenAgain(t *testing.T) {
	w := newWorkspace(t, threeLevels())
	_, first := w.sync()
	w.files["readme.guide.md"] = []byte("# Guide\n\n![diagram](img/flow.png)\n\n```mermaid\ngraph TD; A-->B\n```\n")
	_, report := w.sync()
	guide := report.Pages[1]
	if guide.Planned != syncplanning.Adopt || guide.Outcome != Written || guide.PageID != first.Pages[1].PageID || guide.Annotation.PageID != guide.PageID {
		t.Fatalf("%+v", guide)
	}
	plan, _ := w.sync()
	if plan.Counts()[syncplanning.Unchanged] != 3 {
		t.Fatalf("after the take-back the workspace is steady again: %v", plan.Counts())
	}
}
