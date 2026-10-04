package attachments

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"lore-master/libs/confluence-client/httptransport"
)

func TestListAndDownloadAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/rest/api/content/42/child/attachment":
			_, _ = io.WriteString(w, `{"results":[{"id":"att1","title":"photo.png","metadata":{"comment":"sha256:abc"},"_links":{"download":"/download/attachments/42/photo.png?version=2&api=v2"}}],"size":1,"_links":{}}`)
		case "/wiki/download/attachments/42/photo.png":
			if r.URL.Query().Get("version") != "2" {
				t.Errorf("the download must carry the version, got %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte("PNG-BYTES"))
		default:
			t.Errorf("unexpected request %s", r.URL)
		}
	}))
	t.Cleanup(server.Close)
	client, err := httptransport.New(httptransport.Options{BaseURL: server.URL + "/wiki"})
	if err != nil {
		t.Fatal(err)
	}

	list, err := ListAttachments(context.Background(), client, "42")
	if err != nil || len(list) != 1 {
		t.Fatalf("list %+v, err %v", list, err)
	}
	if got := list[0]; got.Filename != "photo.png" || got.Hash != "sha256:abc" || got.DownloadPath != "/download/attachments/42/photo.png?version=2&api=v2" {
		t.Fatalf("attachment %+v", got)
	}

	content, err := DownloadAttachment(context.Background(), client, list[0].DownloadPath)
	if err != nil || string(content) != "PNG-BYTES" {
		t.Fatalf("content %q, err %v", content, err)
	}
}
