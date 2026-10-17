package odtexport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// template is what an OpenDocument template contributes: its styles part (the page, headers, footers and named styles) and the
// pictures that part refers to, a logo in a header for one.
type template struct {
	styles   string
	pictures []picture
}

// readTemplate reads a template. It must be an OpenDocument package with a styles part.
func readTemplate(path string) (template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return template{}, fmt.Errorf("the reference document: %w", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return template{}, fmt.Errorf("the reference document %s is not an OpenDocument file: %w", path, err)
	}

	var found template
	hasStyles := false
	for _, file := range archive.File {
		wanted := file.Name == "styles.xml" || (strings.HasPrefix(file.Name, "Pictures/") && !strings.HasSuffix(file.Name, "/"))
		if !wanted {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return template{}, err
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return template{}, err
		}
		if file.Name == "styles.xml" {
			found.styles, hasStyles = string(content), true

			continue
		}
		kind := mime.TypeByExtension(strings.ToLower(filepath.Ext(file.Name)))
		found.pictures = append(found.pictures, picture{name: strings.TrimPrefix(file.Name, "Pictures/"), mime: kind, data: content})
	}
	if !hasStyles {
		return template{}, fmt.Errorf("the reference document %s is not an OpenDocument file: it has no styles.xml", path)
	}

	return found, nil
}
