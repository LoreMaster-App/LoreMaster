package siterender

// pageTemplateSource is the HTML shell every page shares. The Markdown itself is not
// converted on our side: it is embedded verbatim (base64, to survive any byte) in a
// non-executable <script> and rendered in the browser by renderScript. Nav labels, the
// title and the hrefs are filled per page so the same shell serves every document.
// pageTemplateSource carries the Markdown as base64 in a data attribute, not inside a
// <script>: html/template escapes a script element's contents as JavaScript (it turns the
// base64 '/' into '\/'), which breaks atob at runtime. An attribute value is HTML-escaped,
// and base64's alphabet has no HTML-special characters, so it survives byte for byte.
const pageTemplateSource = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="stylesheet" href="{{.AssetsPrefix}}assets/lore-master.css">
</head>
<body>
<div class="lm-layout">
<nav class="lm-nav">
<div class="lm-nav-title">{{.SiteTitle}}</div>
<ul>
{{range .Nav}}<li class="lm-depth-{{.Depth}}{{if .Active}} lm-active{{end}}"><a href="{{.Href}}">{{.Label}}</a></li>
{{end}}</ul>
</nav>
<main class="lm-content"><article id="lm-markdown" class="markdown-body"></article></main>
</div>
<div id="lm-source" data-markdown="{{.MarkdownBase64}}" hidden></div>
<script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
<script>{{.RenderScript}}</script>
</body>
</html>
`

// renderScript decodes the embedded Markdown (UTF-8 base64), renders it with marked,
// rewrites internal links to other Markdown files so they point at the generated pages, and
// promotes fenced ` + "```mermaid" + ` blocks into mermaid diagrams. It runs on load; there
// is no build step and nothing is fetched beyond the two CDN libraries above. Because each
// page sits where its Markdown did, relative links resolve once their .md suffix is .html.
const renderScript = `(function () {
  var source = document.getElementById('lm-source');
  if (!source) { return; }
  var bytes = Uint8Array.from(atob(source.getAttribute('data-markdown')), function (c) { return c.charCodeAt(0); });
  var markdown = new TextDecoder('utf-8').decode(bytes);
  var article = document.getElementById('lm-markdown');
  article.innerHTML = marked.parse(markdown);
  article.querySelectorAll('a[href]').forEach(function (link) {
    var href = link.getAttribute('href');
    if (/^[a-z][a-z0-9+.-]*:/i.test(href) || href.charAt(0) === '#' || href.charAt(0) === '/') { return; }
    link.setAttribute('href', href.replace(/\.(md|markdown)(#.*)?$/i, '.html$2'));
  });
  article.querySelectorAll('pre > code.language-mermaid').forEach(function (code) {
    var diagram = document.createElement('div');
    diagram.className = 'mermaid';
    diagram.textContent = code.textContent;
    code.parentNode.replaceWith(diagram);
  });
  if (window.mermaid) {
    mermaid.initialize({ startOnLoad: false });
    mermaid.run({ querySelector: '.mermaid' });
  }
})();`

// defaultCSS is the one bundled theme: a fixed sidebar and a readable content column. A
// theme/CSS choice in the config comes later; for now every site ships this stylesheet.
const defaultCSS = `:root {
  --lm-sidebar: 18rem;
  --lm-text: #1f2328;
  --lm-muted: #656d76;
  --lm-accent: #0969da;
  --lm-border: #d0d7de;
  --lm-bg: #ffffff;
  --lm-nav-bg: #f6f8fa;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  color: var(--lm-text);
  background: var(--lm-bg);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  line-height: 1.6;
}
.lm-layout { display: flex; min-height: 100vh; }
.lm-nav {
  width: var(--lm-sidebar);
  flex: 0 0 var(--lm-sidebar);
  background: var(--lm-nav-bg);
  border-right: 1px solid var(--lm-border);
  padding: 1.5rem 1rem;
  position: sticky;
  top: 0;
  align-self: flex-start;
  height: 100vh;
  overflow-y: auto;
}
.lm-nav-title { font-weight: 600; font-size: 1.1rem; margin-bottom: 1rem; }
.lm-nav ul { list-style: none; margin: 0; padding: 0; }
.lm-nav li { margin: 0.15rem 0; }
.lm-nav a {
  color: var(--lm-text);
  text-decoration: none;
  display: block;
  padding: 0.2rem 0.4rem;
  border-radius: 6px;
  font-size: 0.95rem;
}
.lm-nav a:hover { background: rgba(0, 0, 0, 0.05); }
.lm-depth-1 { padding-left: 1rem; }
.lm-depth-2 { padding-left: 2rem; }
.lm-depth-3 { padding-left: 3rem; }
.lm-depth-4 { padding-left: 4rem; }
.lm-active > a { background: var(--lm-accent); color: #ffffff; font-weight: 600; }
.lm-content {
  flex: 1 1 auto;
  min-width: 0;
  padding: 2.5rem 3rem;
  max-width: 54rem;
}
.markdown-body h1, .markdown-body h2 { border-bottom: 1px solid var(--lm-border); padding-bottom: 0.3rem; }
.markdown-body a { color: var(--lm-accent); }
.markdown-body pre {
  background: var(--lm-nav-bg);
  padding: 1rem;
  border-radius: 6px;
  overflow-x: auto;
}
.markdown-body code {
  background: rgba(175, 184, 193, 0.2);
  padding: 0.15em 0.35em;
  border-radius: 6px;
  font-size: 0.9em;
}
.markdown-body pre code { background: none; padding: 0; }
.markdown-body table { border-collapse: collapse; }
.markdown-body th, .markdown-body td { border: 1px solid var(--lm-border); padding: 0.4rem 0.7rem; }
.markdown-body img { max-width: 100%; }
.markdown-body blockquote {
  margin: 0;
  padding: 0 1rem;
  color: var(--lm-muted);
  border-left: 0.25rem solid var(--lm-border);
}
@media (max-width: 48rem) {
  .lm-layout { flex-direction: column; }
  .lm-nav { width: 100%; height: auto; position: static; border-right: none; border-bottom: 1px solid var(--lm-border); }
  .lm-content { padding: 1.5rem; }
}
`
