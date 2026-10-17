package documentexport

import (
	"lore-master/libs/markdown-export/browserexport"
	"lore-master/libs/markdown-export/exportmodel"
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

	return pictures
}
