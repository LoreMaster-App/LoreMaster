package connection

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"https://acme.atlassian.net", "https://acme.atlassian.net/wiki"},
		{"  https://ACME.atlassian.net/  ", "https://acme.atlassian.net/wiki"},
		{"https://acme.atlassian.net/wiki/", "https://acme.atlassian.net/wiki"},
		{"https://acme.atlassian.net/wiki/spaces/ENG/overview", "https://acme.atlassian.net/wiki"},
		{"https://confluence.acme.com", "https://confluence.acme.com"},
		{"https://acme.com/confluence//", "https://acme.com/confluence"},
		{"https://acme.com:8443/confluence", "https://acme.com:8443/confluence"},
		{"http://localhost:8090", "http://localhost:8090"},
		{"http://127.0.0.1:8090/confluence/", "http://127.0.0.1:8090/confluence"},
		{"http://[::1]:8090", "http://[::1]:8090"},
	}
	for _, tc := range cases {
		got, err := NormalizeBaseURL(tc.raw)
		if err != nil || got != tc.want {
			t.Errorf("NormalizeBaseURL(%q) = %q, %v; want %q", tc.raw, got, err, tc.want)
		}
	}
}

func TestNormalizeBaseURLRejects(t *testing.T) {
	cases := map[string]string{
		"acme.atlassian.net":         `"acme.atlassian.net" is not a Confluence address; expected something like https://example.atlassian.net`,
		"ftp://acme.com":             `"ftp://acme.com" is not a Confluence address; expected something like https://example.atlassian.net`,
		"http://confluence.acme.com": "the Confluence address must use https; plain http is accepted only for localhost",
		"https://me:secret@acme.com": "the Confluence address must not contain a user name or password",
	}
	for raw, want := range cases {
		if _, err := NormalizeBaseURL(raw); err == nil || err.Error() != want {
			t.Errorf("NormalizeBaseURL(%q) error %v, want %q", raw, err, want)
		}
	}
}

func TestIsCloudHost(t *testing.T) {
	for host, want := range map[string]bool{
		"acme.atlassian.net": true, "ACME.Atlassian.NET:443": true,
		"atlassian.net.evil.com": false, "confluence.acme.com": false,
	} {
		if got := IsCloudHost(host); got != want {
			t.Errorf("IsCloudHost(%q) = %v", host, got)
		}
	}
}
