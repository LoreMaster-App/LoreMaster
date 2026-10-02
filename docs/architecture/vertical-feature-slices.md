# ADR: Vertical feature slices, in Go and TypeScript

Status: accepted (2026-10-01). Issue: #48. Enforced in TypeScript by
`@mnci/eslint-config`'s `verticalSlices` rules; in Go by the compiler for two of the three
rules and by `tools/slicecheck` (#47) for the third.

## The shape

**capability → cohesive subfeature ("slice") → flat, role-suffixed files.**

- The package or app is the **capability**: the one thing it does end to end.
- Every folder directly under the capability root is a **slice**: one outcome, one public
  surface. Nothing outside the slice reaches past that surface.
- A slice is **flat**: no nested folders, except test data.
- Only the entry point lives at the root of a capability. Everything else belongs to a slice.
- Dependencies point one way: entry point → orchestration slices → feature slices → leaf
  slices. Two slices never import each other, not even for types.
- A slice with roughly a dozen hand-authored production files is a signal to split, never
  a reason to nest.

Never `helper`, `util`, `manager`, `processor`, `data`, `service`, `middleware`, `shared`,
`common`, `lib` as a role or folder name. Behaviour that several slices need becomes its
own named slice with its own outcome.

## Roles

| Role | Responsibility |
|---|---|
| `handler` | Adapts a transport in (HTTP, queue, CLI, JSON-RPC, editor command) |
| `use-case` | Coordinates one operation, including I/O |
| `algorithm` | Pure computation, no I/O, no business decision |
| `policy` | A reusable decision |
| `model` | A concept with meaning, state or invariants |
| `contract` | Data crossing a boundary: file format, port interface, RPC payload |
| `mapper` | Deterministic conversion between representations |
| `validator` | Accepts or rejects data and says why |
| `repository` | Persistence in domain terms |
| `client` | An external protocol or vendor SDK |
| `store` | Runtime state owned by the module |
| `error` | An error type the slice throws, callers catch |
| `config` | Configuration owned by one module |
| `enum` | A technical enumeration |

Deciding, in order, stop at the first yes: transport → `handler`; coordinates I/O →
`use-case`; reusable decision → `policy`; meaning/state → `model`; crosses a boundary →
`contract`; converts → `mapper`; pure → `algorithm`; validates → `validator`; persists →
`repository`; vendor → `client`; runtime state → `store`. None → rethink the
responsibility; never invent a bucket.

## TypeScript (`apps/lore-master-vscode/src`)

- `<kebab-name>.<role>.ts`; tests `<basename>.spec.ts` beside the file; `fixtures/` for
  test data only.
- One `index.ts` barrel per slice; sibling imports go through it
  (`import { x } from '../other-slice'`); a slice never imports its own barrel.
- Only `index.ts` and `main.ts` at the root of `src/`. The VS Code entry is therefore
  `main.ts`, not `extension.ts`.
- Code outside `src/` (the `integration/` suites, `test/vscode.stub.ts`) is not bound.

## Go (`apps/lore-master-engine`, `libs/*`)

| Rule | Go translation |
|---|---|
| capability | one Nx project (`libs/<name>`, `apps/<name>`) |
| slice + barrel | one package directory directly under the project root; exported identifiers **are** the barrel; `doc.go` states the outcome in its first line |
| root files | a lib root holds only `doc.go`; an app root only `main.go` (+ `main_test.go`, `doc.go`) |
| file naming | `<snake>_<role>.go`, e.g. `plan_sync_use_case.go`, `nesting_policy.go` |
| tests | `<snake>_<role>_test.go` (Go requires `_test.go`) |
| test data | `testdata/` (ignored by the Go tool) |
| no nesting | no sub-packages inside a slice package |
| barrel-only imports | automatic: Go can only import exported names |
| no cycles | automatic: compiler error, type-only included |
| forbidden names | packages named `util utils helper helpers common shared internal lib` |
| package naming | package name = directory name, lowercase, no separators (`documenttree`) |

`tools/slicecheck` walks `apps/*` and `libs/*` and asserts the rows the compiler does not.
It is a safety net the maintainer can delete; the convention stands without it.

## Self-check before finishing a change

- The file's path and role suffix match what the code does.
- The file lives in the slice whose outcome it serves, not the slice that calls it.
- Models, policies, algorithms and mappers contain no I/O.
- Vendor SDKs and protocols stay behind `client` files; `vscode` is imported only by
  `*.handler.ts` and `*.client.ts`.
- No `helper`/`util`/`service`/`shared`/`common`/`lib` name anywhere.
