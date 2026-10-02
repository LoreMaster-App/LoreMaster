package documentparsing

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"

	"lore-master/libs/markdown-workspace/documentdiscovery"
)

// Inventory is every reference a document makes that the sync has to translate: links
// to other Markdown files become links to their pages, and local images become
// attachments. External links, mail links and same-page anchors need nothing and are
// not listed.
type Inventory struct {
	PageLinks []PageLink
	Images    []ImageRef
	Warnings  []string
}

// PageLink is a link to another Markdown file in the workspace.
type PageLink struct {
	// Target is the linked file, resolved against the linking document's directory.
	// It is not checked against the disk: the caller looks it up among the discovered
	// documents, which also forgives a link written with the wrong case.
	Target documentdiscovery.DocumentPath
	// Fragment is the part after '#', without it; empty when there is none.
	Fragment string
	Node     ast.Node
}

// ImageRef is one image: a local file (Path) or a remote one (URL), never both.
type ImageRef struct {
	Path documentdiscovery.DocumentPath
	URL  string
	// Node is the *ast.Image, or the *ast.RawHTML / *ast.HTMLBlock holding an <img>.
	Node ast.Node
}

// InventoryLinks lists a parsed document's page links and images in document order.
// It is pure: the tree in, the inventory out.
func InventoryLinks(document MarkdownDocument) Inventory {
	var inventory Inventory
	base := path.Dir(string(document.Path))
	source := document.Body

	_ = ast.Walk(document.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Link:
			target, fragment, ok, warning := pageTarget(base, string(n.Destination))
			if warning != "" {
				inventory.Warnings = append(inventory.Warnings, warning)
			}
			if ok {
				inventory.PageLinks = append(inventory.PageLinks, PageLink{Target: target, Fragment: fragment, Node: n})
			}
		case *ast.Image:
			inventory.addImage(base, string(n.Destination), n)
		case *ast.RawHTML:
			var html strings.Builder
			for i := range n.Segments.Len() {
				segment := n.Segments.At(i)
				html.Write(segment.Value(source))
			}
			inventory.addHTMLImages(base, html.String(), n)
		case *ast.HTMLBlock:
			var html strings.Builder
			lines := n.Lines()
			for i := range lines.Len() {
				segment := lines.At(i)
				html.Write(segment.Value(source))
			}
			if n.HasClosure() {
				html.Write(n.ClosureLine.Value(source))
			}
			inventory.addHTMLImages(base, html.String(), n)
		}

		return ast.WalkContinue, nil
	})

	return inventory
}

// imgSource finds the src of each <img> tag, whatever its quoting.
var imgSource = regexp.MustCompile(`(?i)<img\b[^>]*?\ssrc\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)

func (inventory *Inventory) addHTMLImages(base string, html string, node ast.Node) {
	for _, match := range imgSource.FindAllStringSubmatch(html, -1) {
		inventory.addImage(base, match[1]+match[2]+match[3], node)
	}
}

func (inventory *Inventory) addImage(base string, destination string, node ast.Node) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return
	}
	if isRemote(destination) {
		inventory.Images = append(inventory.Images, ImageRef{URL: destination, Node: node})

		return
	}
	resolved, ok, warning := resolveLocal(base, stripQueryAndFragment(destination))
	if warning != "" {
		inventory.Warnings = append(inventory.Warnings, warning)
	}
	if ok {
		inventory.Images = append(inventory.Images, ImageRef{Path: resolved, Node: node})
	}
}

// pageTarget resolves a link destination to a Markdown file, or reports that it is
// not one: remote, mail, anchor-only and non-Markdown targets are left alone.
func pageTarget(base string, destination string) (documentdiscovery.DocumentPath, string, bool, string) {
	destination = strings.TrimSpace(destination)
	if destination == "" || strings.HasPrefix(destination, "#") || isRemote(destination) {
		return "", "", false, ""
	}
	target, fragment, _ := strings.Cut(destination, "#")
	target = stripQueryAndFragment(target)
	if !strings.EqualFold(path.Ext(decoded(target)), ".md") {
		return "", "", false, ""
	}
	resolved, ok, warning := resolveLocal(base, target)

	return resolved, fragment, ok, warning
}

// resolveLocal turns a relative (or workspace-absolute, leading '/') reference into a
// workspace-relative path, percent-decoded. A reference that climbs out of the
// workspace cannot be synced and is reported instead.
func resolveLocal(base string, reference string) (documentdiscovery.DocumentPath, bool, string) {
	reference = decoded(reference)
	var joined string
	if strings.HasPrefix(reference, "/") {
		joined = path.Clean(strings.TrimLeft(reference, "/"))
	} else {
		joined = path.Join(base, reference)
	}
	if joined == ".." || strings.HasPrefix(joined, "../") || joined == "." {
		return "", false, fmt.Sprintf("the reference %q points outside the workspace and is left as is", reference)
	}

	return documentdiscovery.DocumentPath(joined), true, ""
}

// isRemote is any destination with a URL scheme (http:, https:, mailto:, data:, …) or a
// protocol-relative "//host" prefix. A one-letter "scheme" is a Windows drive, not a URL.
func isRemote(destination string) bool {
	if strings.HasPrefix(destination, "//") {
		return true
	}
	parsed, err := url.Parse(destination)

	return err == nil && len(parsed.Scheme) > 1
}

func stripQueryAndFragment(reference string) string {
	if cut := strings.IndexAny(reference, "?#"); cut >= 0 {
		return reference[:cut]
	}

	return reference
}

func decoded(reference string) string {
	if unescaped, err := url.PathUnescape(reference); err == nil {
		return unescaped
	}

	return reference
}
