# Releasing

Every merge to `main` releases the extension, `lore-master-vscode`: `nx release` reads the
conventional commits since the last tag, versions it, tags it, creates the GitHub Release,
publishes to the Marketplace, and attaches the engine binaries (`lore-master-engine-<version>`
zips and a `sha256` file; see [the CLI](../cli.md#getting-the-binary-in-ci)).

## What decides the version

- **The commits.** `fix:` is a patch, `feat:` a minor, a breaking change a major, from the
  conventional commits of the PRs merged since the last tag.
- **Below 1.0.0 everything is demoted one level.** That is Nx's rule for 0.x versions: a `feat:`
  applies as a patch and a breaking change as a minor. While the version is `0.x`, a feature
  release is `0.x.(y+1)` unless you force the version (below).
- **What counts as a change.** The extension depends on the engine (`nx.implicitDependencies`
  in its `package.json`), and the engine on the libs, so a change under `apps/lore-master-engine/`
  or `libs/` releases the extension too: the binary ships inside it.

## Forcing an exact version

Put a **version directive** commit in the PR:

```
version(lore-master-vscode)[0.16.0]: the first minor with the Storybook generator
```

The release uses exactly `0.16.0`. No repository variable is needed. This needs `@mnci/cli`
4.63.0 or later and its commitlint config (`npx @mnci/cli@latest upgrade`); until the workspace
is upgraded from 4.44.0, the commit hook rejects the subject and `RELEASE_SPECIFIER` is the way. The `RELEASE_SPECIFIER`
variable still exists as an override for one run, but it wins over a directive, so delete it
after use.
