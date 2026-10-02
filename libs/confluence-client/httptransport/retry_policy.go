package httptransport

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	maxAttempts = 5
	// maxRetryAfter is the longest Retry-After the transport will sleep through; a
	// longer one is returned as an error rather than freezing a sync for minutes.
	maxRetryAfter = 2 * time.Minute
	baseBackoff   = 500 * time.Millisecond
	maxBackoff    = 8 * time.Second
)

// retryDecision says whether to try again and after how long.
type retryDecision struct {
	retry bool
	wait  time.Duration
}

// decideRetry is the whole policy. A 429 is always safe to retry, POST included,
// because Confluence did not process the request; the wait is its Retry-After. A 502,
// 503 or 504, or a transport error, is retried with jittered exponential backoff only
// for idempotent methods: a POST that timed out may have created the page, and
// repeating it could create another, so the use case decides.
func decideRetry(method string, status int, transportFailed bool, header http.Header, attempt int, now time.Time) retryDecision {
	if attempt >= maxAttempts {
		return retryDecision{}
	}
	if status == http.StatusTooManyRequests {
		wait, ok := retryAfter(header.Get("Retry-After"), now)
		if !ok {
			wait = backoff(attempt)
		}
		if wait > maxRetryAfter {
			return retryDecision{}
		}

		return retryDecision{retry: true, wait: wait}
	}
	idempotent := method != http.MethodPost && method != http.MethodPatch
	transient := transportFailed || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
	if idempotent && transient {
		return retryDecision{retry: true, wait: backoff(attempt)}
	}

	return retryDecision{}
}

// retryAfter reads either form RFC 9110 allows: delay-seconds or an HTTP-date.
func retryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return max(time.Duration(seconds)*time.Second, 0), true
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(at.Sub(now), 0), true
	}

	return 0, false
}

// backoff is full-jitter exponential backoff: a random wait up to 500 ms × 2^(attempt-1),
// capped at 8 s, so many clients retrying together do not hit the server in step.
func backoff(attempt int) time.Duration {
	ceiling := min(baseBackoff<<(attempt-1), maxBackoff)

	return time.Duration(rand.Int64N(int64(ceiling)) + 1)
}
