package synccommands

import (
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/platformport"
)

var visibleSpaces = []platformport.Space{
	{ID: "98336", Key: "TLM", Alias: "LORE", Name: "Lore Master"},
	{ID: "131077", Key: "SD", Name: "Software Development"},
	{ID: "7", Key: "LORE", Name: "Another space whose key is the alias of the first"},
}

func TestMatchSpaceKeyWinsOverAnotherSpacesAlias(t *testing.T) {
	space, ok := matchSpace(visibleSpaces, "LORE")
	if !ok || space.ID != "7" {
		t.Fatalf("got %+v ok=%v, want the space whose key is LORE", space, ok)
	}
}

func TestMatchSpaceAcceptsTheAddressAlias(t *testing.T) {
	space, ok := matchSpace(visibleSpaces[:2], "lore")
	if !ok || space.Key != "TLM" {
		t.Fatalf("got %+v ok=%v, want TLM through its alias", space, ok)
	}
}

func TestMatchSpaceIgnoresCaseOfTheKey(t *testing.T) {
	space, ok := matchSpace(visibleSpaces, "sd")
	if !ok || space.Key != "SD" {
		t.Fatalf("got %+v ok=%v, want SD", space, ok)
	}
}

func TestMatchSpaceFindsNothingForAnUnknownOrEmptyName(t *testing.T) {
	if _, ok := matchSpace(visibleSpaces, "NOPE"); ok {
		t.Fatal("matched an unknown space")
	}
	if _, ok := matchSpace(visibleSpaces, "  "); ok {
		t.Fatal("matched an empty name")
	}
}

func TestDescribeSpacesShowsKeyAliasAndName(t *testing.T) {
	got := describeSpaces(visibleSpaces[:2], 10)
	want := `TLM (alias LORE, "Lore Master"), SD ("Software Development")`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDescribeSpacesSaysHowManyMoreThereAre(t *testing.T) {
	got := describeSpaces(visibleSpaces, 2)
	if !strings.HasSuffix(got, ", and 1 more") {
		t.Fatalf("got %q, want a count of the rest", got)
	}
}
