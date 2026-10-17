package browserexport

import "strings"

// hasDiagrams reports whether a page has a Mermaid block to draw.
func hasDiagrams(page string) bool {
	return strings.Contains(page, `class="language-mermaid"`)
}

// diagramScript draws every Mermaid block of the page as a picture. It says it is going to before the page has loaded, so
// the height probe waits for it, and it always ends by recording the height, whether or not the library loaded or a
// diagram was wrong: a diagram that cannot be drawn stays as code.
const diagramScript = `<script>
window.lmWaitForDiagrams = true;
(async function () {
  try {
    mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'default' });
    let count = 0;
    for (const block of document.querySelectorAll('pre > code.language-mermaid')) {
      try {
        const { svg } = await mermaid.render('lore-master-diagram-' + count, block.textContent);
        const holder = document.createElement('div');
        holder.className = 'diagram';
        holder.innerHTML = svg;
        block.parentElement.replaceWith(holder);
      } catch (error) {
        for (const leftover of [document.getElementById('lore-master-diagram-' + count), document.getElementById('dlore-master-diagram-' + count)]) {
          if (leftover) { leftover.remove(); }
        }
      }
      count += 1;
    }
  } catch (error) {
    // The library did not load: the blocks stay as code.
  } finally {
    document.documentElement.setAttribute('data-diagrams', 'done');
    if (window.recordHeight) { window.recordHeight(); }
  }
})();
</script>`

// withDiagrams adds the Mermaid library and the script that draws with it at the end of a page, when the page has diagrams and
// there is a library to draw them with.
func withDiagrams(page, mermaidPath string) string {
	if mermaidPath == "" || !hasDiagrams(page) {
		return page
	}
	scripts := `<script src="` + fileURL(mermaidPath) + `"></script>` + "\n" + diagramScript + "\n"
	if index := strings.LastIndex(page, "</body>"); index >= 0 {
		return page[:index] + scripts + page[index:]
	}

	return page + scripts
}
