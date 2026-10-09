package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lore-master/libs/documentation-sync/workspacesettings"
	"lore-master/libs/github-pages/sitedeployment"
)

const sitePublishingPlanToolName = "site_publishing_plan"

const sitePublishingPlanToolDescription = "Works out how this repository deploys to GitHub Pages (from .github/workflows) and which LoreMaster setting puts the docs site beside that deploy instead of over it: build into the app's artifact folder, publish into a folder of the branch, or own the branch. Returns the .lore-master.yaml output and the workflow step to add, with the warnings that matter. Writes nothing; use it before editing a workflow or the settings."

func sitePublishingPlanTool() toolDescriptor {
	return toolDescriptor{
		Name:        sitePublishingPlanToolName,
		Description: sitePublishingPlanToolDescription,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func sitePublishingPlanResult(workspaceRoot string) toolCallResult {
	if workspaceRoot == "" {
		return noWorkspaceResult()
	}

	workflows, workflowDir := readWorkflows(workspaceRoot)
	deployment := sitedeployment.DetectDeployment(workflows)

	var outputs []sitedeployment.Output
	var notes []string
	loaded, err := workspacesettings.LoadSettings(workspaceRoot)
	if err != nil {
		notes = append(notes, fmt.Sprintf("%s could not be read: %v", workspacesettings.FileName, err))
	}
	for _, output := range loaded.Settings.Outputs {
		if output.Platform == "github-pages" {
			outputs = append(outputs, sitedeployment.Output{Repo: output.Repo, Branch: output.Branch, Path: output.Path})
		}
	}

	plan := sitedeployment.Recommend(deployment, outputs)
	plan.Warnings = append(notes, plan.Warnings...)

	return toolCallResult{
		Content:           []contentBlock{{Type: "text", Text: describePlan(plan, workflowDir)}},
		StructuredContent: plan,
	}
}

// readWorkflows reads the workflow files of the repository the workspace is in: the
// workspace may be a folder of a larger repository, so it looks up from the workspace to the
// directory holding .git. It returns the files by repository-relative path and the folder
// they were read from ("" when there are none).
func readWorkflows(workspaceRoot string) (map[string]string, string) {
	for dir := filepath.Clean(workspaceRoot); ; {
		workflowDir := filepath.Join(dir, ".github", "workflows")
		if entries, err := os.ReadDir(workflowDir); err == nil {
			found := map[string]string{}
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
					continue
				}
				if content, err := os.ReadFile(filepath.Join(workflowDir, name)); err == nil {
					found[".github/workflows/"+name] = string(content)
				}
			}

			return found, workflowDir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return map[string]string{}, ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return map[string]string{}, ""
		}
		dir = parent
	}
}

func describePlan(plan sitedeployment.Plan, workflowDir string) string {
	var text strings.Builder

	switch plan.Deployment.Kind {
	case sitedeployment.KindActionsSource:
		fmt.Fprintf(&text, "This repository deploys GitHub Pages from an artifact (%s)", strings.Join(plan.Deployment.Workflows, ", "))
		if plan.Deployment.ArtifactPath != "" {
			fmt.Fprintf(&text, ", uploading `%s`", plan.Deployment.ArtifactPath)
		}
		text.WriteString(". Each deploy replaces the whole site, so the docs must be built into that artifact.\n")
	case sitedeployment.KindBranchPush:
		fmt.Fprintf(&text, "This repository deploys GitHub Pages by pushing branch `%s` (%s).\n", plan.Deployment.Branch, strings.Join(plan.Deployment.Workflows, ", "))
	default:
		if workflowDir == "" {
			text.WriteString("No .github/workflows folder was found, so no Pages deploy is known.\n")
		} else {
			text.WriteString("No Pages deploy was found in the workflows.\n")
		}
	}
	for _, note := range plan.Deployment.Notes {
		fmt.Fprintf(&text, "Note: %s\n", note)
	}

	switch plan.Approach {
	case "build":
		fmt.Fprintf(&text, "\nRecommended: BUILD the docs into `%s` with the pages action (or `lore-master-engine pages build --out %s`).\n", plan.OutDir, plan.OutDir)
	case "publish":
		fmt.Fprintf(&text, "\nRecommended: PUBLISH into the `%s` folder of the branch by setting `path` on the output.\n", plan.Path)
	default:
		text.WriteString("\nRecommended: PUBLISH the site to the branch; LoreMaster owns it.\n")
	}

	if plan.OutputSnippet != "" {
		fmt.Fprintf(&text, "\n.lore-master.yaml output (under `outputs:`):\n```yaml\n%s```\n", plan.OutputSnippet)
	}
	if plan.WorkflowSnippet != "" {
		fmt.Fprintf(&text, "\nWorkflow step:\n```yaml\n%s```\n", plan.WorkflowSnippet)
	}
	if len(plan.Warnings) > 0 {
		text.WriteString("\nWatch for:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(&text, "- %s\n", warning)
		}
	}

	return strings.TrimRight(text.String(), "\n")
}
