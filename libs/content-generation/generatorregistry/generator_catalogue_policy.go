package generatorregistry

import "slices"

// Entry describes one generator for the people and agents choosing between them.
type Entry struct {
	// Type is the configured name, for example "test-results".
	Type string `json:"type"`
	// Reads says what the generator takes as input.
	Reads string `json:"reads"`
	// Recognised says how the format is identified when type is left out; empty when the type
	// must always be named, because source code cannot be recognised from a file.
	Recognised string `json:"recognised,omitempty"`
	// Writes says what pages come out.
	Writes string `json:"writes"`
	// Needs names what must be installed or provided, when anything must.
	Needs string `json:"needs,omitempty"`
}

// catalogue has one entry per generator in the registry; a test keeps the two equal, so a
// generator cannot be added without saying what it is.
var catalogue = []Entry{
	{
		Type:       "test-results",
		Reads:      "JUnit XML reports (Go, Jest, pytest, Maven and most runners write them)",
		Recognised: "XML whose root is testsuite or testsuites",
		Writes:     "an index with the totals and failures, and a page per test suite with its cases",
	},
	{
		Type:       "coverage",
		Reads:      "Cobertura XML and LCOV tracefiles",
		Recognised: "XML whose root is coverage, or a tracefile with SF: records",
		Writes:     "a page per source folder with coverage bars and counts, nested like the code; numbers only, never source",
	},
	{
		Type:       "openapi-docs",
		Reads:      "OpenAPI 3 descriptions, YAML or JSON",
		Recognised: "a top-level openapi or swagger key",
		Writes:     "a folder per API with its operations by tag, parameters, responses and schemas",
	},
	{
		Type:       "markdown-docs",
		Reads:      "Markdown another tool wrote (TypeDoc's Markdown plugin, dokka, a build/docs folder)",
		Recognised: "files with the .md or .markdown extension",
		Writes:     "the same pages as ordinary pages with unique titles, the images they show copied beside them; input is required unless the command writes the pages straight into the output folder",
	},
	{
		Type:       "storybook-docs",
		Reads:      "a Storybook 7 or later: a static build folder or the URL of a running one (source)",
		Recognised: "an index.json or stories.json of stories, or a source key",
		Writes:     "a page per component with a screenshot of each story, its props table and its source",
		Needs:      "Chrome, Chromium or Edge on the machine (LORE_MASTER_BROWSER overrides the executable)",
	},
	{
		Type:   "notes",
		Reads:  "the tagged comments of the source files (TODO, FIXME, BUG, HACK, NOTE...), by the same ignore rules a sync reads files by",
		Writes: "one page with the count per tag and a table per tag: the text, where it is, and its owner, issue and date",
	},
	{
		Type:   "go-docs",
		Reads:  "the Go packages of the workspace and their doc comments",
		Writes: "an index and a page per package, nested like the folders",
	},
	{
		Type:   "ts-docs",
		Reads:  "TypeScript and JavaScript projects (every folder with a tsconfig.json)",
		Writes: "an index and a page per class, interface, function and type, nested like the folders",
		Needs:  "typedoc and typedoc-plugin-markdown installed in the workspace, and Node.js",
	},
	{
		Type:   "python-docs",
		Reads:  "Python projects (pyproject.toml, setup.py or setup.cfg), read statically",
		Writes: "an index and a page per module, a package being the README of its folder",
		Needs:  "pydoc-markdown (pip install pydoc-markdown)",
	},
	{
		Type:   "csharp-docs",
		Reads:  "C# projects, built so the XML documentation is fresh",
		Writes: "a page per namespace and per type, namespaces nested as folders",
		Needs:  "dotnet and the DefaultDocumentation dotnet tool",
	},
	{
		Type:   "dart-docs",
		Reads:  "Dart packages, the files under lib/",
		Writes: "a page per library file with its classes, mixins, enums, extensions, functions and variables",
		Needs:  "dartdoc_json as a pub global tool",
	},
}

// Catalogue is what every generator reads and writes, in the order of the documentation:
// the formats that are recognised from their content first, then the code documenters.
func Catalogue() []Entry {
	return slices.Clone(catalogue)
}
