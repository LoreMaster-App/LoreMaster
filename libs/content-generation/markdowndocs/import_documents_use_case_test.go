package markdowndocs

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"lore-master/libs/content-generation/generatedfile"
)

func put(t *testing.T, root, name, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pathsOf(files []generatedfile.File) []string {
	out := make([]string, len(files))
	for i, file := range files {
		out[i] = file.Path
	}

	return out
}

func TestGenerateKeepsTheLayoutAndCopiesTheImagesPagesShow(t *testing.T) {
	root := t.TempDir()
	put(t, root, "build/docs/api/cart.md", "# Cart\n\n![diagram](../img/flow.png)\n\n```md\n![example](nope.png)\n```\n")
	put(t, root, "build/docs/guide.md", "# Guide\n\n[Cart](api/cart.md)\n")
	put(t, root, "build/docs/img/flow.png", "PNGDATA")
	put(t, root, "other/readme.md", "# Not selected\n")

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "markdown-docs", Input: []string{"build/docs/"}, Output: "docs/api", Title: "API"})
	if err != nil {
		t.Fatal(err)
	}

	got := pathsOf(output.Files)
	for _, want := range []string{"README.md", "api/cart.md", "guide.md", "img/flow.png"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if slices.Contains(got, "readme.md") || len(got) != 4 {
		t.Errorf("files = %v, want the index, two pages and the image", got)
	}
	for _, file := range output.Files {
		if file.Path == "img/flow.png" && (!file.Asset || string(file.Body) != "PNGDATA") {
			t.Errorf("image = %+v, want an asset with its bytes", file)
		}
		if strings.HasSuffix(file.Path, ".md") && file.Asset {
			t.Errorf("%s must not be an asset", file.Path)
		}
	}
	if len(output.Warnings) != 0 {
		t.Errorf("warnings = %v, want none (the fenced example is not an image)", output.Warnings)
	}
}

func TestGenerateWarnsAboutAMissingImageAndKeepsGoing(t *testing.T) {
	root := t.TempDir()
	put(t, root, "docs/a.md", "# A\n\n![x](missing.png)\n")

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "markdown-docs", Input: []string{"docs/"}, Output: "out"})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Warnings) != 1 || !strings.Contains(output.Warnings[0], "missing.png") {
		t.Fatalf("warnings = %v", output.Warnings)
	}
}

func TestGenerateNeedsAnInput(t *testing.T) {
	for _, input := range [][]string{nil, {"!docs/"}, {" "}} {
		if _, err := Generate(context.Background(), t.TempDir(), generatedfile.Spec{Type: "markdown-docs", Input: input, Output: "out"}); err == nil {
			t.Errorf("input %v: want an error", input)
		}
	}
}

func TestGenerateNeverReadsItsOwnOutput(t *testing.T) {
	root := t.TempDir()
	put(t, root, "docs/a.md", "# A\n")
	put(t, root, "docs/out/old.md", "# Old\n")

	output, err := Generate(context.Background(), root, generatedfile.Spec{Type: "markdown-docs", Input: []string{"docs/"}, Output: "docs/out"})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(pathsOf(output.Files), "out/old.md") {
		t.Fatalf("imported its own output: %v", pathsOf(output.Files))
	}
}

func TestRelativeImagesIgnoresWebAddressesRootPathsAndNonImages(t *testing.T) {
	got := relativeImages("![a](https://x/y.png) ![b](/abs.png) ![c](doc.pdf) ![d](pic%20one.png?raw=1) <img src=\"./h.svg\">")
	want := []string{"pic one.png", "./h.svg"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
