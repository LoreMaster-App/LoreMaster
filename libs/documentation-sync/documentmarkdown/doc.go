// Package documentmarkdown serialises a platform-neutral Document back into Markdown. It is
// the reverse of documentconversion (Markdown into a Document) and one half of the two-way
// sync pull (#160, #95): the other half parses a platform's storage format into a Document.
// It is pure — no I/O — and conservative: every node that came from Markdown round-trips,
// and anything it cannot represent faithfully is flagged rather than mangled.
package documentmarkdown
