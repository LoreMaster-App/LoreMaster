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
		Tree: tree, Output: w.output, SpaceID: "1", Rendered: prepared.Rendered(),
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
		if page.Annotation == nil {
			continue
		}
		var rendered []byte
		if page.PulledBody != nil {
			rendered, err = syncannotation.RenderWithBody(w.files[string(page.Path)], *page.Annotation, page.PulledBody)
		} else {
			rendered, err = syncannotation.Render(w.files[string(page.Path)], *page.Annotation)
		}
		if err != nil {
			w.t.Fatal(err)
		}
		w.files[string(page.Path)] = rendered
		for _, file := range page.PulledAttachments {
			w.files[string(file.Path)] = file.Content
		}
	}

	return plan, report
}

func (w *workspace) result(report SyncReport, path string) PageResult {
	w.t.Helper()
	for _, page := range report.Pages {
		if string(page.Path) == path {
			return page
		}
	}
	w.t.Fatalf("no result for %s", path)

	return PageResult{}
}

func TestTwoWayPullWritesTheRemoteBodyIntoTheFile(t *testing.T) {
	w := newWorkspace(t, map[string]string{"guide.md": "# Guide\n\nold local text\n"})
	w.output.Direction = "two-way"
	_, first := w.sync()
	id := w.result(first, "guide.md").PageID

	// Someone edits the page on the platform: a new body and a newer version.
	page := w.fake.Page(id)
	page.Body = platformport.Document{Blocks: []platformport.Block{
		platformport.Heading{Level: 1, Inlines: []platformport.Inline{platformport.Text{Value: "Guide"}}},
		platformport.Paragraph{Inlines: []platformport.Inline{platformport.Text{Value: "new text from the platform"}}},
	}}
	page.Version++

	_, second := w.sync()
	guide := w.result(second, "guide.md")
	if guide.Outcome != Pulled || guide.Version != page.Version {
		t.Fatalf("outcome %s version %d", guide.Outcome, guide.Version)
	}
	body := string(w.files["guide.md"])
	if !strings.Contains(body, "new text from the platform") || strings.Contains(body, "old local text") {
		t.Fatalf("the file was not rewritten with the pulled body: %q", body)
	}
	if !strings.Contains(body, "<!-- lore-master") {
		t.Fatalf("the annotation was lost on a pull: %q", body)
	}

	// Idempotent: nothing changed on either side, so a third sync leaves the page alone.
	_, third := w.sync()
	if got := w.result(third, "guide.md").Outcome; got != Unchanged {
		t.Fatalf("a pulled file should read back as unchanged, got %s", got)
	}
}

func TestPushEmbedsTheMermaidSourceInTheSVG(t *testing.T) {
	w := newWorkspace(t, map[string]string{"guide.md": "# Guide\n\n```mermaid\ngraph TD; A-->B\n```\n"})
	_, report := w.sync()
	id := w.result(report, "guide.md").PageID

	names, err := w.fake.ListAttachments(context.Background(), id)
	if err != nil || len(names) == 0 {
		t.Fatalf("no attachment uploaded: %v", err)
	}
	svg, err := w.fake.DownloadAttachment(context.Background(), id, names[0].Filename)
	if err != nil {
		t.Fatal(err)
	}
	if source, ok := extractMermaidSource(svg); !ok || !strings.Contains(source, "graph TD; A-->B") {
		t.Fatalf("the uploaded SVG does not carry the Mermaid source: %q ok=%v", source, ok)
	}
}

func TestTwoWayPullRecoversADiagramFromItsSVG(t *testing.T) {
	w := newWorkspace(t, map[string]string{"guide.md": "# Guide\n\ntext\n"})
	w.output.Direction = "two-way"
	_, first := w.sync()
	id := w.result(first, "guide.md").PageID

	// The platform shows the diagram as just its SVG image — its code macro was removed — but
	// the SVG carries the source.
	svg := injectMermaidSource([]byte("<svg></svg>"), "graph LR; X-->Y")
	if _, err := w.fake.UploadFile(context.Background(), id, platformport.File{Name: "mermaid-1.svg", Content: svg}); err != nil {
		t.Fatal(err)
	}
	page := w.fake.Page(id)
	page.Body = platformport.Document{Blocks: []platformport.Block{
		platformport.Heading{Level: 1, Inlines: []platformport.Inline{platformport.Text{Value: "Guide"}}},
		platformport.Paragraph{Inlines: []platformport.Inline{platformport.Image{Source: &platformport.AttachmentRef{Filename: "mermaid-1.svg"}, Alt: "Mermaid diagram"}}},
	}}
	page.Version++

	_, second := w.sync()
	if got := w.result(second, "guide.md").Outcome; got != Pulled {
		t.Fatalf("outcome %s", got)
	}
	if body := string(w.files["guide.md"]); !strings.Contains(body, "```mermaid\ngraph LR; X-->Y\n```") {
		t.Fatalf("the diagram was not recovered from its SVG: %q", body)
	}
	if _, wrote := w.files["assets/mermaid-1.svg"]; wrote {
		t.Fatal("a recovered diagram's SVG must not be written to disk")
	}
	if plan, _ := w.sync(); plan.Counts()[syncplanning.Unchanged] != 1 {
		t.Fatalf("a recovered diagram should read back as unchanged: %v", plan.Counts())
	}
}

func TestTwoWayPullDownloadsANewRemoteImageToAssets(t *testing.T) {
	w := newWorkspace(t, map[string]string{"guide.md": "# Guide\n\ntext\n"})
	w.output.Direction = "two-way"
	_, first := w.sync()
	id := w.result(first, "guide.md").PageID

	// The platform gains an image the file never had, and the body now shows it.
	if _, err := w.fake.UploadFile(context.Background(), id, platformport.File{Name: "diagram.png", Content: []byte("PNG-DATA")}); err != nil {
		t.Fatal(err)
	}
	page := w.fake.Page(id)
	page.Body = platformport.Document{Blocks: []platformport.Block{
		platformport.Heading{Level: 1, Inlines: []platformport.Inline{platformport.Text{Value: "Guide"}}},
		platformport.Paragraph{Inlines: []platformport.Inline{platformport.Image{Source: &platformport.AttachmentRef{Filename: "diagram.png"}, Alt: "d"}}},
	}}
	page.Version++

	_, second := w.sync()
	if got := w.result(second, "guide.md").Outcome; got != Pulled {
		t.Fatalf("outcome %s", got)
	}
	if got := string(w.files["assets/diagram.png"]); got != "PNG-DATA" {
		t.Fatalf("the new image was not written to assets/: %q", got)
	}
	if body := string(w.files["guide.md"]); !strings.Contains(body, "![d](assets/diagram.png)") {
		t.Fatalf("the image reference was not rewritten to the local path: %q", body)
	}

	// Idempotent: the image is now a tracked file, so a third sync changes nothing.
	if plan, _ := w.sync(); plan.Counts()[syncplanning.Unchanged] != 1 {
		t.Fatalf("a pulled image should read back as unchanged: %v", plan.Counts())
	}
}

func TestTwoWayPullLeavesAnUnconvertiblePageAlone(t *testing.T) {
	w := newWorkspace(t, map[string]string{"guide.md": "# Guide\n\nkeep me\n"})
	w.output.Direction = "two-way"
	_, first := w.sync()
	id := w.result(first, "guide.md").PageID

	// A remote edit whose body links to a page no file owns: the serializer cannot resolve it
	// and flags the conversion, so the pull must leave the file untouched.
	page := w.fake.Page(id)
	page.Body = platformport.Document{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{
		platformport.Link{Target: platformport.PageLink{Title: "Nonexistent"}, Inlines: []platformport.Inline{platformport.Text{Value: "x"}}},
	}}}}
	page.Version++

	_, second := w.sync()
	guide := w.result(second, "guide.md")
	if guide.Outcome != Skipped || guide.PulledBody != nil || !strings.Contains(guide.Error, "does not convert") {
		t.Fatalf("outcome %s error %q", guide.Outcome, guide.Error)
	}
	if !strings.Contains(string(w.files["guide.md"]), "keep me") {
		t.Fatalf("an unconvertible pull must not clobber the file: %q", w.files["guide.md"])
	}
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

func TestALinkFollowsTheFileItPointsAt(t *testing.T) {
	steadyAfter := func(t *testing.T, w *workspace, changed string) {
		t.Helper()
		plan, report := w.sync()
		for _, action := range plan.Actions {
			want := syncplanning.Unchanged
			if string(action.Path) == changed {
				want = syncplanning.Update
			}
			if action.Kind != want && action.Kind != syncplanning.Create {
				t.Fatalf("%s: %s, want %s (%q)", action.Path, action.Kind, want, outcomes(report))
			}
		}
		if again, _ := w.sync(); again.Counts()[syncplanning.Unchanged] != len(again.Actions) {
			t.Fatalf("then steady: %v", again.Counts())
		}
	}

	t.Run("the linked file appears", func(t *testing.T) {
		w := newWorkspace(t, map[string]string{"README.md": "# Home\n\nSee [setup](setup.md).\n"})
		w.sync()
		w.files["setup.md"] = []byte("# Setup\n")
		steadyAfter(t, w, "README.md")
		if !strings.Contains(fmt.Sprint(w.fake.Page(pageOf(t, w, "README.md")).Body), "ENG: Setup") {
			t.Fatal("the link now points at the new page")
		}
	})
	t.Run("the linked file is retitled", func(t *testing.T) {
		w := newWorkspace(t, map[string]string{"README.md": "# Home\n\nSee [setup](setup.md).\n", "setup.md": "# Setup\n"})
		w.sync()
		w.files["setup.md"] = []byte(strings.Replace(string(w.files["setup.md"]), "# Setup", "# Install", 1))
		plan, _ := w.sync()
		if plan.Counts()[syncplanning.Update] != 2 {
			t.Fatalf("setup.md changed and README.md shows it: %s", fmt.Sprint(plan.Actions))
		}
	})
	t.Run("a heading it points at is renamed", func(t *testing.T) {
		w := newWorkspace(t, map[string]string{"README.md": "# Home\n\nSee [usage](setup.md#usage).\n", "setup.md": "# Setup\n\n## Usage\n"})
		w.sync()
		// Same slug, other text: README.md is byte for byte the same, its anchor is not.
		w.files["setup.md"] = []byte(strings.Replace(string(w.files["setup.md"]), "## Usage", "## Usage!", 1))
		plan, _ := w.sync()
		if plan.Counts()[syncplanning.Update] != 2 || plan.Actions[0].Path != "README.md" || plan.Actions[0].Kind != syncplanning.Update {
			t.Fatalf("%v", plan.Actions)
		}
	})
}

func pageOf(t *testing.T, w *workspace, path string) string {
	t.Helper()
	read, err := syncannotation.Read(w.files[path])
	if err != nil || read.Annotation == nil {
		t.Fatalf("%s has no annotation", path)
	}

	return read.Annotation.PageID
}

func linkURLs(document platformport.Document) []string {
	var urls []string
	for _, block := range document.Blocks {
		if paragraph, ok := block.(platformport.Paragraph); ok {
			for _, inline := range paragraph.Inlines {
				if link, ok := inline.(platformport.Link); ok {
					if page, ok := link.Target.(platformport.PageLink); ok {
						urls = append(urls, page.Title+"="+page.URL)
					}
				}
			}
		}
	}

	return urls
}

func cycle() map[string]string {
	return map[string]string{
		"README.md": "# Home\n\nSee [setup](setup.md).\n",
		"setup.md":  "# Setup\n\nBack [home](README.md).\n",
	}
}

func TestIDLinksInACycleOfNewPages(t *testing.T) {
	w := newWorkspace(t, cycle())
	w.output.LinkMode = "id"
	_, report := w.sync()
	home, setup := report.Pages[0], report.Pages[1]
	if home.Outcome != Written || setup.Outcome != Written {
		t.Fatalf("%q", outcomes(report))
	}
	if got := linkURLs(w.fake.Page(home.PageID).Body); !slices.Equal(got, []string{"ENG: Setup=" + setup.URL}) {
		t.Fatalf("home links %q", got)
	}
	if got := linkURLs(w.fake.Page(setup.PageID).Body); !slices.Equal(got, []string{"ENG: Home=" + home.URL}) {
		t.Fatalf("setup links %q", got)
	}
	// Home was written before Setup existed, so it was written again; Setup was not.
	if home.Version != 2 || home.Annotation.Version != 2 || setup.Version != 1 || w.fake.Page(home.PageID).Version != 2 {
		t.Fatalf("versions: home %d/%d, setup %d", home.Version, home.Annotation.Version, setup.Version)
	}
	if plan, _ := w.sync(); plan.Counts()[syncplanning.Unchanged] != 2 {
		t.Fatalf("then steady: %v", plan.Actions)
	}
}

func TestTitleLinksNeverNeedASecondPass(t *testing.T) {
	w := newWorkspace(t, cycle())
	_, report := w.sync()
	if report.Pages[0].Version != 1 || linkURLs(w.fake.Page(report.Pages[0].PageID).Body)[0] != "ENG: Setup=" {
		t.Fatalf("%+v", report.Pages[0])
	}
}

func TestALinkToARecreatedPageFollowsItsNewID(t *testing.T) {
	for mode, want := range map[string]syncplanning.ActionKind{"id": syncplanning.Update, "title": syncplanning.Unchanged} {
		t.Run(mode, func(t *testing.T) {
			w := newWorkspace(t, cycle())
			w.output.LinkMode = mode
			_, first := w.sync()
			_ = w.fake.TrashPage(context.Background(), first.Pages[1].PageID)
			plan, report := w.sync()
			if plan.Actions[0].Kind != want || plan.Actions[1].Kind != syncplanning.Create {
				t.Fatalf("%s", fmt.Sprint(plan.Actions))
			}
			if mode == "id" && linkURLs(w.fake.Page(report.Pages[0].PageID).Body)[0] != "ENG: Setup="+report.Pages[1].URL {
				t.Fatal("home now links to the new page's id")
			}
		})
	}
}

// noRelink refuses the second write of a page.
type noRelink struct{ *platformport.InMemoryPlatform }

func (p noRelink) UpdatePage(ctx context.Context, update platformport.PageUpdate) (platformport.RemotePage, error) {
	if update.Message == "Links updated by LoreMaster" {
		return platformport.RemotePage{}, errors.New("503 unavailable")
	}

	return p.InMemoryPlatform.UpdatePage(ctx, update)
}

func TestAFailedRelinkIsRetriedNextTime(t *testing.T) {
	w := newWorkspace(t, cycle())
	w.output.LinkMode = "id"
	w.platform = noRelink{w.fake}
	_, report := w.sync()
	if report.Pages[0].Version != 1 || report.Pages[0].Annotation.Version != 1 || !slices.ContainsFunc(report.Warnings, func(warning string) bool {
		return strings.HasPrefix(warning, "README.md: the page was written, but its links could not be updated")
	}) {
		t.Fatalf("%+v %q", report.Pages[0], report.Warnings)
	}
	w.platform = w.fake
	plan, _ := w.sync()
	if plan.Actions[0].Kind != syncplanning.Update || plan.Actions[1].Kind != syncplanning.Unchanged {
		t.Fatalf("%s", fmt.Sprint(plan.Actions))
	}
	if linkURLs(w.fake.Page(report.Pages[0].PageID).Body)[0] != "ENG: Setup="+report.Pages[1].URL {
		t.Fatal("the retry links by id")
	}
}

func TestEditingAPageInIDModeWritesItOnce(t *testing.T) {
	w := newWorkspace(t, cycle())
	w.output.LinkMode = "id"
	w.sync()
	w.files["README.md"] = append(w.files["README.md"], "More.\n"...)
	calls := len(w.fake.Calls())
	_, report := w.sync()
	// Setup is unchanged and comes after README, yet its URL is known from the plan.
	if got := writes(w.fake, calls); len(got) != 1 || !strings.HasPrefix(got[0], "UpdatePage") {
		t.Fatalf("writes %q", got)
	}
	if linkURLs(w.fake.Page(report.Pages[0].PageID).Body)[0] != "ENG: Setup="+report.Pages[1].URL {
		t.Fatal("linked by id")
	}
}
