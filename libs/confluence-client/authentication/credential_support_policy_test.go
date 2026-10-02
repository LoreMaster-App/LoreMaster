package authentication

import (
	"testing"

	"lore-master/libs/confluence-client/connection"
)

func TestSupports(t *testing.T) {
	unknown := connection.Version{}
	v74 := connection.Version{Major: 7, Minor: 4}
	v79 := connection.Version{Major: 7, Minor: 9}
	v85 := connection.Version{Major: 8, Minor: 5, Patch: 3}
	token := APIToken{Email: "me@acme.com", Token: "t"}
	pat := PAT{Token: "p"}
	password := Basic{User: "me", Password: "pw"}

	cases := []struct {
		name       string
		edition    connection.Edition
		version    connection.Version
		credential Credential
		want       string
	}{
		{"cloud takes an API token", connection.Cloud, unknown, token, ""},
		{"cloud refuses a PAT", connection.Cloud, unknown, pat, "a pat credential does not work on Confluence Cloud, which accepts only an Atlassian account email with an API token (create one at https://id.atlassian.com/manage-profile/security/api-tokens)"},
		{"cloud refuses a password", connection.Cloud, unknown, password, "a basic credential does not work on Confluence Cloud, which accepts only an Atlassian account email with an API token (create one at https://id.atlassian.com/manage-profile/security/api-tokens)"},
		{"data center 8.5 takes a PAT", connection.DataCenter, v85, pat, ""},
		{"server 7.9 takes a PAT", connection.Server, v79, pat, ""},
		{"server 7.4 refuses a PAT", connection.Server, v74, pat, "personal access tokens need Confluence 7.9 or later and this site runs 7.4.0; sign in with a user name and password instead"},
		{"server 7.4 takes a password", connection.Server, v74, password, ""},
		{"data center 8.5 refuses a password", connection.DataCenter, v85, password, "this site runs Confluence 8.5.3, which supports personal access tokens; create one under Profile → Personal Access Tokens and use it instead of a password"},
		{"unknown version: PAT allowed", connection.DataCenter, unknown, pat, ""},
		{"unknown version: password allowed", connection.Server, unknown, password, ""},
		{"data center refuses an API token", connection.DataCenter, v85, token, "API tokens belong to Atlassian Cloud accounts; on Confluence Data Center use a personal access token"},
		{"server refuses an API token", connection.Server, v74, token, "API tokens belong to Atlassian Cloud accounts; on Confluence Server use a personal access token"},
		{"unknown edition", connection.Edition("x"), unknown, pat, `unknown Confluence edition "x"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Supports(tc.edition, tc.version, tc.credential)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
