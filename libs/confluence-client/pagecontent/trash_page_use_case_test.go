package pagecontent

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"lore-master/libs/confluence-client/connection"
	"lore-master/libs/confluence-client/httptransport"
)

func TestTrashPage(t *testing.T) {
	cases := []struct {
		edition connection.Edition
		path    string
	}{
		{connection.Cloud, "/x/api/v2/pages/42"},
		{connection.DataCenter, "/x/rest/api/content/42"},
	}
	for _, tc := range cases {
		pages := pagesOn(t, tc.edition, "/x", func(r *http.Request) (int, string) {
			if r.Method != http.MethodDelete || r.URL.Path != tc.path {
				t.Errorf("%s: %s %s", tc.edition, r.Method, r.URL)
			}

			return 204, ""
		})
		if err := pages.TrashPage(context.Background(), "42"); err != nil {
			t.Fatalf("%s: %v", tc.edition, err)
		}
	}
}

func TestTrashPageTreatsAMissingPageAsTrashed(t *testing.T) {
	pages := pagesOn(t, connection.Cloud, "/x", func(*http.Request) (int, string) { return 404, `{}` })
	if err := pages.TrashPage(context.Background(), "42"); err != nil {
		t.Fatalf("error %v", err)
	}
	forbidden := pagesOn(t, connection.Cloud, "/x", func(*http.Request) (int, string) { return 403, `{}` })
	var apiError *httptransport.APIError
	if err := forbidden.TrashPage(context.Background(), "42"); !errors.As(err, &apiError) || apiError.Status != 403 {
		t.Fatalf("a refusal must surface, got %v", err)
	}
}
