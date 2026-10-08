package openapidocs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

const fixtureWorkspace = "testdata/workspace"

func joined(files []generatedfile.File) string {
	var out strings.Builder
	for _, file := range files {
		fmt.Fprintf(&out, "<!-- file: %s -->\n%s\n", file.Path, file.Body)
	}

	return out.String()
}

func bodyOf(t *testing.T, files []generatedfile.File, path string) string {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return string(file.Body)
		}
	}
	t.Fatalf("no page %s in\n%s", path, joined(files))

	return ""
}

func pathsOf(files []generatedfile.File) []string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.Path
	}

	return paths
}

func workspaceWith(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relative, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestGenerateMatchesTheGoldenPages(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}
	got := joined(output.Files)

	const golden = "testdata/workspace.golden.md"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if normalised := strings.ReplaceAll(string(want), "\r\n", "\n"); got != normalised {
		t.Fatalf("pages differ from %s (run with UPDATE_GOLDEN=1 to refresh it)\n got:\n%s\nwant:\n%s", golden, got, normalised)
	}
	if len(output.Warnings) != 2 || !strings.HasPrefix(output.Warnings[0], "legacy/swagger.yaml: this is a Swagger 2.0") || !strings.HasPrefix(output.Warnings[1], "notes/openapi.yaml: not an OpenAPI") {
		t.Fatalf("warnings %q", output.Warnings)
	}
}

func TestGenerateGivesEachApiAFolderAndPagesPerTagAndSchemas(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs", Title: "Our APIs"})
	if err != nil {
		t.Fatal(err)
	}

	want := "README.md,orders/README.md,orders/orders.md,orders/schemas.md,petstore/README.md,petstore/pets.md,petstore/default.md,petstore/schemas.md"
	if got := strings.Join(pathsOf(output.Files), ","); got != want {
		t.Fatalf("pages %s, want %s", got, want)
	}
	if !strings.HasPrefix(bodyOf(t, output.Files, "README.md"), "# Our APIs\n") {
		t.Fatal("the index title was not used")
	}
}

func TestGenerateListsTagsInDeclaredOrderThenOthersThenDefaultAndUsesTheFirstTag(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	pets := bodyOf(t, output.Files, "petstore/pets.md")
	if strings.Count(pets, "## GET /pets/{petId}") != 1 {
		t.Fatal("an operation with two tags is listed once, under its first")
	}
	overview := bodyOf(t, output.Files, "petstore/README.md")
	if strings.Index(overview, "[pets](pets.md)") > strings.Index(overview, "[default](default.md)") {
		t.Fatalf("overview %s", overview)
	}
	for _, unused := range []string{"owners.md", "store.md"} {
		if slices.Contains(pathsOf(output.Files), "petstore/"+unused) {
			t.Fatalf("a declared tag no operation uses got a page: %s", unused)
		}
	}
}

func TestGenerateResolvesReferencesAndAnOperationParameterOverridesThePathLevelOne(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	pets := bodyOf(t, output.Files, "petstore/pets.md")
	for _, want := range []string{
		"| `status` | query | string | no | Only pets with this status. One of: `available`, `pending`, `sold`. |",
		"| `default` | Something went wrong. | `application/problem+json`: [Problem](schemas.md) |",
		"Required. The pet to add.",
		"| `petId` | path | string | yes | Overrides the path-level description. |",
		"**Remove a pet** · **deprecated**",
		"| `petId` | path | string (uuid) | yes | The pet's id. |",
	} {
		if !strings.Contains(pets, want) {
			t.Errorf("missing %q in\n%s", want, pets)
		}
	}
}

func TestGenerateFlattensAllOfPropertiesAndEscapesWhatBreaksATable(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	schemas := bodyOf(t, output.Files, "petstore/schemas.md")
	for _, want := range []string{
		"Type: all of [NewPet](schemas.md), object",
		"| `id` | string (uuid) | yes |  |",
		"| `owner` | string or null | no | The owner's name, if any. |",
		"| `friends` | array of [Pet](schemas.md) | no |  |",
		"| `status` | string | no | One of: `available`, `pending`, `sold`. Default: available. |",
		`Contains a \| pipe and &lt;angle> brackets.`,
		"Type: one of [Pet](schemas.md), [Problem](schemas.md)",
	} {
		if !strings.Contains(schemas, want) {
			t.Errorf("missing %q in\n%s", want, schemas)
		}
	}
}

func TestGenerateTellsUnresolvedReferencesApart(t *testing.T) {
	root := workspaceWith(t, map[string]string{"openapi.yaml": `openapi: 3.0.0
info: {title: Broken, version: '1'}
paths:
  /x:
    get:
      parameters: [{$ref: '#/components/parameters/Gone'}]
      requestBody: {$ref: '#/components/requestBodies/Gone'}
      responses:
        '200': {$ref: '#/components/responses/Gone'}
`})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	page := bodyOf(t, output.Files, "broken/default.md")
	for _, want := range []string{"unresolved reference #/components/parameters/Gone", "unresolved reference #/components/requestBodies/Gone", "unresolved reference #/components/responses/Gone"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q in\n%s", want, page)
		}
	}
}

func TestGenerateKeepsTitlesAndFoldersUniqueAcrossApis(t *testing.T) {
	spec := "openapi: 3.0.0\ninfo: {title: Same, version: '1'}\npaths:\n  /x: {get: {tags: [t], responses: {}}}\n"
	root := workspaceWith(t, map[string]string{"a/openapi.yaml": spec, "b/openapi.yaml": spec, "c/openapi.yaml": strings.ReplaceAll(spec, "Same", "API reference")})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	titles := map[string]bool{}
	for _, file := range output.Files {
		heading := strings.SplitN(string(file.Body), "\n", 2)[0]
		if titles[strings.ToLower(heading)] {
			t.Fatalf("duplicate page title %q", heading)
		}
		titles[strings.ToLower(heading)] = true
	}
	if want := "README.md,same/README.md,same/t.md,same-2/README.md,same-2/t.md,api-reference/README.md,api-reference/t.md"; strings.Join(pathsOf(output.Files), ",") != want {
		t.Fatalf("pages %v, want %s", pathsOf(output.Files), want)
	}
	if !strings.Contains(bodyOf(t, output.Files, "api-reference/README.md"), "# API reference (c/openapi.yaml)") {
		t.Fatal("an API titled like the index is disambiguated by its file")
	}
}

func TestGenerateHonoursTheInputPatterns(t *testing.T) {
	output, err := Generate(context.Background(), fixtureWorkspace, generatedfile.Spec{Type: "openapi-docs", Input: []string{"orders/"}})
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(pathsOf(output.Files), ","); got != "README.md,orders/README.md,orders/orders.md,orders/schemas.md" || len(output.Warnings) != 0 {
		t.Fatalf("pages %s, warnings %q", got, output.Warnings)
	}
}

func TestGenerateWithNoDescriptionsStillWritesTheIndexSoStalePagesGoAway(t *testing.T) {
	output, err := Generate(context.Background(), workspaceWith(t, nil), generatedfile.Spec{Type: "openapi-docs"})
	if err != nil {
		t.Fatal(err)
	}

	if len(output.Files) != 1 || !strings.Contains(string(output.Files[0].Body), "No OpenAPI descriptions were found.") {
		t.Fatalf("pages %s", joined(output.Files))
	}
}

func TestGenerateSkipsDependenciesAndStopsWhenCancelled(t *testing.T) {
	spec := "openapi: 3.0.0\ninfo: {title: X, version: '1'}\n"
	root := workspaceWith(t, map[string]string{"node_modules/dep/openapi.yaml": spec, "vendor/v/openapi.yaml": spec})

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "openapi-docs"})
	if err != nil || len(output.Files) != 1 {
		t.Fatalf("pages %v, err %v", pathsOf(output.Files), err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Generate(ctx, root, generatedfile.Spec{Type: "openapi-docs"}); err == nil {
		t.Fatal("expected the cancellation")
	}
}
