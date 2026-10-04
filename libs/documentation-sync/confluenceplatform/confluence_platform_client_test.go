package confluenceplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lore-master/libs/confluence-client/authentication"
	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/platformport"
)

func TestGetPageContentParsesTheStorageBody(t *testing.T) {
	const storage = `<h1>Title</h1><p>a <strong>b</strong></p>`
	platform := dataCenter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/confluence/rest/api/content/700" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		if !strings.Contains(r.URL.Query().Get("expand"), "body.storage") {
			t.Errorf("the pull must ask for the body, expand was %q", r.URL.Query().Get("expand"))
		}
		_, _ = io.WriteString(w, `{"id":"700","title":"ENG: Doc","version":{"number":5},"ancestors":[{"id":"500"}],"space":{"key":"ENG"},"body":{"storage":{"value":"`+storage+`"}}}`)
	})
	content, err := platform.GetPageContent(context.Background(), "700")
	if err != nil {
		t.Fatal(err)
	}
	if content.Version != 5 || len(content.Flags) != 0 {
		t.Fatalf("version %d, flags %v", content.Version, content.Flags)
	}
	// The body round-trips: mapping it back to storage reproduces what the page served.
	remapped, err := toStorage(content.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, err := storageformat.Render(remapped, storageformat.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got != storage {
		t.Fatalf("round trip\n got %q\nwant %q", got, storage)
	}
}

func TestGetPageContentFlagsUnconvertibleContent(t *testing.T) {
	platform := dataCenter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"700","title":"X","version":{"number":1},"space":{"key":"ENG"},"body":{"storage":{"value":"<ac:structured-macro ac:name=\"info\"><ac:rich-text-body><p>hi</p></ac:rich-text-body></ac:structured-macro>"}}}`)
	})
	content, err := platform.GetPageContent(context.Background(), "700")
	if err != nil {
		t.Fatal(err)
	}
	if len(content.Flags) == 0 {
		t.Fatalf("an unknown macro should be flagged, body %#v", content.Body)
	}
}

func dataCenter(t *testing.T, handler http.HandlerFunc) *Platform {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	platform, err := New(Options{
		Connection: connection.Connection{BaseURL: server.URL + "/confluence", Edition: connection.DataCenter, Version: connection.Version{Major: 8, Minor: 5}},
		Credential: authentication.PAT{Token: "t"},
	})
	if err != nil {
		t.Fatal(err)
	}

	return platform
}

func TestNewRefusesACredentialTheEditionCannotTake(t *testing.T) {
	_, err := New(Options{
		Connection: connection.Connection{BaseURL: "https://acme.atlassian.net/wiki", Edition: connection.Cloud},
		Credential: authentication.PAT{Token: "t"},
	})
	if err == nil || !strings.Contains(err.Error(), "does not work on Confluence Cloud") {
		t.Fatalf("error %v", err)
	}
}

func TestCreatePageRendersTheBodyAndMapsTheAnswer(t *testing.T) {
	platform := dataCenter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/confluence/rest/api/content" || r.Header.Get("Authorization") != "Bearer t" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		var body struct {
			Title string `json:"title"`
			Body  struct {
				Storage struct {
					Value string `json:"value"`
				} `json:"storage"`
			} `json:"body"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		want := `<p>see <ac:link><ri:page ri:content-title="ENG: Other" /><ac:link-body>ENG: Other</ac:link-body></ac:link></p>`
		if body.Title != "ENG: New" || body.Body.Storage.Value != want {
			t.Errorf("title %q, storage %q", body.Title, body.Body.Storage.Value)
		}
		_, _ = io.WriteString(w, `{"id":"501","title":"ENG: New","version":{"number":1},"ancestors":[{"id":"500"}],"_links":{"webui":"/display/ENG/New","base":"https://c.acme.com/confluence"}}`)
	})
	page, err := platform.CreatePage(context.Background(), platformport.NewPage{
		Space: platformport.SpaceRef{Key: "ENG"}, ParentID: "500", Title: "ENG: New",
		Body: platformport.Document{Blocks: []platformport.Block{platformport.Paragraph{Inlines: []platformport.Inline{
			platformport.Text{Value: "see "}, platformport.Link{Target: platformport.PageLink{Title: "ENG: Other"}},
		}}}},
	})
	want := platformport.RemotePage{ID: "501", Title: "ENG: New", ParentID: "500", Version: 1, URL: "https://c.acme.com/confluence/display/ENG/New"}
	if err != nil || page != want {
		t.Fatalf("page %+v, err %v", page, err)
	}
}

func TestErrorsArriveAsThePortsErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		check  func(error) bool
	}{
		{"conflict", 409, `{"message":"Version must be incremented"}`, func(err error) bool {
			var e *platformport.VersionConflictError

			return errors.As(err, &e) && e.ID == "42" && e.ExpectedVersion == 3
		}},
		{"title taken", 400, `{"message":"A page with this title already exists: A page already exists with the title ENG: X"}`, func(err error) bool {
			var e *platformport.TitleTakenError

			return errors.As(err, &e) && e.Title == "ENG: X"
		}},
		{"not found", 404, `{"message":"No content found"}`, func(err error) bool {
			var e *platformport.PageNotFoundError

			return errors.As(err, &e) && e.ID == "42"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			platform := dataCenter(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := platform.UpdatePage(context.Background(), platformport.PageUpdate{ID: "42", ExpectedVersion: 3, Title: "ENG: X"})
			if !tc.check(err) {
				t.Fatalf("error %T %v", err, err)
			}
		})
	}
}

func TestListMarkedDescendantsUsesTheMarker(t *testing.T) {
	platform := dataCenter(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cql") != `ancestor = "500" and label = "lore-master" and type = page` {
			t.Errorf("cql %s", r.URL.Query().Get("cql"))
		}
		_, _ = io.WriteString(w, `{"results":[{"id":"501","title":"ENG: Ours","ancestors":[{"id":"500"}]}],"size":1,"limit":250,"_links":{}}`)
	})
	found, err := platform.ListMarkedDescendants(context.Background(), "500")
	if err != nil || len(found) != 1 || found[0].ID != "501" || found[0].ParentID != "500" {
		t.Fatalf("found %+v, err %v", found, err)
	}
}
