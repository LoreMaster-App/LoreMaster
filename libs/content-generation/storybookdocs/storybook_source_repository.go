package storybookdocs

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// indexNames are the files a Storybook publishes its list of stories in, newest first.
var indexNames = []string{"index.json", "stories.json"}

// storybook is an opened source: the base URL its pages are served at, and what to call when
// the render is over.
type storybook struct {
	base  string
	close func()
}

// openStorybook makes the source readable over HTTP. A URL is used as it is; a folder is
// served on a loopback port for the length of the run, because a Storybook's pages load their
// scripts as modules and do not work from file URLs.
func openStorybook(workspaceRoot string, value string) (*storybook, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return &storybook{base: strings.TrimRight(value, "/"), close: func() {}}, nil
	}

	folder := filepath.Join(workspaceRoot, filepath.FromSlash(value))
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("the Storybook build %q was not found; build it first (a prepare command can do that)", value)
	}
	if !hasIndex(folder) {
		return nil, fmt.Errorf("%q has no index.json: it is not a Storybook 7+ build (build-storybook writes it)", value)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("serving %s: %w", value, err)
	}
	server := &http.Server{Handler: http.FileServer(http.Dir(folder)), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()

	return &storybook{
		base:  "http://" + listener.Addr().String(),
		close: func() { _ = server.Close() },
	}, nil
}

func hasIndex(folder string) bool {
	for _, name := range indexNames {
		if _, err := os.Stat(filepath.Join(folder, name)); err == nil {
			return true
		}
	}

	return false
}

// readIndex fetches the story index of the Storybook.
func (s *storybook) readIndex(ctx context.Context) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	var lastErr error
	for _, name := range indexNames {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+"/"+name, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			lastErr = err

			continue
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s answered %s", s.base+"/"+name, response.Status)

			continue
		}
		if readErr != nil {
			return nil, readErr
		}

		return body, nil
	}

	return nil, fmt.Errorf("reading the story index of %s: %w", s.base, lastErr)
}
