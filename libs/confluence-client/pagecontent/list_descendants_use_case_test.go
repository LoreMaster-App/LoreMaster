package pagecontent

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"lore-master/libs/confluence-client/connection"
)

func ids(pages []Page) []string {
	var out []string
	for _, page := range pages {
		out = append(out, page.ID)
	}

	return out
}

func TestListDescendantsUsesEachDialect(t *testing.T) {
	cloud := pagesOn(t, connection.Cloud, "/wiki", func(r *http.Request) (int, string) {
		if r.URL.Path != "/wiki/api/v2/pages/10/descendants" {
			t.Errorf("request %s", r.URL)
		}

		return 200, `{"results":[{"id":"11","title":"A","parentId":"10","depth":1},{"id":"12","title":"B","parentId":"11","depth":2}],"_links":{}}`
	})
	got, err := cloud.ListDescendants(context.Background(), "10", false)
	if err != nil || !reflect.DeepEqual(ids(got), []string{"11", "12"}) || got[1].ParentID != "11" {
		t.Fatalf("cloud %+v, err %v", got, err)
	}

	server := pagesOn(t, connection.Server, "/confluence", func(r *http.Request) (int, string) {
		if r.URL.Path != "/confluence/rest/api/content/10/descendant/page" {
			t.Errorf("request %s", r.URL)
		}

		return 200, `{"results":[{"id":"11","title":"A","ancestors":[{"id":"10"}]}],"size":1,"limit":250,"_links":{}}`
	})
	got, err = server.ListDescendants(context.Background(), "10", false)
	if err != nil || !reflect.DeepEqual(ids(got), []string{"11"}) || got[0].ParentID != "10" {
		t.Fatalf("server %+v, err %v", got, err)
	}
}

func TestListDescendantsOnlyMarkedFiltersOnTheServer(t *testing.T) {
	for _, edition := range []connection.Edition{connection.Cloud, connection.DataCenter} {
		pages := pagesOn(t, edition, "/x", func(r *http.Request) (int, string) {
			if r.URL.Path != "/x/rest/api/content/search" || r.URL.Query().Get("cql") != `ancestor = "10" and label = "lore-master" and type = page` {
				t.Errorf("%s: request %s", edition, r.URL)
			}

			return 200, `{"results":[{"id":"11","title":"Ours","ancestors":[{"id":"10"}]}],"size":1,"limit":250,"_links":{}}`
		})
		got, err := pages.ListDescendants(context.Background(), "10", true)
		if err != nil || !reflect.DeepEqual(ids(got), []string{"11"}) {
			t.Fatalf("%s: %+v, err %v", edition, got, err)
		}
	}
}
