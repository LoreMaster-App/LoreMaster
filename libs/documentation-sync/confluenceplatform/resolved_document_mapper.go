package confluenceplatform

import (
	"fmt"

	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/platformport"
)

// toStorage maps the sync's platform-neutral document onto storageformat's contract,
// node for node. An unknown node is an error, so a node added to the port without a
// mapping here fails a test instead of vanishing from pages.
func toStorage(doc platformport.Document) (storageformat.Document, error) {
	blocks, err := mapBlocks(doc.Blocks)

	return storageformat.Document{Blocks: blocks}, err
}

func mapBlocks(blocks []platformport.Block) ([]storageformat.Block, error) {
	out := make([]storageformat.Block, 0, len(blocks))
	for _, block := range blocks {
		mapped, err := mapBlock(block)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}

	return out, nil
}

func mapBlock(block platformport.Block) (storageformat.Block, error) {
	switch b := block.(type) {
	case platformport.Paragraph:
		inlines, err := mapInlines(b.Inlines)

		return storageformat.Paragraph{Inlines: inlines}, err
	case platformport.Heading:
		inlines, err := mapInlines(b.Inlines)

		return storageformat.Heading{Level: b.Level, Inlines: inlines}, err
	case platformport.Blockquote:
		inner, err := mapBlocks(b.Blocks)

		return storageformat.Blockquote{Blocks: inner}, err
	case platformport.List:
		items := make([]storageformat.ListItem, 0, len(b.Items))
		for _, item := range b.Items {
			inner, err := mapBlocks(item.Blocks)
			if err != nil {
				return nil, err
			}
			items = append(items, storageformat.ListItem{Blocks: inner})
		}

		return storageformat.List{Ordered: b.Ordered, Start: b.Start, Items: items}, nil
	case platformport.TaskList:
		items := make([]storageformat.TaskItem, 0, len(b.Items))
		for _, item := range b.Items {
			inlines, err := mapInlines(item.Inlines)
			if err != nil {
				return nil, err
			}
			nested, err := mapBlocks(item.Blocks)
			if err != nil {
				return nil, err
			}
			items = append(items, storageformat.TaskItem{Done: item.Done, Inlines: inlines, Blocks: nested})
		}

		return storageformat.TaskList{Items: items}, nil
	case platformport.Table:
		return mapTable(b)
	case platformport.ThematicBreak:
		return storageformat.ThematicBreak{}, nil
	case platformport.CodeBlock:
		return storageformat.CodeBlock{Language: b.Language, Code: b.Code}, nil
	case platformport.Diagram:
		if b.Language != "mermaid" {
			return storageformat.CodeBlock{Language: b.Language, Code: b.Source}, nil
		}
		diagram := storageformat.Mermaid{Source: b.Source}
		if b.Image != nil {
			diagram.Image = &storageformat.AttachmentRef{Filename: b.Image.Filename}
		}

		return diagram, nil
	}

	return nil, fmt.Errorf("the Confluence adapter cannot render block %T", block)
}

func mapTable(table platformport.Table) (storageformat.Block, error) {
	mapCells := func(cells []platformport.TableCell) ([]storageformat.TableCell, error) {
		out := make([]storageformat.TableCell, 0, len(cells))
		for _, cell := range cells {
			inlines, err := mapInlines(cell.Inlines)
			if err != nil {
				return nil, err
			}
			out = append(out, storageformat.TableCell{Inlines: inlines})
		}

		return out, nil
	}
	header, err := mapCells(table.Header)
	if err != nil {
		return nil, err
	}
	rows := make([][]storageformat.TableCell, 0, len(table.Rows))
	for _, row := range table.Rows {
		cells, err := mapCells(row)
		if err != nil {
			return nil, err
		}
		rows = append(rows, cells)
	}
	align := make([]storageformat.Alignment, len(table.Align))
	for i, a := range table.Align {
		align[i] = storageformat.Alignment(a)
	}

	return storageformat.Table{Header: header, Rows: rows, Align: align}, nil
}

func mapInlines(inlines []platformport.Inline) ([]storageformat.Inline, error) {
	out := make([]storageformat.Inline, 0, len(inlines))
	for _, inline := range inlines {
		mapped, err := mapInline(inline)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}

	return out, nil
}

func mapInline(inline platformport.Inline) (storageformat.Inline, error) {
	wrap := func(inner []platformport.Inline, build func([]storageformat.Inline) storageformat.Inline) (storageformat.Inline, error) {
		mapped, err := mapInlines(inner)

		return build(mapped), err
	}
	switch n := inline.(type) {
	case platformport.Text:
		return storageformat.Text{Value: n.Value}, nil
	case platformport.Emphasis:
		return wrap(n.Inlines, func(in []storageformat.Inline) storageformat.Inline { return storageformat.Emphasis{Inlines: in} })
	case platformport.Strong:
		return wrap(n.Inlines, func(in []storageformat.Inline) storageformat.Inline { return storageformat.Strong{Inlines: in} })
	case platformport.Strikethrough:
		return wrap(n.Inlines, func(in []storageformat.Inline) storageformat.Inline { return storageformat.Strikethrough{Inlines: in} })
	case platformport.CodeSpan:
		return storageformat.CodeSpan{Value: n.Value}, nil
	case platformport.HardBreak:
		return storageformat.HardBreak{}, nil
	case platformport.Image:
		image := storageformat.Image{Alt: n.Alt, Title: n.Title, Width: n.Width}
		switch source := n.Source.(type) {
		case *platformport.AttachmentRef:
			image.Source = &storageformat.AttachmentRef{Filename: source.Filename}
		case *platformport.URLRef:
			image.Source = &storageformat.URLRef{URL: source.URL}
		}

		return image, nil
	case platformport.Link:
		link := storageformat.Link{}
		switch target := n.Target.(type) {
		case platformport.PageLink:
			link.Target = storageformat.PageLink{Title: target.Title, Anchor: target.Anchor, URL: target.URL}
		case *platformport.AttachmentRef:
			link.Target = &storageformat.AttachmentRef{Filename: target.Filename}
		case *platformport.URLRef:
			link.Target = &storageformat.URLRef{URL: target.URL}
		}
		inlines, err := mapInlines(n.Inlines)
		link.Inlines = inlines

		return link, err
	}

	return nil, fmt.Errorf("the Confluence adapter cannot render inline %T", inline)
}

// fromStorage maps storageformat's document back onto the sync's platform-neutral model, the
// reverse of toStorage, for the two-way sync pull (#160). storageformat's blocks are a closed
// set with a neutral counterpart each, so unlike toStorage it cannot meet an unknown node and
// cannot fail; the parser upstream is where unrecognised storage is flagged.
func fromStorage(doc storageformat.Document) platformport.Document {
	return platformport.Document{Blocks: unmapBlocks(doc.Blocks)}
}

func unmapBlocks(blocks []storageformat.Block) []platformport.Block {
	out := make([]platformport.Block, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, unmapBlock(block))
	}

	return out
}

func unmapBlock(block storageformat.Block) platformport.Block {
	switch b := block.(type) {
	case storageformat.Paragraph:
		return platformport.Paragraph{Inlines: unmapInlines(b.Inlines)}
	case storageformat.Heading:
		return platformport.Heading{Level: b.Level, Inlines: unmapInlines(b.Inlines)}
	case storageformat.Blockquote:
		return platformport.Blockquote{Blocks: unmapBlocks(b.Blocks)}
	case storageformat.List:
		items := make([]platformport.ListItem, 0, len(b.Items))
		for _, item := range b.Items {
			items = append(items, platformport.ListItem{Blocks: unmapBlocks(item.Blocks)})
		}

		return platformport.List{Ordered: b.Ordered, Start: b.Start, Items: items}
	case storageformat.TaskList:
		items := make([]platformport.TaskItem, 0, len(b.Items))
		for _, item := range b.Items {
			items = append(items, platformport.TaskItem{Done: item.Done, Inlines: unmapInlines(item.Inlines), Blocks: unmapBlocks(item.Blocks)})
		}

		return platformport.TaskList{Items: items}
	case storageformat.Table:
		return unmapTable(b)
	case storageformat.ThematicBreak:
		return platformport.ThematicBreak{}
	case storageformat.CodeBlock:
		return platformport.CodeBlock{Language: b.Language, Code: b.Code}
	case storageformat.Mermaid:
		diagram := platformport.Diagram{Language: "mermaid", Source: b.Source}
		if b.Image != nil {
			diagram.Image = &platformport.AttachmentRef{Filename: b.Image.Filename}
		}

		return diagram
	}

	// Unreachable: storageformat.Block is a closed set, all handled above. A block added there
	// without a case here fails the round-trip test instead of silently vanishing from a pull.
	panic(fmt.Sprintf("the Confluence adapter cannot read storage block %T", block))
}

func unmapTable(table storageformat.Table) platformport.Block {
	unmapCells := func(cells []storageformat.TableCell) []platformport.TableCell {
		out := make([]platformport.TableCell, 0, len(cells))
		for _, cell := range cells {
			out = append(out, platformport.TableCell{Inlines: unmapInlines(cell.Inlines)})
		}

		return out
	}
	rows := make([][]platformport.TableCell, 0, len(table.Rows))
	for _, row := range table.Rows {
		rows = append(rows, unmapCells(row))
	}
	align := make([]platformport.Alignment, len(table.Align))
	for i, a := range table.Align {
		align[i] = platformport.Alignment(a)
	}

	return platformport.Table{Header: unmapCells(table.Header), Rows: rows, Align: align}
}

func unmapInlines(inlines []storageformat.Inline) []platformport.Inline {
	out := make([]platformport.Inline, 0, len(inlines))
	for _, inline := range inlines {
		out = append(out, unmapInline(inline))
	}

	return out
}

func unmapInline(inline storageformat.Inline) platformport.Inline {
	switch n := inline.(type) {
	case storageformat.Text:
		return platformport.Text{Value: n.Value}
	case storageformat.Emphasis:
		return platformport.Emphasis{Inlines: unmapInlines(n.Inlines)}
	case storageformat.Strong:
		return platformport.Strong{Inlines: unmapInlines(n.Inlines)}
	case storageformat.Strikethrough:
		return platformport.Strikethrough{Inlines: unmapInlines(n.Inlines)}
	case storageformat.CodeSpan:
		return platformport.CodeSpan{Value: n.Value}
	case storageformat.HardBreak:
		return platformport.HardBreak{}
	case storageformat.Image:
		image := platformport.Image{Alt: n.Alt, Title: n.Title, Width: n.Width}
		switch source := n.Source.(type) {
		case *storageformat.AttachmentRef:
			image.Source = &platformport.AttachmentRef{Filename: source.Filename}
		case *storageformat.URLRef:
			image.Source = &platformport.URLRef{URL: source.URL}
		}

		return image
	case storageformat.Link:
		link := platformport.Link{Inlines: unmapInlines(n.Inlines)}
		switch target := n.Target.(type) {
		case storageformat.PageLink:
			link.Target = platformport.PageLink{Title: target.Title, Anchor: target.Anchor, URL: target.URL}
		case *storageformat.AttachmentRef:
			link.Target = &platformport.AttachmentRef{Filename: target.Filename}
		case *storageformat.URLRef:
			link.Target = &platformport.URLRef{URL: target.URL}
		}

		return link
	}

	panic(fmt.Sprintf("the Confluence adapter cannot read storage inline %T", inline))
}
