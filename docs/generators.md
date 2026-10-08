# Generators: pages written from your project's artifacts

A **generator** reads something your project already produces — today, **JUnit test
reports** — and writes it as ordinary Markdown files into your workspace. Those files then
sync like any page you wrote: they join the page tree, keep their annotation, and go to every
storage. Because they are plain files you can review them in git, edit them, or leave them
out.

## Configure one

In `.lore-master.yaml`:

```yaml
version: 1
generators:
  - type: test-results
    input: ["reports/", "**/junit*.xml"]   # optional; see the defaults below
    output: docs/tests                      # the folder the pages are written to
    title: Test results                     # optional; heads the index page
outputs:
  - platform: confluence
    # …
    content:
      - type: markdown
        roots: [docs]                       # the output folder must be inside a root to sync
```

| Key | Meaning |
|---|---|
| `type` | `test-results`, `go-docs`, `openapi-docs`, `ts-docs`, `python-docs` or `csharp-docs`. `dart-docs` is reserved for generators not built yet. |
| `input` | Gitignore-style patterns selecting what to read. A pattern starting with `!` leaves out what it matches. With no selecting pattern (none, or only `!` ones) the type's default applies. |
| `output` | The workspace folder the pages are written to. It cannot be the workspace itself, and two generators cannot write into overlapping folders. |
| `title` | The index page's title. |

## `test-results`

Reads JUnit XML — the format Go (`gotestsum`, `go-junit-report`), Jest (`jest-junit`),
pytest (`--junitxml`), Maven and most other runners write. Without `input` it looks for
`**/junit*.xml`, `**/TEST-*.xml` and `**/*junit.xml`, skipping `node_modules` and `.git`.

It writes:

- **`README.md`** — the totals (tests, passed, failed, errors, skipped, time), the failing
  tests, and a table of suites. Every suite page nests under it.
- **one page per test suite** — its totals, every failure with its message and stack in a
  code block, and a table of its tests. A very large suite lists its first 500 tests; every
  failure is always listed.

A report that is not valid JUnit is reported and skipped; the others are still written. When
there are no reports at all, the index says so, so old results never linger.

## `go-docs`

Reads the Go packages in the workspace and writes their documentation — the same doc
comments `go doc` shows — as Markdown:

- **`README.md`** — a table of every documented package with its synopsis.
- **one page per package**, at the path of its folder (`<output>/pkg/store/README.md`), so
  the pages nest the way your directories do. A page has the import path, the package
  comment, then constants, variables, functions and types with their constructors and
  methods; each declaration is shown as gofmt prints it, without its body, with its comment
  beneath it. The root package, which cannot be `README.md`, is named after the package.

```yaml
generators:
  - type: go-docs
    input: ["libs/", "apps/", "!**/internal/"]   # optional; default is every package
    output: docs/api
```

Folders named `node_modules`, `vendor`, `testdata`, `dist`, or starting with `.` or `_` are
skipped. A package with no exported declarations and no package comment is left out. Build
constraints are ignored (so the pages are the same on every OS), except files marked
`//go:build ignore`. A file that does not parse is reported and skipped.

## `ts-docs`

Documents **TypeScript and JavaScript** projects with [TypeDoc](https://typedoc.org) and its
Markdown plugin. Install them once in the workspace:

```bash
npm i -D typedoc typedoc-plugin-markdown
```

If they are missing, the generator stops and says exactly this; it never installs anything
itself. Node.js must be on the `PATH`.

```yaml
generators:
  - type: ts-docs
    input: ["libs/", "apps/web/", "!libs/legacy/"]   # project folders; default is every project
    output: docs/typescript
    title: TypeScript API
```

- **Which projects:** every folder with a `tsconfig.json` (or `jsconfig.json`) that has
  something to document, narrowed by `input`. A folder with only references and no `src` — such
  as a monorepo root — is skipped.
- **What is documented:** a library's `src/index.ts` documents what it exports, which is its
  public API; a project without an index (an app) documents every module under `src`; a project
  with its own `typedoc.json` is left to it. Nx's `tsconfig.lib.json` / `tsconfig.app.json` are
  preferred over a solution-style `tsconfig.json`.
- **Pages:** an index, then a page per class, interface, function, type alias and variable,
  nested like the folders. A monorepo gets one section per project, and a title that two
  projects share gets the folder added in brackets, since a wiki space needs unique titles.
  Source links and page headers are left out because they change with the commit.

## `python-docs`

Documents **Python** projects with [pydoc-markdown](https://github.com/NiklasRosenstein/pydoc-markdown),
which reads the source statically: nothing is imported or run, so no dependency needs installing.
Install the tool once, in the workspace's `.venv` or anywhere on the `PATH`:

```bash
pip install pydoc-markdown
```

If it is missing, the generator stops and says exactly this; it never installs anything itself.

```yaml
generators:
  - type: python-docs
    input: ["services/", "!services/legacy/"]   # project folders; default is every project
    output: docs/python
    title: Python API
```

- **Which projects:** every folder with a `pyproject.toml`, `setup.py` or `setup.cfg`, narrowed by
  `input`. Its packages (folders with an `__init__.py`) and modules are read from `src/` when that
  holds any, else from the project folder. Tests, docs, examples, virtual environments, build
  output and `setup.py` are left out, and so are private (`_name`) members and undocumented ones.
- **Pages:** an index, then a page per module. A package is the `README.md` of its folder
  (`shop/billing/README.md`) and a module a file beside its siblings (`shop/cart.md`), so the
  pages nest like the packages. Docstrings in the Sphinx, Google and NumPy styles are rendered by
  pydoc-markdown. Titles are the dotted module names, which are unique; a title that two projects
  share gets the folder added in brackets.

## `csharp-docs`

Documents **C#** projects from their XML documentation with
[DefaultDocumentation](https://github.com/Doraku/DefaultDocumentation). It needs the .NET SDK (8 or
newer) and the tool, installed once:

```bash
dotnet tool install -g DefaultDocumentation.Console
```

If the tool is missing, the generator stops before building anything and says exactly this; it
never installs anything itself.

```yaml
generators:
  - type: csharp-docs
    input: ["src/", "!src/Legacy/"]   # project folders; default is every project
    output: docs/csharp
    title: C# API
```

- **Which projects:** every folder with a `.csproj`, narrowed by `input`. Test projects
  (`*.Tests`, `*.UnitTests`, ...) are skipped.
- **Build:** each project is built in `Release` with `GenerateDocumentationFile` switched on, so
  the XML is as fresh as the code (an up-to-date project builds in moments; `bin/` and `obj/` are
  written as for any build). A project that targets several frameworks is documented for the last
  one. A failed build stops the run and shows the compiler's errors.
- **Pages:** an index, then a page per namespace and per type (its constructors, properties and
  methods are on its page), nested by namespace like folders: `Shop.Billing` is
  `Shop/Billing/README.md`, `Shop.Billing.Invoice` is `Shop/Billing/Invoice.md`. A type's title is
  its full name and kind (`Shop.Cart Class`), so titles are unique; links between pages follow
  the new paths.

## `openapi-docs`

Reads **OpenAPI 3.x** descriptions, YAML or JSON, and writes Markdown. Without `input` it looks
for `**/openapi.{yaml,yml,json}`, `**/swagger.{yaml,yml,json}` and `**/*.openapi.{yaml,yml,json}`,
skipping `node_modules`, `vendor` and `.git`. Each API gets its own folder:

- **`README.md`** (the output root) — the list of APIs; **`<api>/README.md`** — title, version,
  description, servers and a table of every operation.
- **`<api>/<tag>.md`** — one page per tag, in the order the description declares them (then
  tags only operations mention, then `default` for the untagged). Each operation shows its
  parameters, request body and responses as tables; `$ref`s into `components` are resolved. An
  operation is listed under its first tag.
- **`<api>/schemas.md`** — every schema in `components`, with its properties, enum values and
  defaults; an `allOf` shows the properties of its inline parts, and a reference links to the
  schemas page.

Paths, responses and properties keep the order the author wrote them in. A Swagger 2 file, or a
file that is not an OpenAPI description, is reported and skipped; the rest are still written.

## Running it

You do not have to edit the YAML. The **Generators** view in the LoreMaster sidebar lists your
generators — what each one is, where it writes, and what its last run did — and its buttons add,
run, edit and remove them: **Add generator** asks which kind, the output folder and what to read,
saves it with your comments kept, warns if no storage syncs that folder, and offers to run it at
once. Removing a generator never deletes the pages it wrote.

The editor's **Run generators** command, the view's run buttons and the engine's `generators/run`
method all run the same thing.

## Generate and sync in one step

**LoreMaster: Generate and sync documentation** (also a row in the Sync view and a button in the
Generators view) runs every generator and then syncs, so the pages that reach the storage are as
fresh as the code. The sync previews its plan as usual, and unchanged generated pages are not in
it. If a generator **fails**, you are asked before the pages go out, since some may be missing or
stale; warnings alone (an input skipped) do not stop it. With no generators configured it syncs
the Markdown and says so.

To make every ordinary **Sync** do this, turn on the setting `loreMaster.generateBeforeSync`. Output is deterministic — the same
reports give the same bytes — so running it again with nothing new rewrites nothing and shows
no diff.

## Rules the generator keeps

- Every generated page carries `generated: <type>` in its `<!-- lore-master -->` annotation.
  A generator **only ever overwrites or removes files that carry its own marker**: a page you
  wrote by hand, or one made by another generator, is left alone and reported.
- When the content of a page changes, its sync annotation (the platform page id and version)
  is kept, so the page is updated on the platform, not recreated.
- A page the generator no longer produces (a suite that was removed from the report) is
  deleted, and folders it empties are removed.
