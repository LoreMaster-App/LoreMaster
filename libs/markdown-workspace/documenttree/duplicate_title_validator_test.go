package documenttree

import (
	"reflect"
	"strings"
	"testing"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

func treeOf(t *testing.T, files map[string]string) DocumentTree {
	t.Helper()
	tree, err := BuildTree(parsed(t, files))
	if err != nil {
		t.Fatal(err)
	}

	return tree
}

func TestPageTitles(t *testing.T) {
	tree := treeOf(t, map[string]string{
		"README.md":     "# Home\n",
		"setup.md":      "<!-- lore-master\ntitle:   Installing  \n-->\n# Setup\n",
		"docs/faq.md":   "#   Questions   and answers\n",
		"no-heading.md": "Text only\n",
	})
	cases := []struct {
		prefix string
		want   map[documentdiscovery.DocumentPath]string
	}{
		{"MNCI", map[documentdiscovery.DocumentPath]string{
			"README.md": "MNCI: Home", "setup.md": "MNCI: Installing", "docs/faq.md": "MNCI: Questions and answers", "no-heading.md": "MNCI: no-heading",
		}},
		{"  MNCI  ", map[documentdiscovery.DocumentPath]string{
			"README.md": "MNCI: Home", "setup.md": "MNCI: Installing", "docs/faq.md": "MNCI: Questions and answers", "no-heading.md": "MNCI: no-heading",
		}},
		{"", map[documentdiscovery.DocumentPath]string{
			"README.md": "Home", "setup.md": "Installing", "docs/faq.md": "Questions and answers", "no-heading.md": "no-heading",
		}},
	}
	for _, tc := range cases {
		titles, err := PageTitles(tree, tc.prefix)
		if err != nil {
			t.Fatalf("prefix %q: %v", tc.prefix, err)
		}
		if !reflect.DeepEqual(titles, tc.want) {
			t.Fatalf("prefix %q\n got: %v\nwant: %v", tc.prefix, titles, tc.want)
		}
	}
}

func TestPageTitlesRejectsClashes(t *testing.T) {
	tree := treeOf(t, map[string]string{
		"a/setup.md":   "# Setup\n",
		"b/setup.md":   "# setup\n",
		"c/install.md": "<!-- lore-master\ntitle: Setup\n-->\n# Install\n",
		"x/guide.md":   "# Guide\n",
		"y/guide.md":   "# Guide\n",
		"unique.md":    "# Unique\n",
	})
	_, err := PageTitles(tree, "MNCI")
	want := "These documents would get the same page title, and Confluence allows a title only once per space:\n" +
		"  a/setup.md, b/setup.md, c/install.md → \"MNCI: Setup\"\n" +
		"  x/guide.md, y/guide.md → \"MNCI: Guide\"\n" +
		"Give all but one of each a \"title:\" line in its lore-master annotation, or change its H1."
	if err == nil || err.Error() != want {
		t.Fatalf("error\n got: %v\nwant: %s", err, want)
	}
}

func TestPageTitlesRejectsOverlongTitles(t *testing.T) {
	exactly := strings.Repeat("é", MaxTitleLength-len("P: "))
	tree := treeOf(t, map[string]string{
		"fits.md":      "# " + exactly + "\n",
		"too-long.md":  "# " + exactly + "x\n",
		"also-long.md": "# " + exactly + "x\n",
	})
	_, err := PageTitles(tree, "P")
	want := "These documents would get the same page title, and Confluence allows a title only once per space:\n" +
		"  also-long.md, too-long.md → \"P: " + exactly + "x\"\n" +
		"Give all but one of each a \"title:\" line in its lore-master annotation, or change its H1.\n\n" +
		"These page titles are longer than Confluence's 255 characters:\n" +
		"  also-long.md → 256 characters\n" +
		"  too-long.md → 256 characters\n" +
		"Shorten the H1, set a shorter \"title:\" in the lore-master annotation, or choose a shorter title prefix."
	if err == nil || err.Error() != want {
		t.Fatalf("error\n got: %v\nwant: %s", err, want)
	}
}
