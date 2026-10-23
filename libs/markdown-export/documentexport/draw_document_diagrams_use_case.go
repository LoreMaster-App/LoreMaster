package documentexport

import (
	"lore-master/libs/markdown-export/browserexport"
	"lore-master/libs/markdown-export/exportmodel"
	"lore-master/libs/markdown-export/htmlexport"
)

// drawDocumentDiagrams draws a document's Mermaid diagrams for an HTML page, by their source, using the browser and the Mermaid
// library when there are both. It never fails the export: with no library, no browser, or a diagram that is not valid, the
// diagrams are left to be written as code, which is what every other format does with them. A preview draws its own, so it gets none.
func drawDocumentDiagrams(document exportmodel.Document, options Options) map[string]string {
	if options.Preview {
		return nil
	}
	sources := mermaidSources(document.Blocks)
	if len(sources) == 0 {
		return nil
	}
	pictures, err := browserexport.DrawDiagrams(sources, browserexport.Options{BrowserPath: options.BrowserPath, MermaidPath: options.MermaidPath})
	if err != nil {
		return nil
	}
	keyed := make(map[string]string, len(pictures))
	for source, picture := range pictures {
		keyed[htmlexport.DiagramKey("mermaid", source)] = picture
	}

	return keyed
}

// drawD2Diagrams draws a document's D2 diagrams with the caller's drawer, by their source. D2 needs no browser, so it is drawn for
// every HTML-based format, PDF and PNG included. A diagram the drawer refuses stays code; the export never fails because of one.
func drawD2Diagrams(document exportmodel.Document, options Options) map[string]string {
	if options.Preview || options.DrawD2 == nil {
		return nil
	}
	var pictures map[string]string
	for _, source := range diagramSources(document.Blocks, "d2") {
		svg, err := options.DrawD2(source)
		if err != nil || svg == "" {
			continue
		}
		if pictures == nil {
			pictures = map[string]string{}
		}
		pictures[htmlexport.DiagramKey("d2", source)] = svg
	}

	return pictures
}

// withD2 adds the D2 pictures to the Mermaid ones.
func withD2(pictures map[string]string, d2 map[string]string) map[string]string {
	if len(d2) == 0 {
		return pictures
	}
	merged := make(map[string]string, len(pictures)+len(d2))
	for key, picture := range pictures {
		merged[key] = picture
	}
	for key, picture := range d2 {
		merged[key] = picture
	}

	return merged
}
