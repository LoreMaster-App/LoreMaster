package storageformat

import "errors"

// link writes a hyperlink. A PageLink becomes <ac:link> to <ri:page ri:content-title>
// (by title, resolved when the page is shown) or, in LinkByURL mode with a known URL,
// a plain <a href>. An AttachmentRef becomes <ac:link> to <ri:attachment>. A URLRef
// becomes <a href>. A link without text shows its target.
func (r *renderer) link(link Link) error {
	switch target := link.Target.(type) {
	case PageLink:
		if target.Title == "" {
			return errors.New("a page link needs the target page's title")
		}
		if r.options.LinkMode == LinkByURL && target.URL != "" {
			href := target.URL
			if target.Anchor != "" {
				href += "#" + target.Anchor
			}

			return r.anchor(href, link.Inlines, target.Title)
		}
		r.out.WriteString("<ac:link")
		if target.Anchor != "" {
			r.out.WriteString(` ac:anchor="` + escapeText(target.Anchor) + `"`)
		}
		r.out.WriteString(`><ri:page ri:content-title="` + escapeText(target.Title) + `" />`)

		return r.linkBody(link.Inlines, target.Title)
	case *AttachmentRef:
		if target == nil || target.Filename == "" {
			return errors.New("an attachment link needs a file name")
		}
		r.out.WriteString(`<ac:link><ri:attachment ri:filename="` + escapeText(target.Filename) + `" />`)

		return r.linkBody(link.Inlines, target.Filename)
	case *URLRef:
		if target == nil || target.URL == "" {
			return errors.New("a URL link needs a URL")
		}

		return r.anchor(target.URL, link.Inlines, target.URL)
	}

	return errors.New("a link needs a page, an attachment or a URL")
}

func (r *renderer) linkBody(inlines []Inline, fallback string) error {
	if len(inlines) == 0 {
		inlines = []Inline{Text{Value: fallback}}
	}

	return r.wrapBlock("<ac:link-body>", inlines, "</ac:link-body></ac:link>")
}

func (r *renderer) anchor(href string, inlines []Inline, fallback string) error {
	if len(inlines) == 0 {
		inlines = []Inline{Text{Value: fallback}}
	}

	return r.wrapBlock(`<a href="`+escapeText(href)+`">`, inlines, "</a>")
}
