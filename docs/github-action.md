# GitHub Action: the docs site in one step

`LoreMaster-App/LoreMaster/actions/pages` renders the GitHub Pages output of
`.lore-master.yaml` (see [getting started](getting-started.md)) either into a folder, so it can
be deployed **beside another site**, or onto a branch.

## Beside a web app (the Actions source)

When the repository already deploys to Pages with `actions/deploy-pages`, each deploy replaces
the whole site, so the docs must be part of the same artifact:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - run: npm ci && npm run build            # your app -> dist/
      - uses: LoreMaster-App/LoreMaster/actions/pages@main
        with:
          out: dist/docs                        # the docs appear at <site>/docs/
      - uses: actions/upload-pages-artifact@v4
        with:
          path: dist
```

Add `docs/**` and `.lore-master.yaml` to the workflow's `paths:` so a docs change redeploys.

## To a branch

```yaml
permissions:
  contents: write
steps:
  - uses: actions/checkout@v7
  - uses: LoreMaster-App/LoreMaster/actions/pages@main
    with:
      mode: publish
```

Set `path:` on the output in `.lore-master.yaml` to publish into a folder of the branch and leave
the rest alone. A branch holding another site is refused without it.

## Inputs and outputs

| Input | Default | |
|---|---|---|
| `mode` | `build` | `build` writes `out`; `publish` pushes to the output's branch |
| `out` | `lore-master-site` | build: the folder (created, replaced when LoreMaster wrote it, refused otherwise) |
| `workspace` | `.` | the folder holding `.lore-master.yaml` |
| `output` | | position of the github-pages output; needed only when there are several |
| `generate` | `false` | run the generators first (their toolchains must already be installed) |
| `token` | `github.token` | publish: the token git pushes with |

Outputs: `path` (build: the absolute folder) and `files` (how many files the site has).

The action builds the engine from its own checkout with Go, so a run costs about half a minute
until release binaries exist (#201). Pin `@<tag>` or a commit SHA for reproducible builds.

The same can be done without the action: `lore-master-engine pages build --out DIR`
([command line](cli.md)).
