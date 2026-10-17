package docxexport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// packagePart is one file of a .docx.
type packagePart struct {
	name string
	data []byte
}

// readReferenceDocument reads a template's parts, in the order they are stored. It must be a Word file: a package with a
// document and a styles part.
func readReferenceDocument(path string) ([]packagePart, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the reference document: %w", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("the reference document %s is not a Word file: %w", path, err)
	}

	var parts []packagePart
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return nil, err
		}
		parts = append(parts, packagePart{name: file.Name, data: content})
	}
	if find(parts, "word/document.xml") < 0 || find(parts, "word/styles.xml") < 0 {
		return nil, fmt.Errorf("the reference document %s is not a Word file: it has no word/document.xml and word/styles.xml", path)
	}

	return parts, nil
}

func find(parts []packagePart, name string) int {
	for index, part := range parts {
		if part.name == name {
			return index
		}
	}

	return -1
}

// renderOnReference writes the document into a template: the template's styles, numbering, theme, headers, footers and page
// setup are kept, and its own text is replaced by the document. Headings, quotes, code and the rest take the look the template
// gives them; styles the template does not have are added to it.
func renderOnReference(writer *body, parts []packagePart) ([]byte, error) {
	text := func(name string) string {
		if index := find(parts, name); index >= 0 {
			return string(parts[index].data)
		}

		return ""
	}

	styled, ids := mergeStyles(text("word/styles.xml"))
	numbering, instanceOffset := mergeNumbering(text("word/numbering.xml"), writer.ordered)
	content := moveNumberingUse(renameStyleReferences(writer.out.String(), ids), instanceOffset)
	section := bodySectionProperties(text("word/document.xml"))
	if section == "" {
		section = defaultSection
	}

	replaced := map[string]string{
		"[Content_Types].xml":          mergeContentTypes(text("[Content_Types].xml"), true),
		"word/styles.xml":              styled,
		"word/numbering.xml":           numbering,
		"word/document.xml":            xmlHeader + `<w:document ` + namespaces + `><w:body>` + content + section + `</w:body></w:document>`,
		"word/_rels/document.xml.rels": mergeRelationships(text("word/_rels/document.xml.rels"), writer.relationships, true),
	}

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	written := map[string]bool{}
	for _, part := range parts {
		data := []byte(replaced[part.name])
		if _, isReplaced := replaced[part.name]; !isReplaced {
			data = part.data
		}
		if err := addFile(archive, part.name, data); err != nil {
			return nil, err
		}
		written[part.name] = true
	}
	// A template without a numbering part gets one; the order is fixed so the same input gives the same file.
	for _, name := range []string{"word/numbering.xml"} {
		if !written[name] {
			if err := addFile(archive, name, []byte(replaced[name])); err != nil {
				return nil, err
			}
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
