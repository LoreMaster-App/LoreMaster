# ADR: One Go engine, thin editor shells

Status: accepted (2026-10-01). Issues: #5 (engine), #6 (VS Code), #10 (Visual Studio).

## Decision

All logic that touches Markdown or a documentation platform lives in Go libraries
(`libs/*`) behind one binary, `lore-master-engine` (`apps/lore-master-engine`). Each editor
extension spawns that binary and talks JSON-RPC 2.0 over stdin/stdout: the
language-server pattern. The engine never imports anything editor-specific.

Think of it as a kitchen and waiters. The waiter (extension) takes the order (which
Confluence, which space, which parent, which files) and brings the plate (progress,
report, errors). The kitchen (engine) is one process in one language that every waiter
shares. The kitchen may ask a waiter for something only the dining room has: a browser
to render a Mermaid diagram.

## RPC surface (source of truth: `apps/lore-master-engine/rpcprotocol`)

Editor → engine: `session/open`, `session/close`, `space/list`, `page/children`,
`page/search`, `sync/plan`, `sync/execute`, `$/cancelRequest`.

Engine → editor: `host/renderDiagram { language, source } → { svg }`,
`host/progress { message, done, total }`.

Credentials arrive in `session/open`, stay in the engine's memory, and are zeroed on
`session/close`. The editor owns secret storage (VS Code `SecretStorage`, Windows
Credential Manager). Nothing secret is ever written to disk by the engine or logged.

## Shipping

Go cross-compiles without cgo from one runner: windows/linux/darwin × amd64/arm64,
`CGO_ENABLED=0`, `-ldflags "-s -w -X main.version=<tag>"`. Each platform-specific `.vsix`
carries its binary under `bin/<goos>-<goarch>/`; `vsce package --target` produces six
packages and `vsce publish` uploads them under one version. The extension project is the
only one `nx release` versions; the engine is stamped from the same tag.

## Alternatives considered

| Option | Why not |
|---|---|
| TypeScript in-process in VS Code, Node single-executable sidecar for Visual Studio later | Cheapest now, but the Visual Studio shipment embeds Node (70–90 MB) and the two shells would run the engine two different ways |
| C#/.NET sidecar | Native for Visual Studio, but the larger audience (VS Code) would spawn a .NET binary and every contributor needs two toolchains |
| Rust → WASM + native | Most elegant runtime; no Rust kind in mnci, highest learning cost, HTTP from WASM needs host shims |
| C++ shared library (N-API + P/Invoke) | Per-platform native builds, ABI pain, weak Markdown/HTTP ecosystem |
| Python | Needs a runtime on the user's machine or a 40 MB PyInstaller bundle |

Go won on: one implementation for every editor, small static binaries, trivial
cross-compilation, a mature Markdown parser (goldmark), and existing `go-app` /
`go-internal-lib` support in `@mnci/cli` (one root `go.mod`, which this repo dogfoods).

## Consequences

- Engine tests are Go tests; the TypeScript shell has only thin Jest tests plus
  `@vscode/test-cli` integration suites.
- The slice architecture is applied in Go by convention plus `tools/slicecheck`, since the
  ESLint rules are TypeScript-only.
- mnci needs a multi-platform `go-app` build (mnci #226) and a `vscode-extension` kind
  that can bundle a sidecar (mnci #225).
