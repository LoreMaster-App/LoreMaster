package confluenceplatform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lore-master/libs/confluence-client/storageformat"
	"lore-master/libs/documentation-sync/workspacesettings"
)

// fakeSite is a fake Data Center 8.5.4 site that accepts the PAT "good-token".
func fakeSite(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case "/rest/applinks/1.0/manifest":
			_, _ = w.Write([]byte(`<manifest><typeId>confluence</typeId><version>8.5.4</version><buildNumber>9012</buildNumber></manifest>`))
		case "/rest/api/user/current":
			if r.Header.Get("Authorization") != "Bearer good-token" {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}
			_, _ = w.Write([]byte(`{"type":"known","username":"ada","userKey":"k1","displayName":"Ada Lovelace"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server, &requests
}

func TestConnectDetectsVerifiesAndDescribes(t *testing.T) {
	server, requests := fakeSite(t)
	connected, err := Connect(context.Background(), ConnectOptions{
		BaseURL: server.URL + "/", SignIn: SignIn{Kind: "pat", Token: "good-token"}, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if connected.Edition != "datacenter" || connected.Version != "8.5.4" || connected.Username != "ada" ||
		connected.DisplayName != "Ada Lovelace" || connected.Platform == nil || connected.BaseURL != server.URL {
		t.Fatalf("%+v", connected)
	}
	if want := []string{"/rest/api/settings/systemInfo", "/rest/applinks/1.0/manifest", "/rest/api/user/current"}; strings.Join(*requests, " ") != strings.Join(want, " ") {
		t.Fatalf("two detection probes, then one verification: %q", *requests)
	}
}

func TestConnectWithAKnownEditionSkipsDetection(t *testing.T) {
	server, requests := fakeSite(t)
	connected, err := Connect(context.Background(), ConnectOptions{
		BaseURL: server.URL, Edition: "datacenter", SignIn: SignIn{Kind: "pat", Token: "good-token"}, HTTPClient: server.Client(),
	})
	if err != nil || connected.Version != "" || len(*requests) != 1 {
		t.Fatalf("%+v %v %q", connected, err, *requests)
	}
}

func TestConnectSaysWhatWentWrong(t *testing.T) {
	server, _ := fakeSite(t)
	cases := map[string]struct {
		options ConnectOptions
		failure ConnectFailure
	}{
		"refused":        {ConnectOptions{BaseURL: server.URL, SignIn: SignIn{Kind: "pat", Token: "wrong-secret"}}, Unauthorized},
		"cloud token":    {ConnectOptions{BaseURL: server.URL, SignIn: SignIn{Kind: "apitoken", Email: "a@b.c", Token: "wrong-secret"}}, UnsupportedCredential},
		"no token":       {ConnectOptions{BaseURL: server.URL, SignIn: SignIn{Kind: "pat"}}, InvalidCredential},
		"unknown kind":   {ConnectOptions{BaseURL: server.URL, SignIn: SignIn{Kind: "oauth", Token: "wrong-secret"}}, InvalidCredential},
		"bad address":    {ConnectOptions{BaseURL: "not a url", SignIn: SignIn{Kind: "pat", Token: "wrong-secret"}}, InvalidAddress},
		"bad edition":    {ConnectOptions{BaseURL: server.URL, Edition: "cloudy", SignIn: SignIn{Kind: "pat", Token: "wrong-secret"}}, InvalidAddress},
		"not confluence": {ConnectOptions{BaseURL: server.URL + "/elsewhere", SignIn: SignIn{Kind: "pat", Token: "wrong-secret"}}, Unreachable},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			c.options.HTTPClient = server.Client()
			if name == "not confluence" {
				nothing := httptest.NewServer(http.NotFoundHandler())
				defer nothing.Close()
				c.options.BaseURL, c.options.HTTPClient = nothing.URL, nothing.Client()
			}
			_, err := Connect(context.Background(), c.options)
			var failed *ConnectError
			if !errors.As(err, &failed) || failed.Failure != c.failure {
				t.Fatalf("got %v", err)
			}
			if strings.Contains(err.Error(), "wrong-secret") {
				t.Fatalf("the error names the secret: %v", err)
			}
		})
	}
}

func TestForOutputRendersAsTheOutputAsks(t *testing.T) {
	base := &Platform{render: storageformat.Options{}}
	cases := map[[2]string]storageformat.Options{
		{"title", "image"}:          {LinkMode: storageformat.LinkByTitle, MermaidMode: storageformat.MermaidImage},
		{"id", "code"}:              {LinkMode: storageformat.LinkByURL, MermaidMode: storageformat.MermaidCode},
		{"title", "html-macro"}:     {LinkMode: storageformat.LinkByTitle, MermaidMode: storageformat.MermaidHTMLMacro},
		{"id", "marketplace-macro"}: {LinkMode: storageformat.LinkByURL, MermaidMode: storageformat.MermaidMarketplaceMacro},
	}
	for modes, want := range cases {
		got := base.ForOutput(workspacesettings.Output{LinkMode: modes[0], MermaidMode: modes[1]})
		if got.render != want || got == base {
			t.Errorf("%v: %+v", modes, got.render)
		}
	}
	if base.render != (storageformat.Options{}) {
		t.Fatal("the original is unchanged")
	}
}

func TestSameSite(t *testing.T) {
	cases := map[[2]string]bool{
		{"https://acme.atlassian.net", "https://ACME.atlassian.net/wiki/"}:                          true,
		{"https://acme.atlassian.net/wiki/spaces/ENG/pages/1/X", "https://acme.atlassian.net/wiki"}: true,
		{"https://confluence.acme.com/confluence/", "https://confluence.acme.com/confluence"}:       true,
		{"https://acme.atlassian.net", "https://other.atlassian.net"}:                               false,
		{"https://confluence.acme.com/a", "https://confluence.acme.com/b"}:                          false,
		{"", ""}: false,
	}
	for pair, want := range cases {
		if SameSite(pair[0], pair[1]) != want {
			t.Errorf("%q: want %v", pair, want)
		}
	}
}
