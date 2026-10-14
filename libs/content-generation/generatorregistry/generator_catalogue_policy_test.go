package generatorregistry

import (
	"slices"
	"testing"
)

func TestEveryGeneratorIsDescribedOnce(t *testing.T) {
	var described []string
	for _, entry := range Catalogue() {
		if entry.Reads == "" || entry.Writes == "" {
			t.Errorf("%s has no description of what it reads and writes", entry.Type)
		}
		described = append(described, entry.Type)
	}
	slices.Sort(described)
	if !slices.Equal(described, Types()) {
		t.Fatalf("the catalogue describes %v but the registry has %v", described, Types())
	}
}
