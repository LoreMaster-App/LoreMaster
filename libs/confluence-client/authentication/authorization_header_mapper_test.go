package authentication

import (
	"fmt"
	"strings"
	"testing"
)

func TestAuthorizationHeader(t *testing.T) {
	cases := []struct {
		credential Credential
		want       string
	}{
		{APIToken{Email: "me@acme.com", Token: "t0k"}, "Basic bWVAYWNtZS5jb206dDBr"},
		{Basic{User: "me", Password: "pw"}, "Basic bWU6cHc="},
		{PAT{Token: "NjQ4MzQ"}, "Bearer NjQ4MzQ"},
		{OAuth{AccessToken: "NjQ4MzQ"}, "Bearer NjQ4MzQ"},
	}
	for _, tc := range cases {
		if got, err := AuthorizationHeader(tc.credential); err != nil || got != tc.want {
			t.Errorf("AuthorizationHeader(%s) = %q, %v; want %q", tc.credential, got, err, tc.want)
		}
	}
	for _, incomplete := range []Credential{APIToken{Email: "me@acme.com"}, APIToken{Token: "t"}, PAT{}, Basic{User: "me"}, OAuth{}, nil} {
		if _, err := AuthorizationHeader(incomplete); err == nil {
			t.Errorf("AuthorizationHeader(%v) should fail", incomplete)
		}
	}
}

func TestCredentialsNeverPrintTheirSecret(t *testing.T) {
	secret := "s3cr3t-value"
	for _, credential := range []Credential{APIToken{Email: "me@acme.com", Token: secret}, PAT{Token: secret}, Basic{User: "me", Password: secret}, OAuth{AccessToken: secret}} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
			if printed := fmt.Sprintf(format, credential); strings.Contains(printed, secret) {
				t.Errorf("%s of %T leaks the secret: %s", format, credential, printed)
			}
		}
		// A struct holding the credential prints it through String as well.
		if printed := fmt.Sprintf("%+v", struct{ C Credential }{credential}); strings.Contains(printed, secret) {
			t.Errorf("nested %T leaks the secret: %s", credential, printed)
		}
	}
	if got := (APIToken{Email: "me@acme.com", Token: "x"}).String(); got != "APIToken{Email: me@acme.com, Token: (redacted)}" {
		t.Errorf("String() = %q", got)
	}
	if got := (PAT{}).String(); got != "PAT{Token: (empty)}" {
		t.Errorf("String() = %q", got)
	}
}
