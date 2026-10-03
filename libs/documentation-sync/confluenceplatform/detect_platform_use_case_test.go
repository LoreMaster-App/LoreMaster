package confluenceplatform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// detectSite answers each path with its status and body, 404 otherwise, under /confluence.
func detectSite(t *testing.T, routes map[string]string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server.URL + "/confluence"
}

func TestDetect(t *testing.T) {
	t.Run("Data Center from the manifest", func(t *testing.T) {
		base := detectSite(t, map[string]string{
			"/confluence/rest/applinks/1.0/manifest": "<manifest><typeId>confluence</typeId><version>8.5.3</version></manifest>",
		})

		got, err := Detect(context.Background(), DetectOptions{BaseURL: base})
		if err != nil {
			t.Fatal(err)
		}
		if got.Edition != "datacenter" || got.Version != "8.5.3" || got.BaseURL != base {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("Server from the manifest, no version on the result unless known", func(t *testing.T) {
		base := detectSite(t, map[string]string{
			"/confluence/rest/applinks/1.0/manifest": "<manifest><typeId>confluence</typeId><version>7.4.0</version></manifest>",
		})

		got, err := Detect(context.Background(), DetectOptions{BaseURL: base})
		if err != nil {
			t.Fatal(err)
		}
		if got.Edition != "server" || got.Version != "7.4.0" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("Cloud behind a custom domain answers systemInfo, empty version", func(t *testing.T) {
		base := detectSite(t, map[string]string{
			"/confluence/rest/api/settings/systemInfo": "{}",
		})

		got, err := Detect(context.Background(), DetectOptions{BaseURL: base})
		if err != nil {
			t.Fatal(err)
		}
		if got.Edition != "cloud" || got.Version != "" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a site that answers neither probe is unreachable", func(t *testing.T) {
		base := detectSite(t, map[string]string{})

		_, err := Detect(context.Background(), DetectOptions{BaseURL: base})
		var failed *ConnectError
		if !errors.As(err, &failed) || failed.Failure != Unreachable {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("a bad address is an invalid-address failure, with no request", func(t *testing.T) {
		_, err := Detect(context.Background(), DetectOptions{BaseURL: "://no-scheme"})
		var failed *ConnectError
		if !errors.As(err, &failed) || failed.Failure != InvalidAddress {
			t.Fatalf("err %v", err)
		}
	})
}
