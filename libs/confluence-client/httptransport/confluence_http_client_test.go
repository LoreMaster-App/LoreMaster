package httptransport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const secretHeader = "Bearer s3cr3t-token"

// newTestClient points a Client at handler under the context path /wiki, with sleeps
// recorded instead of slept and logs captured at debug level.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *[]time.Duration, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	var logs bytes.Buffer
	client, err := New(Options{
		BaseURL:             server.URL + "/wiki/",
		AuthorizationHeader: secretHeader,
		Logger:              slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	if err != nil {
		t.Fatal(err)
	}
	var waits []time.Duration
	client.sleep = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)

		return ctx.Err()
	}
	client.now = func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }

	return client, &waits, &logs
}

func TestGetJSONSendsAuthorizationAndJoinsThePath(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/rest/api/space" || r.URL.Query().Get("type") != "global" {
			t.Errorf("request %s", r.URL)
		}
		if r.Header.Get("Authorization") != secretHeader || r.Header.Get("Accept") != "application/json" {
			t.Errorf("headers %v", r.Header)
		}
		_, _ = io.WriteString(w, `{"key":"ENG"}`)
	})
	var out struct{ Key string }
	if err := client.GetJSON(context.Background(), "/rest/api/space", url.Values{"type": {"global"}}, &out); err != nil || out.Key != "ENG" {
		t.Fatalf("out %+v, err %v", out, err)
	}
}

func TestRetries(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		responses []func(http.ResponseWriter)
		attempts  int
		waits     []time.Duration // nil: only count them
		status    int             // 0: success
	}{
		{"429 honours Retry-After seconds", http.MethodGet, []func(http.ResponseWriter){status(429, "Retry-After", "3"), status(200)}, 2, []time.Duration{3 * time.Second}, 0},
		{"429 honours an HTTP-date", http.MethodGet, []func(http.ResponseWriter){status(429, "Retry-After", "Fri, 02 Oct 2026 12:00:07 GMT"), status(200)}, 2, []time.Duration{7 * time.Second}, 0},
		{"429 is retried for POST too", http.MethodPost, []func(http.ResponseWriter){status(429, "Retry-After", "1"), status(200)}, 2, []time.Duration{time.Second}, 0},
		{"a Retry-After beyond two minutes is an error, not a freeze", http.MethodGet, []func(http.ResponseWriter){status(429, "Retry-After", "600")}, 1, []time.Duration{}, 429},
		{"503 on GET backs off until it succeeds", http.MethodGet, []func(http.ResponseWriter){status(503), status(502), status(200)}, 3, nil, 0},
		{"503 on GET gives up after five attempts", http.MethodGet, []func(http.ResponseWriter){status(503), status(503), status(503), status(503), status(503)}, 5, nil, 503},
		{"503 on PUT is retried", http.MethodPut, []func(http.ResponseWriter){status(504), status(200)}, 2, nil, 0},
		{"503 on POST is not retried: the page may exist", http.MethodPost, []func(http.ResponseWriter){status(503)}, 1, []time.Duration{}, 503},
		{"400 is not retried", http.MethodGet, []func(http.ResponseWriter){status(400)}, 1, []time.Duration{}, 400},
		{"500 is not retried", http.MethodGet, []func(http.ResponseWriter){status(500)}, 1, []time.Duration{}, 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			client, waits, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodGet && string(body) != `{"n":1}` {
					t.Errorf("attempt %d sent body %q", calls.Load()+1, body)
				}
				tc.responses[calls.Add(1)-1](w)
			})
			var err error
			switch tc.method {
			case http.MethodGet:
				err = client.GetJSON(context.Background(), "/x", nil, nil)
			case http.MethodPost:
				err = client.PostJSON(context.Background(), "/x", map[string]int{"n": 1}, nil)
			case http.MethodPut:
				err = client.PutJSON(context.Background(), "/x", map[string]int{"n": 1}, nil)
			}
			if int(calls.Load()) != tc.attempts {
				t.Fatalf("attempts %d, want %d", calls.Load(), tc.attempts)
			}
			if tc.waits != nil && !equalDurations(*waits, tc.waits) {
				t.Fatalf("waits %v, want %v", *waits, tc.waits)
			}
			if tc.waits == nil {
				for _, wait := range *waits {
					if wait <= 0 || wait > maxBackoff {
						t.Fatalf("backoff %v outside (0, %v]", wait, maxBackoff)
					}
				}
			}
			var apiError *APIError
			switch {
			case tc.status == 0 && err != nil:
				t.Fatalf("unexpected error %v", err)
			case tc.status != 0 && (!errors.As(err, &apiError) || apiError.Status != tc.status):
				t.Fatalf("error %v, want status %d", err, tc.status)
			}
		})
	}
}

func status(code int, header ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		for i := 0; i+1 < len(header); i += 2 {
			w.Header().Set(header[i], header[i+1])
		}
		w.WriteHeader(code)
		_, _ = io.WriteString(w, `{}`)
	}
}

func equalDurations(a []time.Duration, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestAPIErrorKeepsConfluencesMessageCut(t *testing.T) {
	long := strings.Repeat("x", 5000)
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"A page with this title already exists"}`+long)
	})
	err := client.PostJSON(context.Background(), "/api/v2/pages", map[string]string{}, nil)
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Path != "/wiki/api/v2/pages" || apiError.Method != "POST" {
		t.Fatalf("error %#v", err)
	}
	if !strings.HasPrefix(apiError.Body, `{"message":"A page with this title already exists"}`) || len(apiError.Body) > maxErrorBody+len("…") {
		t.Fatalf("body of %d bytes: %.80s", len(apiError.Body), apiError.Body)
	}
}

type failingTransport struct{ calls atomic.Int32 }

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.calls.Add(1)

	return nil, errors.New("connection reset by peer")
}

func TestTransportErrorsRetryOnlyIdempotentRequests(t *testing.T) {
	for method, want := range map[string]int32{http.MethodGet: maxAttempts, http.MethodPost: 1} {
		transport := &failingTransport{}
		client, err := New(Options{BaseURL: "https://acme.atlassian.net/wiki", HTTPClient: &http.Client{Transport: transport}})
		if err != nil {
			t.Fatal(err)
		}
		client.sleep = func(context.Context, time.Duration) error { return nil }
		if method == http.MethodGet {
			err = client.GetJSON(context.Background(), "/x", nil, nil)
		} else {
			err = client.PostJSON(context.Background(), "/x", map[string]int{}, nil)
		}
		if err == nil || transport.calls.Load() != want {
			t.Errorf("%s: %d attempts, err %v; want %d", method, transport.calls.Load(), err, want)
		}
	}
}

func TestCancellationStopsTheRetryWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		cancel()
		status(503)(w)
	})
	client.sleep = sleepContext
	if err := client.GetJSON(ctx, "/x", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

func TestPostMultipart(t *testing.T) {
	var calls atomic.Int32
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Atlassian-Token") != "nocheck" {
			t.Errorf("missing X-Atlassian-Token")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(file)
		if header.Filename != "diagram 1.svg" || string(content) != "<svg/>" || r.FormValue("minorEdit") != "true" || header.Header.Get("Content-Type") != "image/svg+xml" {
			t.Errorf("attempt %d: %q %q %q", calls.Load()+1, header.Filename, content, r.FormValue("minorEdit"))
		}
		if calls.Add(1) == 1 {
			status(429, "Retry-After", "1")(w)

			return
		}
		_, _ = io.WriteString(w, `{"results":[{"id":"att1"}]}`)
	})
	var out struct{ Results []struct{ ID string } }
	err := client.PostMultipart(context.Background(), "/rest/api/content/42/child/attachment", map[string]string{"minorEdit": "true"},
		[]MultipartFile{{Field: "file", FileName: "diagram 1.svg", ContentType: "image/svg+xml", Content: []byte("<svg/>")}}, &out)
	if err != nil || calls.Load() != 2 || len(out.Results) != 1 || out.Results[0].ID != "att1" {
		t.Fatalf("calls %d, out %+v, err %v", calls.Load(), out, err)
	}
}

func TestLogsNeverCarryHeadersOrBodies(t *testing.T) {
	client, _, logs := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"response-body-marker"}`)
	})
	_ = client.PostJSON(context.Background(), "/api/v2/pages", map[string]string{"body": "request-body-marker"}, nil)
	_ = client.GetJSON(context.Background(), "/api/v2/spaces", nil, nil)
	text := logs.String()
	for _, leak := range []string{"s3cr3t-token", "Authorization", "request-body-marker", "response-body-marker"} {
		if strings.Contains(text, leak) {
			t.Errorf("log contains %q:\n%s", leak, text)
		}
	}
	for _, expected := range []string{"method=POST", "path=/wiki/api/v2/pages", "status=400", "method=GET"} {
		if !strings.Contains(text, expected) {
			t.Errorf("log lacks %q:\n%s", expected, text)
		}
	}
}

func TestNewRejectsARelativeBaseURL(t *testing.T) {
	if _, err := New(Options{BaseURL: "acme.atlassian.net/wiki"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetBytesSendsItsOwnAccept(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/xml" {
			t.Errorf("Accept %q", r.Header.Get("Accept"))
		}
		_, _ = io.WriteString(w, "<manifest/>")
	})
	body, err := client.GetBytes(context.Background(), "/rest/applinks/1.0/manifest", nil, "application/xml")
	if err != nil || string(body) != "<manifest/>" {
		t.Fatalf("body %q, err %v", body, err)
	}
}
