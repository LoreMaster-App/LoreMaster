package browserexport

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// drawnDiagram is how a drawn diagram comes back out of the browser: in a script of an unknown type, whose text the browser prints
// as it is, so the picture arrives without being changed by the HTML around it.
var drawnDiagram = regexp.MustCompile(`(?s)<script type="text/lore-master-svg" data-index="(\d+)">(.*?)</script>`)

// diagramsPage is a page that draws each of the sources with the Mermaid library and leaves the pictures where DrawDiagrams
// can read them. The sources travel as JSON, with every "<" escaped, so nothing in a diagram can end the script.
func diagramsPage(sources []string, mermaidPath string) (string, error) {
	encoded, err := json.Marshal(sources)
	if err != nil {
		return "", err
	}
	safe := strings.ReplaceAll(string(encoded), "<", `<`)

	return `<!doctype html><html><head><meta charset="utf-8"></head><body>
<script type="application/json" id="sources">` + safe + `</script>
<script src="` + fileURL(mermaidPath) + `"></script>
<script>
(async function () {
  try {
    const sources = JSON.parse(document.getElementById('sources').textContent);
    mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'default' });
    for (let index = 0; index < sources.length; index += 1) {
      try {
        const { svg } = await mermaid.render('lore-master-diagram-' + index, sources[index]);
        const holder = document.createElement('script');
        holder.type = 'text/lore-master-svg';
        holder.setAttribute('data-index', String(index));
        holder.textContent = svg;
        document.body.appendChild(holder);
      } catch (error) {
        // This one stays as code.
      }
    }
  } catch (error) {
    // The library did not load: nothing is drawn.
  }
})();
</script></body></html>`, nil
}

// DrawDiagrams draws Mermaid diagrams with the browser and returns each picture as SVG, by the source it came from. A source that
// is not a valid diagram is left out of the result, so the caller keeps it as code. Without a library or a browser the result is
// empty and so is the error: a document with diagrams is exported as it always was.
func DrawDiagrams(sources []string, options Options) (map[string]string, error) {
	library := LocateMermaid(options.MermaidPath)
	if library == "" || len(sources) == 0 {
		return nil, nil
	}
	page, err := diagramsPage(sources, library)
	if err != nil {
		return nil, err
	}

	pictures := map[string]string{}
	_, err = withBrowser(options, func(run runner) ([]byte, error) {
		file, err := run.writePage(page)
		if err != nil {
			return nil, err
		}
		dump, err := run.browse("--virtual-time-budget=20000", "--dump-dom", file)
		if err != nil {
			return nil, err
		}
		for _, match := range drawnDiagram.FindAllStringSubmatch(string(dump), -1) {
			index, convertError := strconv.Atoi(match[1])
			if convertError != nil || index < 0 || index >= len(sources) {
				continue
			}
			pictures[sources[index]] = match[2]
		}

		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("draw the diagrams: %w", err)
	}

	return pictures, nil
}
