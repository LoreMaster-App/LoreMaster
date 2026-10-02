package syncplanning

import (
	"context"
	"errors"
	"fmt"
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
}

// PlanSync plans a sync of the tree to one output, reading the platform but never
// writing to it. Files are visited parents first:
//
//   - a file without an annotation for this output is created, unless its title is
//     taken (titleCollision decides: adopt, or an error naming the page);
//   - an annotated file whose page is gone is created again, with a warning;
//   - an annotated file whose page changed remotely since the last sync is a conflict;
//   - otherwise its content hash, parent and title decide update, move, rename or
//     unchanged.
//
// Then every page the sync marked under the output's parent with no file pointing at
// it is an orphan. Duplicate titles in the workspace are errors before any request.
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

	space := platformport.SpaceRef{ID: input.SpaceID, Key: input.Output.Space}
	pageIDs := map[documentdiscovery.DocumentPath]string{}
	claimed := map[string]bool{}
	var walkErr error
	input.Tree.Walk(func(node *documenttree.TreeNode, _ int) {
		if walkErr != nil {
			return
		}
		action, warning, planError, err := planDocument(ctx, platform, input.Output, space, node, titles, pageIDs)
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
		pageIDs[node.Document.Path] = action.PageID
		claimed[action.PageID] = true
		plan.Actions = append(plan.Actions, action)
	})
	if walkErr != nil {
		return SyncPlan{}, walkErr
	}

	if len(input.Scope) > 0 {
		plan.Actions = scoped(plan.Actions, input.Scope)

		return plan, nil
	}
	orphans, err := platform.ListMarkedDescendants(ctx, input.Output.ParentPageID)
	if err != nil {
		return SyncPlan{}, err
	}
	for _, page := range orphans {
		if !claimed[page.ID] {
			plan.Actions = append(plan.Actions, Action{
				Kind: Orphan, PageID: page.ID, Title: page.Title, URL: page.URL, RemoteVersion: page.Version,
				Reason: "no file in the workspace points at this page any more; prune moves it to the trash",
			})
		}
	}

	return plan, nil
}

func planDocument(
	ctx context.Context, platform platformport.DocumentationPlatform, output workspacesettings.Output, space platformport.SpaceRef,
	node *documenttree.TreeNode, titles map[documentdiscovery.DocumentPath]string, pageIDs map[documentdiscovery.DocumentPath]string,
) (Action, string, string, error) {
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
		found, err := platform.FindPagesByTitle(ctx, space, action.Title)
		if err != nil {
			return Action{}, "", "", err
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
	action.Kind, action.Changes = detectChange(
		remoteState{annotationVersion: annotation.Version, annotationHash: annotation.ContentHash, remoteVersion: remote.Version, remoteParentID: remote.ParentID, remoteTitle: remote.Title},
		localState{contentHash: action.ContentHash, title: action.Title, parentPageID: action.ParentPageID},
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
