package documentexport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openPackage unzips an export and checks that every XML part is well formed.
func openPackage(t *testing.T, data []byte) (map[string]string, []string) {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	parts := map[string]string{}
	var order []string
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[file.Name] = string(content)
		order = append(order, file.Name)
		if strings.HasSuffix(file.Name, ".xml") || strings.HasSuffix(file.Name, ".rels") {
			decoder := xml.NewDecoder(bytes.NewReader(content))
			for {
				if _, err := decoder.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("%s is not well-formed XML: %v", file.Name, err)
				}
			}
		}
	}

	return parts, order
}

func writePicture(t *testing.T, dir string) {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 40, 20))
	picture.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chart.png"), buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

const officeSample = sample + "\n![A chart](chart.png)\n\n![Missing](nope.png)\n\n3. three\n4. four\n"

func TestDocxIsAWellFormedPackage(t *testing.T) {
	dir := t.TempDir()
	writePicture(t, dir)

	file, err := Export(filepath.Join(dir, "guide.md"), []byte(officeSample), "docx", Options{BaseDir: dir})
	if err != nil || file.Name != "guide.docx" {
		t.Fatalf("name %q, err %v", file.Name, err)
	}
	parts, _ := openPackage(t, file.Content)

	document := parts["word/document.xml"]
	for _, want := range []string{
		`<w:pStyle w:val="Heading1"/>`, `<w:pStyle w:val="Heading2"/>`, `<w:bookmarkStart w:id="1" w:name="guide"/>`,
		`<w:b/>`, `<w:i/>`, `<w:strike/>`, `w:val="CodeChar"`, `<w:pStyle w:val="Code"/>`, `<w:pStyle w:val="Quote"/>`,
		`<w:tbl>`, `<w:tblHeader/>`, `<w:jc w:val="right"/>`, `<w:numPr>`, "\u2612", `<w:drawing>`, "Missing",
	} {
		if !strings.Contains(document, want) {
			t.Errorf("document.xml lacks %s", want)
		}
	}
	if strings.Contains(document, "owner: team") {
		t.Error("front matter leaked")
	}
	if _, found := parts["word/media/image1.png"]; !found {
		t.Error("the picture is not in the package")
	}
	rels := parts["word/_rels/document.xml.rels"]
	for _, want := range []string{`Target="https://example.com/a" TargetMode="External"`, `Target="other.docx#part"`, `Target="media/image1.png"`} {
		if !strings.Contains(rels, want) {
			t.Errorf("relationships lack %s:\n%s", want, rels)
		}
	}
	if numbering := parts["word/numbering.xml"]; !strings.Contains(numbering, `<w:startOverride w:val="3"/>`) {
		t.Errorf("the list that starts at 3 does not restart there:\n%s", numbering)
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/styles.xml", "docProps/core.xml"} {
		if _, found := parts[name]; !found {
			t.Errorf("missing part %s", name)
		}
	}
}

func TestOdtIsAWellFormedPackageWithMimetypeFirst(t *testing.T) {
	dir := t.TempDir()
	writePicture(t, dir)

	file, err := Export(filepath.Join(dir, "guide.md"), []byte(officeSample), "odt", Options{BaseDir: dir})
	if err != nil || file.Name != "guide.odt" {
		t.Fatalf("name %q, err %v", file.Name, err)
	}
	parts, order := openPackage(t, file.Content)

	if order[0] != "mimetype" || parts["mimetype"] != "application/vnd.oasis.opendocument.text" {
		t.Errorf("first entry %q = %q", order[0], parts["mimetype"])
	}
	archive, _ := zip.NewReader(bytes.NewReader(file.Content), int64(len(file.Content)))
	if archive.File[0].Method != zip.Store {
		t.Error("the mimetype entry is compressed")
	}
	// LibreOffice refuses a package whose mimetype entry is followed by a data descriptor.
	if archive.File[0].Flags&0x8 != 0 {
		t.Error("the mimetype entry has a data descriptor")
	}

	content := parts["content.xml"]
	for _, want := range []string{
		`<text:h text:style-name="Heading_20_1" text:outline-level="1">`, `<text:bookmark text:name="guide"/>`, `Strong_20_Emphasis`, `Strikethrough`, `Source_20_Text`,
		`xlink:href="https://example.com/a"`, `xlink:href="other.odt#part"`, `<text:list text:style-name="L_bullet">`, `text:start-value="3"`,
		`<table:table `, `<table:table-header-rows>`, `Cell_20_Right`, `<draw:image xlink:href="Pictures/image1.png"`, "\u2612 done", "Missing", `Quotations`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content.xml lacks %s", want)
		}
	}
	if !strings.Contains(parts["META-INF/manifest.xml"], `Pictures/image1.png" manifest:media-type="image/png"`) {
		t.Error("the picture is not in the manifest")
	}
}

func TestOfficeFormatsKeepCodeIndentation(t *testing.T) {
	source := "# Code\n\n```\nif x {\n    run()\n}\n```\n"

	odt, err := Export("c.md", []byte(source), "odt", Options{})
	if err != nil {
		t.Fatal(err)
	}
	parts, _ := openPackage(t, odt.Content)
	if !strings.Contains(parts["content.xml"], `<text:s text:c="4"/>run()`) {
		t.Errorf("odt lost the indentation:\n%s", parts["content.xml"])
	}
}
