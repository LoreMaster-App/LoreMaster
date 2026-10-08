package generatorregistry

import (
	"slices"
	"testing"
)

func TestForFindsABuiltGeneratorAndNothingElse(t *testing.T) {
	for _, kind := range []string{"test-results", "go-docs", "openapi-docs", "ts-docs", "python-docs"} {
		if generate, found := For(kind); !found || generate == nil {
			t.Fatalf("%s is not registered", kind)
		}
	}
	if _, found := For("csharp-docs"); found {
		t.Fatal("a generator that is not built is registered")
	}
	if _, found := For(""); found {
		t.Fatal("the empty type is registered")
	}
}

func TestTypesListsEveryRegisteredGeneratorSorted(t *testing.T) {
	types := Types()

	if !slices.IsSorted(types) || !slices.Contains(types, "test-results") {
		t.Fatalf("types %q", types)
	}
	for _, kind := range types {
		if _, found := For(kind); !found {
			t.Fatalf("%s is listed but not found", kind)
		}
	}
}
