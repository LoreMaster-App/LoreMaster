package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Options configures a Client. BaseURL is the normalised site URL (connection
// package); AuthorizationHeader is the full header value (authentication package).
type Options struct {
	BaseURL             string
	AuthorizationHeader string
	// HTTPClient defaults to one with a 60 s timeout per attempt.
	HTTPClient *http.Client
	// Logger defaults to slog.Default(). It receives method, path, status and attempt.
	Logger *slog.Logger
}

// Client sends requests to one Confluence site.
type Client struct {
	base          *url.URL
	authorization string
	http          *http.Client
	logger        *slog.Logger
	sleep         func(ctx context.Context, d time.Duration) error
	now           func() time.Time
}

// New returns a Client for options.
func New(options Options) (*Client, error) {
	base, err := url.Parse(strings.TrimRight(options.BaseURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("base URL %q is not absolute", options.BaseURL)
	}
	client := &Client{
		base:          base,
		authorization: options.AuthorizationHeader,
		http:          options.HTTPClient,
		logger:        options.Logger,
		sleep:         sleepContext,
		now:           time.Now,
	}
	if client.http == nil {
		client.http = &http.Client{Timeout: 60 * time.Second}
	}
	if client.logger == nil {
		client.logger = slog.Default()
	}

	return client, nil
}

// GetJSON fetches path (relative to the base URL) with query and decodes the body into
// out.
func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, out any) error {
	return c.doJSON(ctx, http.MethodGet, c.resolve(path, query), nil, out)
}

// GetBytes fetches path and returns the raw body, for the few endpoints that do not
// answer JSON (the application-links manifest is XML). accept is sent as the Accept
// header.
func (c *Client) GetBytes(ctx context.Context, path string, query url.Values, accept string) ([]byte, error) {
	var body []byte
	err := c.do(ctx, http.MethodGet, c.resolve(path, query), nil, http.Header{"Accept": {accept}}, &body)

	return body, err
}

// PostJSON sends body as JSON and decodes the answer into out (nil to discard).
func (c *Client) PostJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, c.resolve(path, nil), body, out)
}

// PutJSON sends body as JSON and decodes the answer into out (nil to discard).
func (c *Client) PutJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPut, c.resolve(path, nil), body, out)
}

// Delete deletes path.
func (c *Client) Delete(ctx context.Context, path string, query url.Values) error {
	return c.doJSON(ctx, http.MethodDelete, c.resolve(path, query), nil, nil)
}

// MultipartFile is one file part of a multipart request.
type MultipartFile struct {
	Field       string
	FileName    string
	ContentType string
	Content     []byte
}

// PostMultipart sends fields and files as multipart/form-data, with the
// X-Atlassian-Token header Confluence requires for uploads (its XSRF guard).
func (c *Client) PostMultipart(ctx context.Context, path string, fields map[string]string, files []MultipartFile, out any) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return err
		}
	}
	for _, file := range files {
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name=%q; filename=%q`, file.Field, file.FileName)}
		header["Content-Type"] = []string{cmpOr(file.ContentType, "application/octet-stream")}
		part, err := writer.CreatePart(header)
		if err != nil {
			return err
		}
		if _, err := part.Write(file.Content); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return c.do(ctx, http.MethodPost, c.resolve(path, nil), body.Bytes(), http.Header{
		"Content-Type":      {writer.FormDataContentType()},
		"X-Atlassian-Token": {"nocheck"},
	}, out)
}

func (c *Client) doJSON(ctx context.Context, method string, target *url.URL, body any, out any) error {
	var payload []byte
	header := http.Header{}
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = encoded
		header.Set("Content-Type", "application/json")
	}

	return c.do(ctx, method, target, payload, header, out)
}

// do sends one logical request, retrying per decideRetry. The body is a byte slice so
// every attempt sends it whole.
func (c *Client) do(ctx context.Context, method string, target *url.URL, payload []byte, header http.Header, out any) error {
	for attempt := 1; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(payload))
		if err != nil {
			return err
		}
		for name, values := range header {
			request.Header[name] = values
		}
		if request.Header.Get("Accept") == "" {
			request.Header.Set("Accept", "application/json")
		}
		if c.authorization != "" {
			request.Header.Set("Authorization", c.authorization)
		}

		response, err := c.http.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Debug("confluence request failed", "method", method, "path", target.Path, "attempt", attempt)
			decision := decideRetry(method, 0, true, nil, attempt, c.now())
			if !decision.retry {
				return fmt.Errorf("%s %s: %w", method, target.Path, err)
			}
			if err := c.sleep(ctx, decision.wait); err != nil {
				return err
			}

			continue
		}

		c.logger.Debug("confluence request", "method", method, "path", target.Path, "status", response.StatusCode, "attempt", attempt)
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return decode(response, out)
		}
		errorBody := readErrorBody(response)
		decision := decideRetry(method, response.StatusCode, false, response.Header, attempt, c.now())
		if !decision.retry {
			return &APIError{Method: method, Path: target.Path, Status: response.StatusCode, Body: errorBody}
		}
		if err := c.sleep(ctx, decision.wait); err != nil {
			return err
		}
	}
}

// resolve joins a path relative to the base URL ("/rest/api/space") with a query.
func (c *Client) resolve(path string, query url.Values) *url.URL {
	resolved := *c.base
	rawPath, rawQuery, _ := strings.Cut(path, "?")
	resolved.Path = c.base.Path + "/" + strings.TrimLeft(rawPath, "/")
	values, _ := url.ParseQuery(rawQuery)
	for name, list := range query {
		values[name] = append(values[name], list...)
	}
	resolved.RawQuery = values.Encode()

	return &resolved
}

// sameOrigin resolves a link Confluence returned (absolute, or absolute-path such as
// Cloud's "/wiki/api/v2/pages?cursor=…") and refuses one that leaves the site: the
// next request would carry the Authorization header to whoever is there.
func (c *Client) sameOrigin(link string) (*url.URL, error) {
	origin := url.URL{Scheme: c.base.Scheme, Host: c.base.Host}
	resolved, err := origin.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("confluence returned an unreadable next link %q", link)
	}
	if resolved.Scheme != c.base.Scheme || !strings.EqualFold(resolved.Host, c.base.Host) {
		return nil, fmt.Errorf("confluence returned a next link to another site (%s); refusing to send credentials there", resolved.Host)
	}

	return resolved, nil
}

func decode(response *http.Response, out any) error {
	defer func() { _ = response.Body.Close() }()
	if out == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, response.Body)

		return nil
	}
	if raw, isRaw := out.(*[]byte); isRaw {
		body, err := io.ReadAll(response.Body)
		*raw = body

		return err
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode %s %s: %w", response.Request.Method, response.Request.URL.Path, err)
	}

	return nil
}

func readErrorBody(response *http.Response) string {
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody+1))
	text := strings.TrimSpace(string(body))
	if len(text) > maxErrorBody {
		text = text[:maxErrorBody] + "…"
	}

	return text
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func cmpOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
