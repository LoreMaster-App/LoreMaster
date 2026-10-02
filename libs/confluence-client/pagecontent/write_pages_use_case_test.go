package pagecontent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

// recordBody decodes the request body into a generic map for exact comparison.
func recordBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("body %s: %v", raw, err)
	}

	return body
}

func asJSON(t *testing.T, value string) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		t.Fatal(err)
	}

	return decoded
}

func TestCreatePageCloud(t *testing.T) {
	pages := pagesOn(t, connection.Cloud, "/wiki", func(r *http.Request) (int, string) {
		if r.Method != http.MethodPost || r.URL.Path != "/wiki/api/v2/pages" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		want := asJSON(t, `{"spaceId":"98306","status":"current","title":"ENG: Architecture","parentId":"98310","body":{"representation":"storage","value":"<p>x</p>"}}`)
		if got := recordBody(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body\n got: %v\nwant: %v", got, want)
		}

		return 200, fixture(t, "page-v2-with-body.json")
	})
	page, err := pages.CreatePage(context.Background(), CreatePageInput{SpaceID: "98306", SpaceKey: "ENG", ParentID: "98310", Title: "ENG: Architecture", BodyStorage: "<p>x</p>"})
	if err != nil || page.ID != "98400" || page.Version != 7 {
		t.Fatalf("page %+v, err %v", page, err)
	}
}

func TestCreatePageDataCenter(t *testing.T) {
	pages := pagesOn(t, connection.DataCenter, "/confluence", func(r *http.Request) (int, string) {
		if r.Method != http.MethodPost || r.URL.Path != "/confluence/rest/api/content" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		want := asJSON(t, `{"type":"page","title":"ENG: Architecture","space":{"key":"ENG"},"ancestors":[{"id":"131080"}],"body":{"storage":{"value":"<p>x</p>","representation":"storage"}}}`)
		if got := recordBody(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body\n got: %v\nwant: %v", got, want)
		}

		return 200, fixture(t, "page-v1-with-body.json")
	})
	page, err := pages.CreatePage(context.Background(), CreatePageInput{SpaceKey: "ENG", ParentID: "131080", Title: "ENG: Architecture", BodyStorage: "<p>x</p>"})
	if err != nil || page.ID != "131090" || page.ParentID != "131080" {
		t.Fatalf("page %+v, err %v", page, err)
	}
}

func TestUpdatePageSendsTheNextVersionAndTheParent(t *testing.T) {
	cases := []struct {
		edition connection.Edition
		path    string
		want    string
		fixture string
	}{
		{connection.Cloud, "/x/api/v2/pages/98400", `{"id":"98400","status":"current","title":"ENG: New title","parentId":"98999","body":{"representation":"storage","value":"<p>v8</p>"},"version":{"number":8,"message":"lore-master sync"}}`, "page-v2-with-body.json"},
		{connection.Server, "/x/rest/api/content/98400", `{"id":"98400","type":"page","title":"ENG: New title","ancestors":[{"id":"98999"}],"body":{"storage":{"value":"<p>v8</p>","representation":"storage"}},"version":{"number":8,"message":"lore-master sync"}}`, "page-v1-with-body.json"},
	}
	for _, tc := range cases {
		pages := pagesOn(t, tc.edition, "/x", func(r *http.Request) (int, string) {
			if r.Method != http.MethodPut || r.URL.Path != tc.path {
				t.Errorf("%s: %s %s", tc.edition, r.Method, r.URL)
			}
			if got := recordBody(t, r); !reflect.DeepEqual(got, asJSON(t, tc.want)) {
				t.Errorf("%s: body\n got: %v\nwant: %s", tc.edition, got, tc.want)
			}

			return 200, fixture(t, tc.fixture)
		})
		_, err := pages.UpdatePage(context.Background(), UpdatePageInput{ID: "98400", ExpectedVersion: 7, Title: "ENG: New title", ParentID: "98999", BodyStorage: "<p>v8</p>", Message: "lore-master sync"})
		if err != nil {
			t.Fatalf("%s: %v", tc.edition, err)
		}
	}
}

func TestUpdatePageConflictsAndTakenTitles(t *testing.T) {
	for _, edition := range []connection.Edition{connection.Cloud, connection.DataCenter} {
		conflict := pagesOn(t, edition, "/x", func(*http.Request) (int, string) {
			return 409, `{"message":"Version must be incremented on update. Current version is: 9"}`
		})
		_, err := conflict.UpdatePage(context.Background(), UpdatePageInput{ID: "1", ExpectedVersion: 7, Title: "T"})
		var versionConflict *VersionConflictError
		if !errors.As(err, &versionConflict) || err.Error() != "Confluence page 1 was edited after version 7; it was not overwritten" {
			t.Errorf("%s: error %v", edition, err)
		}

		fixtureName := map[connection.Edition]string{connection.Cloud: "title-taken-v2.json", connection.DataCenter: "title-taken-v1.json"}[edition]
		taken := pagesOn(t, edition, "/x", func(*http.Request) (int, string) { return 400, fixture(t, fixtureName) })
		_, err = taken.UpdatePage(context.Background(), UpdatePageInput{ID: "1", ExpectedVersion: 7, Title: "ENG: Setup"})
		var titleTaken *TitleTakenError
		if !errors.As(err, &titleTaken) || err.Error() != `another page in this space is already titled "ENG: Setup"; give the document a "title:" override in its lore-master annotation` {
			t.Errorf("%s: error %v", edition, err)
		}
		_, err = taken.CreatePage(context.Background(), CreatePageInput{SpaceID: "1", SpaceKey: "ENG", Title: "ENG: Setup"})
		if !errors.As(err, &titleTaken) {
			t.Errorf("%s: create error %v", edition, err)
		}
	}
}

func TestCreatePageRecoversAfterAGatewayTimeout(t *testing.T) {
	searchResult := `{"results":[{"id":"777","title":"ENG: New","space":{"id":1,"key":"ENG"},"version":{"number":1},"ancestors":[{"id":"10"}]}],"size":1,"limit":250,"_links":{}}`
	cases := []struct {
		name      string
		search    string
		wantID    string
		wantError int
	}{
		{"the page was created: return it", searchResult, "777", 0},
		{"nothing was created: the 504 stands", `{"results":[],"size":0,"_links":{}}`, "", 504},
		{"a same-titled page under another parent is not ours", `{"results":[{"id":"888","title":"ENG: New","space":{"key":"ENG"},"ancestors":[{"id":"99"}]}],"size":1,"limit":250,"_links":{}}`, "", 504},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			posts := 0
			pages := pagesOn(t, connection.DataCenter, "/x", func(r *http.Request) (int, string) {
				if r.Method == http.MethodPost {
					posts++

					return 504, "gateway timeout"
				}

				return 200, tc.search
			})
			page, err := pages.CreatePage(context.Background(), CreatePageInput{SpaceKey: "ENG", ParentID: "10", Title: "ENG: New"})
			if posts != 1 {
				t.Fatalf("the POST was sent %d times; a create must never be repeated", posts)
			}
			var apiError *httptransport.APIError
			switch {
			case tc.wantError == 0 && (err != nil || page.ID != tc.wantID):
				t.Fatalf("page %+v, err %v", page, err)
			case tc.wantError != 0 && (!errors.As(err, &apiError) || apiError.Status != tc.wantError):
				t.Fatalf("error %v, want HTTP %d", err, tc.wantError)
			}
		})
	}
}

func TestWriteInputsAreValidated(t *testing.T) {
	pages := pagesOn(t, connection.Cloud, "/x", func(*http.Request) (int, string) { t.Error("no request expected"); return 500, "" })
	if _, err := pages.CreatePage(context.Background(), CreatePageInput{Title: "T"}); err == nil {
		t.Error("a create without a space key must fail")
	}
	if _, err := pages.UpdatePage(context.Background(), UpdatePageInput{ID: "1", Title: "T"}); err == nil {
		t.Error("an update without the version it was read at must fail")
	}
}
