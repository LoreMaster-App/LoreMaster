package sitepublish

import (
	"context"
	"sort"
	"strings"

	"lore-master/libs/github-pages/siterender"
)

// CheckSite answers "would a publish change anything?" without publishing. It stages the
// site exactly as PublishSite does, in a temporary clone, and reads the pending changes from
// git instead of committing and pushing them, so the answer and the publish never disagree.
// A folder holding another site is refused the same way a publish refuses it.
func CheckSite(ctx context.Context, opts PublishOptions, files []siterender.SiteFile) (CheckResult, error) {
	git := newGitClient()

	staged, err := stageSite(ctx, git, opts, files)
	if err != nil {
		return CheckResult{}, err
	}
	defer staged.cleanup()

	status, err := git.run(ctx, staged.dir, "status", "--porcelain", "--no-renames")
	if err != nil {
		return CheckResult{}, err
	}
	changes := parsePorcelain(status)

	return CheckResult{
		Branch: staged.branch, Remote: staged.remote, UpToDate: len(changes) == 0, Changes: changes, Files: len(files),
	}, nil
}

// parsePorcelain reads `git status --porcelain --no-renames` of a fully staged tree: each
// line is a two-letter state, a space and the path.
func parsePorcelain(status string) []SiteChange {
	var changes []SiteChange
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 4 {
			continue
		}
		path := strings.Trim(line[3:], "\"")
		switch line[0] {
		case 'A':
			changes = append(changes, SiteChange{Path: path, Kind: ChangeAdded})
		case 'D':
			changes = append(changes, SiteChange{Path: path, Kind: ChangeRemoved})
		default:
			changes = append(changes, SiteChange{Path: path, Kind: ChangeModified})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })

	return changes
}
