package spacecatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

func serve(t *testing.T, contextPath string, handler func(r *http.Request) string) *httptransport.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(handler(r)))
	}))
	t.Cleanup(server.Close)
	client, err := httptransport.New(httptransport.Options{BaseURL: server.URL + contextPath})
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func TestListSpacesCloud(t *testing.T) {
	client := serve(t, "/wiki", func(r *http.Request) string {
		if r.URL.Path != "/wiki/api/v2/spaces" || r.URL.Query().Get("status") != "current" {
			t.Errorf("request %s", r.URL)
		}
		if r.URL.Query().Get("cursor") == "" {
			return fixture(t, "spaces-v2-page1.json")
		}

		return fixture(t, "spaces-v2-page2.json")
	})
	spaces, err := ListSpaces(context.Background(), client, connection.Cloud, Options{})
	want := []Space{
		{ID: "98310", Key: "arch", Name: "architecture"},
		{ID: "98306", Key: "ENG", Name: "Engineering", HomepageID: "98400"},
	}
	if err != nil || !reflect.DeepEqual(spaces, want) {
		t.Fatalf("spaces %+v, err %v", spaces, err)
	}
}

func TestListSpacesDataCenterWithPersonal(t *testing.T) {
	client := serve(t, "/confluence", func(r *http.Request) string {
		if r.URL.Path != "/confluence/rest/api/space" || r.URL.Query().Get("expand") != "homepage" {
			t.Errorf("request %s", r.URL)
		}
		if r.URL.Query().Get("start") == "0" {
			return fixture(t, "spaces-v1-page1.json")
		}

		return fixture(t, "spaces-v1-page2.json")
	})
	spaces, err := ListSpaces(context.Background(), client, connection.DataCenter, Options{IncludePersonal: true})
	want := []Space{
		{ID: "131075", Key: "~ada", Name: "Ada Lovelace", HomepageID: "131076", Personal: true},
		{ID: "131073", Key: "ENG", Name: "Engineering", HomepageID: "131074"},
		{ID: "131080", Key: "OPS", Name: "Operations"},
	}
	if err != nil || !reflect.DeepEqual(spaces, want) {
		t.Fatalf("spaces %+v, err %v", spaces, err)
	}
}
