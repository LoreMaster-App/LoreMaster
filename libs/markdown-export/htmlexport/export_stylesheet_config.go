package htmlexport

// stylesheet is inlined in every exported page. It follows the reader's light or dark setting and
// prints cleanly.
const stylesheet = `:root{color-scheme:light dark;--fg:#1f2328;--bg:#fff;--muted:#59636e;--rule:#d1d9e0;--code:#f6f8fa;--link:#0969da}
@media (prefers-color-scheme:dark){:root{--fg:#e6edf3;--bg:#0d1117;--muted:#9198a1;--rule:#3d444d;--code:#151b23;--link:#4493f8}}
body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.6 -apple-system,"Segoe UI",Helvetica,Arial,sans-serif}
main{max-width:860px;margin:0 auto;padding:32px 24px}
h1,h2{border-bottom:1px solid var(--rule);padding-bottom:.3em}
h1,h2,h3,h4,h5,h6{line-height:1.25;margin:1.5em 0 .6em}
a{color:var(--link)}
code,pre{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:.9em}
code{background:var(--code);padding:.15em .35em;border-radius:4px}
pre{background:var(--code);padding:14px 16px;border-radius:6px;overflow:auto}
pre code{background:none;padding:0}
blockquote{margin:1em 0;padding:0 1em;color:var(--muted);border-left:4px solid var(--rule)}
table{border-collapse:collapse;margin:1em 0}
th,td{border:1px solid var(--rule);padding:6px 13px}
th{background:var(--code)}
img{max-width:100%}
.diagram{margin:1em 0;overflow:auto;background:#fff;padding:.75em;border-radius:6px}
.diagram svg{max-width:100%;height:auto}
hr{border:0;border-top:1px solid var(--rule)}
@media print{body{background:#fff;color:#000}main{max-width:none;padding:0}pre{white-space:pre-wrap}}
`
