package generatorregistry

import (
	"slices"

	"lore-master/libs/content-generation/generatedfile"
	"lore-master/libs/content-generation/csharpdocs"
	"lore-master/libs/content-generation/godocs"
	"lore-master/libs/content-generation/openapidocs"
	"lore-master/libs/content-generation/testreporting"
	"lore-master/libs/content-generation/pythondocs"
	"lore-master/libs/content-generation/tsdocs"
)

// generators maps a configured type to the function that generates it.
var generators = map[string]generatedfile.GenerateFunc{
	"csharp-docs":  csharpdocs.Generate,
	"go-docs":      godocs.Generate,
	"openapi-docs": openapidocs.Generate,
	"python-docs":  pythondocs.Generate,
	"test-results": testreporting.Generate,
	"ts-docs":      tsdocs.Generate,
}

// For returns the generator for a configured type.
func For(kind string) (generatedfile.GenerateFunc, bool) {
	generate, found := generators[kind]

	return generate, found
}

// Types are the generator types that exist, sorted.
func Types() []string {
	types := make([]string, 0, len(generators))
	for kind := range generators {
		types = append(types, kind)
	}
	slices.Sort(types)

	return types
}
