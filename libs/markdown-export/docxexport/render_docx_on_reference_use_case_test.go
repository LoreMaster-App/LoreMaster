package docxexport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lore-master/libs/markdown-export/exportmodel"
)

func heading(text string, level int) exportmodel.Block {
	return exportmodel.Block{Kind: exportmodel.HeadingBlock, Level: level, Inlines: []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: text}}}
}

func paragraph(text string) exportmodel.Block {
	return exportmodel.Block{Kind: exportmodel.ParagraphBlock, Inlines: []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: text}}}
}

func numberedList(items ...string) exportmodel.Block {
	list := exportmodel.Block{Kind: exportmodel.ListBlock, Ordered: true, Start: 1}
	for _, item := range items {
		list.Items = append(list.Items, exportmodel.Item{Blocks: []exportmodel.Block{paragraph(item)}})
	}

	return list
}

// aTemplate writes a company template the way Word would: a heading style with another id, a header, a footer and a page of its
// own, numbered headings through its own numbering part, no code styles, and a line of its own text that must not survive.
func aTemplate(t *testing.T) string {
	t.Helper()
	const w = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`
	files := map[string]string{
		"[Content_Types].xml": xmlHeader + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
			`<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>` +
			`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/></Types>`,
		"_rels/.rels": packageRelationships,
		"word/_rels/document.xml.rels": xmlHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>` +
			`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/></Relationships>`,
		"word/styles.xml": xmlHeader + `<w:styles ` + w + `>` +
			`<w:style w:type="paragraph" w:default="1" w:styleId="Standard"><w:name w:val="Normal"/></w:style>` +
			`<w:style w:type="paragraph" w:styleId="Titre1"><w:name w:val="Heading 1"/><w:basedOn w:val="Standard"/><w:pPr><w:numPr><w:numId w:val="1"/></w:numPr></w:pPr><w:rPr><w:color w:val="C00000"/></w:rPr></w:style>` +
			`</w:styles>`,
		"word/numbering.xml": xmlHeader + `<w:numbering ` + w + `><w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/></w:lvl></w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num></w:numbering>`,
		"word/header1.xml": xmlHeader + `<w:hdr ` + w + `><w:p><w:r><w:t>ACME Corp</w:t></w:r></w:p></w:hdr>`,
		"word/document.xml": xmlHeader + `<w:document ` + w + `><w:body><w:p><w:r><w:t>TEMPLATE TEXT</w:t></w:r></w:p>` +
			`<w:sectPr><w:headerReference w:type="default" r:id="rId3"/><w:pgSz w:w="12240" w:h="15840"/></w:sectPr></w:body></w:document>`,
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/_rels/document.xml.rels", "word/styles.xml", "word/numbering.xml", "word/header1.xml"} {
		if err := addFile(archive, name, []byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "template.docx")
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
		if strings.HasSuffix(file.Name, ".xml") || strings.HasSuffix(file.Name, ".rels") {
			decoder := xml.NewDecoder(bytes.NewReader(content))
			for {
				if _, err := decoder.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("%s is not well-formed: %v", file.Name, err)
				}
			}
		}
	}

	return parts
}

func TestADocumentTakesATemplatesStylesHeaderAndPage(t *testing.T) {
	document := exportmodel.Document{Title: "T", Blocks: []exportmodel.Block{heading("Title", 1), paragraph("Body."), numberedList("one", "two")}}

	data, err := Render(document, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	parts := partsOf(t, data)

	body := parts["word/document.xml"]
	if strings.Contains(body, "TEMPLATE TEXT") || !strings.Contains(body, "Body.") {
		t.Errorf("the template's text was kept or the document's was lost:\n%s", body)
	}
	if !strings.Contains(body, `<w:pStyle w:val="Titre1"/>`) || strings.Contains(body, `w:val="Heading1"`) {
		t.Errorf("a heading should use the template's own id for Heading 1:\n%s", body)
	}
	if !strings.Contains(body, `<w:headerReference w:type="default" r:id="rId3"/>`) || !strings.Contains(body, `w:w="12240"`) {
		t.Errorf("the template's header and page were not kept:\n%s", body)
	}
	if parts["word/header1.xml"] == "" || !strings.Contains(parts["word/_rels/document.xml.rels"], `Target="header1.xml"`) {
		t.Error("the template's header part is missing")
	}
}

func TestAStyleTheTemplateLacksIsAddedAndFitsItsDefaults(t *testing.T) {
	data, err := Render(exportmodel.Document{Blocks: []exportmodel.Block{{Kind: exportmodel.CodeBlock, Text: "x"}}}, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	styles := partsOf(t, data)["word/styles.xml"]

	if !strings.Contains(styles, `w:styleId="Code"`) || !strings.Contains(styles, `<w:basedOn w:val="Standard"/>`) {
		t.Errorf("Code should be added, based on the template's own Normal:\n%s", styles)
	}
	if strings.Count(styles, `w:styleId="Titre1"`) != 1 || strings.Contains(styles, `w:styleId="Heading1"`) {
		t.Errorf("a style the template has under another id should not be added again:\n%s", styles)
	}
}

func TestTheTemplatesNumberingIsKeptAndOursSitsBesideIt(t *testing.T) {
	document := exportmodel.Document{Blocks: []exportmodel.Block{numberedList("one", "two"), numberedList("three")}}

	data, err := Render(document, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	parts := partsOf(t, data)
	numbering := parts["word/numbering.xml"]

	if !strings.Contains(numbering, `<w:abstractNum w:abstractNumId="0"><w:lvl w:ilvl="0"><w:numFmt w:val="decimal"/>`) || !strings.Contains(numbering, `<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`) {
		t.Errorf("the template's own numbering was lost:\n%s", numbering)
	}
	if strings.LastIndex(numbering, "</w:abstractNum>") > strings.Index(numbering, "<w:num ") {
		t.Errorf("every abstract definition has to come before any instance:\n%s", numbering)
	}
	// The template has abstract 0 and instance 1, so ours start above them, and the body points at the moved instances.
	if !strings.Contains(numbering, `<w:abstractNum w:abstractNumId="1">`) || !strings.Contains(numbering, `<w:abstractNum w:abstractNumId="2">`) {
		t.Errorf("our abstract definitions should follow the template's:\n%s", numbering)
	}
	body := parts["word/document.xml"]
	for _, moved := range []string{`<w:numId w:val="3"/>`, `<w:numId w:val="4"/>`} {
		if !strings.Contains(body, moved) || !strings.Contains(numbering, strings.Replace(moved, "numId w:val", "num w:numId", 1)[:len(`<w:num w:numId="3"`)]) {
			t.Errorf("the body and the numbering should agree on %s", moved)
		}
	}
	if strings.Contains(body, `<w:numId w:val="1"/>`) || strings.Contains(body, `<w:numId w:val="2"/>`) {
		t.Errorf("a list kept the id of the template's instances:\n%s", body)
	}
}

func TestOurRelationshipsAndImagesCannotMeetTheTemplatesOwn(t *testing.T) {
	link := exportmodel.Inline{Kind: exportmodel.LinkInline, URL: "https://example.com", Children: []exportmodel.Inline{{Kind: exportmodel.TextInline, Text: "x"}}}
	document := exportmodel.Document{Blocks: []exportmodel.Block{{Kind: exportmodel.ParagraphBlock, Inlines: []exportmodel.Inline{link}}}}

	data, err := Render(document, Options{ReferencePath: aTemplate(t)})
	if err != nil {
		t.Fatal(err)
	}
	rels := partsOf(t, data)["word/_rels/document.xml.rels"]

	if !strings.Contains(rels, `Id="rId3"`) || !strings.Contains(rels, `Id="rIdLoreMaster1"`) || !strings.Contains(rels, `TargetMode="External"`) {
		t.Errorf("relationships were not merged:\n%s", rels)
	}
}

func TestSomethingThatIsNotAWordFileIsRefusedWithTheReason(t *testing.T) {
	notZip := filepath.Join(t.TempDir(), "x.docx")
	if err := os.WriteFile(notZip, []byte("plain text"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Render(exportmodel.Document{}, Options{ReferencePath: notZip}); err == nil || !strings.Contains(err.Error(), "not a Word file") {
		t.Errorf("got %v", err)
	}
	if _, err := Render(exportmodel.Document{}, Options{ReferencePath: filepath.Join(t.TempDir(), "missing.docx")}); err == nil {
		t.Error("a missing template was accepted")
	}
}

func TestWithoutATemplateNothingChanges(t *testing.T) {
	document := exportmodel.Document{Blocks: []exportmodel.Block{heading("A", 1)}}

	plain, err := Render(document, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(partsOf(t, plain)["word/document.xml"], `w:val="Heading1"`) {
		t.Error("the shipped styles are no longer used without a template")
	}
}
