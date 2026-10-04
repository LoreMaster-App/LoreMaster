# Two-way sync: pull remote edits back, conflict when both sides change

Decided 2026-10-04 for #95 (E10). Enables `direction: two-way`, which is reserved in the
settings schema and refused until this ships.

## The model

Today the sync is one-way (`to-platform`): it pushes the workspace to Confluence and, when a
page was edited on the platform since the last sync, reports a **conflict** and leaves it
alone. Two-way sync adds the reverse direction — pulling remote edits back into the Markdown
— using the two signals the annotation already gives per page:

- **local changed** = the file's content hash differs from `annotation.ContentHash`
- **remote changed** = the page's version differs from `annotation.Version`

| local | remote | action |
|---|---|---|
| no  | no  | `unchanged` |
| yes | no  | `update` (push — today's behaviour) |
| no  | yes | **`pull`** (new: remote → file) |
| yes | yes | `conflict` (surfaced, never auto-merged) |

### No three-way merge (by decision)

The annotation stores a content **hash**, not the last-synced **text**, so there is no base
to merge against. Rather than add base-content storage and a merge engine, v1 **never merges
and never clobbers**: a page is pulled only when the file is locally unchanged; if both sides
changed it is a conflict the user resolves. (A later version could add base-content storage
for a real three-way merge — out of scope here.)

## The pull converter (the big new piece)

Pulling means converting Confluence **storage format** (XHTML with `ac:`/`ri:` elements) back
into Markdown — the reverse of the existing `storageformat.Render`. It goes through the
platform-neutral document model, mirroring the forward path:

```
push:  Markdown → Document (documentconversion) → storage XHTML (storageformat.Render)
pull:  storage XHTML → Document (new parser) → Markdown (new serializer)
```

Storage format is richer than Markdown, so the conversion is **lossy by nature**. v1 is
**conservative + flag**: it converts the shapes that round-trip faithfully and flags (rather
than silently mangles) anything it cannot.

- **Converted:** headings, paragraphs and inline marks, lists, tables, blockquotes, rules,
  code (`ac:structured-macro name=code` → fenced block), Mermaid (the `code` macro behind an
  image-mode diagram → a ```mermaid fence), `ri:page` links (by prefixed title → the local
  file that owns that title), images/attachments (`ri:attachment` → a local file, the
  attachment pulled to disk).
- **Flagged, not converted:** other macros, page layouts, deeply nested or merged-cell
  tables, and anything whose round-trip is unsafe. The page is reported with a warning and
  left for the user; the pull does not overwrite the file with a lossy approximation.

The author's `<!-- lore-master … -->` annotation and any `title:`/`parent:`/unknown keys are
preserved across a pull; only the body below is rewritten, and the annotation's version and
hashes are updated.

## Safety

- A pull happens **only when the file is locally unchanged** — local edits are never
  overwritten; a page edited on both sides is a conflict.
- A conflict is reported and skipped (as today); `direction: two-way` changes which cases
  become `pull` vs `conflict`, not the no-clobber guarantee.
- Title/parent changes made on the platform are surfaced; renaming or moving the local file
  to match is deferred (a pull rewrites the body, not the filename, in v1).

## Decomposition (sub-issues of #95)

1. **storage → Markdown conversion** — parser (storage XHTML → Document) + serializer
   (Document → Markdown), conservative + flag. Pure, unit-tested.
2. **Planning** — `pull` action and the two-way reconciliation table in change detection.
3. **Execution** — the pull path: fetch the page body in storage, convert, rewrite the file
   below its annotation, update the annotation; flag what could not be converted.
4. **Wiring** — accept `direction: two-way` in settings; surface `pull`/`conflict` in the
   engine plan/execute and the VS Code preview.
5. **End-to-end tests** — extend the hermetic `remotesync` test with pull and both-changed
   conflict cases.
