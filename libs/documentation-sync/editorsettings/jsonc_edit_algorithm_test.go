package editorsettings

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func decodeJSONC(t *testing.T, source []byte) map[string]json.RawMessage {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(stripJSONC(source), &decoded); err != nil {
		t.Fatalf("not valid JSONC: %v\n%s", err, source)
	}

	return decoded
}

func TestSetKeyReplacesInPlaceAndKeepsComments(t *testing.T) {
	source := "{\n  // my editor\n  \"editor.fontSize\": 14, // big\n  \"loreMaster.ignore\": [\"a/\"],\n  \"other\": true,\n}\n"
	out, err := setKey([]byte(source), "loreMaster.ignore", []byte(`["b/","c/"]`))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, kept := range []string{"// my editor", "// big", `"editor.fontSize": 14`, `"other": true`} {
		if !strings.Contains(text, kept) {
			t.Fatalf("lost %q:\n%s", kept, text)
		}
	}
	if strings.Contains(text, `"a/"`) || !strings.Contains(text, `"c/"`) {
		t.Fatalf("value not replaced:\n%s", text)
	}
	decodeJSONC(t, out)
}

func TestSetKeyAddsANewKeyAfterTheLastOne(t *testing.T) {
	for _, source := range []string{"{\n  \"a\": 1\n}\n", "{\n  \"a\": 1,\n}\n", "{}\n", "{\n  // only a comment\n}\n"} {
		out, err := setKey([]byte(source), "loreMaster.skipGitignored", []byte("false"))
		if err != nil {
			t.Fatal(err)
		}
		decoded := decodeJSONC(t, out)
		if string(decoded["loreMaster.skipGitignored"]) != "false" {
			t.Fatalf("not added to %q:\n%s", source, out)
		}
	}
}

func TestSetKeyWithNilRemovesTheKey(t *testing.T) {
	source := "{\n  \"a\": 1,\n  \"loreMaster.ignore\": [\"x\"],\n  \"b\": 2\n}\n"
	out, err := setKey([]byte(source), "loreMaster.ignore", nil)
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeJSONC(t, out)
	if _, present := decoded["loreMaster.ignore"]; present || len(decoded) != 2 {
		t.Fatalf("not removed:\n%s", out)
	}
}

func TestSetKeyCreatesADocumentFromNothing(t *testing.T) {
	out, err := setKey(nil, "loreMaster.ignore", []byte(`["x"]`))
	if err != nil {
		t.Fatal(err)
	}
	var compact strings.Builder
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, decodeJSONC(t, out)["loreMaster.ignore"]); err != nil {
		t.Fatal(err)
	}
	compact.Write(buffer.Bytes())
	if compact.String() != `["x"]` {
		t.Fatalf("got %s", out)
	}
}

func TestSetKeyIgnoresBracketsAndSlashesInsideStrings(t *testing.T) {
	source := "{\n  \"url\": \"https://a.b/{c}\",\n  \"loreMaster.ignore\": [\"x]\", \"//y\"]\n}\n"
	out, err := setKey([]byte(source), "loreMaster.ignore", []byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeJSONC(t, out)
	if string(decoded["loreMaster.ignore"]) != "[]" || string(decoded["url"]) != `"https://a.b/{c}"` {
		t.Fatalf("got %s", out)
	}
}
