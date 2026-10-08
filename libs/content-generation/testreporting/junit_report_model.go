package testreporting

// Status is how one test ended.
type Status string

// Statuses.
const (
	Passed  Status = "passed"
	Failed  Status = "failed"
	Errored Status = "error"
	Skipped Status = "skipped"
)

// Case is one test.
type Case struct {
	Name      string
	ClassName string
	// Seconds is how long it ran.
	Seconds float64
	Status  Status
	// Message is the failure's or error's one-line message, Details its text (usually a
	// stack trace). Both are empty for a test that passed or was skipped without a reason.
	Message string
	Details string
}

// Suite is a named group of tests read from one report file.
type Suite struct {
	Name string
	// Source is the workspace-relative path of the report the suite came from.
	Source    string
	Timestamp string
	Cases     []Case
	// Totals come from the cases, or from the suite's own attributes when the report lists
	// no cases (a summarised suite).
	Tests, Failures, Errors, SkippedCount int
	Seconds                               float64
}

// Passes is the number of tests that passed.
func (s Suite) Passes() int {
	return max(0, s.Tests-s.Failures-s.Errors-s.SkippedCount)
}
