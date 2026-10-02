package syncplanning

import (
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
)

func TestDecideCollision(t *testing.T) {
	one := []platformport.RemotePage{{ID: "7", URL: "https://docs.example/pages/7"}}
	two := append(one, platformport.RemotePage{ID: "8"})

	cases := []struct {
		name  string
		mode  string
		found []platformport.RemotePage
		kind  ActionKind
		error string
	}{
		{"free title, fail", "fail", nil, Create, ""},
		{"free title, adopt", "adopt", nil, Create, ""},
		{"taken, fail names the page", "fail", one, "", "already exists (https://docs.example/pages/7)"},
		{"taken, adopt", "adopt", one, Adopt, ""},
		{"taken twice, adopt refuses to guess", "adopt", two, "", `a.md: 2 pages are titled "T"; adopt needs exactly one`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decideCollision(c.mode, "a.md", "T", c.found)
			if got.kind != c.kind || (c.error == "") != (got.error == "") || !strings.Contains(got.error, c.error) {
				t.Fatalf("got %+v", got)
			}
			if c.kind == Adopt && got.page.ID != "7" {
				t.Fatalf("adopted %+v", got.page)
			}
		})
	}
}
