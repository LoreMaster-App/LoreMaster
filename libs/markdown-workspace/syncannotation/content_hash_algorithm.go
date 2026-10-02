package syncannotation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash fingerprints a document body (as Read returns it) for change detection:
// SHA-256 over the body with any byte-order mark removed and CRLF turned into LF, so a
// checkout with core.autocrlf, or an editor that adds a BOM, does not look like an edit.
// Trailing whitespace is kept: in Markdown two trailing spaces are a line break.
func ContentHash(body []byte) string {
	normalised := bytes.ReplaceAll(bytes.TrimPrefix(body, byteOrderMark), []byte("\r\n"), []byte("\n"))
	sum := sha256.Sum256(normalised)

	return "sha256:" + hex.EncodeToString(sum[:])
}
