package syncexecution

import (
	"bytes"
	"encoding/base64"
)

// diagramMetadata marks the diagram source lore-master embeds in a diagram's rendered SVG, so a
// two-way pull can recover it even if the page's code macro — the source's primary carrier — was
// removed on the platform. The marker names the language ("lore-master:mermaid=", "lore-master:d2=").
// The source is base64-encoded, so it is safe inside an XML comment (base64 never contains "--").
const diagramMetadata = "lore-master:"

// diagramLanguages are the languages a diagram can be in, in the order markers are looked for.
var diagramLanguages = []string{"mermaid", "d2"}

// injectDiagramSource embeds source into an SVG as a comment just inside the <svg> element, so
// the uploaded picture carries its own source. Bytes that are not an SVG are returned unchanged.
func injectDiagramSource(svg []byte, language string, source string) []byte {
	tag := bytes.Index(bytes.ToLower(svg), []byte("<svg"))
	if tag < 0 {
		return svg
	}
	end := bytes.IndexByte(svg[tag:], '>')
	if end < 0 {
		return svg
	}
	at := tag + end + 1
	comment := []byte("<!--" + diagramMetadata + language + "=" + base64.StdEncoding.EncodeToString([]byte(source)) + "-->")
	out := make([]byte, 0, len(svg)+len(comment))
	out = append(out, svg[:at]...)
	out = append(out, comment...)
	out = append(out, svg[at:]...)

	return out
}

// extractDiagramSource reads the language and source injectDiagramSource embedded, if any.
func extractDiagramSource(svg []byte) (string, string, bool) {
	for _, language := range diagramLanguages {
		marker := []byte("<!--" + diagramMetadata + language + "=")
		start := bytes.Index(svg, marker)
		if start < 0 {
			continue
		}
		start += len(marker)
		end := bytes.Index(svg[start:], []byte("-->"))
		if end < 0 {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(string(svg[start : start+end]))
		if err != nil {
			continue
		}

		return language, string(decoded), true
	}

	return "", "", false
}
