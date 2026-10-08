package changedetection

import "sort"

// Changed lists the files that were added, removed or whose content differs between two
// snapshots, sorted. A file that was only touched (same hash) is not changed.
func Changed(before Snapshot, after Snapshot) []string {
	var changed []string
	for path, state := range after.Files {
		if known, found := before.Files[path]; !found || known.Hash != state.Hash {
			changed = append(changed, path)
		}
	}
	for path := range before.Files {
		if _, found := after.Files[path]; !found {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)

	return changed
}
