# Generators: pages written from your project's artifacts

A **generator** reads something your project already produces (test reports, coverage,
API descriptions, Markdown another tool wrote, a Storybook) and writes it as ordinary Markdown
files into your workspace. Those files then sync like any page you wrote: they join the page
tree, keep their annotation, and go to every storage. Because they are plain files you can
review them in git, edit them, or leave them out.

## Three parts: command, output, reader

1. **The command** (`prepare`, optional) produces the files: a test run, a build, TypeDoc,
   `build-storybook`. LoreMaster does not care what it is. It runs only when you ask for it.
2. **The output** (`output`, with an optional `title` for the index page that heads it) is the
   folder the pages are written to.
3. **The reader** is chosen by LoreMaster, not by you. It opens what `input` selects, works out
   what kind of file each is (XML, JSON, YAML, Markdown, LCOV) and then which schema is inside,
   and uses the parser for it. Leave `type` out and that is all there is to configure.

If nothing selected can be identified the run reports a **problem**, not silence: *could not
identify the output, no parser available*, naming how many files it looked at and what kind of
file each is, so you can fix the input or the command. If it finds several formats it names them
and asks for one generator each, since one generator writes one kind of page.

### What we can read

| Format | How it is recognised | Gives |
|---|---|---|
| JUnit XML (`test-results`) | XML whose root is `testsuite` or `testsuites` | A page per suite with its cases, failures and a summary |
| Cobertura XML, LCOV (`coverage`) | XML whose root is `coverage`; a tracefile with `SF:` records | Per-file results in the folder layout of the code |
| OpenAPI 2 and 3 (`openapi-docs`) | YAML or JSON with a top-level `openapi` or `swagger` | Operations, parameters, schemas |
| Markdown (`markdown-docs`) | `.md` or `.markdown` files | The pages, with the images they show |
| Storybook (`storybook-docs`) | An `index.json` or `stories.json` of stories, or a `source` URL | A page per component with rendered screenshots |

Code documentation (`go-docs`, `ts-docs`, `python-docs`, `csharp-docs`, `dart-docs`) reads source
code, which no file can be recognised from, so those keep an explicit `type`. They are shortcuts:
a command that writes Markdown plus `markdown-docs` does the same for any other tool.

Recognition is not re-run by watch mode: a generator with no `type` runs when you ask, after its
command.

## Configure one

In the `loreMaster.generators` setting (and `loreMaster.outputs` for where the pages go):

```jsonc
// .vscode/settings.json
{
  "loreMaster.generators": [
    {
      "type": "test-results",
      "input": ["reports/", "**/junit*.xml"],   // optional; see the defaults below
      "output": "docs/tests",                     // the folder the pages are written to
      "title": "Test results"                     // optional; heads the index page
    }
  ],
  "loreMaster.outputs": [
    {
      "platform": "confluence",
      // …
      "content": [{ "type": "markdown", "roots": ["docs"] }]   // the output folder must be inside a root to sync
    }
  ]
}
```

| Key | Meaning |
|---|---|
| `type` | Optional for the formats in the table above: left out, the format is recognised from what `input` selects. Required for `go-docs`, `ts-docs`, `python-docs`, `csharp-docs` and `dart-docs`. Names: `test-results`, `coverage`, `markdown-docs`, `storybook-docs`, `openapi-docs`, or those five. |
| `source` | Where a generator reads from when that is not a set of files: for `storybook-docs`, the URL of a Storybook or the folder of its build. |
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

## `coverage`

Reads **code coverage reports** and writes the results of each file, organised by your source
folders. It never shows source code, only numbers. Formats, detected from the content:

- **Cobertura XML** — what coverlet (.NET), coverage.py (`coverage xml`) and Istanbul/nyc
  (`cobertura` reporter) write.
- **LCOV** — `lcov.info`, from Jest/Vitest/c8, Dart, gcov and others.

Without `input` it looks for `**/*cobertura*.xml`, `**/coverage.xml`, `**/lcov.info` and
`**/*.lcov`, skipping `node_modules` and `.git`.

```jsonc
{ "type": "coverage", "input": ["coverage/"], "output": "docs/coverage", "title": "Code coverage" }
```

It writes one page per folder, like the Code Coverage tab of Azure DevOps:

- an **overview**: code coverage %, total, covered, partially covered and not covered lines;
- a table of the folder's **files, then subfolders**, each with a coverage bar and percentage and
  the same four counts;
- each subfolder is a link to its own page, which nests under its parent in the page tree, and each
  page links back **Up**.

A line that ran with every branch taken is *covered*, one that ran with a branch missed is
*partially covered*, one that never ran is *not covered*. Several reports for the same file (for
example one per test run) are merged line by line. The root page starts at the deepest folder all
files share, and paths are made relative to the workspace when the report named them inside it;
otherwise they appear as the report wrote them. A report in no known format is reported and skipped.

## `markdown-docs`

Imports **Markdown that another tool wrote** (TypeDoc's Markdown plugin, dokka, a `build/docs`
folder) as ordinary pages. Markdown only: HTML is never parsed, so produce Markdown first.

```jsonc
{ "type": "markdown-docs", "input": ["build/docs/"], "output": "docs/api", "title": "API reference" }
```

`input` is required (importing every Markdown file of a project would be surprising). Files keep
their layout below the deepest folder they share, so relative links between them keep working.
For each page the generator removes front matter and the lines a tool stamps in, makes sure it
opens with one top heading, gives it a title no other page has (a wiki space requires it), and adds
a `README.md` index when the tool wrote none.

Images the pages show through relative paths are copied next to them, byte for byte. They carry no
annotation, so the output folder keeps a `.lore-master-assets.json` listing them: the generator only
ever replaces or removes the files that list names. An image that is missing, or outside the
workspace, is reported and left out. The generator's own output folder is never read as input.

## `storybook-docs`

Renders **every story of a Storybook in a real browser** and writes one page per component: a
screenshot of each story, the props table and the story's source. The screenshots are plain PNG
files, so any output that takes Markdown takes them (Confluence gets them as attachments).

```jsonc
{
  "type": "storybook-docs",
  "source": "storybook-static",
  "prepare": "npm run build-storybook",
  "output": "docs/components",
  "title": "Component library"
}
```

`source` is required: the folder of a static Storybook build (served locally for the run) or the
URL of a running or deployed one. It needs Storybook 7 or later, which writes `index.json`; the
older `stories.json` is read too. `prepare` is the natural place for the build, see
[below](#producing-what-a-generator-reads-prepare).

`input` is optional and filters stories by **title**, with the same gitignore-style patterns:
`["Components/"]` renders only that branch of the sidebar, `["!Internal/"]` leaves one out.

The pages follow the sidebar: a page per folder listing what is in it, a page per component with
an **Examples** section (one screenshot per story, in light mode, 900 px wide) and a **Props**
table from the stories' `argTypes`. Pages change only when a component does: a story that fails to
render, and console errors, are reported as warnings and never written into a page.

The browser is the machine's Chrome, Chromium or Edge, found on `PATH` or in its usual install
folder. Set `LORE_MASTER_BROWSER` to the executable to use another one. A Storybook of a thousand
stories takes some minutes (four are rendered at a time), so run it from CI or on demand; watch
mode never renders it.

Whether the generated pages and images go into git is yours to decide: the generator writes them
to `output`, and the sync uploads them from there. Commit them, or ignore the folder and generate
on CI before each sync.

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

## `dart-docs`

Documents **Dart and Flutter** packages. dartdoc itself has no Markdown output, so the generator
runs [dartdoc_json](https://pub.dev/packages/dartdoc_json), which parses the source, and writes
the Markdown itself. Install the tool once (the Dart SDK, or Flutter, must be on the `PATH`):

```bash
dart pub global activate dartdoc_json
```

If it is missing, the generator stops and says exactly this; it never installs anything itself.

```yaml
generators:
  - type: dart-docs
    input: ["packages/", "!packages/legacy/"]   # package folders; default is every package
    output: docs/dart
    title: Dart API
```

- **Which packages:** every folder with a `pubspec.yaml`, narrowed by `input`. Every `.dart` file
  under its `lib/` is documented, except generated code (`*.g.dart`, `*.freezed.dart`,
  `*.mocks.dart`, ...), a `generated/` folder and files starting with `_`. Private (`_name`)
  declarations are left out.
- **Pages:** an index, then a page per library file, laid out like `lib/` (`lib/src/cart.dart` is
  `src/cart.md`) and titled by its import path (`package:shop/src/cart.dart`), which is unique.
  Each page has its classes, mixins, enums (with their values), extensions, extension types,
  typedefs, functions and variables, each with its signature and doc comment, and a class's
  constructors, fields, getters, setters and methods.
- **Limits:** the source is parsed, not analysed, so types are as written (no inferred types);
  a `this.x` parameter shows its field's type; a typedef shows its name only; and a top-level
  getter is written as a function. Headings inside doc comments become bold text so they cannot
  disturb the page's outline.

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

## Producing what a generator reads: `prepare`

A generator reads artifacts a build or a test run produces (JUnit and coverage reports, a docs
folder, a Storybook build). LoreMaster never guesses how to produce them. If you want it to run
the step for you, say so with an optional `prepare` command on the generator:

```jsonc
{
  "loreMaster.generators": [
    { "type": "test-results", "input": ["reports/"], "output": "docs/tests",
      "prepare": "npm test -- --reporters=jest-junit" },
    { "type": "coverage", "input": ["coverage/"], "output": "docs/coverage",
      "prepare": "npm test -- --coverage --coverageReporters=cobertura" }
  ]
}
```

In VS Code you do not have to write it by hand: **Add generator** asks for the command when you pick
*Recognise it from the output* (required, since it is the only thing that writes the files to read) or
*Storybook components* (optional), and **Edit generator** has a *Command that produces it* field. After
adding, **Generate and sync** runs the generators, asking to approve the command as described below, and
then shows the sync plan.

It is a shell command run in the workspace folder (`cmd` on Windows, `sh` elsewhere), limited to
30 minutes. **It only runs when a run asks for it; the same setting serves the plugin and CI:**

| Where | How it is asked for |
|---|---|
| **VS Code** | **Run generators** and **Generate and sync documentation** consider each selected generator's `prepare`. The setting `loreMaster.runPrepareSteps` says what to do: `ask` (default) shows the exact commands and asks once per session (or **Always run**), `always` runs them, `never` skips them. Nothing runs in a workspace you have not trusted. |
| **CI / command line** | `lore-master-engine generate --prepare`, or `lore-master-engine sync --generate --prepare --yes`. Without `--prepare` no command is ever executed, so a pipeline that already builds can keep its own steps and call `generate` alone. |
| **Watch mode** | Never runs `prepare`: it regenerates when files change, and a build on every save would be surprising. |

A command that fails is that generator's error (with the tail of its output) and the generator
does not run, so pages built from stale artifacts are not written; the others still run. `generate`
exits 1 in that case, which fails the pipeline step.

## Rules the generator keeps

- Every generated page carries `generated: <type>` in its `<!-- lore-master -->` annotation.
  A generator **only ever overwrites or removes files that carry its own marker**: a page you
  wrote by hand, or one made by another generator, is left alone and reported.
- When the content of a page changes, its sync annotation (the platform page id and version)
  is kept, so the page is updated on the platform, not recreated.
- A page the generator no longer produces (a suite that was removed from the report) is
  deleted, and folders it empties are removed.
