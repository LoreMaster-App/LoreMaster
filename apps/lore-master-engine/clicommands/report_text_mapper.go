package clicommands

import (
	"fmt"
	"slices"
	"strings"

	"lore-master/apps/lore-master-engine/rpcprotocol"
)

// generatorLines say what each generator did, with its warnings and errors beneath it.
func generatorLines(runs []rpcprotocol.GeneratorRun) []string {
	var lines []string
	for _, run := range runs {
		label := fmt.Sprintf("%s -> %s", run.Type, run.Output)
		if run.Error != "" {
			lines = append(lines, label+": failed", "  error: "+run.Error)

			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %d written, %d unchanged, %d removed", label, len(run.Written), len(run.Unchanged), len(run.Removed)))
		for _, warning := range run.Warnings {
			lines = append(lines, "  warning: "+warning)
		}
	}

	return lines
}

// treeLines show the pages as the tree they will be, one per line, with the local status.
func treeLines(nodes []rpcprotocol.TreeNode) []string {
	lines := make([]string, 0, len(nodes))
	for _, node := range nodes {
		status := node.Status
		if status == "" {
			status = "-"
		}
		lines = append(lines, fmt.Sprintf("%s%-13s %s (%s)", strings.Repeat("  ", node.Depth), "["+status+"]", node.Title, node.Path))
	}

	return lines
}

// countLine is "create 2, update 1, unchanged 7" in a fixed order, skipping zeros.
func countLine(counts map[string]int) string {
	kinds := make([]string, 0, len(counts))
	for kind, count := range counts {
		if count > 0 {
			kinds = append(kinds, kind)
		}
	}
	slices.Sort(kinds)
	parts := make([]string, len(kinds))
	for i, kind := range kinds {
		parts[i] = fmt.Sprintf("%s %d", kind, counts[kind])
	}
	if len(parts) == 0 {
		return "nothing to do"
	}

	return strings.Join(parts, ", ")
}

// planLines list what a plan would do to each page that is not unchanged.
func planLines(plan rpcprotocol.SyncPlanResult) []string {
	var lines []string
	for _, action := range plan.Actions {
		if action.Kind == "unchanged" {
			continue
		}
		name := action.Path
		if name == "" {
			name = action.Title
		}
		line := fmt.Sprintf("  %-12s %s", action.Kind, name)
		if action.Reason != "" {
			line += "  (" + action.Reason + ")"
		}
		lines = append(lines, line)
	}

	return lines
}

// outcomeLines count what a sync did to the pages and name the ones that failed.
func outcomeLines(result rpcprotocol.SyncExecuteResult) []string {
	counts := map[string]int{}
	var failures []string
	for _, page := range result.Pages {
		counts[page.Outcome]++
		if page.Outcome == "failed" {
			failures = append(failures, fmt.Sprintf("  failed: %s: %s", pageName(page), page.Error))
		}
	}

	return append([]string{"  " + countLine(counts)}, failures...)
}

func pageName(page rpcprotocol.PageOutcome) string {
	if page.Path != "" {
		return page.Path
	}

	return page.Title
}
