package pagecontent

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"lore-master/libs/confluence-client/connection"
)

// recorded is one request as "METHOD path?query body".
type recorder struct {
	requests []string
}

func (r *recorder) record(req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	line := req.Method + " " + req.URL.Path
	if req.URL.RawQuery != "" {
		line += "?" + req.URL.RawQuery
	}
	if len(body) > 0 {
		line += " " + strings.TrimSpace(string(body))
	}
	r.requests = append(r.requests, line)
}

func TestMarkPage(t *testing.T) {
	cases := []struct {
		name    string
		edition connection.Edition
		answers map[string][2]string // "METHOD path" → status, body; default 200 {}
		want    []string
	}{
		{
			name: "Cloud, new property", edition: connection.Cloud,
			answers: map[string][2]string{"GET /x/api/v2/pages/42/properties": {"200", `{"results":[]}`}},
			want: []string{
				`POST /x/rest/api/content/42/label [{"name":"lore-master","prefix":"global"}]`,
				`GET /x/api/v2/pages/42/properties?key=lore-master.source-path`,
				`POST /x/api/v2/pages/42/properties {"key":"lore-master.source-path","value":{"sourcePath":"docs/a.md"}}`,
			},
		},
		{
			name: "Cloud, moved file updates the property at its next version", edition: connection.Cloud,
			answers: map[string][2]string{"GET /x/api/v2/pages/42/properties": {"200", `{"results":[{"id":"777","key":"lore-master.source-path","version":{"number":3}}]}`}},
			want: []string{
				`POST /x/rest/api/content/42/label [{"name":"lore-master","prefix":"global"}]`,
				`GET /x/api/v2/pages/42/properties?key=lore-master.source-path`,
				`PUT /x/api/v2/pages/42/properties/777 {"key":"lore-master.source-path","value":{"sourcePath":"docs/a.md"},"version":{"number":4}}`,
			},
		},
		{
			name: "Data Center, new property", edition: connection.DataCenter,
			answers: map[string][2]string{"GET /x/rest/api/content/42/property/lore-master.source-path": {"404", `{"message":"not found"}`}},
			want: []string{
				`POST /x/rest/api/content/42/label [{"name":"lore-master","prefix":"global"}]`,
				`GET /x/rest/api/content/42/property/lore-master.source-path`,
				`POST /x/rest/api/content/42/property {"key":"lore-master.source-path","value":{"sourcePath":"docs/a.md"}}`,
			},
		},
		{
			name: "Data Center, existing property", edition: connection.DataCenter,
			answers: map[string][2]string{"GET /x/rest/api/content/42/property/lore-master.source-path": {"200", `{"key":"lore-master.source-path","version":{"number":1}}`}},
			want: []string{
				`POST /x/rest/api/content/42/label [{"name":"lore-master","prefix":"global"}]`,
				`GET /x/rest/api/content/42/property/lore-master.source-path`,
				`PUT /x/rest/api/content/42/property/lore-master.source-path {"key":"lore-master.source-path","value":{"sourcePath":"docs/a.md"},"version":{"number":2}}`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var log recorder
			pages := pagesOn(t, tc.edition, "/x", func(r *http.Request) (int, string) {
				log.record(r)
				if answer, ok := tc.answers[r.Method+" "+r.URL.Path]; ok {
					return map[string]int{"200": 200, "404": 404}[answer[0]], answer[1]
				}

				return 200, `{}`
			})
			if err := pages.MarkPage(context.Background(), "42", "docs/a.md"); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(log.requests, tc.want) {
				t.Fatalf("requests\n got: %q\nwant: %q", log.requests, tc.want)
			}
		})
	}
}
