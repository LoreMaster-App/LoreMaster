package syncannotation

import "testing"

func TestLocateParent(t *testing.T) {
	cases := []struct {
		name, content string
		line          int
		value         string
		found         bool
	}{
		{"in the block", "<!-- lore-master\ntitle: T\nparent: ./a.md\n-->\n# H\n", 2, "./a.md", true},
		{"after front matter and CRLF", "---\r\nx: 1\r\n---\r\n<!-- lore-master\r\nparent:   ../b.md\r\n-->\r\n", 4, "../b.md", true},
		{"no annotation", "# H\nparent: x.md\n", 0, "", false},
		{"no parent line", "<!-- lore-master\ntitle: T\n-->\n", 0, "", false},
		{"parent only after the block", "<!-- lore-master\ntitle: T\n-->\nparent: x.md\n", 0, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LocateParent([]byte(tc.content))
			if ok != tc.found || got.Line != tc.line || got.Value != tc.value {
				t.Fatalf("got %+v ok=%v, want line %d value %q ok=%v", got, ok, tc.line, tc.value, tc.found)
			}
		})
	}
}
