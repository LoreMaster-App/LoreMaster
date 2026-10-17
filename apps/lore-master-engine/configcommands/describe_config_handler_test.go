package configcommands

import (
	"context"
	"encoding/json"
	"testing"

	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/documentation-sync/configcatalog"
)

func TestDescribeConfigAnswersWithTheCatalogAsJSON(t *testing.T) {
	result, err := DescribeConfig()(context.Background(), rpcserver.Call{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	var wire struct {
		Platforms []struct {
			ID       string `json:"id"`
			Delivery string `json:"delivery"`
			Fields   []struct {
				Key string `json:"key"`
			} `json:"fields"`
		} `json:"platforms"`
		Sections []struct {
			ID string `json:"id"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Platforms) != len(configcatalog.PlatformIDs()) || wire.Platforms[0].ID != "confluence" || wire.Platforms[0].Delivery != "pages" || len(wire.Platforms[0].Fields) == 0 {
		t.Fatalf("platforms on the wire: %+v", wire.Platforms)
	}
	if len(wire.Sections) == 0 {
		t.Fatal("no settings sections on the wire")
	}
}
