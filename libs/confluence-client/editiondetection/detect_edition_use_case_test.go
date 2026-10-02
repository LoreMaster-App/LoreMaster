package editiondetection

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

// site answers each path in routes with its status and body, and 404 for anything else.
func site(t *testing.T, routes map[string][2]string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		status := map[string]int{"200": 200, "401": 401}[route[0]]
		w.WriteHeader(status)
		_, _ = w.Write([]byte(route[1]))
	}))
	t.Cleanup(server.Close)

	return server.URL + "/confluence"
}

func detect(t *testing.T, baseURL string) (Detection, error) {
	t.Helper()
	client, err := httptransport.New(httptransport.Options{BaseURL: baseURL, AuthorizationHeader: "Bearer x"})
	if err != nil {
		t.Fatal(err)
	}

	return DetectEdition(context.Background(), client, baseURL)
}

func TestDetectEdition(t *testing.T) {
	cases := []struct {
		name   string
		routes map[string][2]string
		want   Detection
	}{
		{"Cloud behind a custom domain answers systemInfo", map[string][2]string{
			"/confluence/rest/api/settings/systemInfo": {"200", fixture(t, "systeminfo-cloud.json")},
		}, Detection{Edition: connection.Cloud}},
		{"Data Center 8.5 from the manifest", map[string][2]string{
			"/confluence/rest/applinks/1.0/manifest": {"200", fixture(t, "manifest-datacenter-8.5.3.xml")},
		}, Detection{Edition: connection.DataCenter, Version: connection.Version{Major: 8, Minor: 5, Patch: 3}}},
		{"Server 7.4 from the manifest", map[string][2]string{
			"/confluence/rest/applinks/1.0/manifest": {"200", fixture(t, "manifest-server-7.4.0.xml")},
		}, Detection{Edition: connection.Server, Version: connection.Version{Major: 7, Minor: 4}}},
		{"Cloud refusing systemInfo but serving a 1000.x manifest", map[string][2]string{
			"/confluence/rest/api/settings/systemInfo": {"401", `{"message":"Unauthorized"}`},
			"/confluence/rest/applinks/1.0/manifest":   {"200", "<manifest><typeId>confluence</typeId><version>1000.0.0-60b5e2e6d6c9</version></manifest>"},
		}, Detection{Edition: connection.Cloud}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detect(t, site(t, tc.routes))
			if err != nil || got != tc.want {
				t.Fatalf("got %+v, %v; want %+v", got, err, tc.want)
			}
		})
	}
}

func TestDetectEditionRecognisesCloudByHostWithoutARequest(t *testing.T) {
	client, err := httptransport.New(httptransport.Options{
		BaseURL:    "https://acme.atlassian.net/wiki",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("no request expected"); return nil, nil })},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DetectEdition(context.Background(), client, "https://acme.atlassian.net/wiki")
	if err != nil || got != (Detection{Edition: connection.Cloud}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDetectEditionRejectsWhatIsNotConfluence(t *testing.T) {
	cases := map[string]struct {
		routes map[string][2]string
		want   string
	}{
		"nothing answers": {map[string][2]string{}, "does not look like a Confluence site"},
		"a Jira manifest": {map[string][2]string{"/confluence/rest/applinks/1.0/manifest": {"200", fixture(t, "manifest-jira.xml")}}, `its application-links manifest describes "jira"`},
		"garbage version": {map[string][2]string{"/confluence/rest/applinks/1.0/manifest": {"200", "<manifest><typeId>confluence</typeId><version>dev</version></manifest>"}}, "reports an unreadable Confluence version"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := detect(t, site(t, tc.routes))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want %q", err, tc.want)
			}
		})
	}
}
