package markdowndocs

import (
	"context"
	"reflect"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func TestInPlaceWhenTheInputIsTheOutputFolderOrMissing(t *testing.T) {
	cases := []struct {
		name string
		spec generatedfile.Spec
		want bool
	}{
		{"same folder", generatedfile.Spec{Type: "markdown-docs", Input: []string{"documentation/structure"}, Output: "documentation/structure"}, true},
		{"slash and glob", generatedfile.Spec{Type: "markdown-docs", Input: []string{"./documentation/structure/**"}, Output: "documentation/structure/"}, true},
		{"no input", generatedfile.Spec{Type: "markdown-docs", Output: "documentation/structure"}, true},
		{"other folder", generatedfile.Spec{Type: "markdown-docs", Input: []string{"build/docs"}, Output: "documentation/structure"}, false},
		{"no type and no input", generatedfile.Spec{Output: "documentation/structure"}, false},
		{"no type, same folder", generatedfile.Spec{Input: []string{"docs/out"}, Output: "docs/out"}, true},
	}
	for _, c := range cases {
		if got := IsInPlace(c.spec); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestInPlaceFilesAreTheMarkdownOfTheOutputFolder(t *testing.T) {
	root := t.TempDir()
	put(t, root, "documentation/structure/tree.md", "# Tree")
	put(t, root, "documentation/structure/sub/more.md", "# More")
	put(t, root, "documentation/structure/image.png", "x")
	put(t, root, "documentation/other/elsewhere.md", "# No")

	files, err := InPlaceFiles(context.Background(), root, generatedfile.Spec{Output: "documentation/structure"})

	want := []string{"documentation/structure/sub/more.md", "documentation/structure/tree.md"}
	if err != nil || !reflect.DeepEqual(files, want) {
		t.Fatalf("got %v, %v", files, err)
	}
}
