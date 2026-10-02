package credentialverification

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lore-master/libs/confluence-client/httptransport"
)

func verify(t *testing.T, status int, body string) (User, error) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/rest/api/user/current" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client, err := httptransport.New(httptransport.Options{BaseURL: server.URL + "/wiki", AuthorizationHeader: "Basic x"})
	if err != nil {
		t.Fatal(err)
	}

	return VerifyCredentials(context.Background(), client)
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func TestVerifyCredentials(t *testing.T) {
	cloud, err := verify(t, 200, fixture(t, "user-cloud.json"))
	if err != nil || cloud.AccountID != "5b10ac8d82e05b22cc7d4ef5" || cloud.DisplayName != "Ada Lovelace" {
		t.Fatalf("cloud %+v, %v", cloud, err)
	}
	dataCenter, err := verify(t, 200, fixture(t, "user-datacenter.json"))
	if err != nil || dataCenter.Username != "ada" || dataCenter.UserKey == "" || dataCenter.DisplayName != "Ada Lovelace" {
		t.Fatalf("data center %+v, %v", dataCenter, err)
	}
}

func TestVerifyCredentialsRefusals(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"401", 401, `{"message":"Client must be authenticated"}`, "Confluence rejected the credential (HTTP 401); check the email or user name and the token"},
		{"403", 403, `{}`, "Confluence recognised the credential but refused access (HTTP 403); the account may lack the \"Can use\" permission, or the token may have expired"},
		{"anonymous answer from Cloud", 200, fixture(t, "user-cloud-anonymous.json"), "Confluence treated the request as anonymous; the credential was not accepted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := verify(t, tc.status, tc.body)
			var unauthorized *UnauthorizedError
			if !errors.As(err, &unauthorized) || err.Error() != tc.want {
				t.Fatalf("error %v, want %q", err, tc.want)
			}
		})
	}
}

func TestVerifyCredentialsPassesOtherFailuresThrough(t *testing.T) {
	_, err := verify(t, 400, `{"message":"bad"}`)
	var apiError *httptransport.APIError
	if !errors.As(err, &apiError) || apiError.Status != 400 {
		t.Fatalf("error %v", err)
	}
}
