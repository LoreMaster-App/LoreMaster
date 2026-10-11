package editorsettings

import (
	"encoding/json"
	"testing"
)

func TestStripJSONCLeavesPlainJSONDecodable(t *testing.T) {
	source := `{
  // a line comment
  "loreMaster.confluence.baseUrl": "https://example.atlassian.net/wiki", /* inline */
  "loreMaster.ignore": ["a//b", "c",], // trailing comma above
}`
	var decoded map[string]any
	if err := json.Unmarshal(stripJSONC([]byte(source)), &decoded); err != nil {
		t.Fatal(err)
	}
	if got := decoded["loreMaster.confluence.baseUrl"]; got != "https://example.atlassian.net/wiki" {
		t.Fatalf("baseUrl = %v", got)
	}
	ignore, _ := decoded["loreMaster.ignore"].([]any)
	if len(ignore) != 2 || ignore[0] != "a//b" {
		t.Fatalf("ignore = %v", decoded["loreMaster.ignore"])
	}
}

func TestStripJSONCKeepsEscapedQuotesInsideStrings(t *testing.T) {
	var decoded map[string]string
	if err := json.Unmarshal(stripJSONC([]byte(`{"k": "say \"//hi\"" , }`)), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["k"] != `say "//hi"` {
		t.Fatalf("k = %q", decoded["k"])
	}
}
