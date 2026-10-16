package odtexport

import (
	"archive/zip"
	"bytes"
	"hash/crc32"
	"strings"

	"lore-master/libs/markdown-export/exportmodel"
)

// Options say where the document lives.
type Options struct {
	// BaseDir is the folder the document's relative image paths are read from.
	BaseDir string
}

// Render writes the document as the bytes of an .odt file.
func Render(document exportmodel.Document, options Options) ([]byte, error) {
	writer := &content{baseDir: options.BaseDir}
	writer.blocks(document.Blocks)

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)

	if err := addMimetype(archive); err != nil {
		return nil, err
	}

	parts := []struct{ name, content string }{
		{"META-INF/manifest.xml", manifest(writer.pictures)},
		{"meta.xml", xmlHeader + `<office:document-meta ` + namespaces + `><office:meta><dc:title>` + escape(document.Title) + `</dc:title></office:meta></office:document-meta>`},
		{"styles.xml", xmlHeader + `<office:document-styles ` + namespaces + `>` + namedStyles() + `</office:document-styles>`},
		{"content.xml", xmlHeader + `<office:document-content ` + namespaces + `>` + automaticStyles() + `<office:body><office:text>` + writer.out.String() + `</office:text></office:body></office:document-content>`},
	}
	for _, part := range parts {
		if err := addFile(archive, part.name, []byte(part.content)); err != nil {
			return nil, err
		}
	}
	for _, each := range writer.pictures {
		if err := addFile(archive, "Pictures/"+each.name, each.data); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// addMimetype writes the package's first entry: stored uncompressed, with its size and checksum in its
// own header and no data descriptor after it, which is how a reader recognises an OpenDocument file
// from its first bytes.
func addMimetype(archive *zip.Writer) error {
	data := []byte(mimetype)
	entry, err := archive.CreateRaw(&zip.FileHeader{
		Name:               "mimetype",
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(data),
		CompressedSize64:   uint64(len(data)),
		UncompressedSize64: uint64(len(data)),
	})
	if err != nil {
		return err
	}
	_, err = entry.Write(data)

	return err
}

func addFile(archive *zip.Writer, name string, data []byte) error {
	file, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(data)

	return err
}

// manifest lists every file of the package with its media type.
func manifest(pictures []picture) string {
	var out strings.Builder
	out.WriteString(xmlHeader + `<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">`)
	out.WriteString(`<manifest:file-entry manifest:full-path="/" manifest:version="1.2" manifest:media-type="` + mimetype + `"/>`)
	for _, name := range []string{"content.xml", "styles.xml", "meta.xml"} {
		out.WriteString(`<manifest:file-entry manifest:full-path="` + name + `" manifest:media-type="text/xml"/>`)
	}
	for _, each := range pictures {
		out.WriteString(`<manifest:file-entry manifest:full-path="Pictures/` + each.name + `" manifest:media-type="` + each.mime + `"/>`)
	}
	out.WriteString("</manifest:manifest>")

	return out.String()
}
