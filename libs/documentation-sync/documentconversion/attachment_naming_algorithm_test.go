package documentconversion

import (
	"reflect"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

func TestNameAttachments(t *testing.T) {
	cases := []struct {
		name  string
		paths []documentdiscovery.DocumentPath
		want  []string
	}{
		{"unique names stay", []documentdiscovery.DocumentPath{"docs/a.png", "b.png"}, []string{"a.png", "b.png"}},
		{"repeats collapse", []documentdiscovery.DocumentPath{"a.png", "a.png"}, []string{"a.png"}},
		{"same name, two folders", []documentdiscovery.DocumentPath{"docs/diagram.png", "api/diagram.png"}, []string{"docs__diagram.png", "api__diagram.png"}},
		{"same parent folder too", []documentdiscovery.DocumentPath{"docs/img/d.png", "api/img/d.png", "x.png"}, []string{"docs__img__d.png", "api__img__d.png", "x.png"}},
		{"names differ only in case", []documentdiscovery.DocumentPath{"a/D.png", "b/d.png"}, []string{"a__D.png", "b__d.png"}},
		{"root file against a folder", []documentdiscovery.DocumentPath{"d.png", "img/d.png"}, []string{"d.png", "img__d.png"}},
		{"a prefixed name already taken", []documentdiscovery.DocumentPath{"img__d.png", "img/d.png", "x/d.png"}, []string{"img__d.png", "2__d.png", "x__d.png"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, attachment := range nameAttachments(c.paths) {
				got = append(got, attachment.Filename)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
