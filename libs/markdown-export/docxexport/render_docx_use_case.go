package docxexport

import (
	"archive/zip"
	"bytes"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Options say where the document lives.
type Options struct {
	// BaseDir is the folder the document's relative image paths are read from.
	BaseDir string
	// ReferencePath is a Word file whose styles, numbering, headers, footers and page setup the document takes, as a company
	// template; its own text is not used. Empty writes the document with the styles this package ships with.
	ReferencePath string
}

// defaultSection is the page this package writes when there is no template: A4 with 2.5 cm margins.
const defaultSection = `<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`

// Render writes the document as the bytes of a .docx file.
func Render(document exportmodel.Document, options Options) ([]byte, error) {
	writer := &body{baseDir: options.BaseDir}
	writer.blocks(document.Blocks, 0)

	if options.ReferencePath != "" {
		parts, err := readReferenceDocument(options.ReferencePath)
		if err != nil {
			return nil, err
		}

		return renderOnReference(writer, parts)
	}

	parts := []struct{ name, content string }{
		{"[Content_Types].xml", contentTypes},
		{"_rels/.rels", packageRelationships},
		{"docProps/core.xml", coreProperties(document.Title)},
		{"word/document.xml", xmlHeader + `<w:document ` + namespaces + `><w:body>` + writer.out.String() + defaultSection + `</w:body></w:document>`},
		{"word/_rels/document.xml.rels", documentRelationships(writer.relationships)},
		{"word/styles.xml", styles},
		{"word/numbering.xml", numberingPart(writer.ordered)},
	}

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, part := range parts {
		if err := addFile(archive, part.name, []byte(part.content)); err != nil {
			return nil, err
		}
	}
	for _, picture := range writer.media {
		if err := addFile(archive, "word/media/"+picture.name, picture.data); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func addFile(archive *zip.Writer, name string, content []byte) error {
	file, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(content)

	return err
}

func coreProperties(title string) string {
	return xmlHeader + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>` + escape(title) + `</dc:title></cp:coreProperties>`
}

func documentRelationships(relationships []relationship) string {
	var out strings.Builder
	out.WriteString(xmlHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	out.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	out.WriteString(`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>`)
	for _, each := range relationships {
		mode := ""
		if each.external {
			mode = ` TargetMode="External"`
		}
		out.WriteString(`<Relationship Id="` + each.id + `" Type="` + each.kind + `" Target="` + escape(each.target) + `"` + mode + `/>`)
	}
	out.WriteString("</Relationships>")

	return out.String()
}
