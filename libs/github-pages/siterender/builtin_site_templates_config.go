package siterender

// BuiltinTemplate is the template LoreMaster ships under name: default, minimal, handbook or
// headless. ok is false for any other name.
func BuiltinTemplate(name string) (Template, bool) {
	switch name {
	case "", "default":
		return Template{Name: "default", Shell: defaultShell, LayoutCSS: sidebarLayoutCSS}, true
	case "minimal":
		return Template{Name: "minimal", Shell: minimalShell, LayoutCSS: minimalLayoutCSS}, true
	case "handbook":
		return Template{Name: "handbook", Shell: handbookShell, LayoutCSS: handbookLayoutCSS}, true
	case "headless":
		return Template{Name: "headless", Headless: true}, true
	}

	return Template{}, false
}

// minimalShell is one centred column with the page tree folded away under the title.
const minimalShell = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{title}}</title>
{{style}}
</head>
<body class="lm-minimal">
<header class="lm-top">
<a class="lm-site" href="{{root}}index.html">{{site_title}}</a>
<details><summary>Pages</summary>
{{nav}}
</details>
</header>
<main class="lm-column">
{{breadcrumbs}}
{{content}}
</main>
</body>
</html>
`

const minimalLayoutCSS = `* { box-sizing: border-box; }
body.lm-minimal {
  margin: 0;
  color: var(--lm-text);
  background: var(--lm-bg);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  line-height: 1.65;
}
.lm-top { max-width: 44rem; margin: 0 auto; padding: 1.25rem 1.5rem 0; }
.lm-site { font-weight: 600; color: var(--lm-text); text-decoration: none; }
.lm-top details { margin-top: 0.5rem; color: var(--lm-muted); }
.lm-top summary { cursor: pointer; }
.lm-top ul { list-style: none; margin: 0.5rem 0 0; padding: 0; }
.lm-top a { color: var(--lm-accent); text-decoration: none; }
.lm-depth-1 { padding-left: 1rem; }
.lm-depth-2 { padding-left: 2rem; }
.lm-depth-3 { padding-left: 3rem; }
.lm-active > a { font-weight: 600; }
.lm-column { max-width: 44rem; margin: 0 auto; padding: 1rem 1.5rem 4rem; }
.lm-breadcrumbs { font-size: 0.9rem; color: var(--lm-muted); margin-bottom: 1rem; }
.lm-breadcrumbs a { color: var(--lm-muted); }
`

// handbookShell is a printed-manual look: a banner, a contents column and serif text.
const handbookShell = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{title}} · {{site_title}}</title>
{{style}}
</head>
<body class="lm-handbook">
<header class="lm-banner"><a href="{{root}}index.html">{{site_title}}</a></header>
<div class="lm-book">
<nav class="lm-contents">
<div class="lm-contents-title">Contents</div>
{{nav}}
</nav>
<main class="lm-page">
{{breadcrumbs}}
{{content}}
<footer class="lm-source">Source: {{source}}</footer>
</main>
</div>
</body>
</html>
`

const handbookLayoutCSS = `* { box-sizing: border-box; }
body.lm-handbook {
  margin: 0;
  color: #2b2b2b;
  background: #faf8f3;
  font-family: Georgia, "Times New Roman", serif;
  line-height: 1.7;
}
.lm-banner { background: #243447; padding: 1rem 2rem; }
.lm-banner a { color: #ffffff; text-decoration: none; font-size: 1.25rem; letter-spacing: 0.02em; }
.lm-book { display: flex; max-width: 72rem; margin: 0 auto; }
.lm-contents {
  flex: 0 0 16rem;
  padding: 2rem 1.25rem;
  border-right: 1px solid #d9d2c3;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  font-size: 0.92rem;
}
.lm-contents-title { text-transform: uppercase; font-size: 0.75rem; letter-spacing: 0.08em; color: var(--lm-muted); margin-bottom: 0.75rem; }
.lm-contents ul { list-style: none; margin: 0; padding: 0; }
.lm-contents a { color: #243447; text-decoration: none; display: block; padding: 0.2rem 0; }
.lm-depth-1 { padding-left: 0.9rem; }
.lm-depth-2 { padding-left: 1.8rem; }
.lm-depth-3 { padding-left: 2.7rem; }
.lm-active > a { font-weight: 700; }
.lm-page { flex: 1 1 auto; min-width: 0; padding: 2rem 3rem 4rem; }
.lm-breadcrumbs { font-family: -apple-system, "Segoe UI", sans-serif; font-size: 0.85rem; color: var(--lm-muted); margin-bottom: 1rem; }
.lm-source { margin-top: 3rem; padding-top: 1rem; border-top: 1px solid #d9d2c3; font-size: 0.8rem; color: var(--lm-muted); }
@media (max-width: 48rem) {
  .lm-book { flex-direction: column; }
  .lm-contents { border-right: none; border-bottom: 1px solid #d9d2c3; }
  .lm-page { padding: 1.5rem; }
}
`
