/** A kind of generator the engine can run, as the sidebar presents it. */
export interface GeneratorType {
  /** The `type:` value in .lore-master.yaml. */
  type:          string
  label:         string
  description:   string
  /** A codicon id. */
  icon:          string
  /** Where the pages go unless the user picks another folder. */
  defaultOutput: string
  /** What to tell the user when asking which files or folders to read. */
  inputHint:     string
}

export const GENERATOR_TYPES: readonly GeneratorType[] = [
  {
    type:          'test-results',
    label:         'Test results',
    description:   'JUnit XML reports from Go, Jest, pytest, Maven and most runners',
    icon:          'beaker',
    defaultOutput: 'docs/tests',
    inputHint:     'Patterns for the report files, separated by commas, e.g. reports/, **/junit*.xml. Leave empty for the usual names.',
  },
  {
    type:          'go-docs',
    label:         'Go package docs',
    description:   'The doc comments of your Go packages, as `go doc` shows them',
    icon:          'package',
    defaultOutput: 'docs/api',
    inputHint:     'Folders with Go code, separated by commas, e.g. libs/, apps/, !**/internal/. Leave empty for every package.',
  },
  {
    type:          'ts-docs',
    label:         'TypeScript API docs',
    description:   'Classes, interfaces and functions of TypeScript or JavaScript projects, with TypeDoc',
    icon:          'symbol-class',
    defaultOutput: 'docs/typescript',
    inputHint:     'Project folders, separated by commas, e.g. libs/, apps/web/. Needs typedoc and typedoc-plugin-markdown installed in the workspace. Leave empty for every project.',
  },
  {
    type:          'python-docs',
    label:         'Python API docs',
    description:   'Modules, classes and functions of Python projects, with pydoc-markdown',
    icon:          'symbol-method',
    defaultOutput: 'docs/python',
    inputHint:     'Project folders, separated by commas, e.g. services/billing/. Needs pydoc-markdown installed (pip install pydoc-markdown). Leave empty for every project.',
  },
  {
    type:          'csharp-docs',
    label:         'C# API docs',
    description:   'Namespaces and types of C# projects, from their XML documentation, with DefaultDocumentation',
    icon:          'symbol-namespace',
    defaultOutput: 'docs/csharp',
    inputHint:     'Project folders, separated by commas, e.g. src/Shop/. Needs the .NET SDK and DefaultDocumentation (dotnet tool install -g DefaultDocumentation.Console). Leave empty for every project.',
  },
  {
    type:          'dart-docs',
    label:         'Dart API docs',
    description:   'Classes, mixins, enums and functions of Dart and Flutter packages, with dartdoc_json',
    icon:          'symbol-interface',
    defaultOutput: 'docs/dart',
    inputHint:     'Package folders, separated by commas, e.g. packages/shop/. Needs the Dart SDK and dartdoc_json (dart pub global activate dartdoc_json). Leave empty for every package.',
  },
  {
    type:          'openapi-docs',
    label:         'OpenAPI docs',
    description:   'OpenAPI 3 descriptions (YAML or JSON): operations, parameters, schemas',
    icon:          'globe',
    defaultOutput: 'docs/openapi',
    inputHint:     'Patterns for the description files, separated by commas, e.g. api/. Leave empty for openapi.yaml, swagger.json and the like.',
  },
]

/** The presentation of a configured type; a type this version does not know keeps its own name. */
export function generatorTypeOf (type: string): GeneratorType {
  return GENERATOR_TYPES.find(known => known.type === type) ?? {
    type,
    label:         type,
    description:   'A generator this version of the extension does not know',
    icon:          'question',
    defaultOutput: 'docs/generated',
    inputHint:     'Patterns, separated by commas. Leave empty for the default.',
  }
}
