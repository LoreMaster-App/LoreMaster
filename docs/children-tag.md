# The `[children]` tag

Put `[children]` on a line of its own and the page lists its child pages there.

```markdown
# Architecture

How the system fits together.

[children]
```

The tag follows the page tree LoreMaster already builds (explicit `parent:`, dotted file names,
directory indexes; see [choosing what goes where](choosing-what-goes-where.md)), so it lists what
is nested under the page, in the tree's order.

| Where | What the tag becomes |
|---|---|
| **Confluence** | The **Children Display** macro, so the list is live: pages added on Confluence appear in it too. |
| **GitHub Pages** and the **GitHub wiki** | A bulleted list of links to the child pages, written when the site or wiki is built, with links that work on that platform. |
| Your editor | Stays as `[children]`: a plain line of text in the preview. |

## Options

| Tag | Lists |
|---|---|
| `[children]` | The direct children. |
| `[children depth=2]` | Children and grandchildren (the nesting stays visible as indented bullets). |
| `[children all]` | Every descendant. |

Options are case-insensitive and can be combined; `all` wins over `depth`.

## What is not a tag

A tag has to be the only thing in its paragraph. `[children]` inside a sentence, in inline code or
in a code block is left as written, so you can document the tag itself.

## Two-way sync

A Children Display macro on a Confluence page is pulled back as `[children]` (its `all` and
`depth` settings are kept; its other settings, such as sort order, are not). So a page that gained
the macro on Confluence gets the tag in its Markdown on the next pull.
