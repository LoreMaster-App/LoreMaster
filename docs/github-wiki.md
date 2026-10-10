# Publishing to a GitHub wiki

A `github-wiki` output writes your Markdown into the repository's own wiki
(`<repo>.wiki.git`), so the docs show up under the repository's **Wiki** tab with no site, no
branch and no workflow. It uses your `git`, like the GitHub Pages output, so there is nothing
to configure beyond the repository.

```yaml
version: 1
outputs:
  - platform: github-wiki
    direction: to-platform
    repo: owner/name        # optional; empty means this workspace's origin
    content:
      - type: markdown
        roots: ["."]
        template: default
```

## Before the first publish

GitHub creates a wiki's git repository only when its first page is written. Open the
repository's **Wiki** tab and create the first page (anything; it is replaced), then publish.
Without it the publish stops with a message saying so.

## What gets written

- Each Markdown file becomes one wiki page named after its first heading
  (`# User guide` becomes `User-guide.md`). Two documents with the same title get `-2`, `-3`.
- The top-level `README.md` (or `index.md`) becomes `Home.md`.
- `_Sidebar.md` mirrors the page tree, so the nesting rules show in the wiki's sidebar.
- Links between documents are rewritten to the wiki page names, keeping `#anchors`. Images and
  linked files are published at their workspace-relative paths, and the links to them follow.
- Fenced code blocks are left exactly as written.

## What is left alone

A wiki is edited by people too, so LoreMaster never owns the folder. It records the files it
wrote in `.lore-master-wiki` and, on the next publish, replaces or removes only those. A page
somebody wrote on GitHub survives, unless a document of yours has the same page name, in which
case yours replaces it.

## Commands

```bash
lore-master-engine pages check   --output N [--exit-code]   # would a publish change anything?
lore-master-engine pages publish --output N
lore-master-engine pages build   --out DIR --output N        # the pages, into a local folder
```

In VS Code the same output appears in the Storages and Pages views, **Sync** publishes it and
**Check remote status** compares it with the published wiki.

## Limits

- One level of folders only: wiki pages are a flat list, so nesting shows in the sidebar, not in
  the page addresses.
- There is no per-page status in the Pages view for a wiki, only the whole-wiki check.
- A wiki and a Pages site of the same repository are two separate outputs.
