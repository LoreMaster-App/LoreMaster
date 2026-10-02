package pagecontent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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

// pagesOn serves routes (path + "?" + one distinguishing query value, or just path) and
// returns Pages for edition. Unknown requests are a 404.
func pagesOn(t *testing.T, edition connection.Edition, contextPath string, route func(r *http.Request) (int, string)) *Pages {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, body := route(r)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client, err := httptransport.New(httptransport.Options{BaseURL: server.URL + contextPath})
	if err != nil {
		t.Fatal(err)
	}

	return New(client, edition)
}

func TestGetPageCloud(t *testing.T) {
	pages := pagesOn(t, connection.Cloud, "/wiki", func(r *http.Request) (int, string) {
		if r.URL.Path != "/wiki/api/v2/pages/98400" || r.URL.Query().Get("body-format") != "storage" {
			t.Errorf("request %s", r.URL)
		}

		return 200, fixture(t, "page-v2-with-body.json")
	})
	page, err := pages.GetPage(context.Background(), "98400", true)
	want := Page{
		ID: "98400", Title: "ENG: Architecture", SpaceID: "98306", ParentID: "98310", Version: 7,
		WebURL:      "https://acme.atlassian.net/wiki/spaces/ENG/pages/98400/ENG+Architecture",
		BodyStorage: "<h1>Architecture</h1><p>Text</p>",
	}
	if err != nil || page != want {
		t.Fatalf("\n got: %+v\nwant: %+v\nerr %v", page, want, err)
	}
}

func TestGetPageDataCenter(t *testing.T) {
	pages := pagesOn(t, connection.DataCenter, "/confluence", func(r *http.Request) (int, string) {
		if r.URL.Path != "/confluence/rest/api/content/131090" || r.URL.Query().Get("expand") != "version,ancestors,space,body.storage" {
			t.Errorf("request %s", r.URL)
		}

		return 200, fixture(t, "page-v1-with-body.json")
	})
	page, err := pages.GetPage(context.Background(), "131090", true)
	want := Page{
		ID: "131090", Title: "ENG: Architecture", SpaceID: "131073", SpaceKey: "ENG", ParentID: "131080", Version: 4,
		WebURL:      "https://confluence.acme.com/confluence/display/ENG/ENG%3A+Architecture",
		BodyStorage: "<h1>Architecture</h1>",
	}
	if err != nil || page != want {
		t.Fatalf("\n got: %+v\nwant: %+v\nerr %v", page, want, err)
	}
}

func TestGetPageNotFound(t *testing.T) {
	for _, edition := range []connection.Edition{connection.Cloud, connection.Server} {
		pages := pagesOn(t, edition, "/wiki", func(*http.Request) (int, string) { return 404, `{"message":"No content found"}` })
		_, err := pages.GetPage(context.Background(), "42", false)
		var notFound *PageNotFoundError
		if !errors.As(err, &notFound) || notFound.ID != "42" {
			t.Errorf("%s: error %v", edition, err)
		}
	}
}

func TestListChildren(t *testing.T) {
	cloud := pagesOn(t, connection.Cloud, "/wiki", func(r *http.Request) (int, string) {
		if r.URL.Query().Get("cursor") == "" {
			return 200, fixture(t, "children-v2-page1.json")
		}

		return 200, fixture(t, "children-v2-page2.json")
	})
	got, err := cloud.ListChildren(context.Background(), "98400")
	if err != nil || len(got) != 3 || got[2].ID != "98403" || got[0].ParentID != "98400" || got[1].Title != "ENG: API" {
		t.Fatalf("cloud children %+v, err %v", got, err)
	}

	server := pagesOn(t, connection.DataCenter, "/confluence", func(r *http.Request) (int, string) {
		if r.URL.Path != "/confluence/rest/api/content/131090/child/page" {
			t.Errorf("request %s", r.URL)
		}
		switch r.URL.Query().Get("start") {
		case "0":
			return 200, fixture(t, "children-v1-page1.json")
		case "1":
			return 200, fixture(t, "children-v1-page2.json")
		}

		return 200, `{"results":[],"start":2,"limit":1,"size":0,"_links":{}}`
	})
	got, err = server.ListChildren(context.Background(), "131090")
	if err != nil || len(got) != 2 || got[1].ID != "131092" || got[1].Version != 1 || got[0].ParentID != "131090" ||
		!strings.HasSuffix(got[0].WebURL, "/confluence/display/ENG/ENG%3A+Database") {
		t.Fatalf("data center children %+v, err %v", got, err)
	}
}

func TestSearchPagesEscapesAndFiltersToEqualTitles(t *testing.T) {
	for _, edition := range []connection.Edition{connection.Cloud, connection.DataCenter} {
		pages := pagesOn(t, edition, "/confluence", func(r *http.Request) (int, string) {
			if r.URL.Path != "/confluence/rest/api/content/search" {
				t.Errorf("path %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("cql"); got != `space = "E\\NG" and title = "ENG: Say \"hi\"" and type = page` {
				t.Errorf("cql %s", got)
			}

			return 200, fixture(t, "search-v1.json")
		})
		found, err := pages.SearchPages(context.Background(), `E\NG`, `ENG: Say "hi"`)
		var ids []string
		for _, page := range found {
			ids = append(ids, page.ID)
		}
		if err != nil || !reflect.DeepEqual(ids, []string{"131095"}) || found[0].ParentID != "131074" || found[0].SpaceKey != "ENG" {
			t.Fatalf("%s: found %+v, err %v", edition, found, err)
		}
	}
}

func TestCQLString(t *testing.T) {
	cases := map[string]string{
		`plain`:            `"plain"`,
		`a "quoted" title`: `"a \"quoted\" title"`,
		`back\slash`:       `"back\\slash"`,
		`end\"`:            `"end\\\""`,
	}
	for in, want := range cases {
		if got := cqlString(in); got != want {
			t.Errorf("cqlString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestListChildrenOfAMissingPageIsNotFound(t *testing.T) {
	for _, edition := range []connection.Edition{connection.Cloud, connection.DataCenter} {
		pages := pagesOn(t, edition, "", func(*http.Request) (int, string) { return 404, `{"message":"No content found"}` })
		_, err := pages.ListChildren(context.Background(), "404404")
		var missing *PageNotFoundError
		if !errors.As(err, &missing) || missing.ID != "404404" {
			t.Fatalf("%s: %v", edition, err)
		}
	}
}
