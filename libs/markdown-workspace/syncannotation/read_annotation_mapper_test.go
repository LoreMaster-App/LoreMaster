package syncannotation

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

const bom = "\xEF\xBB\xBF"

func TestReadWithoutAnnotation(t *testing.T) {
	for name, content := range map[string]string{
		"plain":                  "# Title\n\nText\n",
		"bom and crlf":           bom + "# Title\r\n",
		"comment that is not it": "<!-- lore-masterful -->\n# Title\n",
		"empty":                  "",
	} {
		t.Run(name, func(t *testing.T) {
			document, err := Read([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if document.Annotation != nil {
				t.Fatalf("unexpected annotation %+v", document.Annotation)
			}
			if got, want := string(document.Body), strings.TrimPrefix(content, bom); got != want {
				t.Fatalf("body %q, want %q", got, want)
			}
		})
	}
}

func TestReadParsesEveryKey(t *testing.T) {
	content := "<!-- lore-master\r\n" +
		"title:   Overridden title  \r\n" +
		"platform: confluence\r\n" +
		"base-url: https://acme.atlassian.net/wiki\r\n" +
		"space: ENG\r\n" +
		"page-id: 123456\r\n" +
		"\r\n" +
		"parent-id: 123000\r\n" +
		"version: 7\r\n" +
		"content-hash: sha256:abc\r\n" +
		"attachments: {\"diagram-1.svg\": \"sha256:def\"}\r\n" +
		"synced-at: 2026-10-01T12:00:00Z\r\n" +
		"parent: ../index.md\r\n" +
		"owner: docs-team\r\n" +
		"-->\r\n" +
		"# Title\r\n"
	document, err := Read([]byte(bom + content))
	if err != nil {
		t.Fatal(err)
	}
	want := Annotation{
		Platform: "confluence", BaseURL: "https://acme.atlassian.net/wiki", Space: "ENG",
		PageID: "123456", ParentID: "123000", Version: 7, ContentHash: "sha256:abc",
		Attachments: map[string]string{"diagram-1.svg": "sha256:def"},
		SyncedAt:    time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		Title:       "Overridden title", Parent: "../index.md",
		Unknown: []Field{{Key: "owner", Value: "docs-team"}},
	}
	if !reflect.DeepEqual(*document.Annotation, want) {
		t.Fatalf("annotation\n got: %+v\nwant: %+v", *document.Annotation, want)
	}
	if string(document.Body) != "# Title\r\n" {
		t.Fatalf("body %q", document.Body)
	}
	if document.Layout != (Layout{ByteOrderMark: true, LineEnding: "\r\n"}) {
		t.Fatalf("layout %+v", document.Layout)
	}
	if want := []string{`the lore-master annotation has an unknown key "owner"; it is kept as is`}; !reflect.DeepEqual(document.Warnings, want) {
		t.Fatalf("warnings %q", document.Warnings)
	}
}

func TestReadFindsTheAnnotationAfterFrontMatter(t *testing.T) {
	document, err := Read([]byte("---\ntags: [a]\n---\n<!-- lore-master\npage-id: 9\n-->\n# Title\n"))
	if err != nil {
		t.Fatal(err)
	}
	if document.Annotation == nil || document.Annotation.PageID != "9" {
		t.Fatalf("annotation %+v", document.Annotation)
	}
	if string(document.Body) != "---\ntags: [a]\n---\n# Title\n" {
		t.Fatalf("body %q", document.Body)
	}
}

func TestReadRejectsAnAnnotationItCannotTrust(t *testing.T) {
	cases := map[string]struct{ content, message string }{
		"unclosed":         {"<!-- lore-master\npage-id: 1\n# Title\n", `the lore-master annotation opened on line 1 is never closed with "-->"`},
		"not key value":    {"<!-- lore-master\npage-id 1\n-->\n", `line 2 of the lore-master annotation is not "key: value": "page-id 1"`},
		"key with a space": {"<!-- lore-master\npage id: 1\n-->\n", `line 2 of the lore-master annotation is not "key: value": "page id: 1"`},
		"repeated key":     {"---\na: b\n---\n<!-- lore-master\npage-id: 1\npage-id: 2\n-->\n", `line 6 of the lore-master annotation repeats the key "page-id"`},
		"version":          {"<!-- lore-master\nversion: seven\n-->\n", `the lore-master annotation's "version" value "seven" is invalid`},
		"synced-at":        {"<!-- lore-master\nsynced-at: yesterday\n-->\n", `the lore-master annotation's "synced-at" value "yesterday" is invalid`},
		"attachments":      {"<!-- lore-master\nattachments: a.svg\n-->\n", `the lore-master annotation's "attachments" value "a.svg" is invalid`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Read([]byte(tc.content))
			if err == nil || !strings.HasPrefix(err.Error(), tc.message) {
				t.Fatalf("error %v, want prefix %q", err, tc.message)
			}
		})
	}
}
