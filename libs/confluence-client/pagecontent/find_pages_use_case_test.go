package pagecontent

import (
	"context"
	"net/http"
	"testing"

	"lore-master/libs/confluence-client/connection"
)

func TestFindPages(t *testing.T) {
	cases := []struct {
		name  string
		query string
		limit int
		cql   string
		want  []string
	}{
		{"words narrow the search", `Say "hi"`, 50,
			`space = "ENG" and type = page and title ~ "Say hi*" order by title`, []string{`ENG: Say "hi"`, `ENG: Say "hi" again`}},
		{"the filter is a real contains", "AGAIN", 50,
			`space = "ENG" and type = page and title ~ "AGAIN*" order by title`, []string{`ENG: Say "hi" again`}},
		{"syntax is neutralised", `(again) OR title ~ x*`, 50,
			`space = "ENG" and type = page and title ~ "again OR title x*" order by title`, nil},
		{"nothing typed lists the space", "  ", 50,
			`space = "ENG" and type = page order by title`, []string{`ENG: Say "hi"`, `ENG: Say "hi" again`}},
		{"the limit stops the search", "", 1,
			`space = "ENG" and type = page order by title`, []string{`ENG: Say "hi"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pages := pagesOn(t, connection.DataCenter, "/confluence", func(r *http.Request) (int, string) {
				if r.URL.Path != "/confluence/rest/api/content/search" || r.URL.Query().Get("cql") != c.cql {
					t.Errorf("request %s cql %q", r.URL.Path, r.URL.Query().Get("cql"))
				}

				return 200, fixture(t, "search-v1.json")
			})
			found, err := pages.FindPages(context.Background(), "ENG", c.query, c.limit)
			if err != nil {
				t.Fatal(err)
			}
			var titles []string
			for _, page := range found {
				titles = append(titles, page.Title)
			}
			if len(titles) != len(c.want) || (len(titles) > 0 && titles[len(titles)-1] != c.want[len(c.want)-1]) {
				t.Fatalf("got %q, want %q", titles, c.want)
			}
		})
	}
}
