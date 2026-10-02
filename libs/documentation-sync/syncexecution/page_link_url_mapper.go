package syncexecution

import "lore-master/libs/documentation-sync/platformport"

// withLinkURLs returns a copy of blocks in which every page link whose target page is
// known carries its URL, and reports whether one is still unknown (its page is created
// later in this sync). The input is not modified.
func withLinkURLs(blocks []platformport.Block, urls map[string]string) ([]platformport.Block, bool) {
	mapper := linkMapper{urls: urls}

	return mapper.blocks(blocks), mapper.missing
}

type linkMapper struct {
	urls    map[string]string
	missing bool
}

func (m *linkMapper) blocks(blocks []platformport.Block) []platformport.Block {
	if blocks == nil {
		return nil
	}
	copied := make([]platformport.Block, len(blocks))
	for i, block := range blocks {
		switch b := block.(type) {
		case platformport.Paragraph:
			copied[i] = platformport.Paragraph{Inlines: m.inlines(b.Inlines)}
		case platformport.Heading:
			copied[i] = platformport.Heading{Level: b.Level, Inlines: m.inlines(b.Inlines)}
		case platformport.Blockquote:
			copied[i] = platformport.Blockquote{Blocks: m.blocks(b.Blocks)}
		case platformport.List:
			list := platformport.List{Ordered: b.Ordered, Start: b.Start, Items: make([]platformport.ListItem, len(b.Items))}
			for j, item := range b.Items {
				list.Items[j] = platformport.ListItem{Blocks: m.blocks(item.Blocks)}
			}
			copied[i] = list
		case platformport.TaskList:
			tasks := platformport.TaskList{Items: make([]platformport.TaskItem, len(b.Items))}
			for j, item := range b.Items {
				tasks.Items[j] = platformport.TaskItem{Done: item.Done, Inlines: m.inlines(item.Inlines)}
			}
			copied[i] = tasks
		case platformport.Table:
			table := platformport.Table{Header: m.cells(b.Header), Align: b.Align, Rows: make([][]platformport.TableCell, len(b.Rows))}
			for j, row := range b.Rows {
				table.Rows[j] = m.cells(row)
			}
			copied[i] = table
		default:
			copied[i] = block
		}
	}

	return copied
}

func (m *linkMapper) cells(cells []platformport.TableCell) []platformport.TableCell {
	if cells == nil {
		return nil
	}
	copied := make([]platformport.TableCell, len(cells))
	for i, cell := range cells {
		copied[i] = platformport.TableCell{Inlines: m.inlines(cell.Inlines)}
	}

	return copied
}

func (m *linkMapper) inlines(inlines []platformport.Inline) []platformport.Inline {
	if inlines == nil {
		return nil
	}
	copied := make([]platformport.Inline, len(inlines))
	for i, inline := range inlines {
		switch n := inline.(type) {
		case platformport.Emphasis:
			copied[i] = platformport.Emphasis{Inlines: m.inlines(n.Inlines)}
		case platformport.Strong:
			copied[i] = platformport.Strong{Inlines: m.inlines(n.Inlines)}
		case platformport.Strikethrough:
			copied[i] = platformport.Strikethrough{Inlines: m.inlines(n.Inlines)}
		case platformport.Link:
			if page, isPage := n.Target.(platformport.PageLink); isPage {
				page.URL = m.urls[page.Title]
				if page.URL == "" {
					m.missing = true
				}
				n.Target = page
			}
			n.Inlines = m.inlines(n.Inlines)
			copied[i] = n
		default:
			copied[i] = inline
		}
	}

	return copied
}
