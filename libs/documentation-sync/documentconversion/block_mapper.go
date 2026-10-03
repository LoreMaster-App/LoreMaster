package documentconversion

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"

	"lore-master/libs/documentation-sync/platformport"
)

func (c *converter) blocks(parent ast.Node) []platformport.Block {
	var blocks []platformport.Block
	for node := parent.FirstChild(); node != nil; node = node.NextSibling() {
		blocks = append(blocks, c.block(node)...)
	}

	return blocks
}

func (c *converter) block(node ast.Node) []platformport.Block {
	switch n := node.(type) {
	case *ast.Heading:
		if n == c.document.TitleHeading {
			return nil
		}

		return []platformport.Block{platformport.Heading{Level: n.Level, Inlines: c.inlines(n)}}
	case *ast.Paragraph, *ast.TextBlock:
		if inlines := c.inlines(n); len(inlines) > 0 {
			return []platformport.Block{platformport.Paragraph{Inlines: inlines}}
		}

		return nil
	case *ast.Blockquote:
		return []platformport.Block{platformport.Blockquote{Blocks: c.blocks(n)}}
	case *ast.List:
		return []platformport.Block{c.list(n)}
	case *ast.ThematicBreak:
		return []platformport.Block{platformport.ThematicBreak{}}
	case *ast.FencedCodeBlock:
		language := string(n.Language(c.source))
		if strings.EqualFold(language, "mermaid") {
			return []platformport.Block{platformport.Diagram{Language: "mermaid", Source: c.code(n)}}
		}

		return []platformport.Block{platformport.CodeBlock{Language: language, Code: c.code(n)}}
	case *ast.CodeBlock:
		return []platformport.Block{platformport.CodeBlock{Code: c.code(n)}}
	case *extast.Table:
		return []platformport.Block{c.table(n)}
	case *ast.HTMLBlock:
		return c.htmlBlock(n)
	}
	c.warn("line %d: a %s block is not synced", c.line(node), node.Kind())

	return nil
}

func (c *converter) code(node ast.Node) string {
	var code bytes.Buffer
	lines := node.Lines()
	for i := range lines.Len() {
		segment := lines.At(i)
		code.Write(segment.Value(c.source))
	}

	return strings.TrimSuffix(code.String(), "\n")
}

// list is a task list when every item starts with a checkbox, and a plain list
// otherwise (a stray checkbox then stays as "[ ]" text).
func (c *converter) list(n *ast.List) platformport.Block {
	tasks := n.ChildCount() > 0
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		if checkbox(item) == nil {
			tasks = false
		}
	}
	if tasks {
		var list platformport.TaskList
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			first := item.FirstChild()
			var inlines []platformport.Inline
			for node := checkbox(item).NextSibling(); node != nil; node = node.NextSibling() {
				for _, inline := range c.inline(node) {
					inlines = appendInline(inlines, inline)
				}
			}
			var nested []platformport.Block
			for node := first.NextSibling(); node != nil; node = node.NextSibling() {
				nested = append(nested, c.block(node)...)
			}
			list.Items = append(list.Items, platformport.TaskItem{Done: checkbox(item).IsChecked, Inlines: inlines, Blocks: nested})
		}

		return list
	}
	list := platformport.List{Ordered: n.IsOrdered()}
	if list.Ordered {
		list.Start = n.Start
	}
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		list.Items = append(list.Items, platformport.ListItem{Blocks: c.blocks(item)})
	}

	return list
}

// checkbox is the task checkbox opening a list item, or nil.
func checkbox(item ast.Node) *extast.TaskCheckBox {
	if first := item.FirstChild(); first != nil {
		if box, isBox := first.FirstChild().(*extast.TaskCheckBox); isBox {
			return box
		}
	}

	return nil
}

func (c *converter) table(n *extast.Table) platformport.Block {
	var table platformport.Table
	for _, alignment := range n.Alignments {
		table.Align = append(table.Align, map[extast.Alignment]platformport.Alignment{
			extast.AlignLeft: platformport.AlignLeft, extast.AlignCenter: platformport.AlignCenter, extast.AlignRight: platformport.AlignRight,
		}[alignment])
	}
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []platformport.TableCell
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, platformport.TableCell{Inlines: c.inlines(cell)})
		}
		if _, isHeader := row.(*extast.TableHeader); isHeader {
			table.Header = cells
		} else {
			table.Rows = append(table.Rows, cells)
		}
	}

	return table
}

// htmlBlock keeps the images an HTML block shows (the centred logo of many READMEs)
// and drops comments quietly. Any other HTML has no platform-neutral form: it is
// reported.
func (c *converter) htmlBlock(n *ast.HTMLBlock) []platformport.Block {
	if n.HTMLBlockType == ast.HTMLBlockType2 {
		return nil
	}
	images := c.images[n]
	if len(images) == 0 {
		c.warn("line %d: an HTML block is not synced", c.line(n))

		return nil
	}
	var inlines []platformport.Inline
	for _, image := range images {
		inlines = append(inlines, platformport.Image{Source: c.imageSource(image)})
	}

	return []platformport.Block{platformport.Paragraph{Inlines: inlines}}
}

// line is a block's 1-based line in the body.
func (c *converter) line(node ast.Node) int {
	for ; node != nil; node = node.FirstChild() {
		if lines := node.Lines(); lines != nil && lines.Len() > 0 {
			return bytes.Count(c.source[:lines.At(0).Start], []byte("\n")) + 1
		}
		if node.Type() != ast.TypeBlock {
			break
		}
	}

	return 0
}
