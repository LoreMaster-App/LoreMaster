package odtexport

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/markdown-export/exportmodel"
)

const companyStyles = xmlHeader + `<office:document-styles ` + namespaces + `><office:styles>` +
	`<style:style style:name="Standard" style:family="paragraph" style:class="text"/>` +
	`<style:style style:name="Heading_20_1" style:display-name="Heading 1" style:family="paragraph"><style:text-properties fo:color="#c00000"/></style:style>` +
	`</office:styles><office:automatic-styles><style:page-layout style:name="Mpm1"><style:page-layout-properties fo:page-width="21.59cm"/></style:page-layout></office:automatic-styles>` +
	`<office:master-styles><style:master-page style:name="Standard" style:page-layout-name="Mpm1"><style:header><text:p>ACME Corp</text:p></style:header></style:master-page></office:master-styles>` +
	`</office:document-styles>`

// aTemplate writes a company template: a page with a header, its own Heading 1, no code style, a logo, and text of its own.
func aTemplate(t *testing.T) string {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	if err := addMimetype(archive); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"styles.xml":   companyStyles,
		"content.xml":  xmlHeader + `<office:document-content ` + namespaces + `><office:body><office:text><text:p>TEMPLATE TEXT</text:p></office:text></office:body></office:document-content>`,
		"Pictures/logo.png": "not really a png",
	} {
		if err := addFile(archive, name, []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "template.odt")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func partsOf(t *testing.T, data []byte) map[string]string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, file := range archive.File {
		reader, _ := file.Open()
		content, _ := io.ReadAll(reader)
		_ = reader.Close()
		parts[file.Name] = string(content)
	}

	return parts
}

var sample = exportmodel.Document{Title: "T", Blocks: []exportmodel.Block{
	{Kind: exportmodel.HeadingBlock, Level: 1, Inlines: []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: "Title"}}},
	{Kind: exportmodel.CodeBlock, Text: "x := 1"},
}}

func TestADocumentTakesATemplatesPageAndHeaderAndKeepsItsOwnHeadingStyle(t *testing.T) {
	data, err := Render(sample, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	parts := partsOf(t, data)

	if strings.Contains(parts["content.xml"], "TEMPLATE TEXT") || !strings.Contains(parts["content.xml"], "Title") {
		t.Errorf("the template's text was kept or the document's was lost:\n%s", parts["content.xml"])
	}
	styles := parts["styles.xml"]
	if !strings.Contains(styles, "ACME Corp") || !strings.Contains(styles, `fo:page-width="21.59cm"`) {
		t.Errorf("the template's header and page were not kept:\n%s", styles)
	}
	if strings.Count(styles, `style:name="Heading_20_1"`) != 1 || !strings.Contains(styles, `fo:color="#c00000"`) {
		t.Errorf("a style the template has must be used as it is, not added again:\n%s", styles)
	}
}

func TestAStyleTheTemplateLacksIsAdded(t *testing.T) {
	data, err := Render(sample, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	styles := partsOf(t, data)["styles.xml"]

	if !strings.Contains(styles, `style:name="Preformatted_20_Text"`) || !strings.Contains(styles, `style:name="Source_20_Text"`) {
		t.Errorf("the code styles should be added:\n%s", styles)
	}
	if strings.Count(styles, `style:name="Standard"`) < 2 {
		t.Errorf("the template's own styles must stay:\n%s", styles)
	}
}

func TestATemplatesPicturesTravelWithItAndAreListedInTheManifest(t *testing.T) {
	data, err := Render(sample, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	parts := partsOf(t, data)

	if parts["Pictures/logo.png"] != "not really a png" {
		t.Error("the template's picture was not copied")
	}
	if !strings.Contains(parts["META-INF/manifest.xml"], `manifest:full-path="Pictures/logo.png" manifest:media-type="image/png"`) {
		t.Errorf("the picture is not in the manifest:\n%s", parts["META-INF/manifest.xml"])
	}
}

func TestSomethingThatIsNotAnOpenDocumentFileIsRefused(t *testing.T) {
	notZip := filepath.Join(t.TempDir(), "x.odt")
	if err := os.WriteFile(notZip, []byte("plain text"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Render(sample, Options{ReferencePath: notZip}); err == nil || !strings.Contains(err.Error(), "not an OpenDocument file") {
		t.Errorf("got %v", err)
	}
}

func TestWithoutATemplateTheShippedStylesAreUsed(t *testing.T) {
	data, err := Render(sample, Options{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(partsOf(t, data)["styles.xml"], `style:name="Preformatted_20_Text"`) {
		t.Error("the shipped styles are gone")
	}
}
