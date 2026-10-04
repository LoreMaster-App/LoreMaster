package syncexecution

import (
	"bytes"
	"encoding/base64"
)

// mermaidMetadata marks the Mermaid source lore-master embeds in a diagram's rendered SVG, so a
// two-way pull can recover it even if the page's code macro — the source's primary carrier — was
// removed on the platform. The source is base64-encoded, so it is safe inside an XML comment
// (base64 never contains "--").
const mermaidMetadata = "lore-master:mermaid="

// injectMermaidSource embeds source into an SVG as a comment just inside the <svg> element, so
// the uploaded picture carries its own source. Bytes that are not an SVG are returned unchanged.
func injectMermaidSource(svg []byte, source string) []byte {
	tag := bytes.Index(bytes.ToLower(svg), []byte("<svg"))
	if tag < 0 {
		return svg
	}
	end := bytes.IndexByte(svg[tag:], '>')
	if end < 0 {
		return svg
	}
	at := tag + end + 1
	comment := []byte("<!--" + mermaidMetadata + base64.StdEncoding.EncodeToString([]byte(source)) + "-->")
	out := make([]byte, 0, len(svg)+len(comment))
	out = append(out, svg[:at]...)
	out = append(out, comment...)
	out = append(out, svg[at:]...)

	return out
}

// extractMermaidSource reads the Mermaid source injectMermaidSource embedded, if any.
func extractMermaidSource(svg []byte) (string, bool) {
	marker := []byte("<!--" + mermaidMetadata)
	start := bytes.Index(svg, marker)
	if start < 0 {
		return "", false
	}
	start += len(marker)
	end := bytes.Index(svg[start:], []byte("-->"))
	if end < 0 {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(string(svg[start : start+end]))
	if err != nil {
		return "", false
	}

	return string(decoded), true
}
