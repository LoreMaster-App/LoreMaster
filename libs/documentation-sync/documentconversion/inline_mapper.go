package documentconversion

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentparsing"
)

func (c *converter) inlines(parent ast.Node) []platformport.Inline {
	var inlines []platformport.Inline
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		for _, inline := range c.inline(node) {
			inlines = appendInline(inlines, inline)
		}
	}

	return inlines
}

// appendInline joins adjacent text, which goldmark splits at every escape and line.
func appendInline(inlines []platformport.Inline, inline platformport.Inline) []platformport.Inline {
	if text, isText := inline.(platformport.Text); isText && len(inlines) > 0 {
		if previous, wasText := inlines[len(inlines)-1].(platformport.Text); wasText {
			inlines[len(inlines)-1] = platformport.Text{Value: previous.Value + text.Value}

			return inlines
		}
	}

	return append(inlines, inline)
}

func (c *converter) inline(node ast.Node) []platformport.Inline {
	switch n := node.(type) {
	case *ast.Text:
		inlines := []platformport.Inline{platformport.Text{Value: string(documentparsing.Unescaped(n.Segment.Value(c.source)))}}
		switch {
		case n.HardLineBreak():
			inlines = append(inlines, platformport.HardBreak{})
		case n.SoftLineBreak():
			inlines = append(inlines, platformport.Text{Value: " "})
		}

		return inlines
	case *ast.String:
		return []platformport.Inline{platformport.Text{Value: string(n.Value)}}
	case *ast.CodeSpan:
		var code strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if text, isText := child.(*ast.Text); isText {
				code.Write(text.Segment.Value(c.source))
				if text.SoftLineBreak() {
					code.WriteByte(' ')
				}
			}
		}

		return []platformport.Inline{platformport.CodeSpan{Value: code.String()}}
	case *ast.Emphasis:
		if n.Level >= 2 {
			return []platformport.Inline{platformport.Strong{Inlines: c.inlines(n)}}
		}

		return []platformport.Inline{platformport.Emphasis{Inlines: c.inlines(n)}}
	case *extast.Strikethrough:
		return []platformport.Inline{platformport.Strikethrough{Inlines: c.inlines(n)}}
	case *ast.Link:
		return c.link(n)
	case *ast.AutoLink:
		url := string(n.URL(c.source))
		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(url), "mailto:") {
			url = "mailto:" + url
		}

		return []platformport.Inline{platformport.Link{Target: &platformport.URLRef{URL: url}, Inlines: []platformport.Inline{platformport.Text{Value: string(n.Label(c.source))}}}}
	case *ast.Image:
		return c.image(n)
	case *ast.RawHTML:
		return c.rawHTML(n)
	case *extast.TaskCheckBox:
		if n.IsChecked {
			return []platformport.Inline{platformport.Text{Value: "[x] "}}
		}

		return []platformport.Inline{platformport.Text{Value: "[ ] "}}
	}

	return c.inlines(node)
}

// link resolves a link's destination. Links to Markdown files and to headings become
// page links; anything with a scheme is a URL; a link that can be neither keeps its
// text, unlinked, because a relative path means nothing on the platform.
func (c *converter) link(n *ast.Link) []platformport.Inline {
	text := c.inlines(n)
	if page, isPage := c.pageLinks[n]; isPage {
		target, warning := c.workspace.resolvePageLink(page)
		if warning != "" {
			c.warn("%s", warning)
		}
		if target == nil {
			return text
		}

		return []platformport.Inline{platformport.Link{Target: *target, Inlines: text}}
	}
	destination := strings.TrimSpace(string(n.Destination))
	switch {
	case strings.HasPrefix(destination, "#"):
		target, warning := c.workspace.resolveFragment(c.document.Path, destination[1:])
		if warning != "" {
			c.warn("%s", warning)
		}

		return []platformport.Inline{platformport.Link{Target: *target, Inlines: text}}
	case documentparsing.IsRemote(destination):
		return []platformport.Inline{platformport.Link{Target: &platformport.URLRef{URL: destination}, Inlines: text}}
	case destination == "" || strings.HasSuffix(strings.ToLower(strings.SplitN(destination, "#", 2)[0]), ".md"):
		// No destination, or a Markdown file outside the workspace, which the
		// inventory has already reported.
		return text
	}
	c.warn("the link to %s points at a local file, which is not synced; it is left as plain text", destination)

	return text
}

func (c *converter) image(n *ast.Image) []platformport.Inline {
	alt := documentparsing.PlainText(n, c.source)
	images := c.images[n]
	if len(images) == 0 {
		// Outside the workspace: reported by the inventory; the alt text stays.
		if alt == "" {
			return nil
		}

		return []platformport.Inline{platformport.Text{Value: alt}}
	}

	return []platformport.Inline{platformport.Image{Source: c.imageSource(images[0]), Alt: alt, Title: string(n.Title)}}
}

var (
	lineBreakTag = regexp.MustCompile(`(?i)^<br\s*/?>$`)
	altAttribute = regexp.MustCompile(`(?i)\salt\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	widthAttr    = regexp.MustCompile(`(?i)\swidth\s*=\s*["']?(\d+)`)
)

// rawHTML keeps what inline HTML means in plain Markdown: <br> is a line break and
// <img> an image. Other tags are dropped and the text between them stays, so
// "<kbd>Ctrl</kbd>" reads "Ctrl".
func (c *converter) rawHTML(n *ast.RawHTML) []platformport.Inline {
	var html strings.Builder
	for i := range n.Segments.Len() {
		segment := n.Segments.At(i)
		html.Write(segment.Value(c.source))
	}
	tag := strings.TrimSpace(html.String())
	if lineBreakTag.MatchString(tag) {
		return []platformport.Inline{platformport.HardBreak{}}
	}
	var inlines []platformport.Inline
	for _, ref := range c.images[n] {
		image := platformport.Image{Source: c.imageSource(ref)}
		if match := altAttribute.FindStringSubmatch(tag); match != nil {
			image.Alt = match[1] + match[2] + match[3]
		}
		if match := widthAttr.FindStringSubmatch(tag); match != nil {
			image.Width, _ = strconv.Atoi(match[1])
		}
		inlines = append(inlines, image)
	}

	return inlines
}
