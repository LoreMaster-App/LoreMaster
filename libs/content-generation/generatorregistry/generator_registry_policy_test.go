package generatorregistry

import (
	"slices"
	"testing"
)

func TestForFindsABuiltGeneratorAndNothingElse(t *testing.T) {
	if generate, found := For("test-results"); !found || generate == nil {
		t.Fatal("test-results is not registered")
	}
	if _, found := For("go-docs"); found {
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
