package synccommands

import (
	"context"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/apps/lore-master-engine/sessionlifecycle"
	"lore-master/libs/documentation-sync/confluenceplatform"
	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/documentation-sync/syncexecution"
	"lore-master/libs/documentation-sync/syncplanning"
	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documenttree"
)

// PlanSync handles sync/plan: settings, discovery, parsing, tree, conversion and the
// plan, writing nothing anywhere.
func PlanSync(sessions *sessionlifecycle.Store, plans *PlanStore) rpcserver.Method {
	return func(ctx context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.SyncPlanParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.WorkspaceRoot) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "workspaceRoot must be an absolute path, got %q", params.WorkspaceRoot)
		}
		session, err := sessions.Get(params.SessionID)
		if err != nil {
			return nil, err
		}
		output, discovery, err := outputToSync(params, session.BaseURL)
		if err != nil {
			return nil, err
		}
		space, err := findSpace(ctx, session.Platform, output.Space)
		if err != nil {
			return nil, err
		}

		documents, warnings, problems, err := loadDocuments(ctx, params.WorkspaceRoot, output, discovery)
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
		}
		tree, err := documenttree.BuildTree(documents)
		if err != nil {
			problems = append(problems, err.Error())
		}
		titles, _ := documenttree.PageTitles(tree, output.TitlePrefix)
		prepared := syncexecution.PreparePages(documents, titles, fileReader(params.WorkspaceRoot))
		scope := make([]documentdiscovery.DocumentPath, len(params.Scope))
		for i, path := range params.Scope {
			scope[i] = documentdiscovery.DocumentPath(path)
		}
		plan, err := syncplanning.PlanSync(ctx, session.Platform, syncplanning.Input{
			Tree: tree, Output: output, SpaceID: space.ID, Scope: scope, Rendered: prepared.Rendered(),
		})
		if err != nil {
			return nil, sessionlifecycle.PlatformError(err)
		}
		plan.Errors = append(problems, plan.Errors...)
		plan.Warnings = append(append(warnings, plan.Warnings...), prepared.Warnings...)

		id := plans.put(&storedPlan{
			sessionID: session.ID, root: params.WorkspaceRoot, output: output, space: space, plan: plan, prepared: prepared,
		})

		return planResult(id, plan), nil
	}
}

// outputToSync loads and checks the settings and picks the output, which must belong
// to the session's site.
func outputToSync(params rpcprotocol.SyncPlanParams, sessionURL string) (workspacesettings.Output, workspacesettings.DiscoveryScope, error) {
	loaded, err := workspacesettings.LoadSettings(params.WorkspaceRoot)
	if err != nil {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if loaded.FirstSync {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s has no site, space, parent page or title prefix yet; run the first sync to choose them", workspacesettings.FileName)
	}
	if err := workspacesettings.Validate(loaded.Settings); err != nil {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidSettings, "%s", err.Error())
	}
	if params.Output < 0 || params.Output >= len(loaded.Settings.Outputs) {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d does not exist; %s has %d", params.Output, workspacesettings.FileName, len(loaded.Settings.Outputs))
	}
	output := loaded.Settings.Outputs[params.Output]
	if !confluenceplatform.SameSite(output.BaseURL, sessionURL) {
		return workspacesettings.Output{}, workspacesettings.DiscoveryScope{}, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "output %d syncs to %s but the session is open on %s", params.Output, output.BaseURL, sessionURL)
	}

	return output, loaded.Settings.DiscoveryScope(), nil
}

func findSpace(ctx context.Context, platform platformport.DocumentationPlatform, key string) (platformport.SpaceRef, error) {
	spaces, err := platform.ListSpaces(ctx)
	if err != nil {
		return platformport.SpaceRef{}, sessionlifecycle.PlatformError(err)
	}
	for _, space := range spaces {
		if space.Key == key {
			return platformport.SpaceRef{ID: space.ID, Key: space.Key}, nil
		}
	}

	return platformport.SpaceRef{}, rpcprotocol.Errorf(rpcprotocol.CodeNotFound, "the space %q does not exist or the session's user cannot see it", key)
}

func planResult(id string, plan syncplanning.SyncPlan) rpcprotocol.SyncPlanResult {
	// Actions is initialised so an empty plan marshals as [] rather than null, which the
	// editor's plan preview would otherwise call .map on.
	result := rpcprotocol.SyncPlanResult{PlanID: id, Counts: map[string]int{}, Actions: []rpcprotocol.PlanAction{}, Warnings: plan.Warnings, Errors: plan.Errors}
	for kind, count := range plan.Counts() {
		result.Counts[string(kind)] = count
	}
	for _, action := range plan.Actions {
		wire := rpcprotocol.PlanAction{
			Kind: string(action.Kind), Path: string(action.Path), Title: action.Title, PageID: action.PageID,
			URL: action.URL, ParentPath: string(action.ParentPath), Reason: action.Reason,
		}
		for _, change := range action.Changes {
			wire.Changes = append(wire.Changes, string(change))
		}
		result.Actions = append(result.Actions, wire)
	}

	return result
}

// planHasErrors refuses to execute a plan that has errors.
func planHasErrors(plan syncplanning.SyncPlan) error {
	if len(plan.Errors) == 0 {
		return nil
	}

	return rpcprotocol.Errorf(rpcprotocol.CodePlanHasErrors, "the plan has %d error(s); fix them and plan again", len(plan.Errors))
}
