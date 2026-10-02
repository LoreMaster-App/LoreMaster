package syncplanning

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documenttree"
	"lore-master/libs/markdown-workspace/syncannotation"
)

// Input is what a plan is made from.
type Input struct {
	Tree   documenttree.DocumentTree
	Output workspacesettings.Output
	// SpaceID is the space's id, needed by platforms that create by id.
	SpaceID string
	// Scope limits the plan to these files and the ancestors they need created; empty
	// plans everything, orphans included.
	Scope []documentdiscovery.DocumentPath
	// Rendered is, per document, what its page shows beyond the Markdown, so a page
	// still updates when that changes and the file does not. A document missing from
	// the map shows nothing beyond it. Nil skips the comparison.
	Rendered map[documentdiscovery.DocumentPath]RenderedPage
}

// RenderedPage is what a page shows as converted now: the hash of its body, which
// follows links into other files, and its attachments (name → content hash).
type RenderedPage struct {
	RenderHash  string
	Attachments map[string]string
	// LinkedPages are the documents the page links to.
	LinkedPages []documentdiscovery.DocumentPath
}

// PlanSync plans a sync of the tree to one output, reading the platform but never
// writing to it. Files are visited parents first:
//
//   - a file without an annotation for this output is created, unless its title is
//     taken. A page this sync marked that no annotation claims is the file's own,
//     whose annotation was lost: it is taken back. Any other page with the title is
//     left to titleCollision (adopt, or an error naming the page);
//   - an annotated file whose page is gone is created again, with a warning;
//   - an annotated file whose page changed remotely since the last sync is a conflict;
//   - otherwise its content hash, parent and title decide update, move, rename or
//     unchanged.
//
// Then every page the sync marked under the output's parent with no file pointing at
// it is an orphan; a page a person made is never marked, so it is never one. Duplicate titles in the workspace are errors before any request.
// Only platform failures are returned as an error; everything about the workspace is
// in the plan.
func PlanSync(ctx context.Context, platform platformport.DocumentationPlatform, input Input) (SyncPlan, error) {
	var plan SyncPlan
	titles, err := documenttree.PageTitles(input.Tree, input.Output.TitlePrefix)
	if err != nil {
		plan.Errors = append(plan.Errors, err.Error())

		return plan, nil
	}
	plan.Warnings = append(plan.Warnings, input.Tree.Warnings...)

	// The pages this sync marked under the parent are read once: they are what a file
	// that lost its annotation is matched back to, and what is left over is orphaned.
	marked, err := platform.ListMarkedDescendants(ctx, input.Output.ParentPageID)
	if err != nil {
		return SyncPlan{}, err
	}
	run := &planner{
		ctx: ctx, platform: platform, input: input, titles: titles,
		space:     platformport.SpaceRef{ID: input.SpaceID, Key: input.Output.Space},
		pageIDs:   map[documentdiscovery.DocumentPath]string{},
		marked:    map[string]bool{},
		annotated: map[string]bool{},
	}
	for _, page := range marked {
		run.marked[page.ID] = true
	}
	input.Tree.Walk(func(node *documenttree.TreeNode, _ int) {
		if annotation := node.Document.Annotation; annotation != nil && annotation.PageID != "" && sameTarget(annotation, input.Output) {
			run.annotated[annotation.PageID] = true
		}
	})

	claimed := map[string]bool{}
	var walkErr error
	input.Tree.Walk(func(node *documenttree.TreeNode, _ int) {
		if walkErr != nil {
			return
		}
		action, warning, planError, err := run.planDocument(node)
		if err != nil {
			walkErr = err

			return
		}
		if warning != "" {
			plan.Warnings = append(plan.Warnings, warning)
		}
		if planError != "" {
			plan.Errors = append(plan.Errors, planError)

			return
		}
		run.pageIDs[node.Document.Path] = action.PageID
		claimed[action.PageID] = true
		plan.Actions = append(plan.Actions, action)
	})
	if walkErr != nil {
		return SyncPlan{}, walkErr
	}

	if input.Output.LinkMode == "id" {
		relinkToNewPages(plan.Actions, input.Rendered)
	}

	if len(input.Scope) > 0 {
		plan.Actions = scoped(plan.Actions, input.Scope)

		return plan, nil
	}
	for _, page := range marked {
		if !claimed[page.ID] {
			plan.Actions = append(plan.Actions, Action{
				Kind: Orphan, PageID: page.ID, Title: page.Title, URL: page.URL, RemoteVersion: page.Version,
				Reason: "no file in the workspace points at this page any more; prune moves it to the trash",
			})
		}
	}

	return plan, nil
}

// planner holds what one plan needs while the tree is walked.
type planner struct {
	ctx      context.Context
	platform platformport.DocumentationPlatform
	input    Input
	space    platformport.SpaceRef
	titles   map[documentdiscovery.DocumentPath]string
	// pageIDs are the pages planned so far, for their children's parent.
	pageIDs map[documentdiscovery.DocumentPath]string
	// marked are the pages the sync marked under the parent; annotated are those an
	// annotation in the workspace points at.
	marked, annotated map[string]bool
}

func (run *planner) planDocument(node *documenttree.TreeNode) (Action, string, string, error) {
	ctx, platform, output := run.ctx, run.platform, run.input.Output
	titles, pageIDs := run.titles, run.pageIDs
	document := node.Document
	action := Action{
		Path: document.Path, Title: titles[document.Path], ContentHash: syncannotation.ContentHash(document.Body),
		ParentPageID: output.ParentPageID,
	}
	if parent := node.Decision.Parent; parent != nil {
		action.ParentPath = *parent
		action.ParentPageID = pageIDs[*parent]
	}

	annotation := document.Annotation
	if annotation == nil || annotation.PageID == "" || !sameTarget(annotation, output) {
		found, err := platform.FindPagesByTitle(ctx, run.space, action.Title)
		if err != nil {
			return Action{}, "", "", err
		}
		if page, ours := run.lostAnnotation(found); ours {
			action.Kind, action.PageID, action.RemoteVersion, action.URL = Adopt, page.ID, page.Version, page.URL
			action.Changes = []Change{ChangeContent, ChangeParent, ChangeTitle}
			action.Reason = "this file's page was synced before and its annotation is gone; the page is taken back"

			return action, fmt.Sprintf("%s: %s (%s)", document.Path, action.Reason, page.URL), "", nil
		}
		decision := decideCollision(output.TitleCollision, string(document.Path), action.Title, found)
		if decision.error != "" {
			return Action{}, "", decision.error, nil
		}
		action.Kind = decision.kind
		if decision.kind == Adopt {
			action.PageID, action.RemoteVersion, action.URL = decision.page.ID, decision.page.Version, decision.page.URL
			action.Changes = []Change{ChangeContent, ChangeParent, ChangeTitle}
		}

		return action, "", "", nil
	}

	remote, err := platform.GetPage(ctx, annotation.PageID)
	var missing *platformport.PageNotFoundError
	if errors.As(err, &missing) {
		action.Kind = Create
		action.Reason = fmt.Sprintf("page %s no longer exists; it will be created again", annotation.PageID)

		return action, fmt.Sprintf("%s: %s", document.Path, action.Reason), "", nil
	}
	if err != nil {
		return Action{}, "", "", err
	}
	action.PageID, action.RemoteVersion, action.URL = remote.ID, remote.Version, remote.URL
	renderedChanged := false
	if rendered := run.input.Rendered; rendered != nil {
		now := rendered[document.Path]
		renderedChanged = now.RenderHash != annotation.RenderHash || !maps.Equal(now.Attachments, annotation.Attachments)
	}
	action.Kind, action.Changes = detectChange(
		remoteState{annotationVersion: annotation.Version, annotationHash: annotation.ContentHash, remoteVersion: remote.Version, remoteParentID: remote.ParentID, remoteTitle: remote.Title},
		localState{contentHash: action.ContentHash, title: action.Title, parentPageID: action.ParentPageID, renderedChanged: renderedChanged},
	)
	if action.Kind == Conflict {
		action.Reason = fmt.Sprintf("the page was edited on the platform after the last sync (version %d, synced at %d); it was left alone", remote.Version, annotation.Version)
	}

	return action, "", "", nil
}

// sameTarget reports whether the annotation was written by a sync to this output. A
// file synced to another site or space starts afresh here rather than updating a page
// id that means nothing on this platform.
func sameTarget(annotation *syncannotation.Annotation, output workspacesettings.Output) bool {
	return (annotation.BaseURL == "" || annotation.BaseURL == output.BaseURL) && (annotation.Space == "" || annotation.Space == output.Space)
}

// scoped keeps the scoped files' actions and, for each, the ancestors it needs created
// (a create or adopt) so it has a parent; ancestors that already exist are left out.
func scoped(actions []Action, scope []documentdiscovery.DocumentPath) []Action {
	byPath := map[documentdiscovery.DocumentPath]Action{}
	for _, action := range actions {
		byPath[action.Path] = action
	}
	keep := map[documentdiscovery.DocumentPath]bool{}
	for _, path := range scope {
		keep[path] = true
		for parent := byPath[path].ParentPath; parent != ""; parent = byPath[parent].ParentPath {
			if kind := byPath[parent].Kind; kind == Create || kind == Adopt {
				keep[parent] = true
			}
		}
	}

	return slices.DeleteFunc(slices.Clone(actions), func(action Action) bool { return !keep[action.Path] })
}

// lostAnnotation finds the page a file without an annotation was synced to before: of
// the pages with its title, exactly one that this sync marked and that no annotation
// in the workspace claims. Workspace titles are unique ignoring case, so no two files
// can match the same page. A page a person made is never marked, so it never matches
// and titleCollision still guards it.
func (run *planner) lostAnnotation(found []platformport.RemotePage) (platformport.RemotePage, bool) {
	var ours []platformport.RemotePage
	for _, page := range found {
		if run.marked[page.ID] && !run.annotated[page.ID] {
			ours = append(ours, page)
		}
	}
	if len(ours) != 1 {
		return platformport.RemotePage{}, false
	}

	return ours[0], true
}

// relinkToNewPages updates, in id link mode, an otherwise unchanged page that links to
// a page this sync creates: its links carry page ids, and the new page's id is new
// even when its title is not (a page deleted remotely and created again).
func relinkToNewPages(actions []Action, rendered map[documentdiscovery.DocumentPath]RenderedPage) {
	created := map[documentdiscovery.DocumentPath]bool{}
	for _, action := range actions {
		if action.Kind == Create {
			created[action.Path] = true
		}
	}
	for i, action := range actions {
		if action.Kind != Unchanged {
			continue
		}
		for _, target := range rendered[action.Path].LinkedPages {
			if created[target] {
				actions[i].Kind, actions[i].Changes = Update, []Change{ChangeContent}
				actions[i].Reason = fmt.Sprintf("it links to %s, which gets a new page and so a new id", target)

				break
			}
		}
	}
}
