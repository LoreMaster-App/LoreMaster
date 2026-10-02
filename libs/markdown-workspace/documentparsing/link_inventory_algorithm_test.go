package documentparsing

import (
	"reflect"
	"testing"

	"github.com/yuin/goldmark/ast"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

type linkRow struct {
	Target   string
	Fragment string
	Node     ast.NodeKind
}

type imageRow struct {
	Path string
	URL  string
	Node ast.NodeKind
}

func inventoryOf(t *testing.T, documentPath string, content string) ([]linkRow, []imageRow, []string) {
	t.Helper()
	document, err := ParseDocument(documentdiscovery.DocumentPath(documentPath), []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	inventory := InventoryLinks(document)
	var links []linkRow
	for _, link := range inventory.PageLinks {
		links = append(links, linkRow{string(link.Target), link.Fragment, link.Node.Kind()})
	}
	var images []imageRow
	for _, image := range inventory.Images {
		images = append(images, imageRow{string(image.Path), image.URL, image.Node.Kind()})
	}

	return links, images, inventory.Warnings
}

func TestInventoryLinks(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		content  string
		links    []linkRow
		images   []imageRow
		warnings []string
	}{
		{
			name:    "sibling directory with a fragment",
			path:    "docs/guide/intro.md",
			content: "See [x](../sibling/x.md#sec).\n",
			links:   []linkRow{{"docs/sibling/x.md", "sec", ast.KindLink}},
		},
		{
			name:    "encoded spaces, angle brackets, query string, upper-case extension",
			path:    "docs/a.md",
			content: "[a](my%20file.md) [b](<other file.md>) [c](c.md?plain=1#top) [d](./D.MD)\n",
			links: []linkRow{
				{"docs/my file.md", "", ast.KindLink}, {"docs/other file.md", "", ast.KindLink},
				{"docs/c.md", "top", ast.KindLink}, {"docs/D.MD", "", ast.KindLink},
			},
		},
		{
			name:    "workspace-absolute target",
			path:    "docs/deep/a.md",
			content: "[root](/README.md)\n",
			links:   []linkRow{{"README.md", "", ast.KindLink}},
		},
		{
			name:    "image inside a link",
			path:    "a.md",
			content: "[![Diagram](img/d.png)](design.md)\n",
			links:   []linkRow{{"design.md", "", ast.KindLink}},
			images:  []imageRow{{"img/d.png", "", ast.KindImage}},
		},
		{
			name:    "reference-style links and images",
			path:    "docs/a.md",
			content: "[Setup][s] and ![logo][l]\n\n[s]: ../setup.md#install\n[l]: https://x.io/logo.svg\n",
			links:   []linkRow{{"setup.md", "install", ast.KindLink}},
			images:  []imageRow{{"", "https://x.io/logo.svg", ast.KindImage}},
		},
		{
			name: "external, mail, anchor-only and non-Markdown links pass through",
			path: "a.md",
			content: "[w](https://example.com/x.md) [m](mailto:a@b.c) [h](#local) [p](report.pdf) " +
				"[r](//cdn.example.com/x.md) <https://auto.link> https://linkified.example\n",
		},
		{
			name: "HTML img inline and as a block, any quoting",
			path: "docs/a.md",
			content: "Inline <img src=\"inline.png\" alt=\"i\"> here.\n\n" +
				"<p align=\"center\">\n  <img width=40 src='block.svg'>\n  <IMG SRC=bare.gif>\n</p>\n\n" +
				"<img src=\"https://x.io/remote.png\">\n",
			images: []imageRow{
				{"docs/inline.png", "", ast.KindRawHTML},
				{"docs/block.svg", "", ast.KindHTMLBlock},
				{"docs/bare.gif", "", ast.KindHTMLBlock},
				{"", "https://x.io/remote.png", ast.KindHTMLBlock},
			},
		},
		{
			name:     "a reference that leaves the workspace is reported, not listed",
			path:     "docs/a.md",
			content:  "[up](../../outside.md) ![up](../../logo.png)\n",
			warnings: []string{`the reference "../../outside.md" points outside the workspace and is left as is`, `the reference "../../logo.png" points outside the workspace and is left as is`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			links, images, warnings := inventoryOf(t, tc.path, tc.content)
			if !reflect.DeepEqual(links, tc.links) {
				t.Errorf("links\n got: %+v\nwant: %+v", links, tc.links)
			}
			if !reflect.DeepEqual(images, tc.images) {
				t.Errorf("images\n got: %+v\nwant: %+v", images, tc.images)
			}
			if !reflect.DeepEqual(warnings, tc.warnings) {
				t.Errorf("warnings\n got: %q\nwant: %q", warnings, tc.warnings)
			}
		})
	}
}
