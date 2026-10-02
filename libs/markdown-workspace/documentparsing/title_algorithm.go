package documentparsing

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

// headingTitle returns the plain text of the document's first top-level H1 and that
// heading. A heading nested in a block quote or list is content, not the document's
// title, and an H1 with no text is skipped.
func headingTitle(document ast.Node, source []byte) (string, *ast.Heading) {
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		heading, isHeading := node.(*ast.Heading)
		if !isHeading || heading.Level != 1 {
			continue
		}
		if title := PlainText(heading, source); title != "" {
			return title, heading
		}
	}

	return "", nil
}

// PlainText is a node's text with inline formatting dropped ("Hello *world*" is "Hello
// world"), escapes and character references resolved, code spans kept verbatim, raw
// HTML left out, and whitespace collapsed.
func PlainText(node ast.Node, source []byte) string {
	var text strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.CodeSpan:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if segment, isText := child.(*ast.Text); isText {
					text.Write(segment.Segment.Value(source))
				}
			}

			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			text.Write(n.Label(source))
		case *ast.String:
			text.Write(n.Value)
		case *ast.Text:
			text.Write(Unescaped(n.Segment.Value(source)))
			if n.SoftLineBreak() || n.HardLineBreak() {
				text.WriteByte(' ')
			}
		}

		return ast.WalkContinue, nil
	})

	return strings.Join(strings.Fields(text.String()), " ")
}

// Unescaped resolves a text segment's backslash escapes and character references, the
// way a Markdown renderer prints it.
func Unescaped(value []byte) []byte {
	return util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(value)))
}

// orderPrefix is a sibling-order prefix: digits, then at least one of '-', '_' or a
// space. Not '.', which already means nesting: "01.setup.md" is a child of "01.md".
var orderPrefix = regexp.MustCompile(`^(\d+)[-_ ]+(.+)$`)

// fileNameTitle is the fallback title and the sibling order, both taken from the last
// dotted segment of the file name, because the segments before it only say where the
// file nests: "readme.02-setup.md" is "setup", second among readme's children.
func fileNameTitle(documentPath string) (string, int, bool) {
	name := path.Base(documentPath)
	name = strings.TrimSuffix(name, path.Ext(name))
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 && dot < len(name)-1 {
		name = name[dot+1:]
	}
	match := orderPrefix.FindStringSubmatch(name)
	if match == nil {
		return name, 0, false
	}
	order, err := strconv.Atoi(match[1])
	if err != nil {
		return name, 0, false
	}

	return match[2], order, true
}
