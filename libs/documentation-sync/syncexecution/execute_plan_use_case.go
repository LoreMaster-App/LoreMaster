package syncexecution

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"mime"
	"path"
	"time"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// Options are the user's choices for one run.
type Options struct {
	// Force overwrites pages edited on the platform since the last sync.
	Force bool
	// Prune moves orphan pages to the trash.
	Prune bool
}

// ExecuteInput is a plan and what is needed to carry it out.
type ExecuteInput struct {
	Plan     syncplanning.SyncPlan
	Prepared Prepared
	Output   workspacesettings.Output
	Space    platformport.SpaceRef
	Read     FileReader
	// Renderer draws Mermaid diagrams; nil shows them as code.
	Renderer platformport.DiagramRenderer
	Options  Options
	// Progress, when set, is told after each action.
	Progress func(done int, total int, message string)
	// Now stamps synced-at; nil is time.Now.
	Now func() time.Time
}

// ExecutePlan carries out the plan in its order, which is parents first, so a new
// page's id is known before its children are created. A page the platform refuses
// fails alone: its siblings go on, and only the children it was to be the parent of
// are skipped. Every page's outcome is in the report, with the annotation its file
// should now carry.
func ExecutePlan(ctx context.Context, platform platformport.DocumentationPlatform, input ExecuteInput) SyncReport {
	run := &execution{input: input, platform: platform, ids: map[documentdiscovery.DocumentPath]string{}, now: input.Now}
	if run.now == nil {
		run.now = time.Now
	}
	report := SyncReport{Warnings: append([]string(nil), input.Prepared.Warnings...)}
	total := len(input.Plan.Actions)
	for i, action := range input.Plan.Actions {
		result := run.execute(ctx, action)
		if action.Path != "" && result.PageID != "" {
			run.ids[action.Path] = result.PageID
		}
		report.Pages = append(report.Pages, result)
		if input.Progress != nil {
			input.Progress(i+1, total, fmt.Sprintf("%s: %s", result.Outcome, result.Title))
		}
	}
	report.Warnings = append(report.Warnings, run.warnings...)

	return report
}

type execution struct {
	input            ExecuteInput
	platform         platformport.DocumentationPlatform
	ids              map[documentdiscovery.DocumentPath]string
	now              func() time.Time
	warnings         []string
	warnedNoRenderer bool
}

func (run *execution) warn(format string, args ...any) {
	run.warnings = append(run.warnings, fmt.Sprintf(format, args...))
}

func (run *execution) execute(ctx context.Context, action syncplanning.Action) PageResult {
	result := PageResult{Path: action.Path, Title: action.Title, Planned: action.Kind, PageID: action.PageID, Version: action.RemoteVersion, URL: action.URL}
	if err := ctx.Err(); err != nil {
		result.Outcome, result.Error = Skipped, err.Error()

		return result
	}
	switch action.Kind {
	case syncplanning.Unchanged:
		result.Outcome = Unchanged

		return result
	case syncplanning.Conflict:
		if !run.input.Options.Force {
			result.Outcome, result.Error = Skipped, action.Reason

			return result
		}
	case syncplanning.Orphan:
		if !run.input.Options.Prune {
			result.Outcome = Reported

			return result
		}
		if err := run.platform.TrashPage(ctx, action.PageID); err != nil {
			result.Outcome, result.Error = Failed, err.Error()

			return result
		}
		result.Outcome = Trashed

		return result
	}

	return run.write(ctx, action, result)
}

// write creates or updates the page, then uploads what changed and marks it.
func (run *execution) write(ctx context.Context, action syncplanning.Action, result PageResult) PageResult {
	page, prepared := run.input.Prepared.Pages[action.Path]
	if !prepared {
		result.Outcome, result.Error = Failed, "the file was not prepared for this sync"

		return result
	}
	parentID := action.ParentPageID
	if parentID == "" && action.ParentPath != "" {
		parentID = run.ids[action.ParentPath]
		if parentID == "" {
			result.Outcome, result.Error = Skipped, fmt.Sprintf("its parent page (%s) was not written", action.ParentPath)

			return result
		}
	}

	diagrams := run.renderDiagrams(ctx, action.Path, page)
	images := make(map[string]string, len(diagrams))
	for source, diagram := range diagrams {
		images[source] = diagram.name
	}
	body := platformport.Document{Blocks: withDiagramImages(page.Converted.Document.Blocks, images)}

	var remote platformport.RemotePage
	var err error
	if action.Kind == syncplanning.Create {
		remote, err = run.platform.CreatePage(ctx, platformport.NewPage{Space: run.input.Space, ParentID: parentID, Title: action.Title, Body: body})
	} else {
		remote, err = run.platform.UpdatePage(ctx, platformport.PageUpdate{
			ID: action.PageID, ExpectedVersion: action.RemoteVersion, Title: action.Title, ParentID: parentID, Body: body,
			Message: "Synced by Lore Master from " + string(action.Path),
		})
	}
	if err != nil {
		result.Outcome, result.Error = Failed, describe(err)

		return result
	}
	result.Outcome, result.PageID, result.Version, result.URL = Written, remote.ID, remote.Version, remote.URL
	if action.Kind == syncplanning.Create || action.Kind == syncplanning.Adopt {
		if err := run.platform.MarkPage(ctx, remote.ID, string(action.Path)); err != nil {
			run.warn("%s: the page was written but could not be marked as synced (%v); prune will not see it", action.Path, err)
		}
	}

	attachments := run.uploadFiles(ctx, action, page, remote.ID)
	for _, diagram := range diagrams {
		if _, err := run.platform.UploadFile(ctx, remote.ID, platformport.File{Name: diagram.name, ContentType: "image/svg+xml", Content: diagram.svg}); err != nil {
			run.warn("%s: the diagram %s could not be uploaded (%v)", action.Path, diagram.name, err)
		}
	}
	result.Annotation = run.annotation(page, remote, parentID, action.ContentHash, attachments)

	return result
}

// uploadFiles uploads each attachment whose hash differs from the one recorded at the
// last sync, or every one on a page new to the sync, and returns the hashes the page
// now carries. A failed upload is left out so the next sync tries again.
func (run *execution) uploadFiles(ctx context.Context, action syncplanning.Action, page PreparedPage, pageID string) map[string]string {
	recorded := map[string]string{}
	fresh := action.Kind == syncplanning.Create || action.Kind == syncplanning.Adopt
	for _, file := range page.Files {
		if !fresh && page.Annotation != nil && page.Annotation.Attachments[file.Name] == file.Hash {
			recorded[file.Name] = file.Hash

			continue
		}
		content, err := run.input.Read(file.Path)
		if err != nil {
			run.warn("%s: the image %s could not be read (%v)", action.Path, file.Path, err)

			continue
		}
		contentType := mime.TypeByExtension(path.Ext(file.Name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if _, err := run.platform.UploadFile(ctx, pageID, platformport.File{Name: file.Name, ContentType: contentType, Content: content}); err != nil {
			run.warn("%s: the image %s could not be uploaded (%v)", action.Path, file.Path, err)

			continue
		}
		recorded[file.Name] = attachmentHash(content)
	}
	if len(recorded) == 0 {
		return nil
	}

	return recorded
}

type renderedDiagram struct {
	name string
	svg  []byte
}

// renderDiagrams draws the page's Mermaid diagrams when the output shows them as
// images. A diagram that cannot be drawn stays as code, with a warning.
func (run *execution) renderDiagrams(ctx context.Context, documentPath documentdiscovery.DocumentPath, page PreparedPage) map[string]renderedDiagram {
	if run.input.Output.MermaidMode != "image" {
		return nil
	}
	sources := mermaidSources(page.Converted.Document.Blocks)
	if len(sources) == 0 {
		return nil
	}
	if run.input.Renderer == nil {
		if !run.warnedNoRenderer {
			run.warnedNoRenderer = true
			run.warn("no diagram renderer is available, so Mermaid diagrams are shown as code")
		}

		return nil
	}
	diagrams := map[string]renderedDiagram{}
	for _, source := range sources {
		svg, err := run.input.Renderer.Render(ctx, "mermaid", source)
		if err != nil {
			run.warn("%s: a Mermaid diagram could not be drawn (%v); it is shown as code", documentPath, err)

			continue
		}
		diagrams[source] = renderedDiagram{name: "mermaid-" + attachmentHash([]byte(source))[len("sha256:"):][:12] + ".svg", svg: svg}
	}

	return diagrams
}

// annotation is the file's annotation after a successful write: what the author set
// (title and parent overrides, unknown keys) is kept, and the sync's own fields are
// replaced.
func (run *execution) annotation(page PreparedPage, remote platformport.RemotePage, parentID string, contentHash string, attachments map[string]string) *syncannotation.Annotation {
	var annotation syncannotation.Annotation
	if page.Annotation != nil {
		annotation = *page.Annotation
		annotation.Unknown = append([]syncannotation.Field(nil), page.Annotation.Unknown...)
	}
	output := run.input.Output
	annotation.Platform, annotation.BaseURL, annotation.Space = output.Platform, output.BaseURL, output.Space
	annotation.PageID, annotation.ParentID, annotation.Version = remote.ID, parentID, remote.Version
	annotation.ContentHash = contentHash
	annotation.Attachments = maps.Clone(attachments)
	annotation.SyncedAt = run.now().UTC().Truncate(time.Second)

	return &annotation
}

// describe words a platform error for the report.
func describe(err error) string {
	var conflict *platformport.VersionConflictError
	var taken *platformport.TitleTakenError
	var missing *platformport.PageNotFoundError
	switch {
	case errors.As(err, &conflict):
		return fmt.Sprintf("the page was edited on the platform while syncing (expected version %d); sync again to see the conflict", conflict.ExpectedVersion)
	case errors.As(err, &taken):
		return fmt.Sprintf("another page is already titled %q", taken.Title)
	case errors.As(err, &missing):
		return fmt.Sprintf("page %s no longer exists; sync again to create it", missing.ID)
	}

	return err.Error()
}
