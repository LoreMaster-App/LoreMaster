# Sync state: `lore-master.json`

A sync has to remember which page each Markdown file became. That memory lives in **one file,
`lore-master.json`, at the workspace root**. It used to be a `<!-- lore-master … -->` comment at the
top of every synced file, which meant every sync touched every file in git.

## What it looks like

```json
{
  "version": 1,
  "storages": {
    "Technical docs": {
      "platform": "confluence",
      "baseUrl": "https://acme.atlassian.net/wiki",
      "space": "ENG",
      "pages": {
        "README.md": {
          "pageId": "123456",
          "parentId": "123000",
          "version": 7,
          "contentHash": "sha256:…",
          "renderHash": "sha256:…",
          "attachments": { "diagram-1.svg": "sha256:…" },
          "syncedAt": "2026-10-15T12:00:00Z"
        }
      }
    },
    "User manuals": { "…": "…" }
  }
}
```

- It is keyed by the **storage name** (`outputs[].name`; see
  [Naming your storages](configuration.md#naming-your-storages)). Two Confluences get two entries,
  and one file can be a page in both.
- Pages are keyed by the file's workspace-relative path, with `/` separators.
- Keys are sorted and the file is indented, so a diff shows exactly which pages a sync changed.
- Commit it. It is what lets a re-run update pages instead of adopting them again, and what lets a
  teammate's sync continue yours.

## What stays in the Markdown

Only what **you** decide about a file stays in its annotation: `title:`, `parent:` and (for
generated files) `generated:`. A file with none of those has no annotation at all.

```markdown
<!-- lore-master
parent: ./architecture/README.md
-->
# Payments
```

## Migrating an existing workspace

Nothing to do. The first sync after updating moves the page id, version and hashes of every file
that still carries them into `lore-master.json` and removes them from the file, keeping `title:`
and `parent:`. Until then the old annotations are read as before.

A file annotated for **another** site or space keeps its annotation until a sync to that storage
moves it.

## Things to know

- **Renaming the storage** starts it afresh: the state is found by name, so the next sync adopts the
  pages by their titles instead of updating them by id. Rename the key in `lore-master.json` along
  with `outputs[].name` to keep it.
- **Moving or renaming a file** by hand drops its entry the same way; the next sync adopts the page
  by title.
- **Pages whose file is gone** lose their entry at the next sync.
- **A broken file.** If `lore-master.json` is not valid JSON, or was written by a newer LoreMaster,
  the sync reports it and leaves the file alone.
- **Rollback** restores `lore-master.json` along with the files, so the state matches the pages.
