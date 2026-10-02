package syncexecution

// Outcome is what happened to one page.
type Outcome string

// Outcomes.
const (
	// Written: the page was created or changed on the platform.
	Written Outcome = "written"
	// Unchanged: nothing needed doing.
	Unchanged Outcome = "unchanged"
	// Skipped: a conflict not forced, or a page whose parent could not be written.
	Skipped Outcome = "skipped"
	// Reported: an orphan left alone because prune was not asked for.
	Reported Outcome = "reported"
	// Trashed: an orphan moved to the trash.
	Trashed Outcome = "trashed"
	// Failed: the platform refused; Error says why.
	Failed Outcome = "failed"
)
