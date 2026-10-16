// Package imageloading reads the local images a document refers to, with the size each office
// format needs to place them.
package imageloading

import (
	"bytes"
	"image"
	_ "image/gif"  // registers the decoder LoadImage uses
	_ "image/jpeg" // registers the decoder LoadImage uses
	_ "image/png"  // registers the decoder LoadImage uses
	"os"
	"path/filepath"
	"strings"
)

// Image is a picture read from disk.
type Image struct {
	Data []byte
	// Extension is the file ending without its dot, always lower case ("png", "jpeg", "gif").
	Extension string
	// MIME is the media type, such as "image/png".
	MIME          string
	Width, Height int
}

// LoadImage reads the image a document refers to by source, relative to baseDir. It reports false
// for what it cannot embed: no base folder, an address on the web, a missing file, or a format
// other than PNG, JPEG and GIF.
func LoadImage(baseDir, source string) (Image, bool) {
	if baseDir == "" || source == "" || strings.Contains(source, "://") || strings.HasPrefix(source, "data:") {
		return Image{}, false
	}
	data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(source)))
	if err != nil {
		return Image{}, false
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Image{}, false
	}

	return Image{Data: data, Extension: format, MIME: "image/" + format, Width: config.Width, Height: config.Height}, true
}
