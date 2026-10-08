package watchcommands

import (
	"context"
	"slices"
	"time"
)

// Options are the loop's timing.
type Options struct {
	// Interval is how often the source is asked for changes.
	Interval time.Duration
	// Quiet is how long nothing may change before the pending changes are applied, so a burst of
	// saves (a formatter, a branch switch, two quick edits) is one batch.
	Quiet time.Duration
	// MaxBackoff caps the wait before a failed batch is tried again; the wait starts at Quiet
	// and doubles with each failure.
	MaxBackoff time.Duration
}

// Source returns the files that changed since it was last asked.
type Source func(ctx context.Context) ([]string, error)

// Apply handles one batch of changed files (sorted, without repeats). While it runs nothing
// else does: changes made meanwhile wait in the source for the next batch.
type Apply func(ctx context.Context, changed []string) error

// Reporter is told what the loop does, for the user.
type Reporter interface {
	// Changes says files changed and a batch is waiting for quiet.
	Changes(changed []string)
	// Applied says a batch was handled.
	Applied(changed []string)
	// Failed says a batch could not be handled and when it will be tried again.
	Failed(changed []string, err error, retryIn time.Duration)
}

// Watch asks source for changes every Interval and applies them in one batch once nothing has
// changed for Quiet. A batch that fails is kept, joined by anything that changes later, and tried
// again after a growing wait (Quiet, twice that, ... up to MaxBackoff); a batch that succeeds
// resets the wait. It runs until ctx ends, which is not an error, but a source that fails is.
func Watch(ctx context.Context, source Source, apply Apply, options Options, reporter Reporter) error {
	pending := map[string]bool{}
	var lastChange, notBefore time.Time
	failures := 0

	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		changed, err := source(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return err
		}
		if len(changed) > 0 {
			for _, file := range changed {
				pending[file] = true
			}
			lastChange = time.Now()
			reporter.Changes(sorted(pending))
		}
		if len(pending) == 0 || time.Since(lastChange) < options.Quiet || time.Now().Before(notBefore) {
			continue
		}

		batch := sorted(pending)
		if err := apply(ctx, batch); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			failures++
			wait := backoff(options, failures)
			notBefore = time.Now().Add(wait)
			reporter.Failed(batch, err, wait)

			continue
		}
		clear(pending)
		failures, notBefore = 0, time.Time{}
		reporter.Applied(batch)
	}
}

func sorted(set map[string]bool) []string {
	files := make([]string, 0, len(set))
	for file := range set {
		files = append(files, file)
	}
	slices.Sort(files)

	return files
}

// backoff is Quiet doubled for each failure after the first, capped at MaxBackoff.
func backoff(options Options, failures int) time.Duration {
	wait := options.Quiet
	for i := 1; i < failures && wait < options.MaxBackoff; i++ {
		wait *= 2
	}

	return min(wait, max(options.MaxBackoff, options.Quiet))
}
