package documentconversion

import (
	"fmt"

	"lore-master/libs/documentation-sync/platformport"
	"lore-master/libs/markdown-workspace/documentdiscovery"
	"lore-master/libs/markdown-workspace/documentparsing"
)

// resolvePageLink turns a link to a Markdown file into a link to its page, by the
// page's final title, so it needs no page id and works even for a page created in
// the same sync. A fragment becomes the anchor of the heading it names. A file that
// is not being synced cannot be linked to: nil is returned with the reason.
func (w Workspace) resolvePageLink(link documentparsing.PageLink) (*platformport.PageLink, documentdiscovery.DocumentPath, string) {
	found, ok := w.lookup.Lookup(string(link.Target))
	if !ok {
		return nil, "", fmt.Sprintf("the link to %s is not synced (that file is not part of the sync), so it is left as plain text", link.Target)
	}
	target, warning := w.pageLink(found, link.Fragment)

	return target, found, warning
}

// resolveFragment is a same-page link ("#setup").
func (w Workspace) resolveFragment(self documentdiscovery.DocumentPath, fragment string) (*platformport.PageLink, string) {
	return w.pageLink(self, fragment)
}

func (w Workspace) pageLink(path documentdiscovery.DocumentPath, fragment string) (*platformport.PageLink, string) {
	page := w.targets[path]
	link := &platformport.PageLink{Title: page.title}
	if fragment == "" {
		return link, ""
	}
	heading, ok := documentparsing.FindHeading(page.headings, fragment)
	switch {
	case !ok:
		return link, fmt.Sprintf("%s has no heading for #%s, so the link goes to the top of the page", path, fragment)
	case heading.Node == page.titleHeading:
		// The title heading is not in the body: the top of the page is where it is.
	default:
		link.Anchor = heading.Text
	}

	return link, ""
}
