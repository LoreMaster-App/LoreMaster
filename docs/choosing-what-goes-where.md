# Choosing what goes where

A repository often feeds several places at once: some pages on a public GitHub Pages site,
internal documents on Confluence, public details in the repository wiki. LoreMaster reads every
Markdown file by default and lets you say what each **storage** (an output of
`.lore-master.yaml`) leaves out, or takes back.

## The model

- **Everything not ignored by git is in.** Every storage reads every `.md` file under its
  `roots` (the whole workspace by default).
- **`exclude` leaves things out, `include` takes them back.** Each entry is a file, or a folder
  ending in `/`, in gitignore syntax (so globs work too: `**/*.draft.md`).
- **Two scopes:**
  - the top-level `ignore` list leaves a path out of **every** storage;
  - each storage has its own `exclude` and `include`.
- **The most specific entry wins.** The deeper path decides, so excluding `internal/` and
  including `internal/public-faq.md` publishes that one file. On a tie the include wins.
- **Git-ignored files are never synced**, whatever any list says (see `skipGitignored` below).
- **`roots` is where a storage starts**, not an include list. It also decides what becomes the
  Pages site's index or the wiki's `Home`.

```yaml
version: 1
ignore:                       # in no storage, unless a storage includes it
  - internal/
  - notes/scratch.md
outputs:
  - platform: github-pages    # public site
    include: [internal/public-faq.md]
    exclude: [docs/adr/]
    content: [{ type: markdown, roots: ["docs"] }]

  - platform: github-wiki     # public details
    content: [{ type: markdown, roots: ["docs"] }]

  - platform: confluence      # internal: takes the whole internal folder back
    include: [internal/]
    baseUrl: https://acme.atlassian.net/wiki
    space: ENG
    parentPageId: "100"
    content: [{ type: markdown, roots: ["."] }]
```

A storage's `exclude` is added to its content entries' `excludes` (still accepted). The same
path in both of a storage's lists is refused: it would say nothing.

## In the editor

The **Pages** view shows what each storage syncs, and a **Local** node above them shows every
Markdown file of the repository and where each one goes:

```
Local                      every Markdown file in the repository
  README.md                README.md · → GitHub Pages, GitHub wiki
  internal/plan.md         plan.md · → Confluence · ENG
  scratch.md               scratch.md · ignored by git, never synced
Confluence · ENG
GitHub Pages
```

- **Arrange:** the button in the view title switches between the *storage tree* (pages as the
  storage nests them), a *flat list* and the *repository tree* (the files where they sit, folders
  included). The setting is `loreMaster.pages.view`.
- **Left-out files:** the eye button shows them faded, with the setting that left each out, or
  hides them (`loreMaster.pages.excluded`).
- **Do not sync…** (right-click a page or folder) asks which storages to leave it out of, or
  *All storages*, and writes the entry to `.lore-master.yaml`.
- **Sync this file…** (right-click a left-out file) brings it into the storages you pick: it
  removes the storage's own exclusion of exactly that file, or adds an include, which beats a
  broader exclusion. A file that git ignores, or that sits outside the storage's `roots`, has no
  such menu: the first is never synced, the second is not read at all.

## Links between pages

A page can only link to pages the same storage publishes. A link to a Markdown file the storage
leaves out is reported as a warning by `pages build`, `pages check` and `pages publish`
(`README.md: links to internal/plan.md, which this output leaves out`), and a wiki turns it
into plain text, so a public page never points at, or names, an internal document.

## Public repositories and private ones

GitHub decides who can see what a storage writes, whatever the lists say:

- A **private repository's wiki** is visible only to its collaborators, so it cannot hold public
  details. Put them in a Pages site, or give the wiki output `repo: owner/other-public-repo`.
- **GitHub Pages from a private repository** needs a plan that allows it, and the site is public
  unless the plan makes it private.

## `skipGitignored`

`skipGitignored` is `true` by default and is the setting behind "git-ignored files are never
synced". Setting it to `false` is still accepted for workspaces that sync generated, ignored
Markdown, but the editor then no longer marks those files as ignored.
