package clicommands

import (
	"strings"
	"testing"

	"lore-master/libs/documentation-sync/configcatalog"
)

// The variables the description names for Confluence sign-in.
const (
	envEmail    = "LORE_MASTER_EMAIL"
	envToken    = "LORE_MASTER_TOKEN"
	envPAT      = "LORE_MASTER_PAT"
	envUser     = "LORE_MASTER_USER"
	envPassword = "LORE_MASTER_PASSWORD"
)

func environment(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestCredentialFromTheEnvironmentPrefersAnApiTokenThenAPatThenBasic(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
		user string
	}{
		{"cloud", map[string]string{envEmail: " me@acme.com ", envToken: "tok", envPAT: "pat"}, "apitoken", "me@acme.com"},
		{"pat", map[string]string{envPAT: "pat", envUser: "u", envPassword: "p"}, "pat", ""},
		{"basic", map[string]string{envUser: "u", envPassword: "p"}, "basic", ""},
	}
	for _, tc := range cases {
		credential, err := credentialFromEnvironment(environment(tc.env))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if credential.Kind != tc.want || credential.Email != tc.user {
			t.Errorf("%s: %+v", tc.name, credential)
		}
	}
}

func TestCredentialFromTheEnvironmentReadsTheSecretsThrough(t *testing.T) {
	credential, _ := credentialFromEnvironment(environment(map[string]string{envEmail: "e", envToken: " secret "}))
	if credential.Token != "secret" {
		t.Fatalf("token %q", credential.Token)
	}
	basic, _ := credentialFromEnvironment(environment(map[string]string{envUser: "u", envPassword: "p"}))
	if basic.User != "u" || basic.Password != "p" {
		t.Fatalf("basic %+v", basic)
	}
}

func TestCredentialFromTheEnvironmentNamesTheVariablesWhenThereIsNone(t *testing.T) {
	for _, env := range []map[string]string{{}, {envEmail: "e"}, {envToken: "t"}, {envUser: "u"}} {
		_, err := credentialFromEnvironment(environment(env))
		if err == nil {
			t.Fatalf("%v: expected an error", env)
		}
		for _, name := range []string{envEmail, envToken, envPAT, envUser, envPassword} {
			if !strings.Contains(err.Error(), name) {
				t.Fatalf("the message does not name %s: %v", name, err)
			}
		}
	}
}

func TestTheCommandLineHelpNamesEveryVariableTheDescriptionDoes(t *testing.T) {
	_, connection := signIn()
	for _, kind := range connection.Kinds {
		for _, field := range kind.Fields {
			if !strings.Contains(usage, field.Env) {
				t.Errorf("the help does not mention %s (%s)", field.Env, kind.ID)
			}
		}
	}
}

func TestTheConstantsAboveAreTheDescriptionsVariables(t *testing.T) {
	names := map[string]bool{}
	for _, platform := range configcatalog.Platforms() {
		if platform.Connection == nil {
			continue
		}
		for _, kind := range platform.Connection.Kinds {
			for _, field := range kind.Fields {
				names[field.Env] = true
			}
		}
	}
	for _, name := range []string{envEmail, envToken, envPAT, envUser, envPassword} {
		if !names[name] {
			t.Errorf("%s is not a variable the description names", name)
		}
	}
}
