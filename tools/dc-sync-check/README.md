# DC sync check

A manual smoke check: sync one throwaway page to a real Confluence **Data Center** space
and then prune it, proving the engine talks to a real site end to end. It is **not CI** and
not a product — a maintainer runs it locally, and it **cleans up after itself**, so it is
safe to run against a shared automation space.

## Run it

Build (or locate) the engine, then set the token in the environment — **never on the
command line history or in a committed file** — and run:

```bash
# an engine binary for this platform (e.g. from `go build`, or an extension's bin/)
export LORE_MASTER_ENGINE_BIN=/path/to/lore-master-engine

export CONFLUENCE_DC_BASE_URL=https://atlassian.jato.com/confluence
export CONFLUENCE_DC_PAT=<a Data Center personal access token>
export CONFLUENCE_DC_SPACE=AUT            # the space key to write into (default AUT)
# export CONFLUENCE_DC_PARENT=123456      # optional: a parent page id; defaults to the space home page

node tools/dc-sync-check/check.mjs
```

It opens a session with the PAT, creates a uniquely-titled page under the space's home page
(or `CONFLUENCE_DC_PARENT`), reports the created page's URL, then removes the file and prunes
the page to the trash. A non-zero exit means the sync or the cleanup failed.

It is validated against an in-process fake Confluence during development; the real value is
running it against an actual Data Center site.
