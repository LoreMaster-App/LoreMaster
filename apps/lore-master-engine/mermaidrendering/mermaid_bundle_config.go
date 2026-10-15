package mermaidrendering

import (
	"bytes"
	"compress/gzip"
	_ "embed" // the Mermaid bundle is embedded, so the engine needs nothing installed beside it
	"io"
	"sync"
)

//go:embed mermaid.min.js.gz
var bundleGzip []byte

var (
	bundleOnce sync.Once
	bundleText string
	bundleErr  error
)

// bundle is the Mermaid library as the script to run in a page, unpacked on first use.
func bundle() (string, error) {
	bundleOnce.Do(func() {
		reader, err := gzip.NewReader(bytes.NewReader(bundleGzip))
		if err != nil {
			bundleErr = err

			return
		}
		defer func() { _ = reader.Close() }()
		unpacked, err := io.ReadAll(reader)
		if err != nil {
			bundleErr = err

			return
		}
		bundleText = string(unpacked)
	})

	return bundleText, bundleErr
}
