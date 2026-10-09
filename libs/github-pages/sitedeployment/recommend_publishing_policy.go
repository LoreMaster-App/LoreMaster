package sitedeployment

import (
	"fmt"
	"strings"
)

const (
	defaultBranch   = "gh-pages"
	docsFolder      = "docs"
	defaultBuildDir = "dist"
	actionRef       = "LoreMaster-App/LoreMaster/actions/pages@main"
)

// Recommend turns what the repository deploys today and what .lore-master.yaml says (outputs
// is every github-pages output, possibly none) into the one approach that puts the docs site
// beside the existing deploy, with the snippets to apply it and the warnings that matter.
func Recommend(deployment Deployment, outputs []Output) Plan {
	plan := Plan{Deployment: deployment}
	current := firstOutput(outputs)

	switch deployment.Kind {
	case KindActionsSource:
		plan.Approach = "build"
		base := deployment.ArtifactPath
		if base == "" {
			base = defaultBuildDir
		}
		plan.OutDir = base + "/" + docsFolder
		plan.OutputSnippet = pagesOutputSnippet("", "", "")
		plan.WorkflowSnippet = buildStep(plan.OutDir)
		plan.Warnings = append(plan.Warnings,
			"Put the step after the app is built and before upload-pages-artifact, so the docs are inside the uploaded folder; the app stays at the site root and the docs appear under /"+docsFolder+"/.",
			"Add the docs folders and .lore-master.yaml to the workflow's `paths:` filter, or a docs-only change will not redeploy.",
			"Do not publish this repository's docs to a branch: with the Actions source Pages ignores branches, and a branch publish would be invisible.")
		if deployment.ArtifactPath == "" {
			plan.Warnings = append(plan.Warnings, "The uploaded folder was not found in the workflow, so "+plan.OutDir+" is a guess; use the folder passed to upload-pages-artifact.")
		}
		if deployment.UsesLoreMasterAction {
			plan.Warnings = append(plan.Warnings, "A workflow already uses the LoreMaster pages action; check its `out` is inside the uploaded folder.")
		}
	case KindBranchPush:
		plan.Approach = "publish"
		plan.Path = docsFolder
		branch := deployment.Branch
		if branch == "" {
			branch = defaultBranch
		}
		plan.OutputSnippet = pagesOutputSnippet(branch, plan.Path, "")
		plan.WorkflowSnippet = "- uses: " + actionRef + "\n  with:\n    mode: publish\n"
		plan.Warnings = append(plan.Warnings,
			"The job needs `permissions: contents: write`.",
			"Publishing without `path` to a branch that holds the app is refused and leaves the branch untouched; `path` makes LoreMaster replace only that folder.")
		if !deployment.KeepsOtherFiles {
			plan.Warnings = append(plan.Warnings, "The app's own deploy does not say it keeps files it did not write (peaceiris `keep_files: true`, JamesIves `clean: false`), so its next deploy will delete the docs folder.")
		}
		if current != nil && current.Path == "" {
			plan.Warnings = append(plan.Warnings, "The existing github-pages output has no `path`, so a publish would be refused here; set `path: "+docsFolder+"`.")
		}
	default:
		plan.Approach = "publish-root"
		plan.OutputSnippet = pagesOutputSnippet("", "", "")
		plan.WorkflowSnippet = "- uses: " + actionRef + "\n  with:\n    mode: publish\n"
		plan.Warnings = append(plan.Warnings,
			"No Pages deploy was found in .github/workflows, so LoreMaster can own the gh-pages branch; set the repository's Pages source to that branch.",
			"If something else deploys to Pages outside these workflows, use `path` (or build into its artifact) instead.")
	}

	if len(outputs) == 0 {
		plan.Warnings = append(plan.Warnings, "There is no github-pages output in .lore-master.yaml yet; add the one shown.")
	}
	if len(outputs) > 1 {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("There are %d github-pages outputs; pass `output:` (a position) to the action so it builds the right one.", len(outputs)))
	}

	return plan
}

func firstOutput(outputs []Output) *Output {
	if len(outputs) == 0 {
		return nil
	}

	return &outputs[0]
}

func pagesOutputSnippet(branch, path, repo string) string {
	var out strings.Builder
	out.WriteString("- platform: github-pages\n  direction: to-platform\n")
	if repo != "" {
		fmt.Fprintf(&out, "  repo: %s\n", repo)
	}
	if branch != "" {
		fmt.Fprintf(&out, "  branch: %s\n", branch)
	}
	if path != "" {
		fmt.Fprintf(&out, "  path: %s\n", path)
	}
	out.WriteString("  content:\n    - type: markdown\n      roots: [\"docs\"]\n      template: default\n")

	return out.String()
}

func buildStep(outDir string) string {
	return "- uses: " + actionRef + "\n  with:\n    out: " + outDir + "\n"
}
