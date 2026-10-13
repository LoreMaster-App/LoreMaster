// Package markdowndocs is the markdown-docs generator: it imports Markdown another tool wrote
// (TypeDoc's Markdown plugin, dokka, a docs folder under build/) as ordinary pages. The folder
// layout is kept, so relative links keep working; each page gets one top heading and a title no
// other page has, an index page is added when the tool wrote none, and the images the pages show
// are copied next to them. Markdown only: HTML is never parsed.
package markdowndocs
