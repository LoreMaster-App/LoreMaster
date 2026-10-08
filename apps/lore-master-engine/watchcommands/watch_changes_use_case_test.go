package watchcommands

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

var quick = Options{Interval: 5 * time.Millisecond, Quiet: 100 * time.Millisecond, MaxBackoff: 400 * time.Millisecond}

// script feeds changes to the loop, and records what it was asked to apply.
type script struct {
	mu      sync.Mutex
	queue   [][]string
	batches [][]string
	failing int
	events  []string
}

func (s *script) source(context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return nil, nil
	}
	next := s.queue[0]
	s.queue = s.queue[1:]

	return next, nil
}

func (s *script) apply(_ context.Context, changed []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches = append(s.batches, slices.Clone(changed))
	if s.failing > 0 {
		s.failing--

		return errors.New("confluence is down")
	}

	return nil
}

func (s *script) push(changed ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, changed)
}

func (s *script) applied() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.batches)
}

func (s *script) Changes([]string) {}
func (s *script) Applied([]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "applied")
}
func (s *script) Failed(_ []string, _ error, retryIn time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, "failed:"+retryIn.String())
}

// run starts the loop and stops it after d.
func run(t *testing.T, s *script, d time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	if err := Watch(ctx, s.source, s.apply, quick, s); err != nil {
		t.Fatal(err)
	}
}

func TestAQuietWorkspaceAppliesNothing(t *testing.T) {
	s := &script{}

	run(t, s, 300*time.Millisecond)

	if len(s.applied()) != 0 {
		t.Fatalf("batches %v", s.applied())
	}
}

func TestTwoQuickEditsAreOneBatch(t *testing.T) {
	s := &script{}
	go func() {
		s.push("b.md")
		time.Sleep(10 * time.Millisecond)
		s.push("a.md", "b.md")
	}()

	run(t, s, 600*time.Millisecond)

	if got := s.applied(); len(got) != 1 || !slices.Equal(got[0], []string{"a.md", "b.md"}) {
		t.Fatalf("batches %v", got)
	}
}

func TestEditsFarApartAreSeparateBatches(t *testing.T) {
	s := &script{}
	go func() {
		s.push("a.md")
		time.Sleep(350 * time.Millisecond)
		s.push("b.md")
	}()

	run(t, s, 1000*time.Millisecond)

	if got := s.applied(); len(got) != 2 || !slices.Equal(got[0], []string{"a.md"}) || !slices.Equal(got[1], []string{"b.md"}) {
		t.Fatalf("batches %v", got)
	}
}

func TestAFailedBatchIsRetriedWithItsFilesAndAGrowingWait(t *testing.T) {
	s := &script{failing: 2}
	s.push("a.md")

	run(t, s, 1500*time.Millisecond)

	got := s.applied()
	if len(got) != 3 {
		t.Fatalf("batches %v", got)
	}
	for _, batch := range got {
		if !slices.Equal(batch, []string{"a.md"}) {
			t.Fatalf("batches %v", got)
		}
	}
	if !slices.Equal(s.events, []string{"failed:100ms", "failed:200ms", "applied"}) {
		t.Fatalf("events %v", s.events)
	}
}

func TestChangesDuringAFailureJoinTheRetry(t *testing.T) {
	s := &script{failing: 1}
	s.push("a.md")
	go func() {
		time.Sleep(150 * time.Millisecond)
		s.push("b.md")
	}()

	run(t, s, 1200*time.Millisecond)

	got := s.applied()
	if len(got) < 2 || !slices.Equal(got[len(got)-1], []string{"a.md", "b.md"}) {
		t.Fatalf("batches %v", got)
	}
}

func TestASourceThatFailsEndsTheWatch(t *testing.T) {
	broken := errors.New("cannot read the workspace")

	err := Watch(context.Background(), func(context.Context) ([]string, error) { return nil, broken }, (&script{}).apply, quick, &script{})

	if !errors.Is(err, broken) {
		t.Fatalf("error %v", err)
	}
}

func TestBackoffDoublesFromQuietAndIsCapped(t *testing.T) {
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond, 400 * time.Millisecond}
	for i, expected := range want {
		if got := backoff(quick, i+1); got != expected {
			t.Errorf("failure %d: %v, want %v", i+1, got, expected)
		}
	}
}
