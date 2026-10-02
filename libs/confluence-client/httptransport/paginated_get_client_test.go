package httptransport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type item struct {
	ID string `json:"id"`
}

func TestPageV2FollowsNextLinks(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/spaces" || r.Header.Get("Authorization") != secretHeader {
			t.Errorf("request %s", r.URL)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			if r.URL.Query().Get("limit") != "2" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"results":[{"id":"1"},{"id":"2"}],"_links":{"next":"/wiki/api/v2/spaces?cursor=c2&limit=2"}}`)
		case "c2":
			_, _ = io.WriteString(w, `{"results":[{"id":"3"}],"_links":{}}`)
		}
	})
	var ids []string
	err := PageV2(context.Background(), client, "/api/v2/spaces", url.Values{"limit": {"2"}}, func(i item) error {
		ids = append(ids, i.ID)

		return nil
	})
	if err != nil || !reflect.DeepEqual(ids, []string{"1", "2", "3"}) {
		t.Fatalf("ids %v, err %v", ids, err)
	}
}

func TestPageV2RefusesANextLinkToAnotherSite(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"results":[{"id":"1"}],"_links":{"next":"https://evil.example/wiki/api/v2/spaces?cursor=x"}}`)
	})
	err := PageV2(context.Background(), client, "/api/v2/spaces", nil, func(item) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "refusing to send credentials there") {
		t.Fatalf("error %v", err)
	}
}

func TestPageV1StopsOnAShortPage(t *testing.T) {
	var starts []string
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		starts = append(starts, start)
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("type") != "global" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		switch start {
		case "0":
			_, _ = io.WriteString(w, `{"results":[{"id":"a"},{"id":"b"}],"size":2}`)
		case "2":
			_, _ = io.WriteString(w, `{"results":[{"id":"c"}],"size":1}`)
		}
	})
	var ids []string
	err := PageV1(context.Background(), client, "/rest/api/space", url.Values{"type": {"global"}}, 2, func(i item) error {
		ids = append(ids, i.ID)

		return nil
	})
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "b", "c"}) || !reflect.DeepEqual(starts, []string{"0", "2"}) {
		t.Fatalf("ids %v, starts %v, err %v", ids, starts, err)
	}
}

func TestPagingStopsWhenVisitFails(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"results":[{"id":"1"},{"id":"2"}],"size":2,"_links":{"next":"/wiki/again"}}`)
	})
	stop := fmt.Errorf("stop")
	visited := 0
	err := PageV2(context.Background(), client, "/api/v2/spaces", nil, func(item) error {
		visited++

		return stop
	})
	if err != stop || visited != 1 {
		t.Fatalf("visited %d, err %v", visited, err)
	}
}

func TestPageV1KeepsGoingWhenTheServerClampsTheLimit(t *testing.T) {
	var starts []string
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		starts = append(starts, start)
		switch start {
		case "0":
			_, _ = io.WriteString(w, `{"results":[{"id":"a"},{"id":"b"}],"start":0,"limit":2,"size":2,"_links":{"next":"/rest/api/space?start=2&limit=2"}}`)
		case "2":
			_, _ = io.WriteString(w, `{"results":[{"id":"c"},{"id":"d"}],"start":2,"limit":2,"size":2,"_links":{"next":"/rest/api/space?start=4&limit=2"}}`)
		case "4":
			_, _ = io.WriteString(w, `{"results":[{"id":"e"}],"start":4,"limit":2,"size":1,"_links":{}}`)
		}
	})
	var ids []string
	err := PageV1(context.Background(), client, "/rest/api/space", nil, 250, func(i item) error {
		ids = append(ids, i.ID)

		return nil
	})
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "b", "c", "d", "e"}) || !reflect.DeepEqual(starts, []string{"0", "2", "4"}) {
		t.Fatalf("ids %v, starts %v, err %v", ids, starts, err)
	}
}

func TestPageV1StopsOnAFullLastPageWithoutANextLink(t *testing.T) {
	calls := 0
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			_, _ = io.WriteString(w, `{"results":[{"id":"a"},{"id":"b"}],"limit":2,"size":2,"_links":{}}`)

			return
		}
		_, _ = io.WriteString(w, `{"results":[],"limit":2,"size":0,"_links":{}}`)
	})
	err := PageV1(context.Background(), client, "/rest/api/space", nil, 2, func(item) error { return nil })
	if err != nil || calls != 2 {
		t.Fatalf("calls %d, err %v", calls, err)
	}
}

func TestPageV1FollowsACursorNextLink(t *testing.T) {
	var cursors []string
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/rest/api/content/search" || r.URL.Query().Get("cql") != `type = page` {
			t.Errorf("request %s", r.URL)
		}
		cursor := r.URL.Query().Get("cursor")
		cursors = append(cursors, cursor)
		if cursor == "" {
			_, _ = io.WriteString(w, `{"results":[{"id":"1"}],"limit":1,"size":1,"_links":{"next":"/rest/api/content/search?cql=type+%3D+page&limit=1&cursor=raNDoM"}}`)

			return
		}
		_, _ = io.WriteString(w, `{"results":[{"id":"2"}],"limit":2,"size":1,"_links":{}}`)
	})
	var ids []string
	err := PageV1(context.Background(), client, "/rest/api/content/search", url.Values{"cql": {"type = page"}}, 1, func(i item) error {
		ids = append(ids, i.ID)

		return nil
	})
	if err != nil || !reflect.DeepEqual(ids, []string{"1", "2"}) || !reflect.DeepEqual(cursors, []string{"", "raNDoM"}) {
		t.Fatalf("ids %v, cursors %v, err %v", ids, cursors, err)
	}
}

func TestPageV1RefusesAnAbsoluteNextLinkToAnotherSite(t *testing.T) {
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"results":[{"id":"1"}],"limit":1,"size":1,"_links":{"next":"https://evil.example/rest/api/space?start=1"}}`)
	})
	err := PageV1(context.Background(), client, "/rest/api/space", nil, 1, func(item) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "refusing to send credentials there") {
		t.Fatalf("error %v", err)
	}
}

func TestPagingGivesUpOnAnEndlessServer(t *testing.T) {
	calls := 0
	client, _, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"results":[],"_links":{"next":"/wiki/api/v2/spaces?cursor=again"}}`)
	})
	err := PageV2(context.Background(), client, "/api/v2/spaces", nil, func(item) error { return nil })
	if err != errTooManyPages || calls != maxPages {
		t.Fatalf("calls %d, err %v", calls, err)
	}
}
