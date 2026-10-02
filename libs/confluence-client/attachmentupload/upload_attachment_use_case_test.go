package attachmentupload

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"lore-master/libs/confluence-client/httptransport"
)

type upload struct {
	path, filename, contentType, content, comment, minorEdit, xsrf string
}

// site answers the listing with listing and records every upload.
func site(t *testing.T, listing string, answer string) (*httptransport.Client, *[]upload, *int) {
	t.Helper()
	var uploads []upload
	lists := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			lists++
			if r.URL.Path != "/wiki/rest/api/content/42/child/attachment" || r.URL.Query().Get("filename") != "diagram 1.svg" {
				t.Errorf("listing %s", r.URL)
			}
			_, _ = io.WriteString(w, listing)

			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(file)
		uploads = append(uploads, upload{
			path: r.URL.Path, filename: header.Filename, contentType: header.Header.Get("Content-Type"),
			content: string(content), comment: r.FormValue("comment"), minorEdit: r.FormValue("minorEdit"),
			xsrf: r.Header.Get("X-Atlassian-Token"),
		})
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(server.Close)
	client, err := httptransport.New(httptransport.Options{BaseURL: server.URL + "/wiki"})
	if err != nil {
		t.Fatal(err)
	}

	return client, &uploads, &lists
}

var svg = []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)

func TestUploadCreatesANewAttachment(t *testing.T) {
	client, uploads, _ := site(t, `{"results":[],"size":0}`, `{"results":[{"id":"att901","title":"diagram 1.svg"}],"size":1}`)
	got, err := UploadAttachment(context.Background(), client, "42", AttachmentInput{Filename: "diagram 1.svg", Content: svg})
	if err != nil || got != (Attachment{ID: "att901", Filename: "diagram 1.svg", Hash: ContentHash(svg)}) {
		t.Fatalf("got %+v, err %v", got, err)
	}
	want := upload{
		path: "/wiki/rest/api/content/42/child/attachment", filename: "diagram 1.svg", contentType: "image/svg+xml",
		content: string(svg), comment: ContentHash(svg), minorEdit: "true", xsrf: "nocheck",
	}
	if len(*uploads) != 1 || (*uploads)[0] != want {
		t.Fatalf("uploads\n got: %+v\nwant: %+v", *uploads, want)
	}
}

func TestUploadAddsANewVersionWhenTheContentChanged(t *testing.T) {
	listing := `{"results":[{"id":"att901","title":"diagram 1.svg","metadata":{"comment":"sha256:old"}}],"size":1}`
	client, uploads, _ := site(t, listing, `{"id":"att901","title":"diagram 1.svg","version":{"number":2}}`)
	got, err := UploadAttachment(context.Background(), client, "42", AttachmentInput{Filename: "diagram 1.svg", ContentType: "image/svg+xml", Content: svg})
	if err != nil || got.ID != "att901" || got.Skipped {
		t.Fatalf("got %+v, err %v", got, err)
	}
	if len(*uploads) != 1 || (*uploads)[0].path != "/wiki/rest/api/content/42/child/attachment/att901/data" || (*uploads)[0].comment != ContentHash(svg) {
		t.Fatalf("uploads %+v", *uploads)
	}
}

func TestUploadSkipsAnUnchangedFile(t *testing.T) {
	for name, listing := range map[string]string{
		"comment in metadata":   `{"results":[{"id":"att901","title":"diagram 1.svg","metadata":{"comment":"` + ContentHash(svg) + `"}}]}`,
		"comment in extensions": `{"results":[{"id":"att901","title":"diagram 1.svg","extensions":{"comment":"` + ContentHash(svg) + `"}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			client, uploads, lists := site(t, listing, `{}`)
			got, err := UploadAttachment(context.Background(), client, "42", AttachmentInput{Filename: "diagram 1.svg", Content: svg})
			if err != nil || !got.Skipped || got.ID != "att901" || len(*uploads) != 0 || *lists != 1 {
				t.Fatalf("got %+v, uploads %d, err %v", got, len(*uploads), err)
			}
		})
	}
}

func TestUploadIgnoresAListingEntryWithAnotherName(t *testing.T) {
	listing := `{"results":[{"id":"att7","title":"diagram 10.svg","metadata":{"comment":"` + ContentHash(svg) + `"}}]}`
	client, uploads, _ := site(t, listing, `{"results":[{"id":"att902"}]}`)
	got, err := UploadAttachment(context.Background(), client, "42", AttachmentInput{Filename: "diagram 1.svg", Content: svg})
	if err != nil || got.ID != "att902" || len(*uploads) != 1 || (*uploads)[0].path != "/wiki/rest/api/content/42/child/attachment" {
		t.Fatalf("got %+v, uploads %+v, err %v", got, *uploads, err)
	}
}

func TestUploadNeedsAPageAndAName(t *testing.T) {
	client, _, _ := site(t, `{}`, `{}`)
	if _, err := UploadAttachment(context.Background(), client, "", AttachmentInput{Filename: "a.png"}); err == nil {
		t.Fatal("expected an error")
	}
}
