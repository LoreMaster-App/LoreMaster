package generatorrouting

import (
	"path"
	"slices"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/inputselection"
)

// Plan is what a batch of changed files calls for.
type Plan struct {
	// Everything is true when the settings file itself changed: every generator is affected and
	// the whole workspace is synced, since what is generated or synced may have changed with it.
	Everything bool
	// Generators are the indexes of the generators that read a changed file, ascending.
	Generators []int
	// Markdown are the changed Markdown files, sorted. Generated pages a regeneration writes are
	// added by whoever ran the generators.
	Markdown []string
}

// settingsFile is the configuration file's name.
const settingsFile = ".lore-master.yaml"

// reads says which files a generator type reads: extensions and file names, and the patterns
// that select its inputs when the generator has no input of its own.
var reads = map[string]struct {
	extensions []string
	names      []string
	defaults   []string
}{
	"test-results": {extensions: []string{".xml"}, defaults: []string{"**/junit*.xml", "**/TEST-*.xml", "**/*junit.xml"}},
	"openapi-docs": {extensions: []string{".yaml", ".yml", ".json"}, defaults: []string{"**/openapi.*", "**/swagger.*", "**/*.openapi.*"}},
	"go-docs":      {extensions: []string{".go"}, names: []string{"go.mod"}},
	"ts-docs":      {extensions: []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx"}, names: []string{"tsconfig.json", "jsconfig.json", "typedoc.json", "package.json"}},
	"python-docs":  {extensions: []string{".py"}, names: []string{"pyproject.toml", "setup.cfg"}},
	"csharp-docs":  {extensions: []string{".cs"}, names: []string{"directory.build.props"}},
	"dart-docs":    {extensions: []string{".dart"}, names: []string{"pubspec.yaml"}},
}

// Route decides what a batch of changed files (workspace-relative, '/'-separated) calls for. A
// file inside a generator's own output folder never makes that generator run again: those pages
// are its result, not its input.
func Route(changed []string, generators []generatedfile.Spec) Plan {
	var plan Plan
	affected := map[int]bool{}
	selectors := make([]inputselection.Selector, len(generators))
	for i, spec := range generators {
		selectors[i] = inputselection.NewSelector(spec.Input, reads[spec.Type].defaults)
	}

	for _, file := range changed {
		if file == settingsFile {
			plan.Everything = true
		}
		if strings.EqualFold(path.Ext(file), ".md") {
			plan.Markdown = append(plan.Markdown, file)
		}
		for index, spec := range generators {
			if reading(spec, selectors[index], file) {
				affected[index] = true
			}
		}
	}
	if plan.Everything {
		for index := range generators {
			affected[index] = true
		}
	}

	for index := range affected {
		plan.Generators = append(plan.Generators, index)
	}
	slices.Sort(plan.Generators)
	slices.Sort(plan.Markdown)

	return plan
}

// reading reports whether the generator reads the file.
func reading(spec generatedfile.Spec, selector inputselection.Selector, file string) bool {
	output := strings.TrimSuffix(spec.Output, "/")
	if output != "" && (file == output || strings.HasPrefix(file, output+"/")) {
		return false
	}
	kind, known := reads[spec.Type]
	if !known {
		return false
	}
	name := strings.ToLower(path.Base(file))
	if !slices.Contains(kind.extensions, strings.ToLower(path.Ext(file))) && !slices.Contains(kind.names, name) {
		return false
	}

	return selector.Selects(file, false)
}
