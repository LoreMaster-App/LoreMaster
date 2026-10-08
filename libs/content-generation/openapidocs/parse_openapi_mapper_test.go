package openapidocs

import (
	"slices"
	"strings"
	"testing"
)

func TestParseDescriptionReadsYAMLAndJSONAndKeepsTheAuthorsOrder(t *testing.T) {
	yamlDoc := "openapi: 3.0.0\ninfo: {title: T, version: '1'}\npaths:\n  /z: {get: {responses: {'200': {description: ok}}}}\n  /a: {get: {responses: {'404': {description: no}, '200': {description: ok}}}}\n"
	parsed, err := parseDescription([]byte(yamlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed.Paths.Keys, []string{"/z", "/a"}) {
		t.Fatalf("paths %v: the written order was lost", parsed.Paths.Keys)
	}
	if got := parsed.Paths.Values["/a"].Get.Responses.Keys; !slices.Equal(got, []string{"404", "200"}) {
		t.Fatalf("responses %v", got)
	}

	jsonDoc := `{"openapi": "3.1.0", "info": {"title": "J", "version": "2"}, "paths": {"/b": {"post": {"summary": "s"}}}}`
	parsed, err = parseDescription([]byte(jsonDoc))
	if err != nil || parsed.Info.Title != "J" || parsed.Paths.Values["/b"].Post.Summary != "s" {
		t.Fatalf("json: %+v, %v", parsed, err)
	}
}

func TestParseDescriptionReadsATypeAsANameOrAList(t *testing.T) {
	doc := "openapi: 3.1.0\ncomponents:\n  schemas:\n    A: {type: string}\n    B: {type: [integer, 'null']}\n"
	parsed, err := parseDescription([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Components.Schemas.Values["A"].Type; !slices.Equal(got, names{"string"}) {
		t.Fatalf("A: %v", got)
	}
	if got := parsed.Components.Schemas.Values["B"].Type; !slices.Equal(got, names{"integer", "null"}) {
		t.Fatalf("B: %v", got)
	}
}

func TestParseDescriptionRefusesWhatItCannotRenderAndSaysWhy(t *testing.T) {
	cases := map[string]string{
		"swagger 2":     "swagger: '2.0'\ninfo: {title: T}\n",
		"no openapi":    "title: not an api\n",
		"openapi 2":     "openapi: 2.0.0\n",
		"openapi 4":     "openapi: 4.0.0\n",
		"broken yaml":   "openapi: [3.0.0\n",
		"not a mapping": "- a\n- b\n",
	}
	want := map[string]string{
		"swagger 2":     "this is a Swagger 2.0 description",
		"no openapi":    "there is no openapi field",
		"openapi 2":     "OpenAPI 2.0.0 is not supported",
		"openapi 4":     "OpenAPI 4.0.0 is not supported",
		"broken yaml":   "not valid YAML or JSON",
		"not a mapping": "not valid YAML or JSON",
	}
	for name, doc := range cases {
		_, err := parseDescription([]byte(doc))
		if err == nil || !strings.Contains(err.Error(), want[name]) {
			t.Errorf("%s: got %v, want it to contain %q", name, err, want[name])
		}
	}
}
