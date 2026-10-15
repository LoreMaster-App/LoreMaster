# Page templates

A template decides how an output's pages are laid out around the content LoreMaster writes.
Every output has one `template` value; empty is the platform's own layout.

```jsonc
"loreMaster.outputs": [
  { "platform": "github-pages", "template": "handbook" },
  { "platform": "github-wiki",  "template": "templates/wiki.md" },
  { "platform": "confluence", "space": "ENG", "template": "98765" }
]
```

You can also set it in the storage form (**Page template**).

A template is a shell with a slot: it holds `{{content}}`, and LoreMaster puts each page's
content there. `{{content}}` is the only required placeholder.

## GitHub Pages

| Value | Result |
|---|---|
| empty, `default` | A sidebar with the page tree and the content beside it. |
| `minimal` | One centred column; the page tree folds away under the title. |
| `handbook` | A manual look: a banner, a contents column, serif text and the source file at the foot. |
| `headless` | No pages. Every document's Markdown at its own path and a `site.json` describing the tree, for a front end you build yourself (React, Astro, anything). |
| a folder, for example `templates/site` | Your own shell. |

### Your own site template

A folder with a `template.html`. Every other file in the folder is published under
`assets/template/`, so a stylesheet at `templates/site/theme.css` is
`{{root}}assets/template/theme.css` in the shell.

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{title}} · {{site_title}}</title>
  {{style}}
  <link rel="stylesheet" href="{{root}}assets/template/theme.css">
</head>
<body>
  <header><a href="{{root}}index.html">{{site_title}}</a></header>
  <aside>{{nav}}</aside>
  <main>{{breadcrumbs}}{{content}}</main>
</body>
</html>
```

| Placeholder | Becomes |
|---|---|
| `{{content}}` | The rendered page. **Required.** |
| `{{title}}` | The page's title. |
| `{{site_title}}` | The site's title. |
| `{{nav}}` | The whole page tree as a list, with the current page marked `lm-active`. |
| `{{breadcrumbs}}` | The page's ancestors as links; empty for a top-level page. |
| `{{root}}` | The way back to the site's root from this page (`""`, `"../"`, …). |
| `{{style}}` | A link to the styles of the rendered Markdown. Leave it out to style the content yourself. |
| `{{source}}` | The Markdown file the page came from. |

A placeholder LoreMaster does not know is an error, so a typo is found when you build, not in
the browser. A folder whose name is also a built-in's (`./minimal`) is written with the `./`.

### Building your own front end

`headless` is for teams that want a React app (or any other) to be the site. The output is:

```
site.json          { "title", "home", "pages": [ { "path", "title", "depth", "parent", "children" } ] }
README.md          every document, at its own path, with [children] expanded and diagrams drawn
docs/setup.md
...
```

Your app reads `site.json`, fetches the Markdown files and renders them its own way.

## GitHub wiki

A Markdown file with `{{content}}`. It wraps every page; the sidebar is not wrapped.
`{{title}}` and `{{source}}` are also available.

```markdown
> Generated from `{{source}}`. Edit it in the repository, not here.

{{content}}
```

## Confluence

The template is a Confluence page, or a file in the workspace, in storage format. Put
`{{content}}` in a paragraph of its own where the page should go.

1. Build the layout in the Confluence editor (an info panel, a table of contents macro, a footer).
2. Type `{{content}}` in a paragraph where the page goes. The editor turns it into monospace; that
   is understood too.
3. Set `template` to the page's id or its address.

Each synced page is the template with `{{content}}` replaced by the page's content; `{{title}}`
is replaced by the page's title.

- **Editing the template re-publishes the pages.** The template is part of what a page is compared
  with, so the next sync updates every page that used it.
- **Two-way pull keeps working.** The content sits between two invisible anchor macros, and a
  pull reads only what is between them; the template's own text never reaches your Markdown.
  Do not delete the anchors.
- A template page is not synced; it is only read. Keep it outside the parent page LoreMaster owns.

## Not yet

A page choosing its own template through front matter. One template per output covers the
common case; tell us if you need more.
