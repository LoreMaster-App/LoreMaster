package documentexport

import (
	"bytes"
	"image/png"
	"testing"

	"lore-master/libs/markdown-export/browserexport"
)

// withBrowser skips a test where there is no Chrome, Edge or Chromium to print with.
func withBrowser(t *testing.T) string {
	t.Helper()
	browser, err := browserexport.LocateBrowser("")
	if err != nil {
		t.Skip("no browser installed: " + err.Error())
	}

	return browser
}

func TestPDFAndPNGAreBinaryFormatsNamedAfterTheDocument(t *testing.T) {
	for id, name := range map[string]string{"pdf": "guide.pdf", "png": "guide.png"} {
		format, found := FormatByID(id)
		if !found || !format.Binary || resultName("docs/guide.md", format) != name {
			t.Errorf("%s: %+v", id, format)
		}
	}
}

func TestPDFAndPNGArePrintedFromTheDocument(t *testing.T) {
	browser := withBrowser(t)

	pdf, err := Export("guide.md", []byte(sample), "pdf", Options{BrowserPath: browser, PageSize: "Letter"})
	if err != nil || !bytes.HasPrefix(pdf.Content, []byte("%PDF-")) {
		t.Fatalf("pdf: %v, %q", err, pdf.Content[:min(len(pdf.Content), 10)])
	}

	image, err := Export("guide.md", []byte(sample), "png", Options{BrowserPath: browser, Width: 640})
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(image.Content))
	if err != nil || config.Width != 640 || config.Height < 200 {
		t.Fatalf("png: %v, %+v", err, config)
	}
}

func TestPDFWithoutABrowserFailsWithTheWayToFixIt(t *testing.T) {
	if _, err := Export("guide.md", []byte(sample), "pdf", Options{BrowserPath: "/no/such/browser"}); err == nil {
		t.Fatal("a missing browser was accepted")
	}
}
