package httptransport

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		value string
		want  time.Duration
		ok    bool
	}{
		{"0", 0, true},
		{"120", 2 * time.Minute, true},
		{"Fri, 02 Oct 2026 12:00:30 GMT", 30 * time.Second, true},
		{"Fri, 02 Oct 2026 11:00:00 GMT", 0, true},
		{"", 0, false},
		{"soon", 0, false},
	}
	for _, tc := range cases {
		got, ok := retryAfter(tc.value, now)
		if got != tc.want || ok != tc.ok {
			t.Errorf("retryAfter(%q) = %v, %v; want %v, %v", tc.value, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDecideRetryStopsAtTheLastAttempt(t *testing.T) {
	header := http.Header{"Retry-After": {"1"}}
	if decideRetry(http.MethodGet, 429, false, header, maxAttempts, time.Now()).retry {
		t.Fatal("no retry after the last attempt")
	}
	if !decideRetry(http.MethodGet, 429, false, http.Header{}, 1, time.Now()).retry {
		t.Fatal("a 429 without Retry-After still backs off and retries")
	}
}

func TestBackoffStaysWithinItsCeiling(t *testing.T) {
	for attempt := 1; attempt <= 10; attempt++ {
		ceiling := min(baseBackoff<<(attempt-1), maxBackoff)
		for range 100 {
			if wait := backoff(attempt); wait <= 0 || wait > ceiling {
				t.Fatalf("attempt %d: %v outside (0, %v]", attempt, wait, ceiling)
			}
		}
	}
}
