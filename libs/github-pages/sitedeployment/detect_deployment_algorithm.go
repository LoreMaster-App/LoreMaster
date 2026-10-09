package sitedeployment

import (
	"regexp"
	"sort"
	"strings"
)

const loreMasterAction = "loremaster-app/loremaster/actions/pages"

var (
	valueAfterKey = regexp.MustCompile(`^\s*(?:-\s*)?([A-Za-z_-]+)\s*:\s*['"]?([^'"#]*?)['"]?\s*(?:#.*)?$`)
	pushToBranch  = regexp.MustCompile(`git\s+push[^\n]*\b(gh-pages|pages)\b`)
)

// DetectDeployment reads the workflow files (path to text) and says how the repository
// deploys to GitHub Pages. An Actions-source deploy wins over a branch push when both are
// present, because that is the one that replaces the whole site; the note says so.
func DetectDeployment(workflows map[string]string) Deployment {
	paths := make([]string, 0, len(workflows))
	for path := range workflows {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var actions, branch Deployment
	var actionsFiles, branchFiles []string
	usesAction := false

	for _, path := range paths {
		text := workflows[path]
		lower := strings.ToLower(text)
		if strings.Contains(lower, loreMasterAction) {
			usesAction = true
		}
		lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

		if strings.Contains(lower, "actions/deploy-pages") || strings.Contains(lower, "actions/upload-pages-artifact") {
			actionsFiles = append(actionsFiles, path)
			if actions.ArtifactPath == "" {
				actions.ArtifactPath = withinStep(lines, "upload-pages-artifact", "path")
			}
		}
		if reads := branchDeploy(lines, lower); reads.found {
			branchFiles = append(branchFiles, path)
			if branch.Branch == "" {
				branch.Branch = reads.branch
			}
			branch.KeepsOtherFiles = branch.KeepsOtherFiles || reads.keeps
		}
	}

	result := Deployment{Kind: KindNone, UsesLoreMasterAction: usesAction}
	switch {
	case len(actionsFiles) > 0:
		result.Kind = KindActionsSource
		result.Workflows = actionsFiles
		result.ArtifactPath = strings.TrimRight(actions.ArtifactPath, "/")
		if len(branchFiles) > 0 {
			result.Notes = append(result.Notes, "A branch push to Pages was also found ("+strings.Join(branchFiles, ", ")+"); only one Pages source can be active, so the Actions deploy is assumed.")
		}
		if result.ArtifactPath == "" {
			result.Notes = append(result.Notes, "The folder the deploy uploads (upload-pages-artifact `path`) could not be read.")
		}
	case len(branchFiles) > 0:
		result.Kind = KindBranchPush
		result.Workflows = branchFiles
		result.Branch = branch.Branch
		result.KeepsOtherFiles = branch.KeepsOtherFiles
	}

	return result
}

type branchRead struct {
	found  bool
	branch string
	keeps  bool
}

// branchDeploy recognises the common ways a workflow writes a Pages branch.
func branchDeploy(lines []string, lower string) branchRead {
	var read branchRead
	switch {
	case strings.Contains(lower, "peaceiris/actions-gh-pages"):
		read.found = true
		read.branch = withinStep(lines, "peaceiris/actions-gh-pages", "publish_branch")
		read.keeps = strings.EqualFold(withinStep(lines, "peaceiris/actions-gh-pages", "keep_files"), "true")
	case strings.Contains(lower, "jamesives/github-pages-deploy-action"):
		read.found = true
		read.branch = withinStep(lines, "github-pages-deploy-action", "branch")
		read.keeps = strings.EqualFold(withinStep(lines, "github-pages-deploy-action", "clean"), "false")
	case pushToBranch.MatchString(lower):
		read.found = true
		if match := pushToBranch.FindStringSubmatch(lower); len(match) > 1 && match[1] == "gh-pages" {
			read.branch = "gh-pages"
		}
	}
	if read.found && read.branch == "" {
		read.branch = "gh-pages"
	}

	return read
}

// withinStep returns the value of key in the lines that follow the first line containing
// marker, stopping at the next step ("- " at a shallower or equal indent) or after a bounded
// number of lines.
func withinStep(lines []string, marker, key string) string {
	for i, line := range lines {
		if !strings.Contains(strings.ToLower(line), strings.ToLower(marker)) {
			continue
		}
		for j := i + 1; j < len(lines) && j <= i+12; j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") && strings.Contains(strings.ToLower(lines[j]), "uses:") {
				break
			}
			if match := valueAfterKey.FindStringSubmatch(lines[j]); match != nil && match[1] == key {
				return strings.TrimSpace(match[2])
			}
		}
	}

	return ""
}
